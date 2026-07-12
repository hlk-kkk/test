"""Tests for async API: auth dependencies, standalone edit endpoint, schemas, and service."""

import uuid
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi import FastAPI, HTTPException
from httpx import ASGITransport, AsyncClient
from starlette.requests import Request

from pictactic.api.dependencies import (
    effective_user_id,
    get_current_user_or_api_key,
)
from pictactic.api.routes.generation import router
from pictactic.models.enums import GenerationStatus, ImageProvider
from pictactic.models.schemas import StandaloneEditRequest
from pictactic.services.auth_service import User, UserStatus


# ========== Helpers ==========


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


def _api_key_sentinel():
    """Create the API-key sentinel user."""
    return User(
        user_id="__api_key__",
        email="internal@api.local",
        status=UserStatus.ACTIVE,
    )


def _make_request(headers: dict | None = None) -> Request:
    """Build a minimal Starlette Request with the given headers."""
    scope = {
        "type": "http",
        "method": "GET",
        "path": "/",
        "headers": [(k.lower().encode(), v.encode()) for k, v in (headers or {}).items()],
    }
    return Request(scope)


# ========== 1. Auth Dependency Tests ==========


class TestEffectiveUserId:
    """Tests for effective_user_id helper."""

    def test_normal_user(self):
        user = User(user_id="user-123", email="a@b.com", status=UserStatus.ACTIVE)
        assert effective_user_id(user) == "user-123"

    def test_api_key_sentinel(self):
        user = _api_key_sentinel()
        assert effective_user_id(user) is None


class TestGetCurrentUserOrApiKey:
    """Tests for get_current_user_or_api_key dependency."""

    @pytest.mark.asyncio
    async def test_jwt_auth_success(self):
        expected_user = _mock_user()

        with (
            patch("pictactic.api.dependencies.auth_service") as mock_auth,
        ):
            mock_auth.verify_access_token.return_value = "user-123"
            mock_auth.get_user_by_id = AsyncMock(return_value=expected_user)

            request = _make_request()
            result = await get_current_user_or_api_key(request, token="valid-jwt")

            assert result.user_id == "user-123"
            mock_auth.verify_access_token.assert_called_once_with("valid-jwt")
            mock_auth.get_user_by_id.assert_awaited_once_with("user-123")

    @pytest.mark.asyncio
    async def test_api_key_auth_success(self):
        with patch("pictactic.api.dependencies.auth_service") as mock_auth:
            # JWT path not triggered (no token)
            mock_auth.verify_access_token.return_value = None

            with patch("pictactic.core.config.settings") as mock_settings:
                mock_settings.internal_api_key = "secret-key-123"

                request = _make_request({"X-API-Key": "secret-key-123"})
                result = await get_current_user_or_api_key(request, token=None)

                assert result.user_id == "__api_key__"
                assert result.email == "internal@api.local"

    @pytest.mark.asyncio
    async def test_no_auth_401(self):
        with patch("pictactic.api.dependencies.auth_service") as mock_auth:
            mock_auth.verify_access_token.return_value = None

            request = _make_request()
            with pytest.raises(HTTPException) as exc_info:
                await get_current_user_or_api_key(request, token=None)

            assert exc_info.value.status_code == 401

    @pytest.mark.asyncio
    async def test_wrong_api_key_401(self):
        with patch("pictactic.api.dependencies.auth_service") as mock_auth:
            mock_auth.verify_access_token.return_value = None

            with patch("pictactic.core.config.settings") as mock_settings:
                mock_settings.internal_api_key = "correct-key"

                request = _make_request({"X-API-Key": "wrong-key"})
                with pytest.raises(HTTPException) as exc_info:
                    await get_current_user_or_api_key(request, token=None)

                assert exc_info.value.status_code == 401

    @pytest.mark.asyncio
    async def test_jwt_user_disabled_falls_through(self):
        """Disabled JWT user should fall through; if no API key → 401."""
        disabled_user = User(
            user_id="user-disabled",
            email="d@b.com",
            status=UserStatus.DISABLED,
        )
        with patch("pictactic.api.dependencies.auth_service") as mock_auth:
            mock_auth.verify_access_token.return_value = "user-disabled"
            mock_auth.get_user_by_id = AsyncMock(return_value=disabled_user)

            request = _make_request()
            with pytest.raises(HTTPException) as exc_info:
                await get_current_user_or_api_key(request, token="some-jwt")

            assert exc_info.value.status_code == 401


