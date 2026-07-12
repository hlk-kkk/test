"""CLI edit command implementation."""

import asyncio
import base64
import logging
import shutil
from pathlib import Path
from uuid import uuid4

import httpx
import typer

from .output import OutputFormat, OutputFormatter

logger = logging.getLogger(__name__)

edit_app = typer.Typer(name="edit", help="Edit an existing image with a text prompt.")

_EDIT_ENHANCE_SYSTEM_PROMPT = """You are an expert at writing instructions for AI image editing.

Your goal is to transform user edit instructions into clear, specific directives that produce:
- Precise and targeted edits
- Consistent visual style with the original
- Professional and aesthetically pleasing results

Guidelines:
- Preserve the user's core intent exactly
- Add specific visual details (lighting direction, color temperature, texture details)
- Clarify ambiguous terms with concrete descriptors
- Keep instructions focused (max 100 words)

DO NOT:
- Change what the user asked to edit
- Add unrelated modifications
- Include explanations or meta-commentary
- Use quotation marks around the instruction"""


async def _load_and_normalize_sources(sources: list[str]) -> list[str]:
    """Validate source images and normalize them to JPEG data URIs.

    Local files are validated for existence before normalization.
    All sources (local paths, HTTP URLs, data URIs) are converted to
    JPEG data URIs with max_long_edge=2048.
    """
    from ..agents.tools._image_utils import normalize_reference_images

    for src in sources:
        if not src.startswith(("http://", "https://", "data:")):
            path = Path(src)
            if not path.exists():
                raise typer.BadParameter(f"Source image not found: {src}")

    return await normalize_reference_images(sources, max_long_edge=2048, keep_http_urls=False)


async def _enhance_edit_prompt(prompt: str) -> str:
    """Enhance an edit instruction using Vision LLM. Falls back to original on failure."""
    from langchain_core.messages import HumanMessage, SystemMessage

    from ..core.config import create_vision_llm

    try:
        llm = create_vision_llm(temperature=0.6)
        response = await llm.ainvoke(
            [
                SystemMessage(content=_EDIT_ENHANCE_SYSTEM_PROMPT),
                HumanMessage(
                    content=(
                        f"Enhance this image edit instruction:\n\n{prompt}"
                        "\n\nOutput ONLY the enhanced instruction, nothing else."
                    )
                ),
            ]
        )
        return response.content.strip().strip("\"'")
    except Exception as e:
        logger.warning("_enhance_edit_prompt failed, using original prompt: %s", e)
        return prompt


async def _save_edited_images(
    results: list,
    output_dir: Path,
) -> list[dict]:
    """Save edited GenerationResult objects to output_dir.

    Handles three cases in priority order:
    1. result.local_path exists on disk (Gemini provider saves locally first)
    2. result.url is a data URI (base64-encoded inline image)
    3. result.url is an HTTP/HTTPS URL (download with httpx)
    """
    output_dir.mkdir(parents=True, exist_ok=True)
    saved = []

    async with httpx.AsyncClient(timeout=60) as client:
        for i, r in enumerate(results):
            img_id = f"edit_{uuid4().hex[:8]}_{i}"
            filename = f"pictactic_{img_id}.png"
            filepath = output_dir / filename

            if r.local_path and Path(r.local_path).exists():
                shutil.copy2(r.local_path, filepath)
            elif r.url.startswith("data:"):
                _, encoded = r.url.split(",", 1)
                filepath.write_bytes(base64.b64decode(encoded))
            elif r.url.startswith(("http://", "https://")):
                try:
                    resp = await client.get(r.url)
                    resp.raise_for_status()
                    filepath.write_bytes(resp.content)
                except httpx.HTTPError:
                    logger.warning(
                        "Failed to download edited image from %s", r.url[:60]
                    )
                    saved.append({"id": img_id, "url": r.url, "score": 0.0})
                    continue
            else:
                logger.warning(
                    "Cannot save edited image %d: no local path or usable URL", i
                )
                saved.append({"id": img_id, "url": r.url, "score": 0.0})
                continue

            saved.append({"id": img_id, "url": r.url, "local_path": str(filepath), "score": 0.0})

    return saved


