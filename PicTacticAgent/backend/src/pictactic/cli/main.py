"""PicTactic CLI entry point."""

import typer

from .edit import edit_app
from .generate import generate_app
from .providers import providers_app
from .result import result_app
from .status import status_app
from .submit import submit_app
from .video import video_app

app = typer.Typer(
    name="pictactic",
    help="PicTactic - AI-driven image & video generation from the command line.",
    no_args_is_help=True,
)

# Register sub-commands
app.add_typer(generate_app, name="generate")
app.add_typer(edit_app, name="edit")
app.add_typer(video_app, name="video")
app.add_typer(providers_app, name="providers")
app.add_typer(status_app, name="status")
app.add_typer(submit_app, name="submit")
app.add_typer(result_app, name="result")


if __name__ == "__main__":
    app()
