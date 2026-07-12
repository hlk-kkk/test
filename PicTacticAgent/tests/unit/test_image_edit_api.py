"""Tests for image edit API endpoint."""

from datetime import datetime
import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from fastapi import FastAPI
from httpx import AsyncClient, ASGITransport

from pictactic.api.dependencies import get_current_user_or_api_key
from pictactic.api.routes.generation import router
from pictactic.core.exceptions import PicTacticError, TaskNotFoundError
from pictactic.models.enums import GenerationStatus, ImageProvider
from pictactic.models.schemas import (
    GenerationResult,
    GeneratedImage,
    ImageEditResult,
)
from pictactic.services.auth_service import User, UserStatus


def _create_test_app():
    """Create test FastAPI application with generation router."""
    app = FastAPI()
    app.include_router(router, prefix="/api/v1/generation")
    return app


def _mock_user():
    """Create a mock authenticated user."""
    return User(
        user_id="user-123",
        email="test@example.com",
        display_name="Test User",
        status=UserStatus.ACTIVE,
    )


def _mock_edit_result(
    original_image_id: str = "img-1",
    enhanced: bool = False,
) -> ImageEditResult:
    """Create a mock ImageEditResult."""
    new_image = GeneratedImage(
        id="new-img-1",
        url="https://example.com/edited.png",
        local_path="/tmp/edited.png",
        prompt="edited prompt",
        provider=ImageProvider.GEMINI,
        round_number=1,
        score=0.0,
        metadata={"edited_from": original_image_id, "edit_prompt": "Edit this"},
    )
    return ImageEditResult(
        original_image_id=original_image_id,
        new_image=new_image,
        edit_prompt="Edit this",
        enhanced_edit_prompt="Enhanced edit prompt" if enhanced else None,
    )


class TestEditImageEndpoint:
    """Tests for POST /api/v1/generation/{task_id}/images/{image_id}/edit."""

    @pytest.mark.asyncio
    async def test_edit_image_success(self):
        """Test successful image edit via API."""
        app = _create_test_app()
        mock_result = _mock_edit_result()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(return_value=mock_result)
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/generation/task-1/images/img-1/edit",
                    json={
                        "edit_prompt": "Make the background blue",
                        "enhance_prompt": False,
                    },
                )

                assert response.status_code == 200
                data = response.json()
                assert data["original_image_id"] == "img-1"
                assert data["new_image"]["url"] == "https://example.com/edited.png"
                assert data["edit_prompt"] == "Edit this"

        app.dependency_overrides.clear()


class TestTaskStatusEndpoint:
    """Tests for GET /api/v1/generation/{task_id} status fallback behavior."""

    @pytest.mark.asyncio
    async def test_get_task_status_falls_back_to_db_result_when_memory_missed(self):
        """get_task_status now uses _ensure_task_loaded which does DB fallback internally.

        The route simply awaits service.get_task_status(); if the task is only in DB,
        the service restores it and returns a TaskProgress.
        """
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        from pictactic.models.schemas import TaskProgress

        # Simulate the service restoring a completed task from DB via _ensure_task_loaded
        restored_progress = TaskProgress(
            task_id="task-1",
            status=GenerationStatus.COMPLETED,
            current_round=2,
            max_rounds=2,
            images_per_round=5,
            aspect_ratio="16:9",
            size="2K",
            template_images=[],
            images_generated=10,
            images_in_current_round=0,
            current_best_score=0.0,
            quality_threshold=0.7,
            message="Completed",
            original_prompt="Create a cover",
            enhanced_prompt="Enhanced cover prompt",
            recent_images=[],
            error=None,
        )

        with patch("pictactic.api.routes.generation.get_generation_service") as mock_get_service:
            mock_service = MagicMock()
            mock_service.get_task_status = AsyncMock(return_value=restored_progress)
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/generation/task-1")

                assert response.status_code == 200
                data = response.json()
                assert data["task_id"] == "task-1"
                assert data["status"] == "completed"
                assert data["images_generated"] == 10
                assert data["current_round"] == 2

        app.dependency_overrides.clear()


    @pytest.mark.asyncio
    async def test_edit_image_with_enhancement(self):
        """Test image edit with prompt enhancement enabled."""
        app = _create_test_app()
        mock_result = _mock_edit_result(enhanced=True)
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(return_value=mock_result)
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/generation/task-1/images/img-1/edit",
                    json={
                        "edit_prompt": "Make the background blue",
                        "enhance_prompt": True,
                    },
                )

                assert response.status_code == 200
                data = response.json()
                assert data["enhanced_edit_prompt"] == "Enhanced edit prompt"

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_task_not_found(self):
        """Test edit returns 404 for non-existent task."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(
                side_effect=TaskNotFoundError("nonexistent-task")
            )
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/generation/nonexistent-task/images/img-1/edit",
                    json={"edit_prompt": "Edit this"},
                )

                assert response.status_code == 404

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_bad_request(self):
        """Test edit returns 400 for PicTacticError (e.g., image not found)."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(
                side_effect=PicTacticError("Image not found: img-999")
            )
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/generation/task-1/images/img-999/edit",
                    json={"edit_prompt": "Edit this"},
                )

                assert response.status_code == 400
                assert "Image not found" in response.json()["detail"]

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_validation_error(self):
        """Test edit returns 422 for invalid request body."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        transport = ASGITransport(app=app)
        async with AsyncClient(
            transport=transport, base_url="http://test"
        ) as client:
            # Missing required edit_prompt
            response = await client.post(
                "/api/v1/generation/task-1/images/img-1/edit",
                json={},
            )

            assert response.status_code == 422

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_empty_prompt(self):
        """Test edit returns 422 for empty edit_prompt."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        transport = ASGITransport(app=app)
        async with AsyncClient(
            transport=transport, base_url="http://test"
        ) as client:
            response = await client.post(
                "/api/v1/generation/task-1/images/img-1/edit",
                json={"edit_prompt": ""},
            )

            assert response.status_code == 422

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_server_error(self):
        """Test edit returns 500 for unexpected errors."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(
                side_effect=RuntimeError("Unexpected error")
            )
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/generation/task-1/images/img-1/edit",
                    json={"edit_prompt": "Edit this"},
                )

                assert response.status_code == 500

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_edit_image_passes_user_id(self):
        """Test that the endpoint passes the authenticated user's ID to the service."""
        app = _create_test_app()
        mock_result = _mock_edit_result()
        mock_user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: mock_user

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.edit_image = AsyncMock(return_value=mock_result)
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                await client.post(
                    "/api/v1/generation/task-1/images/img-1/edit",
                    json={"edit_prompt": "Edit this"},
                )

                # Verify user_id was passed to service
                mock_service.edit_image.assert_awaited_once()
                call_kwargs = mock_service.edit_image.call_args[1]
                assert call_kwargs.get("user_id") == "user-123"

        app.dependency_overrides.clear()
