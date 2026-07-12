"""CLI video command — generate videos from text or image prompts."""

import asyncio
import json
import logging
from pathlib import Path
from uuid import uuid4

import typer
from rich.console import Console

from .generate import load_template_images
from .output import OutputFormat

logger = logging.getLogger(__name__)
console = Console()
err_console = Console(stderr=True)

video_app = typer.Typer(name="video", help="Generate videos from text or image prompts.")


async def save_videos(videos: list[dict], output_dir: Path) -> list[dict]:
    """Ensure generated videos are saved to *output_dir*.

    Videos that already have a ``local_path`` are copied/symlinked;
    remote URLs are downloaded.
    """
    import shutil

    import httpx

    output_dir.mkdir(parents=True, exist_ok=True)
    updated: list[dict] = []

    async with httpx.AsyncClient(timeout=120) as client:
        for vid in videos:
            vid_id = vid.get("id", str(uuid4())[:8])
            local = vid.get("local_path")
            url = vid.get("url", "")

            if local and Path(local).exists():
                dest = output_dir / Path(local).name
                if dest != Path(local):
                    shutil.copy2(local, dest)
                vid["local_path"] = str(dest)
            elif url.startswith(("http://", "https://")):
                dest = output_dir / f"video_{vid_id}.mp4"
                try:
                    resp = await client.get(url)
                    resp.raise_for_status()
                    dest.write_bytes(resp.content)
                    vid["local_path"] = str(dest)
                except (httpx.HTTPError, OSError):
                    logger.warning("Failed to download video %s", vid_id, exc_info=True)

            updated.append(vid)
    return updated


async def run_video_generation(
    prompt: str,
    start_image: str | None = None,
    end_image: str | None = None,
    reference_images: list[str] | None = None,
    max_rounds: int = 2,
    videos_per_round: int = 2,
    duration: int = 5,
    resolution: str | None = None,
    aspect_ratio: str = "16:9",
    generate_audio: bool = False,
    provider: str = "kling",
    enhance_prompt: bool = True,
    negative_prompt: str | None = None,
    cfg_scale: float | None = None,
    professional: bool | None = None,
    seed: int | None = None,
    sample_count: int = 1,
    output_dir: Path = Path("./output"),
    fmt: OutputFormat = OutputFormat.TEXT,
) -> dict:
    """Run the video generation workflow directly (no HTTP needed)."""
    from ..agents.video_state import create_initial_video_state
    from ..agents.video_workflow import create_video_generation_workflow

    _show_config(
        fmt,
        {
            "Prompt": prompt,
            "Provider": provider,
            "Rounds": f"{max_rounds} x {videos_per_round} videos",
            "Duration": f"{duration}s",
            "Aspect Ratio": aspect_ratio,
            "Audio": "Yes" if generate_audio else "No",
            "Enhance": "Yes" if enhance_prompt else "No",
        },
    )

    workflow = create_video_generation_workflow()
    initial_state = create_initial_video_state(
        user_prompt=prompt,
        start_image=start_image,
        end_image=end_image,
        reference_images=reference_images,
        max_rounds=max_rounds,
        videos_per_round=videos_per_round,
        duration=duration,
        resolution=resolution,
        aspect_ratio=aspect_ratio,
        generate_audio=generate_audio,
        provider=provider,
        enhance_prompt_enabled=enhance_prompt,
        negative_prompt=negative_prompt,
        cfg_scale=cfg_scale,
        professional=professional,
        seed=seed,
        sample_count=sample_count,
    )

    config = {"configurable": {"thread_id": f"cli-video-{uuid4()}"}}
    current_round = 0
    all_videos: list[dict] = []

    async for event in workflow.astream(initial_state, config=config, stream_mode="custom"):
        if not isinstance(event, dict):
            continue

        event_type = event.get("type", "")

        if event_type == "status_change":
            new_round = event.get("current_round", current_round)
            if new_round > current_round:
                current_round = new_round
                _show_round(fmt, current_round, max_rounds)
            _show_status(fmt, event.get("status", ""))

        elif event_type == "video_ready":
            video_info = event.get("video", {})
            all_videos.append(video_info)
            _show_video_ready(fmt, video_info, current_round)

        elif event_type == "video_progress":
            progress = event.get("progress_percent", 0)
            _show_progress(fmt, progress, event.get("eta", 0))

    # Get final state
    final_snapshot = await workflow.aget_state(config)
    final_state = final_snapshot.values if final_snapshot else {}
    final_videos = final_state.get("generated_videos", all_videos)

    # Save to output_dir
    saved = await save_videos(final_videos, output_dir)
    _show_results(fmt, saved, output_dir)

    return {"videos": saved, "total_videos": len(saved), "final_state": final_state}