async def run_edit(
    prompt: str,
    raw_sources: list[str],
    count: int = 1,
    aspect_ratio: str = "16:9",
    size: str = "2K",
    provider_name: str = "gemini",
    enhance: bool = False,
    output_dir: Path = Path("./output"),
    formatter: OutputFormatter | None = None,
) -> dict:
    """Run the image edit workflow directly (no HTTP server needed)."""
    from ..providers import GenerationMode, GenerationParams, get_provider

    if formatter is None:
        formatter = OutputFormatter()

    formatter.show_config(
        {
            "Prompt": prompt,
            "Sources": f"{len(raw_sources)} image(s)",
            "Provider": provider_name,
            "Count": str(count),
            "Aspect Ratio": aspect_ratio,
            "Size": size,
            "Enhance": "Yes" if enhance else "No",
        }
    )

    # Load and normalize source images
    formatter.show_status("Loading and normalizing source images...")
    source_images = await _load_and_normalize_sources(raw_sources)

    # Validate provider supports IMAGE_TO_IMAGE
    provider = get_provider(provider_name)
    if GenerationMode.IMAGE_TO_IMAGE not in provider.supported_modes:
        raise typer.BadParameter(
            f"Provider '{provider_name}' does not support image editing (IMAGE_TO_IMAGE mode). "
            "Use 'gemini' or 'jiekouai' instead."
        )

    # Optionally enhance prompt
    active_prompt = prompt
    if enhance:
        formatter.show_status("Enhancing edit prompt...")
        active_prompt = await _enhance_edit_prompt(prompt)
        if active_prompt != prompt:
            formatter.show_status(f"Enhanced: {active_prompt[:80]}...")

    # Build params — reference_images triggers IMAGE_TO_IMAGE mode
    params = GenerationParams(
        prompt=active_prompt,
        reference_images=source_images,
        aspect_ratio=aspect_ratio,
        size=size,
    )

    # Generate
    formatter.show_status("Generating edited image(s)...")
    if count == 1:
        result = await provider.generate(params, index=0)
        results = [result]
    else:
        results = await provider.generate_batch_stream(params, count=count)

    # Save to disk
    saved = await _save_edited_images(results, output_dir)
    formatter.show_results(saved, len(saved), output_dir)

    return {"edited_images": saved, "total": len(saved)}


@edit_app.callback(invoke_without_command=True)
def edit(
    prompt: str = typer.Argument(..., help="Text instruction for image editing"),
    source: list[str] | None = typer.Option(
        None,
        "--source",
        "-s",
        help="Source image path(s) or URL(s) to edit (can specify multiple times)",
    ),
    count: int = typer.Option(1, "--count", "-n", help="Number of edited images to generate"),
    aspect_ratio: str = typer.Option("16:9", "--aspect-ratio", "-a", help="Image aspect ratio"),
    size: str = typer.Option("2K", "--size", help="Image size (1K, 2K, 4K)"),
    provider: str = typer.Option(
        None,
        "--provider",
        "-p",
        help="Image provider (gemini, jiekouai)",
    ),
    enhance: bool = typer.Option(False, "--enhance", help="Enable prompt enhancement"),
    output_dir: Path = typer.Option(
        Path("./output"), "--output-dir", "-o", help="Output directory"
    ),
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
) -> None:
    """Edit an existing image guided by a text instruction."""
    formatter = OutputFormatter(fmt)

    if not source:
        formatter.show_error(
            "At least one source image is required. Use --source/-s to specify."
        )
        raise typer.Exit(code=1)

    if provider is None:
        from ..core.config import settings

        provider = settings.default_provider

    try:
        result = asyncio.run(
            run_edit(
                prompt=prompt,
                raw_sources=list(source),
                count=count,
                aspect_ratio=aspect_ratio,
                size=size,
                provider_name=provider,
                enhance=enhance,
                output_dir=output_dir,
                formatter=formatter,
            )
        )
        if not result.get("edited_images"):
            raise typer.Exit(code=3)
    except typer.BadParameter as e:
        formatter.show_error(str(e))
        raise typer.Exit(code=1) from e
    except KeyboardInterrupt:
        formatter.show_error("Interrupted by user")
        raise typer.Exit(code=1) from None
    except typer.Exit:
        raise
    except Exception as e:
        formatter.show_error(f"Edit failed: {e}")
        raise typer.Exit(code=2) from e
