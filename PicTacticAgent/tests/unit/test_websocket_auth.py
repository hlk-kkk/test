"""Unit tests for WebSocket auth and ownership checks."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient
from starlette.websockets import WebSocketDisconnect

from pictactic.api.app import app
from pictactic.models.enums import GenerationStatus
from pictactic.models.schemas import TaskProgress


def _progress(task_id: str, status: GenerationStatus = GenerationStatus.COMPLETED) -> TaskProgress:
    return TaskProgress(
        task_id=task_id,
        status=status,
        current_round=1,
        max_rounds=1,
        images_per_round=1,
        aspect_ratio="16:9",
        size="1K",
        template_images=[],
        images_generated=1,
        images_in_current_round=1,
        current_best_score=0.9,
        quality_threshold=0.5,
        message="ok",
        original_prompt="test",
        enhanced_prompt=None,
        recent_images=[],
        error=None,
    )


def test_ws_requires_token():
    client = TestClient(app)
    with pytest.raises((WebSocketDisconnect, RuntimeError)):
        with client.websocket_connect("/ws/progress/task-123"):
            pass


def test_ws_rejects_invalid_token():
    client = TestClient(app)
    with patch(
        "pictactic.api.websocket.auth_service.verify_access_token",
        return_value=None,
    ):
        with pytest.raises((WebSocketDisconnect, RuntimeError)):
            with client.websocket_connect("/ws/progress/task-123?token=bad"):
                pass


def test_ws_rejects_wrong_owner():
    client = TestClient(app)
    mock_service = MagicMock()
    mock_service.get_task_owner = AsyncMock(return_value="user-2")

    with patch(
        "pictactic.api.websocket.auth_service.verify_access_token",
        return_value="user-1",
    ):
        with patch(
            "pictactic.api.websocket.get_generation_service",
            return_value=mock_service,
        ):
            with pytest.raises((WebSocketDisconnect, RuntimeError)):
                with client.websocket_connect("/ws/progress/task-123?token=good"):
                    pass


def test_ws_allows_owner_and_streams_progress():
    client = TestClient(app)
    mock_service = MagicMock()
    mock_service.get_task_owner = AsyncMock(return_value="user-1")
    mock_service.get_task_status = AsyncMock(return_value=_progress("task-123"))

    with patch(
        "pictactic.api.websocket.auth_service.verify_access_token",
        return_value="user-1",
    ):
        with patch(
            "pictactic.api.websocket.get_generation_service",
            return_value=mock_service,
        ):
            with client.websocket_connect("/ws/progress/task-123?token=good") as ws:
                data = ws.receive_json()
                assert data["task_id"] == "task-123"
                assert data["status"] == GenerationStatus.COMPLETED