# ---- Output helpers ----


def _show_config(fmt: OutputFormat, cfg: dict) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"type": "config", **cfg}, ensure_ascii=False))
    elif fmt == OutputFormat.TEXT:
        from rich.panel import Panel
        from rich.table import Table

        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        for k, v in cfg.items():
            table.add_row(f"{k}:", str(v))
        console.print(Panel(table, title="Video Generation", border_style="blue"))


def _show_round(fmt: OutputFormat, round_num: int, max_rounds: int) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"type": "round", "round": round_num, "max": max_rounds}))
    elif fmt == OutputFormat.TEXT:
        console.print(f"\n[bold cyan]━━ Round {round_num}/{max_rounds} ━━[/bold cyan]")


def _show_status(fmt: OutputFormat, status: str) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"type": "status", "status": status}))
    elif fmt == OutputFormat.TEXT:
        console.print(f"  [dim]Status:[/dim] {status}")


def _show_video_ready(fmt: OutputFormat, video: dict, round_num: int) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"type": "video_ready", "round": round_num, **video}, ensure_ascii=False))
    elif fmt == OutputFormat.TEXT:
        vid_id = video.get("id", "?")[:8]
        provider = video.get("provider", "?")
        console.print(f"  [green]✓[/green] Video {vid_id} ready ({provider})")


def _show_progress(fmt: OutputFormat, percent: int, eta: int) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"type": "progress", "percent": percent, "eta": eta}))


def _show_results(fmt: OutputFormat, videos: list[dict], output_dir: Path) -> None:
    if fmt == OutputFormat.QUIET:
        for v in videos:
            lp = v.get("local_path")
            if lp:
                print(lp)
    elif fmt == OutputFormat.JSON:
        print(
            json.dumps(
                {
                    "type": "result",
                    "total_videos": len(videos),
                    "videos": [
                        {"id": v.get("id"), "local_path": v.get("local_path")} for v in videos
                    ],
                },
                ensure_ascii=False,
            )
        )
    else:
        from rich.panel import Panel
        from rich.table import Table

        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        table.add_row("Videos:", str(len(videos)))
        table.add_row("Output:", str(output_dir))
        for i, v in enumerate(videos, 1):
            table.add_row(f"  #{i}:", v.get("local_path", "N/A"))
        console.print(Panel(table, title="Results", border_style="green"))


def _show_error(fmt: OutputFormat, message: str) -> None:
    if fmt == OutputFormat.JSON:
        print(json.dumps({"error": message}, ensure_ascii=False))
    else:
        err_console.print(f"[bold red]Error:[/bold red] {message}")


# ---- CLI entry point ----


@video_app.callback(invoke_without_command=True)
def video(
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
    cfg_scale: float | None = typer.Option(None, "--cfg-scale", help="Kling CFG scale (0-1)"),
    professional: bool | None = typer.Option(None, "--professional", help="Sora professional mode"),
    seed: int | None = typer.Option(None, "--seed", help="Random seed"),
    sample_count: int = typer.Option(1, "--sample-count", help="Veo samples per call (1-4)"),
    output_dir: Path = typer.Option(
        Path("./output"), "--output-dir", "-o", help="Output directory"
    ),
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
) -> None:
    """Generate videos from a text/image prompt using the PicTactic workflow."""
    if provider is None:
        from ..core.config import settings

        provider = settings.default_video_provider

    # Convert local image paths to base64 data URIs
    start_data = _resolve_image(start_image, fmt)
    end_data = _resolve_image(end_image, fmt)

    try:
        result = asyncio.run(
            run_video_generation(
                prompt=prompt,
                start_image=start_data,
                end_image=end_data,
                max_rounds=rounds,
                videos_per_round=videos_per_round,
                duration=duration,
                resolution=resolution,
                aspect_ratio=aspect_ratio,
                generate_audio=audio,
                provider=provider,
                enhance_prompt=not no_enhance,
                negative_prompt=negative_prompt,
                cfg_scale=cfg_scale,
                professional=professional,
                seed=seed,
                sample_count=sample_count,
                output_dir=output_dir,
                fmt=fmt,
            )
        )
        if not result.get("videos"):
            raise typer.Exit(code=3)
    except KeyboardInterrupt:
        _show_error(fmt, "Interrupted by user")
        raise typer.Exit(code=1) from None
    except typer.Exit:
        raise
    except Exception as e:
        _show_error(fmt, f"Video generation failed: {e}")
        raise typer.Exit(code=2) from e


def _resolve_image(image_path: str | None, fmt: OutputFormat) -> str | None:
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