# ========== 2. Standalone Edit Endpoint Tests ==========


class TestStandaloneEditEndpoint:
    """Tests for POST /api/v1/generation/edit."""

    @pytest.mark.asyncio
    async def test_create_edit_task_success(self):
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.create_edit_task = AsyncMock(return_value="task-id-1")
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "Make the sky purple",
                        "source_images": ["data:image/png;base64,abc123"],
                        "count": 1,
                        "aspect_ratio": "16:9",
                        "size": "2K",
                        "provider": "gemini",
                        "enhance_prompt": False,
                    },
                )

            assert response.status_code == 200
            data = response.json()
            assert data["task_id"] == "task-id-1"
            assert data["status"] == "pending"
            assert "message" in data

            mock_service.create_edit_task.assert_awaited_once()

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_create_edit_task_missing_source(self):
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.post(
                "/api/v1/generation/edit",
                json={
                    "prompt": "Edit this image",
                    # source_images missing
                },
            )

        assert response.status_code == 422
        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_create_edit_task_empty_prompt(self):
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.post(
                "/api/v1/generation/edit",
                json={
                    "prompt": "",
                    "source_images": ["data:image/png;base64,abc"],
                },
            )

        assert response.status_code == 422
        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_create_edit_task_passes_user_id(self):
        """Verify that effective_user_id(user) is passed to service."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.create_edit_task = AsyncMock(return_value="task-2")
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "Change colors",
                        "source_images": ["data:image/png;base64,abc"],
                        "provider": "gemini",
                    },
                )

            call_kwargs = mock_service.create_edit_task.call_args
            assert call_kwargs[1].get("user_id") == "user-123"

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_create_edit_task_api_key_user_id_is_none(self):
        """API-key sentinel should pass user_id=None to service."""
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _api_key_sentinel()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.create_edit_task = AsyncMock(return_value="task-3")
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "Edit it",
                        "source_images": ["data:image/png;base64,abc"],
                        "provider": "gemini",
                    },
                )

            call_kwargs = mock_service.create_edit_task.call_args
            assert call_kwargs[1].get("user_id") is None

        app.dependency_overrides.clear()

    @pytest.mark.asyncio
    async def test_create_edit_task_service_error_returns_400(self):
        app = _create_test_app()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: _mock_user()

        with patch(
            "pictactic.api.routes.generation.get_generation_service"
        ) as mock_get_service:
            mock_service = MagicMock()
            mock_service.create_edit_task = AsyncMock(
                side_effect=ValueError("Invalid source image")
            )
            mock_get_service.return_value = mock_service

            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "Edit it",
                        "source_images": ["data:image/png;base64,abc"],
                        "provider": "gemini",
                    },
                )

            assert response.status_code == 400

        app.dependency_overrides.clear()


# ========== 3. Schema Tests ==========


class TestStandaloneEditRequestSchema:
    """Tests for StandaloneEditRequest pydantic model."""

    def test_valid_request(self):
        req = StandaloneEditRequest(
            prompt="Make background blue",
            source_images=["data:image/png;base64,abc123"],
            count=2,
            aspect_ratio="1:1",
            size="1K",
            provider=ImageProvider.GEMINI,
            enhance_prompt=True,
        )
        assert req.prompt == "Make background blue"
        assert len(req.source_images) == 1
        assert req.count == 2
        assert req.provider == ImageProvider.GEMINI

    def test_default_provider_from_settings(self):
        with patch("pictactic.core.config.settings") as mock_settings:
            mock_settings.default_provider = "jiekouai"

            req = StandaloneEditRequest(
                prompt="Edit this",
                source_images=["data:image/png;base64,abc"],
            )
            assert req.provider == "jiekouai"

    def test_min_prompt_length(self):
        from pydantic import ValidationError

        with pytest.raises(ValidationError):
            StandaloneEditRequest(
                prompt="",
                source_images=["data:image/png;base64,abc"],
            )

    def test_min_source_images_required(self):
        from pydantic import ValidationError

        with pytest.raises(ValidationError):
            StandaloneEditRequest(
                prompt="Edit this",
                source_images=[],
            )

    def test_count_bounds(self):
        from pydantic import ValidationError

        with pytest.raises(ValidationError):
            StandaloneEditRequest(
                prompt="Edit",
                source_images=["data:image/png;base64,abc"],
                count=0,
                provider=ImageProvider.GEMINI,
            )

        with pytest.raises(ValidationError):
            StandaloneEditRequest(
                prompt="Edit",
                source_images=["data:image/png;base64,abc"],
                count=11,
                provider=ImageProvider.GEMINI,
            )

    def test_defaults(self):
        with patch("pictactic.core.config.settings") as mock_settings:
            mock_settings.default_provider = "gemini"

            req = StandaloneEditRequest(
                prompt="Edit this",
                source_images=["data:image/png;base64,abc"],
            )
            assert req.count == 1
            assert req.aspect_ratio == "16:9"
            assert req.size == "2K"
            assert req.enhance_prompt is False


# ========== 4. Service Layer Tests ==========


class TestCreateEditTask:
    """Tests for GenerationService.create_edit_task."""

    def _make_service(self):
        """Create a GenerationService with mocked dependencies."""
        with (
            patch("pictactic.services.generation_service.create_generation_workflow"),
            patch("pictactic.services.generation_service.TaskRepository") as MockRepo,
        ):
            mock_repo_instance = MagicMock()
            mock_repo_instance.create_task = AsyncMock()
            mock_repo_instance.save_result = AsyncMock()
            mock_repo_instance.update_status = AsyncMock()
            MockRepo.return_value = mock_repo_instance

            from pictactic.services.generation_service import GenerationService

            service = GenerationService()
            return service, mock_repo_instance

    @pytest.mark.asyncio
    async def test_creates_task_in_memory(self):
        service, _ = self._make_service()
        req = StandaloneEditRequest(
            prompt="Edit this image",
            source_images=["data:image/png;base64,abc"],
            provider=ImageProvider.GEMINI,
        )

        task_id = await service.create_edit_task(req, user_id="user-123")

        assert task_id in service._tasks
        task_info = service._tasks[task_id]
        assert task_info["status"] == GenerationStatus.PENDING
        assert task_info["user_id"] == "user-123"
        assert task_info["edit_request"] is req

    @pytest.mark.asyncio
    async def test_returns_uuid_format(self):
        service, _ = self._make_service()
        req = StandaloneEditRequest(
            prompt="Edit it",
            source_images=["data:image/png;base64,abc"],
            provider=ImageProvider.GEMINI,
        )

        task_id = await service.create_edit_task(req)

        # Should be valid UUID
        parsed = uuid.UUID(task_id)
        assert str(parsed) == task_id

    @pytest.mark.asyncio
    async def test_persists_to_db(self):
        service, mock_repo = self._make_service()
        req = StandaloneEditRequest(
            prompt="Edit the sky",
            source_images=["data:image/png;base64,abc"],
            provider=ImageProvider.GEMINI,
        )

        task_id = await service.create_edit_task(req, user_id="user-456")

        mock_repo.create_task.assert_awaited_once()
        call_kwargs = mock_repo.create_task.call_args[1]
        assert call_kwargs["task_id"] == task_id
        assert call_kwargs["original_prompt"] == "Edit the sky"
        assert call_kwargs["user_id"] == "user-456"
        assert call_kwargs["config"]["mode"] == "edit"

    @pytest.mark.asyncio
    async def test_db_failure_does_not_raise(self):
        """DB persistence failure should be swallowed (logged warning)."""
        service, mock_repo = self._make_service()
        mock_repo.create_task = AsyncMock(side_effect=RuntimeError("DB down"))

        req = StandaloneEditRequest(
            prompt="Edit",
            source_images=["data:image/png;base64,abc"],
            provider=ImageProvider.GEMINI,
        )

        # Should not raise
        task_id = await service.create_edit_task(req)
        assert task_id in service._tasks


class TestRunEditTask:
    """Tests for GenerationService.run_edit_task."""

    def _make_service_with_task(self, provider_name="gemini"):
        """Create service with a pre-populated edit task."""
        with (
            patch("pictactic.services.generation_service.create_generation_workflow"),
            patch("pictactic.services.generation_service.TaskRepository") as MockRepo,
        ):
            mock_repo_instance = MagicMock()
            mock_repo_instance.create_task = AsyncMock()
            mock_repo_instance.save_result = AsyncMock()
            mock_repo_instance.update_status = AsyncMock()
            MockRepo.return_value = mock_repo_instance

            from pictactic.services.generation_service import GenerationService

            service = GenerationService()

        edit_req = StandaloneEditRequest(
            prompt="Make sky blue",
            source_images=["data:image/png;base64,abc123"],
            count=1,
            provider=ImageProvider(provider_name),
        )
        gen_req = MagicMock()
        gen_req.prompt = edit_req.prompt

        task_id = "test-task-id"
        from datetime import datetime

        service._tasks[task_id] = {
            "id": task_id,
            "user_id": None,
            "status": GenerationStatus.PENDING,
            "request": gen_req,
            "edit_request": edit_req,
            "state": None,
            "created_at": datetime.utcnow(),
            "updated_at": datetime.utcnow(),
            "result": None,
            "error": None,
        }

        return service, mock_repo_instance, task_id

    @pytest.mark.asyncio
    async def test_successful_edit(self):
        service, mock_repo, task_id = self._make_service_with_task("gemini")

        mock_result = MagicMock()
        mock_result.url = "https://example.com/edited.png"
        mock_result.local_path = "/tmp/edited.png"
        mock_result.metadata = {}

        mock_provider = MagicMock()
        mock_provider.supported_modes = [MagicMock(value="image_to_image")]
        mock_provider.generate = AsyncMock(return_value=mock_result)

        from pictactic.providers.base import GenerationMode

        mock_provider.supported_modes = [GenerationMode.IMAGE_TO_IMAGE]

        with (
            patch(
                "pictactic.agents.tools._image_utils.normalize_reference_images",
                new_callable=AsyncMock,
                return_value=["data:image/png;base64,normalized"],
            ),
            patch(
                "pictactic.providers.get_provider",
                return_value=mock_provider,
            ),
        ):
            await service.run_edit_task(task_id)

        assert service._tasks[task_id]["status"] == GenerationStatus.COMPLETED
        assert service._tasks[task_id]["result"] is not None
        mock_repo.save_result.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_unsupported_provider_fails(self):
        service, mock_repo, task_id = self._make_service_with_task("gemini")

        mock_provider = MagicMock()
        # Only TEXT_TO_IMAGE, not IMAGE_TO_IMAGE
        from pictactic.providers.base import GenerationMode

        mock_provider.supported_modes = [GenerationMode.TEXT_TO_IMAGE]

        with (
            patch(
                "pictactic.agents.tools._image_utils.normalize_reference_images",
                new_callable=AsyncMock,
                return_value=["data:image/png;base64,normalized"],
            ),
            patch(
                "pictactic.providers.get_provider",
                return_value=mock_provider,
            ),
        ):
            await service.run_edit_task(task_id)

        assert service._tasks[task_id]["status"] == GenerationStatus.FAILED
        assert "IMAGE_TO_IMAGE" in service._tasks[task_id]["error"]

    @pytest.mark.asyncio
    async def test_task_not_found_raises(self):
        with (
            patch("pictactic.services.generation_service.create_generation_workflow"),
            patch("pictactic.services.generation_service.TaskRepository"),
        ):
            from pictactic.core.exceptions import TaskNotFoundError
            from pictactic.services.generation_service import GenerationService

            service = GenerationService()

            with pytest.raises(TaskNotFoundError):
                await service.run_edit_task("nonexistent-task-id")
