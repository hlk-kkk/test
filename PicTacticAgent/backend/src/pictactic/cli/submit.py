"""CLI submit command — submit async generation/edit tasks to the backend API."""

import json
from pathlib import Path

import typer
from rich.console import Console
from rich.panel import Panel

from .api_client import DEFAULT_API_URL, ApiError, api_post, ensure_backend_running
from .generate import load_template_images
from .output import OutputFormat

console = Console()
err_console = Console(stderr=True)

submit_app = typer.Typer(
    name="submit",
    help="Submit async tasks to the backend (returns task_id immediately).",
)


@submit_app.command("generate")
def submit_generate(
    prompt: str = typer.Argument(..., help="Text prompt for image generation"),
    template: list[Path] | None = typer.Option(
        None, "--template", "-t", help="Template image path(s)"
    ),
    rounds: int = typer.Option(3, "--rounds", "-r", help="Maximum generation rounds"),
    images: int = typer.Option(5, "--images", "-n", help="Images per round"),
    top_k: int = typer.Option(3, "--top-k", "-k", help="Top-scored images to output"),
    threshold: float = typer.Option(0.7, "--threshold", "-q", help="Quality threshold"),
    aspect_ratio: str = typer.Option("16:9", "--aspect-ratio", "-a", help="Aspect ratio"),
    size: str = typer.Option("2K", "--size", "-s", help="Image size"),
    provider: str = typer.Option(None, "--provider", "-p", help="Image provider"),
    no_enhance: bool = typer.Option(False, "--no-enhance", help="Disable prompt enhancement"),
    no_ai_scoring: bool = typer.Option(False, "--no-ai-scoring", help="Disable AI scoring"),
    fmt: OutputFormat = typer.Option(OutputFormat.QUIET, "--format", "-f", help="Output format"),
    api_url: str = typer.Option(DEFAULT_API_URL, "--api-url", help="Backend API URL"),
) -> None:
    """Submit an image generation task to the backend (async)."""
    if provider is None:
        from ..core.config import settings

        provider = settings.default_provider

    # Load template images to base64
    template_images = []
    if template:
        try:
            template_images = load_template_images(template)
        except typer.BadParameter as e:
            _show_error(fmt, str(e))
            raise typer.Exit(code=1) from e

    # Ensure backend is running
    try:
        ensure_backend_running(api_url)
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e

    # Build request payload
    payload = {
        "prompt": prompt,
        "template_images": template_images,
        "max_rounds": rounds,
        "images_per_round": images,
        "quality_threshold": threshold,
        "enable_ai_scoring": not no_ai_scoring,
        "aspect_ratio": aspect_ratio,
        "size": size,
        "enhance_prompt": not no_enhance,
        "provider": provider,
    }

    try:
        resp = api_post(f"{api_url}/api/v1/generation/", json=payload)
        data = resp.json()
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e
    except Exception as e:
        _show_error(fmt, f"提交失败: {e}")
        raise typer.Exit(code=2) from e

    task_id = data.get("task_id", "")
    _show_submit_result(fmt, task_id, data.get("status", "pending"), data.get("message", ""))


@submit_app.command("edit")
def submit_edit(
    prompt: str = typer.Argument(..., help="Edit instruction prompt"),
    source: list[str] | None = typer.Option(
        None, "--source", "-s", help="Source image path(s) or URL(s)"
    ),
    count: int = typer.Option(1, "--count", "-n", help="Number of outputs"),
    aspect_ratio: str = typer.Option("16:9", "--aspect-ratio", "-a", help="Aspect ratio"),
    size: str = typer.Option("2K", "--size", help="Image size"),
    provider: str = typer.Option(None, "--provider", "-p", help="Image provider"),
    enhance: bool = typer.Option(False, "--enhance", help="Enable prompt enhancement"),
    fmt: OutputFormat = typer.Option(OutputFormat.QUIET, "--format", "-f", help="Output format"),
    api_url: str = typer.Option(DEFAULT_API_URL, "--api-url", help="Backend API URL"),
) -> None:
    """Submit an image edit task to the backend (async)."""
    if not source:
        _show_error(fmt, "至少需要一个源图片。使用 --source/-s 指定。")
        raise typer.Exit(code=1)

    if provider is None:
        from ..core.config import settings

        provider = settings.default_provider

    # Convert local file paths to base64 data URIs
    source_images = []
    for src in source:
        if not src.startswith(("http://", "https://", "data:")):
            path = Path(src)
            if not path.exists():
                _show_error(fmt, f"源图片不存在: {src}")
                raise typer.Exit(code=1)
            source_images.extend(load_template_images([path]))
        else:
            source_images.append(src)

    # Ensure backend is running
    try:
        ensure_backend_running(api_url)
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e

    # Build request payload
    payload = {
        "prompt": prompt,
        "source_images": source_images,
        "count": count,
        "aspect_ratio": aspect_ratio,
        "size": size,
        "provider": provider,
        "enhance_prompt": enhance,
    }

    try:
        resp = api_post(f"{api_url}/api/v1/generation/edit", json=payload)
        data = resp.json()
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e
    except Exception as e:
        _show_error(fmt, f"提交失败: {e}")
        raise typer.Exit(code=2) from e

    task_id = data.get("task_id", "")
    _show_submit_result(fmt, task_id, data.get("status", "pending"), data.get("message", ""))


