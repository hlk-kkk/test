"""Output formatting for CLI (text/json/quiet modes)."""

import json
from enum import Enum
from pathlib import Path

from rich.console import Console
from rich.panel import Panel
from rich.progress import BarColumn, Progress, SpinnerColumn, TextColumn
from rich.table import Table

console = Console()
err_console = Console(stderr=True)


class OutputFormat(str, Enum):
    TEXT = "text"
    JSON = "json"
    QUIET = "quiet"


class OutputFormatter:
    """Format CLI output based on selected mode."""

    def __init__(self, fmt: OutputFormat = OutputFormat.TEXT):
        self.fmt = fmt

    def show_config(self, config: dict) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line({"event": "start", "config": config})
            return
        if self.fmt == OutputFormat.QUIET:
            return

        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        for key, value in config.items():
            table.add_row(f"{key}:", str(value))
        console.print(Panel(table, title="PicTactic Generation", border_style="cyan"))

    def show_round_start(self, round_num: int, max_rounds: int) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line({"event": "round_start", "round": round_num, "max_rounds": max_rounds})
            return
        if self.fmt == OutputFormat.QUIET:
            return
        console.print(f"\n[bold][Round {round_num}/{max_rounds}][/bold]")

    def show_image_ready(self, image: dict, round_num: int) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line(
                {
                    "event": "image_ready",
                    "round": round_num,
                    "image": {
                        "id": image.get("id", ""),
                        "url": image.get("url", ""),
                        "local_path": image.get("local_path"),
                        "score": image.get("score", 0.0),
                    },
                }
            )
            return
        if self.fmt == OutputFormat.QUIET:
            return
        score = image.get("score", 0.0)
        img_id = image.get("id", "?")
        console.print(f"  Image [dim]{img_id}[/dim] score: [green]{score:.2f}[/green]")

    def show_round_end(self, round_num: int, best_score: float) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line(
                {
                    "event": "round_end",
                    "round": round_num,
                    "best_score": best_score,
                }
            )
            return
        if self.fmt == OutputFormat.QUIET:
            return
        console.print(f"  Best score this round: [bold green]{best_score:.2f}[/bold green]")

    def show_status(self, status: str) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line({"event": "status", "status": status})
            return
        if self.fmt == OutputFormat.QUIET:
            return
        console.print(f"[dim]{status}[/dim]")

    def show_results(self, best_images: list[dict], total_images: int, output_dir: Path) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line(
                {
                    "event": "complete",
                    "total_images": total_images,
                    "best_score": best_images[0]["score"] if best_images else 0.0,
                    "best_images": [
                        {
                            "id": img.get("id"),
                            "local_path": img.get("local_path"),
                            "score": img.get("score", 0.0),
                        }
                        for img in best_images
                    ],
                }
            )
            return

        if self.fmt == OutputFormat.QUIET:
            for img in best_images:
                lp = img.get("local_path")
                if lp:
                    print(lp)
            return

        table = Table.grid(padding=(0, 2))
        table.add_column(style="bold cyan", justify="right")
        table.add_column()
        table.add_row("Total images:", str(total_images))
        table.add_row("Best score:", f"{best_images[0]['score']:.2f}" if best_images else "N/A")
        table.add_row("Output:", str(output_dir))
        table.add_row("", "")
        table.add_row("Best images:", "")
        for i, img in enumerate(best_images, 1):
            lp = img.get("local_path", img.get("url", "?"))
            table.add_row("", f"  {i}. {lp}  ({img.get('score', 0.0):.2f})")
        console.print(Panel(table, title="Results", border_style="green"))

    def show_error(self, message: str) -> None:
        if self.fmt == OutputFormat.JSON:
            self._json_line({"event": "error", "message": message})
            return
        err_console.print(f"[bold red]Error:[/bold red] {message}")

    def create_progress(self, description: str, total: int) -> Progress:
        return Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            BarColumn(),
            TextColumn("[progress.percentage]{task.percentage:>3.0f}%"),
            console=console,
        )

    def _json_line(self, data: dict) -> None:
        print(json.dumps(data, ensure_ascii=False), flush=True)
