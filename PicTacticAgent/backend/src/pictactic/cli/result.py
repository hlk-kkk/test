"""CLI result command — download completed task results."""

import asyncio
import json
from pathlib import Path

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from .api_client import DEFAULT_API_URL, ApiError, api_get
from .generate import save_images
from .output import OutputFormat

console = Console()
err_console = Console(stderr=True)

result_app = typer.Typer(name="result", help="Download results of a completed task.")


@result_app.callback(invoke_without_command=True)
def get_result(
    task_id: str = typer.Argument(..., help="Task ID to get results for"),
    video: bool = typer.Option(False, "--video", "-v", help="Download video task results"),
    output_dir: Path = typer.Option(
        Path("./output"), "--output-dir", "-o", help="Output directory"
    ),
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
    api_url: str = typer.Option(DEFAULT_API_URL, "--api-url", help="Backend API URL"),
) -> None:
    """Download results of a completed generation/edit/video task."""
    prefix = "video" if video else "generation"
    # Fetch task result from API
    try:
        resp = api_get(f"{api_url}/api/v1/{prefix}/{task_id}/result")
        data = resp.json()
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=1) from e
    except Exception as e:
        _show_error(fmt, f"获取结果失败: {e}")
        raise typer.Exit(code=1) from e

    status = data.get("status", "")
    if status != "completed":
        msg = f"任务未完成 (status={status})"
        if data.get("error"):
            msg += f": {data['error']}"
        _show_error(fmt, msg)
        raise typer.Exit(code=3)

    if video:
        _download_video_results(task_id, data, output_dir, fmt)
    else:
        _download_image_results(task_id, data, output_dir, fmt)


def _download_image_results(task_id: str, data: dict, output_dir: Path, fmt: OutputFormat) -> None:
    """Download image task results."""
    best_images = data.get("best_images", [])
    if not best_images:
        _show_error(fmt, "任务已完成但无可用图片")
        raise typer.Exit(code=3)

    try:
        saved = asyncio.run(save_images(best_images, output_dir))
    except Exception as e:
        _show_error(fmt, f"图片下载失败: {e}")
        raise typer.Exit(code=2) from e

    if fmt == OutputFormat.QUIET:
        for img in saved:
            lp = img.get("local_path")
            if lp:
                print(lp)
    elif fmt == OutputFormat.JSON:
        print(
            json.dumps(
                {
                    "task_id": task_id,
                    "status": "completed",
                    "images": [
                        {
                            "id": img.get("id"),
                            "local_path": img.get("local_path"),
                            "score": img.get("score", 0.0),
                        }
                        for img in saved
                    ],
                },
                ensure_ascii=False,
            )
        )
    else:
        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        table.add_row("Task ID:", task_id)
        table.add_row("Images:", str(len(saved)))
        table.add_row("Output:", str(output_dir))
        table.add_row("", "")
        for i, img in enumerate(saved, 1):
            lp = img.get("local_path", "N/A")
            table.add_row(f"  #{i}:", f"{lp}  ({img.get('score', 0.0):.2f})")
        console.print(Panel(table, title="Results Downloaded", border_style="green"))


def _download_video_results(task_id: str, data: dict, output_dir: Path, fmt: OutputFormat) -> None:
    """Download video task results."""
    from .video import save_videos

    videos = data.get("videos", [])
    if not videos:
        _show_error(fmt, "任务已完成但无可用视频")
        raise typer.Exit(code=3)

    try:
        saved = asyncio.run(save_videos(videos, output_dir))
    except Exception as e:
        _show_error(fmt, f"视频下载失败: {e}")
        raise typer.Exit(code=2) from e

    if fmt == OutputFormat.QUIET:
        for v in saved:
            lp = v.get("local_path")
            if lp:
                print(lp)
    elif fmt == OutputFormat.JSON:
        print(
            json.dumps(
                {
                    "task_id": task_id,
                    "status": "completed",
                    "videos": [
                        {"id": v.get("id"), "local_path": v.get("local_path")} for v in saved
                    ],
                },
                ensure_ascii=False,
            )
        )
    else:
        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        table.add_row("Task ID:", task_id)
        table.add_row("Videos:", str(len(saved)))
        table.add_row("Output:", str(output_dir))
        table.add_row("", "")
        for i, v in enumerate(saved, 1):
            table.add_row(f"  #{i}:", v.get("local_path", "N/A"))
        console.print(Panel(table, title="Video Results Downloaded", border_style="green"))


def _show_error(fmt: OutputFormat, message: str) -> None:
    """Show error in the requested format."""
    if fmt == OutputFormat.JSON:
        print(json.dumps({"error": message}, ensure_ascii=False))
    else:
        err_console.print(f"[bold red]Error:[/bold red] {message}")
