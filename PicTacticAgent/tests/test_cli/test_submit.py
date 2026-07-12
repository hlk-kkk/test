"""Tests for CLI submit command (submit generate & submit edit)."""

import json
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from typer.testing import CliRunner

from pictactic.cli.api_client import ApiError
from pictactic.cli.main import app

runner = CliRunner()

# Common mock response for successful submission
_SUBMIT_RESP_DATA = {"task_id": "abc-123", "status": "pending", "message": "ok"}


def _make_mock_response(data: dict, status_code: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status_code
    resp.json.return_value = data
    resp.raise_for_status = MagicMock()
    return resp


class TestSubmitGenerate:
    """Tests for submit generate command."""

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_generate_quiet(self, mock_ensure, mock_post):
        result = runner.invoke(
            app, ["submit", "generate", "--format", "quiet", "test prompt"]
        )
        assert result.exit_code == 0
        assert "abc-123" in result.output.strip()

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_generate_json(self, mock_ensure, mock_post):
        result = runner.invoke(
            app, ["submit", "generate", "--format", "json", "test prompt"]
        )
        assert result.exit_code == 0
        data = json.loads(result.output.strip())
        assert data["task_id"] == "abc-123"
        assert data["status"] == "pending"

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_generate_text(self, mock_ensure, mock_post):
        result = runner.invoke(
            app, ["submit", "generate", "--format", "text", "test prompt"]
        )
        assert result.exit_code == 0
        assert "Task Submitted" in result.output

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_generate_with_template(self, mock_ensure, mock_post, tmp_path):
        img_file = tmp_path / "template.png"
        img_file.write_bytes(b"\x89PNG\r\n\x1a\n")

        result = runner.invoke(
            app,
            [
                "submit", "generate",
                "--template", str(img_file),
                "--format", "quiet",
                "test prompt",
            ],
        )
        assert result.exit_code == 0
        # Verify api_post was called with template_images
        call_kwargs = mock_post.call_args
        payload = call_kwargs[1].get("json") or call_kwargs.kwargs.get("json")
        assert payload is not None
        assert len(payload["template_images"]) > 0

    @patch("pictactic.cli.submit.ensure_backend_running")
    @patch("pictactic.cli.submit.api_post", side_effect=ApiError("connection refused"))
    def test_submit_generate_api_error(self, mock_post, mock_ensure):
        result = runner.invoke(
            app, ["submit", "generate", "--format", "quiet", "test prompt"]
        )
        assert result.exit_code == 2

    @patch(
        "pictactic.cli.submit.ensure_backend_running",
        side_effect=ApiError("backend unreachable"),
    )
    def test_submit_generate_backend_not_running(self, mock_ensure):
        result = runner.invoke(
            app, ["submit", "generate", "--format", "quiet", "test prompt"]
        )
        assert result.exit_code == 2


class TestSubmitEdit:
    """Tests for submit edit command."""

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_edit_quiet(self, mock_ensure, mock_post, tmp_path):
        img_file = tmp_path / "source.png"
        img_file.write_bytes(b"\x89PNG\r\n\x1a\n")

        result = runner.invoke(
            app,
            [
                "submit", "edit",
                "--source", str(img_file),
                "--format", "quiet",
                "make it brighter",
            ],
        )
        assert result.exit_code == 0
        assert "abc-123" in result.output.strip()

    def test_submit_edit_no_source(self):
        result = runner.invoke(
            app, ["submit", "edit", "--format", "quiet", "make it brighter"]
        )
        assert result.exit_code == 1

    def test_submit_edit_source_not_found(self):
        result = runner.invoke(
            app,
            [
                "submit", "edit",
                "--source", "/nonexistent/path/image.png",
                "--format", "quiet",
                "make it brighter",
            ],
        )
        assert result.exit_code == 1

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_edit_json(self, mock_ensure, mock_post, tmp_path):
        img_file = tmp_path / "source.png"
        img_file.write_bytes(b"\x89PNG\r\n\x1a\n")

        result = runner.invoke(
            app,
            [
                "submit", "edit",
                "--source", str(img_file),
                "--format", "json",
                "make it brighter",
            ],
        )
        assert result.exit_code == 0
        data = json.loads(result.output.strip())
        assert data["task_id"] == "abc-123"

    @patch("pictactic.cli.submit.api_post", return_value=_make_mock_response(_SUBMIT_RESP_DATA))
    @patch("pictactic.cli.submit.ensure_backend_running")
    def test_submit_edit_with_url_source(self, mock_ensure, mock_post):
        result = runner.invoke(
            app,
            [
                "submit", "edit",
                "--source", "https://example.com/image.png",
                "--format", "quiet",
                "make it brighter",
            ],
        )
        assert result.exit_code == 0
        # Verify url was passed directly without conversion
        call_kwargs = mock_post.call_args
        payload = call_kwargs[1].get("json") or call_kwargs.kwargs.get("json")
        assert "https://example.com/image.png" in payload["source_images"]
