"""Tests for CLI output formatting."""

import json

import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock

from pictactic.cli.output import OutputFormat, OutputFormatter


class TestOutputFormat:
    """Tests for OutputFormat enum."""

    def test_text_value(self):
        assert OutputFormat.TEXT == "text"

    def test_json_value(self):
        assert OutputFormat.JSON == "json"

    def test_quiet_value(self):
        assert OutputFormat.QUIET == "quiet"

    def test_is_string_enum(self):
        assert isinstance(OutputFormat.TEXT, str)


class TestOutputFormatterShowConfig:
    """Tests for OutputFormatter.show_config."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        config = {"Prompt": "test prompt", "Provider": "gemini"}
        formatter.show_config(config)
        output = capsys.readouterr().out
        data = json.loads(output)
        assert data["event"] == "start"
        assert data["config"]["Prompt"] == "test prompt"

    def test_quiet_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_config({"Prompt": "test"})
        assert capsys.readouterr().out == ""

    def test_text_format(self):
        """Text format uses rich console - just verify it doesn't crash."""
        formatter = OutputFormatter(OutputFormat.TEXT)
        formatter.show_config({"Prompt": "test", "Provider": "gemini"})


class TestOutputFormatterShowRoundStart:
    """Tests for OutputFormatter.show_round_start."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_round_start(1, 3)
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "round_start"
        assert data["round"] == 1
        assert data["max_rounds"] == 3

    def test_quiet_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_round_start(1, 3)
        assert capsys.readouterr().out == ""


class TestOutputFormatterShowImageReady:
    """Tests for OutputFormatter.show_image_ready."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        image = {
            "id": "img-1",
            "url": "https://example.com/img.png",
            "local_path": "/tmp/img.png",
            "score": 0.85,
        }
        formatter.show_image_ready(image, round_num=2)
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "image_ready"
        assert data["round"] == 2
        assert data["image"]["id"] == "img-1"
        assert data["image"]["score"] == 0.85

    def test_json_format_defaults(self, capsys):
        """Test JSON output with missing optional fields."""
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_image_ready({}, round_num=1)
        data = json.loads(capsys.readouterr().out)
        assert data["image"]["id"] == ""
        assert data["image"]["score"] == 0.0

    def test_quiet_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_image_ready({"id": "img-1", "score": 0.9}, round_num=1)
        assert capsys.readouterr().out == ""


class TestOutputFormatterShowRoundEnd:
    """Tests for OutputFormatter.show_round_end."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_round_end(2, 0.92)
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "round_end"
        assert data["round"] == 2
        assert data["best_score"] == 0.92

    def test_quiet_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_round_end(1, 0.8)
        assert capsys.readouterr().out == ""


class TestOutputFormatterShowStatus:
    """Tests for OutputFormatter.show_status."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_status("Generating images...")
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "status"
        assert data["status"] == "Generating images..."

    def test_quiet_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_status("Generating...")
        assert capsys.readouterr().out == ""


class TestOutputFormatterShowResults:
    """Tests for OutputFormatter.show_results."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        best_images = [
            {"id": "img-1", "local_path": "/tmp/img1.png", "score": 0.95},
            {"id": "img-2", "local_path": "/tmp/img2.png", "score": 0.88},
        ]
        formatter.show_results(best_images, total_images=10, output_dir=Path("./output"))
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "complete"
        assert data["total_images"] == 10
        assert data["best_score"] == 0.95
        assert len(data["best_images"]) == 2

    def test_json_format_empty_best(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_results([], total_images=5, output_dir=Path("./output"))
        data = json.loads(capsys.readouterr().out)
        assert data["best_score"] == 0.0
        assert data["best_images"] == []

    def test_quiet_format_prints_paths(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        best_images = [
            {"local_path": "/tmp/img1.png"},
            {"local_path": "/tmp/img2.png"},
        ]
        formatter.show_results(best_images, total_images=5, output_dir=Path("./output"))
        output = capsys.readouterr().out
        assert "/tmp/img1.png" in output
        assert "/tmp/img2.png" in output

    def test_quiet_format_skips_no_path(self, capsys):
        formatter = OutputFormatter(OutputFormat.QUIET)
        formatter.show_results([{"score": 0.9}], total_images=1, output_dir=Path("./output"))
        assert capsys.readouterr().out == ""

    def test_text_format(self):
        """Text format uses rich console - verify it doesn't crash."""
        formatter = OutputFormatter(OutputFormat.TEXT)
        best = [{"local_path": "/tmp/x.png", "score": 0.8}]
        formatter.show_results(best, total_images=3, output_dir=Path("./output"))


class TestOutputFormatterShowError:
    """Tests for OutputFormatter.show_error."""

    def test_json_format(self, capsys):
        formatter = OutputFormatter(OutputFormat.JSON)
        formatter.show_error("Something went wrong")
        data = json.loads(capsys.readouterr().out)
        assert data["event"] == "error"
        assert data["message"] == "Something went wrong"

    def test_text_format_uses_stderr(self):
        """Text error output goes to stderr via rich Console(stderr=True)."""
        formatter = OutputFormatter(OutputFormat.TEXT)
        # Just verify it doesn't crash
        formatter.show_error("Oops")


class TestOutputFormatterCreateProgress:
    """Tests for OutputFormatter.create_progress."""

    def test_returns_progress_object(self):
        formatter = OutputFormatter()
        progress = formatter.create_progress("Loading...", total=10)
        from rich.progress import Progress

        assert isinstance(progress, Progress)


class TestOutputFormatterDefault:
    """Tests for default OutputFormatter behavior."""

    def test_default_format_is_text(self):
        formatter = OutputFormatter()
        assert formatter.fmt == OutputFormat.TEXT

    def test_explicit_text_format(self):
        formatter = OutputFormatter(OutputFormat.TEXT)
        assert formatter.fmt == OutputFormat.TEXT
