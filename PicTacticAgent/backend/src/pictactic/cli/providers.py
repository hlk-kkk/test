"""CLI providers command implementation."""

import json

import typer
from rich.console import Console
from rich.table import Table

from ..core.config import settings
from ..providers import _PROVIDERS, get_provider_capabilities
from .output import OutputFormat

console = Console()

providers_app = typer.Typer(name="providers", help="List available image generation providers.")


@providers_app.callback(invoke_without_command=True)
def list_providers(
    fmt: OutputFormat = typer.Option(OutputFormat.TEXT, "--format", "-f", help="Output format"),
) -> None:
    """List available image generation providers and their capabilities."""
    capabilities = []
    for name in _PROVIDERS:
        cap = get_provider_capabilities(name)
        cap["is_default"] = name == settings.default_provider
        capabilities.append(cap)

    if fmt == OutputFormat.JSON:
        print(json.dumps(capabilities, ensure_ascii=False))
        return

    table = Table(title="Available Providers")
    table.add_column("Provider", style="bold")
    table.add_column("Modes")
    table.add_column("Batch")
    table.add_column("Default")

    for cap in capabilities:
        table.add_row(
            cap["name"],
            ", ".join(cap["supported_modes"]),
            "Yes" if cap.get("supports_batch") else "No",
            "*" if cap.get("is_default") else "",
        )

    console.print(table)