@submit_app.command("video")
def submit_video(
    prompt: str = typer.Argument(..., help="Text prompt for video generation"),
    start_image: str | None = typer.Option(
        None, "--start-image", "-i", help="Start frame image path or URL"
    ),
    end_image: str | None = typer.Option(
        None, "--end-image", "-e", help="End frame image path or URL"
    ),
    rounds: int = typer.Option(2, "--rounds", "-r", help="Generation rounds (1-5)"),
    videos_per_round: int = typer.Option(2, "--videos", "-n", help="Videos per round (1-4)"),
    duration: int = typer.Option(5, "--duration", "-d", help="Duration in seconds"),
    resolution: str | None = typer.Option(None, "--resolution", help="Resolution (720p, 1080p)"),
    aspect_ratio: str = typer.Option("16:9", "--aspect-ratio", "-a", help="Aspect ratio"),
    audio: bool = typer.Option(False, "--audio", help="Generate audio"),
    provider: str | None = typer.Option(
        None, "--provider", "-p", help="Video provider (kling, sora, veo)"
    ),
    no_enhance: bool = typer.Option(False, "--no-enhance", help="Disable prompt enhancement"),
    negative_prompt: str | None = typer.Option(None, "--negative-prompt", help="Negative prompt"),
    seed: int | None = typer.Option(None, "--seed", help="Random seed"),
    sample_count: int = typer.Option(1, "--sample-count", help="Veo samples per call (1-4)"),
    fmt: OutputFormat = typer.Option(OutputFormat.QUIET, "--format", "-f", help="Output format"),
    api_url: str = typer.Option(DEFAULT_API_URL, "--api-url", help="Backend API URL"),
) -> None:
    """Submit a video generation task to the backend (async)."""
    if provider is None:
        from ..core.config import settings

        provider = settings.default_video_provider

    # Resolve local images to base64 data URIs
    start_data = _resolve_image_for_submit(start_image, fmt)
    end_data = _resolve_image_for_submit(end_image, fmt)

    # Ensure backend is running
    try:
        ensure_backend_running(api_url)
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e

    payload = {
        "prompt": prompt,
        "start_image": start_data,
        "end_image": end_data,
        "max_rounds": rounds,
        "videos_per_round": videos_per_round,
        "duration": duration,
        "resolution": resolution,
        "aspect_ratio": aspect_ratio,
        "generate_audio": audio,
        "provider": provider,
        "enhance_prompt": not no_enhance,
        "negative_prompt": negative_prompt,
        "seed": seed,
        "sample_count": sample_count,
    }

    try:
        resp = api_post(f"{api_url}/api/v1/video/", json=payload)
        data = resp.json()
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=2) from e
    except Exception as e:
        _show_error(fmt, f"提交失败: {e}")
        raise typer.Exit(code=2) from e

    task_id = data.get("task_id", "")
    _show_submit_result(fmt, task_id, data.get("status", "pending"), data.get("message", ""))


def _resolve_image_for_submit(image_path: str | None, fmt: OutputFormat) -> str | None:
    """Convert a local file path to base64 data URI, or pass through URLs."""
    if not image_path:
        return None
    if image_path.startswith(("http://", "https://", "data:")):
        return image_path
    path = Path(image_path)
    if not path.exists():
        _show_error(fmt, f"Image not found: {image_path}")
        raise typer.Exit(code=1)
    images = load_template_images([path])
    return images[0] if images else None


def _show_submit_result(fmt: OutputFormat, task_id: str, status: str, message: str) -> None:
    """Output the submit result in the requested format."""
    if fmt == OutputFormat.QUIET:
        print(task_id)
    elif fmt == OutputFormat.JSON:
        print(
            json.dumps(
                {"task_id": task_id, "status": status, "message": message},
                ensure_ascii=False,
            )
        )
    else:
        from rich.table import Table

        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        table.add_row("Task ID:", task_id)
        table.add_row("Status:", status)
        table.add_row("Message:", message)
        console.print(Panel(table, title="Task Submitted", border_style="green"))


def _show_error(fmt: OutputFormat, message: str) -> None:
    """Show error in the requested format."""
    if fmt == OutputFormat.JSON:
        print(json.dumps({"error": message}, ensure_ascii=False))
    else:
        err_console.print(f"[bold red]Error:[/bold red] {message}")
