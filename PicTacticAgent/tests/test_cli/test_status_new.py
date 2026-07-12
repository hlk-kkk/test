"""Tests for CLI status command (new async version)."""

import json
from unittest.mock import MagicMock, patch

import pytest
from typer.testing import CliRunner

from pictactic.cli.api_client import ApiError
from pictactic.cli.main import app

runner = CliRunner()

_TASK_PROGRESS = {
    "task_id": "task-xyz",
    "status": "generating",
    "current_round": 2,
    "max_rounds": 3,
    "images_generated": 8,
    "current_best_score": 0.85,
    "message": "Round 2 in progress",
    "original_prompt": "a cat in space",
    "enhanced_prompt": "a cute cat floating in outer space",
}


def _make_mock_response(data: dict, status_code: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status_code
    resp.json.return_value = data
    resp.raise_for_status = MagicMock()
    return resp


class TestCheckStatus:
    """Tests for status command."""

    @patch(
        "pictactic.cli.status.api_get",
        return_value=_make_mock_response(_TASK_PROGRESS),
    )
    def test_status_json(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "json", "task-xyz"])
        assert result.exit_code == 0
        data = json.loads(result.output.strip())
        assert data["task_id"] == "task-xyz"
        assert data["status"] == "generating"
        assert data["current_round"] == 2

    @patch(
        "pictactic.cli.status.api_get",
        return_value=_make_mock_response(_TASK_PROGRESS),
    )
    def test_status_quiet(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "quiet", "task-xyz"])
        assert result.exit_code == 0
        output = result.output.strip()
        # Format: status round/max images best_score
        assert "generating" in output
        assert "2/3" in output
        assert "8" in output
        assert "0.85" in output

    @patch(
        "pictactic.cli.status.api_get",
        return_value=_make_mock_response(_TASK_PROGRESS),
    )
    def test_status_text(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "text", "task-xyz"])
        assert result.exit_code == 0
        assert "Task task-xyz" in result.output

    @patch("pictactic.cli.status.api_get", side_effect=ApiError("connection refused"))
    def test_status_api_error(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "quiet", "task-xyz"])
        assert result.exit_code == 1

    def test_status_no_task_id(self):
        result = runner.invoke(app, ["status"])
        assert result.exit_code == 0
        # Should show usage hint
        assert "task_id" in result.output.lower() or "pictactic status" in result.output

    @patch(
        "pictactic.cli.status.api_get",
        return_value=_make_mock_response(
            {
                "task_id": "task-done",
                "status": "completed",
                "current_round": 3,
                "max_rounds": 3,
                "images_generated": 15,
                "current_best_score": 0.97,
                "message": "Done",
                "original_prompt": "sunset over mountains",
            }
        ),
    )
    def test_status_completed_task(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "quiet", "task-done"])
        assert result.exit_code == 0
        assert "completed" in result.output
        assert "0.97" in result.output

    @patch("pictactic.cli.status.api_get", side_effect=Exception("network error"))
    def test_status_generic_exception(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "text", "task-xyz"])
        assert result.exit_code == 1

    @patch(
        "pictactic.cli.status.api_get",
        return_value=_make_mock_response(
            {
                "task_id": "task-err",
                "status": "failed",
                "current_round": 1,
                "max_rounds": 3,
                "images_generated": 0,
                "current_best_score": 0.0,
                "message": "",
                "error": "Provider API key invalid",
            }
        ),
    )
    def test_status_text_shows_error_field(self, mock_get):
        result = runner.invoke(app, ["status", "--format", "text", "task-err"])
        assert result.exit_code == 0
        assert "Provider API key invalid" in result.output
