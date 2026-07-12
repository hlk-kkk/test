"""CLI status command implementation."""

import json

import typer
from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from .api_client import DEFAULT_API_URL, ApiError, api_get
from .output import OutputFormat

console = Console()
err_console = Console(stderr=True)

status_app = typer.Typer(name="status", help="Check task status (requires backend running).")


@status_app.callback(invoke_without_command=True)
def check_status(
    task_id: str | None = typer.Argument(None, help="Task ID to check (omit to list all)"),
    video: bool = typer.Option(False, "--video", "-v", help="Query a video task"),
    api_url: str = typer.Option(DEFAULT_API_URL, "--api-url", help="Backend API URL"),
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
) -> None:
    """Query task status from the running backend service."""
    if task_id:
        _show_task(task_id, api_url, fmt, is_video=video)
    else:
        _list_tasks(api_url, fmt)


def _show_task(task_id: str, api_url: str, fmt: OutputFormat, *, is_video: bool = False) -> None:
    """Show details of a single task."""
    prefix = "video" if is_video else "generation"
    url = f"{api_url}/api/v1/{prefix}/{task_id}"
    try:
        resp = api_get(url, timeout=10)
    except ApiError as e:
        _show_error(fmt, str(e))
        raise typer.Exit(code=1) from None
    except Exception:
        _show_error(fmt, "无法连接后端。后端是否在运行？")
        raise typer.Exit(code=1) from None

    data = resp.json()

    if fmt == OutputFormat.JSON:
        print(json.dumps(data, ensure_ascii=False))
        return

    if fmt == OutputFormat.QUIET:
        status = data.get("status", "unknown")
        current_round = data.get("current_round", 0)
        max_rounds = data.get("max_rounds", 0)
        if is_video:
            # Format: status round/max videos_count [progress%]
            videos = data.get("videos_generated", 0)
            progress = data.get("current_video_progress")
            if progress is not None:
                print(f"{status} {current_round}/{max_rounds} {videos} {progress}%")
            else:
                print(f"{status} {current_round}/{max_rounds} {videos}")
        else:
            # Format: status round/max images_count best_score
            images = data.get("images_generated", 0)
            best_score = data.get("current_best_score", 0.0)
            print(f"{status} {current_round}/{max_rounds} {images} {best_score:.2f}")
        return

    table = Table.grid(padding=(0, 2))
    table.add_column(style="bold cyan", justify="right")
    table.add_column()
    table.add_row("Task ID:", data.get("task_id", task_id))
    table.add_row("Status:", data.get("status", "unknown"))
    table.add_row("Round:", f"{data.get('current_round', 0)}/{data.get('max_rounds', 0)}")
    if is_video:
        table.add_row("Videos:", str(data.get("videos_generated", 0)))
    else:
        table.add_row("Images:", str(data.get("images_generated", 0)))
        table.add_row("Best Score:", f"{data.get('current_best_score', 0.0):.2f}")
    table.add_row("Message:", data.get("message", ""))

    prompt = data.get("original_prompt") or data.get("enhanced_prompt", "N/A")
    table.add_row("Prompt:", prompt[:80] + ("..." if len(prompt) > 80 else ""))

    if data.get("error"):
        table.add_row("Error:", data["error"])

    console.print(Panel(table, title=f"Task {task_id}", border_style="cyan"))


def _list_tasks(api_url: str, fmt: OutputFormat) -> None:
    """List all tasks."""
    _show_error(
        fmt,
        "请提供 task_id。用法: pictactic status <task_id>",
    )


def _show_error(fmt: OutputFormat, message: str) -> None:
    """Show error in the requested format."""
    if fmt == OutputFormat.JSON:
        print(json.dumps({"error": message}, ensure_ascii=False))
    else:
        err_console.print(f"[bold red]Error:[/bold red] {message}")
