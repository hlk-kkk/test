"""Tests for CLI result command — download completed task results."""

import json
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from typer.testing import CliRunner

from pictactic.cli.api_client import ApiError
from pictactic.cli.main import app

runner = CliRunner()

_COMPLETED_DATA = {
    "status": "completed",
    "best_images": [
        {"id": "img-001", "url": "https://example.com/1.png", "score": 0.95},
        {"id": "img-002", "url": "https://example.com/2.png", "score": 0.88},
    ],
}

_SAVED_IMAGES = [
    {"id": "img-001", "local_path": "/tmp/output/pictactic_img-001.png", "score": 0.95},
    {"id": "img-002", "local_path": "/tmp/output/pictactic_img-002.png", "score": 0.88},
]


def _make_mock_response(data: dict, status_code: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status_code
    resp.json.return_value = data
    resp.raise_for_status = MagicMock()
    return resp


class TestGetResult:
    """Tests for result command."""

    @patch("pictactic.cli.result.save_images", new_callable=AsyncMock, return_value=_SAVED_IMAGES)
    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response(_COMPLETED_DATA),
    )
    def test_result_quiet(self, mock_get, mock_save):
        result = runner.invoke(app, ["result", "--format", "quiet", "task-abc"])
        assert result.exit_code == 0
        lines = result.output.strip().splitlines()
        assert "/tmp/output/pictactic_img-001.png" in lines
        assert "/tmp/output/pictactic_img-002.png" in lines

    @patch("pictactic.cli.result.save_images", new_callable=AsyncMock, return_value=_SAVED_IMAGES)
    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response(_COMPLETED_DATA),
    )
    def test_result_json(self, mock_get, mock_save):
        result = runner.invoke(app, ["result", "--format", "json", "task-abc"])
        assert result.exit_code == 0
        data = json.loads(result.output.strip())
        assert data["task_id"] == "task-abc"
        assert data["status"] == "completed"
        assert len(data["images"]) == 2
        assert data["images"][0]["score"] == 0.95

    @patch("pictactic.cli.result.save_images", new_callable=AsyncMock, return_value=_SAVED_IMAGES)
    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response(_COMPLETED_DATA),
    )
    def test_result_text(self, mock_get, mock_save):
        result = runner.invoke(app, ["result", "--format", "text", "task-abc"])
        assert result.exit_code == 0
        assert "Results Downloaded" in result.output

    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response({"status": "generating"}),
    )
    def test_result_not_completed(self, mock_get):
        result = runner.invoke(app, ["result", "--format", "quiet", "task-abc"])
        assert result.exit_code == 3

    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response({"status": "completed", "best_images": []}),
    )
    def test_result_no_images(self, mock_get):
        result = runner.invoke(app, ["result", "--format", "quiet", "task-abc"])
        assert result.exit_code == 3

    @patch("pictactic.cli.result.api_get", side_effect=ApiError("server down"))
    def test_result_api_error(self, mock_get):
        result = runner.invoke(app, ["result", "--format", "quiet", "task-abc"])
        assert result.exit_code == 1

    @patch(
        "pictactic.cli.result.api_get",
        return_value=_make_mock_response(
            {"status": "failed", "error": "provider timeout"}
        ),
    )
    def test_result_failed_with_error_message(self, mock_get):
        result = runner.invoke(app, ["result", "--format", "json", "task-abc"])
        assert result.exit_code == 3
        data = json.loads(result.output.strip())
        assert "error" in data
