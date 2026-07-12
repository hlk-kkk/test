"""CLI generate command implementation."""

import asyncio
import base64
import logging
from pathlib import Path
from uuid import uuid4

import httpx
import typer

from .output import OutputFormat, OutputFormatter

logger = logging.getLogger(__name__)

generate_app = typer.Typer(name="generate", help="Generate images from text prompts.")


def load_template_images(paths: list[Path]) -> list[str]:
    """Convert local image files to base64 data URIs."""
    result = []
    for path in paths:
        if not path.exists():
            raise typer.BadParameter(f"Template image not found: {path}")
        data = path.read_bytes()
        suffix = path.suffix.lower().lstrip(".")
        mime = {
            "png": "image/png",
            "jpg": "image/jpeg",
            "jpeg": "image/jpeg",
            "webp": "image/webp",
        }.get(suffix, "image/png")
        encoded = base64.b64encode(data).decode()
        result.append(f"data:{mime};base64,{encoded}")
    return result


async def save_images(images: list[dict], output_dir: Path) -> list[dict]:
    """Download generated images to local directory, updating local_path in-place."""
    output_dir.mkdir(parents=True, exist_ok=True)
    updated = []
    async with httpx.AsyncClient(timeout=60) as client:
        for img in images:
            url = img.get("url", "")
            img_id = img.get("id", str(uuid4())[:8])
            filename = f"pictactic_{img_id}.png"
            filepath = output_dir / filename

            try:
                if url.startswith("data:"):
                    # base64 data URI - decode and save
                    _, encoded = url.split(",", 1)
                    filepath.write_bytes(base64.b64decode(encoded))
                elif url.startswith(("http://", "https://")):
                    resp = await client.get(url)
                    resp.raise_for_status()
                    filepath.write_bytes(resp.content)
                else:
                    logger.warning("Unknown URL scheme for image %s: %s", img_id, url[:50])
                    updated.append(img)
                    continue

                img["local_path"] = str(filepath)
            except (httpx.HTTPError, OSError, ValueError):
                logger.warning("Failed to save image %s", img_id, exc_info=True)

            updated.append(img)
    return updated


async def run_generation(
    prompt: str,
    template_images: list[str] | None = None,
    max_rounds: int = 3,
    images_per_round: int = 5,
    top_k: int = 3,
    quality_threshold: float = 0.7,
    aspect_ratio: str = "16:9",
    size: str = "2K",
    provider: str = "gemini",
    enhance_prompt: bool = True,
    enable_ai_scoring: bool = True,
    output_dir: Path = Path("./output"),
    formatter: OutputFormatter | None = None,
) -> dict:
    """Run the generation workflow directly (no HTTP server needed)."""
    from ..agents.state import create_initial_state
    from ..agents.workflow import create_generation_workflow

    if formatter is None:
        formatter = OutputFormatter()

    formatter.show_config(
        {
            "Prompt": prompt,
            "Provider": provider,
            "Rounds": f"{max_rounds} x {images_per_round} images",
            "Output Top-K": str(top_k),
            "Threshold": f"{quality_threshold:.2f}",
            "Aspect Ratio": aspect_ratio,
            "Size": size,
            "Enhance": "Yes" if enhance_prompt else "No",
            "AI Scoring": "Yes" if enable_ai_scoring else "No",
        }
    )

    workflow = create_generation_workflow()
    initial_state = create_initial_state(
        user_prompt=prompt,
        template_images=template_images or [],
        max_rounds=max_rounds,
        images_per_round=images_per_round,
        top_k=top_k,
        quality_threshold=quality_threshold,
        aspect_ratio=aspect_ratio,
        image_size=size,
        provider=provider,
        enhance_prompt_enabled=enhance_prompt,
        enable_ai_scoring=enable_ai_scoring,
    )

    config = {"configurable": {"thread_id": f"cli-{uuid4()}"}}
    current_round = 0
    all_images: list[dict] = []

    async for event in workflow.astream(initial_state, config=config, stream_mode="custom"):
        if not isinstance(event, dict):
            continue

        event_type = event.get("type", "")

        if event_type == "status_change":
            status = event.get("status", "")
            new_round = event.get("round", current_round)
            if new_round > current_round:
                current_round = new_round
                formatter.show_round_start(current_round, max_rounds)
            formatter.show_status(status)

        elif event_type == "image_ready":
            image_info = event.get("image", {})
            round_num = image_info.get("round_number", current_round)
            if round_num > current_round:
                current_round = round_num
                formatter.show_round_start(current_round, max_rounds)
            formatter.show_image_ready(image_info, current_round)
            all_images.append(image_info)

    # Get final state
    final_snapshot = await workflow.aget_state(config)
    final_state = final_snapshot.values if final_snapshot else {}
    best_images = final_state.get("best_images", [])

    # Save images to output dir
    images_to_save = best_images if best_images else all_images
    saved_images = await save_images(images_to_save, output_dir)

    formatter.show_results(saved_images, len(all_images), output_dir)

    return {
        "best_images": saved_images,
        "total_images": len(all_images),
        "final_state": final_state,
    }


@generate_app.callback(invoke_without_command=True)
def generate(
    prompt: str = typer.Argument(..., help="Text prompt for image generation"),
    template: list[Path] | None = typer.Option(
        None, "--template", "-t", help="Template image path(s)"
    ),
    rounds: int = typer.Option(3, "--rounds", "-r", help="Maximum generation rounds"),
    images: int = typer.Option(5, "--images", "-n", help="Images per round (candidate pool size)"),
    top_k: int = typer.Option(3, "--top-k", "-k", help="Number of top-scored images to output"),
    threshold: float = typer.Option(0.7, "--threshold", "-q", help="Quality threshold (0.0-1.0)"),
    aspect_ratio: str = typer.Option("16:9", "--aspect-ratio", "-a", help="Image aspect ratio"),
    size: str = typer.Option("2K", "--size", "-s", help="Image size (1K, 2K, 4K)"),
    provider: str = typer.Option(
        None, "--provider", "-p", help="Image provider (gemini, jiekouai, antigravity)"
    ),
    no_enhance: bool = typer.Option(False, "--no-enhance", help="Disable prompt enhancement"),
    no_ai_scoring: bool = typer.Option(False, "--no-ai-scoring", help="Disable AI scoring"),
    output_dir: Path = typer.Option(
        Path("./output"), "--output-dir", "-o", help="Output directory"
    ),
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
) -> None:
    """Generate images from a text prompt using the PicTactic workflow."""
    formatter = OutputFormatter(fmt)

    # Load provider from settings if not specified
    if provider is None:
        from ..core.config import settings

        provider = settings.default_provider

    # Load template images
    template_images = []
    if template:
        try:
            template_images = load_template_images(template)
        except typer.BadParameter as e:
            formatter.show_error(str(e))
            raise typer.Exit(code=1) from e

    try:
        result = asyncio.run(
            run_generation(
                prompt=prompt,
                template_images=template_images,
                max_rounds=rounds,
                images_per_round=images,
                top_k=top_k,
                quality_threshold=threshold,
                aspect_ratio=aspect_ratio,
                size=size,
                provider=provider,
                enhance_prompt=not no_enhance,
                enable_ai_scoring=not no_ai_scoring,
                output_dir=output_dir,
                formatter=formatter,
            )
        )
        if not result.get("best_images"):
            raise typer.Exit(code=3)
    except KeyboardInterrupt:
        formatter.show_error("Interrupted by user")
        raise typer.Exit(code=1) from None
    except typer.Exit:
        raise
    except Exception as e:
        formatter.show_error(f"Generation failed: {e}")
        raise typer.Exit(code=2) from e
