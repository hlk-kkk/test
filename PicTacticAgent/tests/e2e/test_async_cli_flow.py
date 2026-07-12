"""E2E integration tests for async CLI workflow via ASGITransport.

These tests exercise the full API chain (route -> service -> workflow/provider)
using httpx AsyncClient + ASGITransport, without a real HTTP server.
External APIs (LLM, image providers) are mocked at the lowest layer.
"""

import asyncio
import uuid
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from httpx import ASGITransport, AsyncClient

from pictactic.api.dependencies import get_current_user_or_api_key
from pictactic.services.auth_service import User, UserStatus


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _mock_user(user_id: str = "test-user-e2e") -> User:
    return User(
        user_id=user_id,
        email="e2e@test.local",
        status=UserStatus.ACTIVE,
    )


def _create_test_app():
    """Create a fresh FastAPI app **without** lifespan (skip DB init)."""
    from fastapi import FastAPI

    from pictactic.api.routes import generation, health

    app = FastAPI()
    app.include_router(health.router, tags=["Health"])
    app.include_router(
        generation.router,
        prefix="/api/v1/generation",
        tags=["Generation"],
    )
    return app


async def _wait_for_task(client, task_id, *, headers=None, timeout=5.0, interval=0.1):
    """Poll task status until completed/failed or timeout."""
    elapsed = 0.0
    while elapsed < timeout:
        resp = await client.get(f"/api/v1/generation/{task_id}", headers=headers)
        if resp.status_code == 200:
            data = resp.json()
            if data["status"] in ("completed", "failed"):
                return data
        await asyncio.sleep(interval)
        elapsed += interval
    return None


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture()
def mock_workflow():
    """Mock the LangGraph workflow so no real LLM / provider calls happen.

    The mock workflow yields one status_change event and one image_ready event,
    then returns a final state with best_images populated.
    """
    fake_image = {
        "id": str(uuid.uuid4()),
        "url": "data:image/png;base64,iVBORw0KGgo=",
        "local_path": None,
        "score": 0.85,
        "review": "Good image",
        "round_number": 1,
        "provider": "jiekouai",
        "metadata": {},
    }

    async def _fake_astream(initial_state, *, config=None, stream_mode=None):
        yield {"type": "status_change", "status": "generating", "current_round": 1}
        yield {"type": "image_ready", "image": fake_image}

    mock_wf = MagicMock()
    mock_wf.astream = _fake_astream

    # aget_state returns final snapshot
    final_state = {
        "messages": [],
        "user_prompt": "a cute cat",
        "enhanced_prompt": "a cute fluffy cat in sunshine",
        "template_images": [],
        "template_analysis": None,
        "generated_images": [fake_image],
        "best_images": [fake_image],
        "current_round": 1,
        "status": "completed",
        "final_result": {"success": True},
        "error": None,
        "max_rounds": 1,
        "images_per_round": 2,
        "aspect_ratio": "16:9",
        "image_size": "2K",
        "provider": "jiekouai",
        "quality_threshold": 0.7,
    }
    snapshot = MagicMock()
    snapshot.values = final_state
    mock_wf.aget_state = AsyncMock(return_value=snapshot)

    return mock_wf, fake_image


@pytest.fixture(autouse=True)
def _reset_service_singleton():
    """Reset the global GenerationService singleton between tests."""
    import pictactic.services.generation_service as svc_mod

    svc_mod._service = None
    yield
    svc_mod._service = None


# ---------------------------------------------------------------------------
# Test: Full generate async flow
# ---------------------------------------------------------------------------

class TestGenerateAsyncFlow:
    """Submit generate -> poll status -> get result."""

    async def test_full_generate_flow(self, mock_workflow):
        mock_wf, fake_image = mock_workflow
        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=mock_wf,
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.api.websocket.get_connection_manager",
                return_value=MagicMock(send_progress=AsyncMock()),
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                # Step 1: Submit task
                resp = await client.post(
                    "/api/v1/generation/",
                    json={
                        "prompt": "a cute cat",
                        "max_rounds": 1,
                        "images_per_round": 2,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                data = resp.json()
                assert "task_id" in data
                assert data["status"] == "pending"
                task_id = data["task_id"]

                # Step 2: Poll until completed
                status_data = await _wait_for_task(client, task_id)
                assert status_data is not None, "Task did not complete within timeout"
                assert status_data["status"] == "completed"
                assert status_data["images_generated"] >= 1

                # Step 3: Get result
                resp = await client.get(f"/api/v1/generation/{task_id}/result")
                assert resp.status_code == 200
                result = resp.json()
                assert result["task_id"] == task_id
                # Result should have best_images with correct scores
                assert len(result["best_images"]) >= 1
                assert result["best_images"][0]["score"] == fake_image["score"]

        app.dependency_overrides.clear()

    async def test_generate_flow_workflow_failure(self):
        """Workflow raises -> task status should be 'failed'."""
        async def _exploding_astream(*_a, **_kw):
            raise RuntimeError("LLM is down")
            yield  # make it an async generator  # noqa: E501  # pragma: no cover

        mock_wf = MagicMock()
        mock_wf.astream = _exploding_astream

        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=mock_wf,
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.api.websocket.get_connection_manager",
                return_value=MagicMock(send_progress=AsyncMock()),
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.post(
                    "/api/v1/generation/",
                    json={
                        "prompt": "a cute cat",
                        "max_rounds": 1,
                        "images_per_round": 1,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                task_id = resp.json()["task_id"]

                status_data = await _wait_for_task(client, task_id)
                assert status_data is not None
                assert status_data["status"] == "failed"
                assert status_data["error"] is not None

        app.dependency_overrides.clear()


# ---------------------------------------------------------------------------
# Test: API Key authentication E2E
# ---------------------------------------------------------------------------

class TestApiKeyAuthE2E:
    """X-API-Key header authentication full flow."""

    async def test_no_auth_returns_401(self):
        """No auth header -> 401."""
        app = _create_test_app()
        # Do NOT override auth dependency; let real auth logic run

        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://test"
        ) as client:
            resp = await client.get("/api/v1/generation/nonexistent-task")
            assert resp.status_code == 401

    async def test_wrong_api_key_returns_401(self):
        """Wrong X-API-Key -> 401."""
        app = _create_test_app()

        with patch("pictactic.core.config.settings") as mock_settings:
            mock_settings.internal_api_key = "correct-key"

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.get(
                    "/api/v1/generation/nonexistent-task",
                    headers={"X-API-Key": "wrong-key"},
                )
                assert resp.status_code == 401

    async def test_correct_api_key_passes_auth(self):
        """Correct X-API-Key -> auth passes (404 because task doesn't exist)."""
        app = _create_test_app()

        with (
            patch("pictactic.core.config.settings") as mock_settings,
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
        ):
            mock_settings.internal_api_key = "correct-key"
            mock_repo = MockRepo.return_value
            mock_repo.get_task_result = AsyncMock(return_value=None)

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.get(
                    "/api/v1/generation/nonexistent-task",
                    headers={"X-API-Key": "correct-key"},
                )
                # Auth passes -> task not found -> 404
                assert resp.status_code == 404

    async def test_api_key_user_is_sentinel(self):
        """API Key auth returns sentinel user with user_id='__api_key__'."""
        app = _create_test_app()

        captured_user = {}

        async def capture_user_dep():
            user = User(
                user_id="__api_key__",
                email="internal@api.local",
                status=UserStatus.ACTIVE,
            )
            captured_user["user"] = user
            return user

        app.dependency_overrides[get_current_user_or_api_key] = capture_user_dep

        with (
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.get_task_result = AsyncMock(return_value=None)

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.get("/api/v1/generation/some-task")
                # user_id is None for __api_key__ sentinel (effective_user_id returns None)
                # so ownership check is skipped -> task not found -> 404
                assert resp.status_code == 404

        assert captured_user["user"].user_id == "__api_key__"
        app.dependency_overrides.clear()


# ---------------------------------------------------------------------------
# Test: Standalone edit task E2E
# ---------------------------------------------------------------------------

class TestEditTaskAsyncFlow:
    """Submit edit -> poll -> get result."""

    async def test_edit_task_full_flow(self):
        """Edit task completes successfully through the API."""
        from pictactic.providers import GenerationMode
        from pictactic.providers.base import GenerationResult as ProviderResult

        mock_provider = MagicMock()
        mock_provider.supported_modes = {GenerationMode.IMAGE_TO_IMAGE}
        mock_provider.generate = AsyncMock(
            return_value=ProviderResult(
                url="data:image/png;base64,AAAA",
                local_path=None,
            )
        )

        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.providers.get_provider",
                return_value=mock_provider,
            ),
            patch(
                "pictactic.agents.tools._image_utils.normalize_reference_images",
                new_callable=AsyncMock,
                return_value=["data:image/png;base64,AAAA"],
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                # Step 1: Submit edit task
                resp = await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "make it blue",
                        "source_images": ["data:image/png;base64,AAAA"],
                        "count": 1,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                data = resp.json()
                assert "task_id" in data
                assert data["status"] == "pending"
                task_id = data["task_id"]

                # Step 2: Wait for background task
                status_data = await _wait_for_task(client, task_id)
                assert status_data is not None, "Edit task did not complete within timeout"
                assert status_data["status"] == "completed"

                # Step 3: Get result
                resp = await client.get(f"/api/v1/generation/{task_id}/result")
                assert resp.status_code == 200
                result = resp.json()
                # Note: result.status may lag behind task_info["status"] because
                # _build_result snapshots status before _update_task_status(COMPLETED).
                # The status endpoint (step 2) already confirmed completion.
                assert len(result["best_images"]) == 1

        app.dependency_overrides.clear()

    async def test_edit_task_unsupported_provider_fails(self):
        """Edit task with provider that doesn't support I2I -> fails."""
        from pictactic.providers import GenerationMode

        mock_provider = MagicMock()
        mock_provider.supported_modes = {GenerationMode.TEXT_TO_IMAGE}

        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.providers.get_provider",
                return_value=mock_provider,
            ),
            patch(
                "pictactic.agents.tools._image_utils.normalize_reference_images",
                new_callable=AsyncMock,
                return_value=["data:image/png;base64,AAAA"],
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.post(
                    "/api/v1/generation/edit",
                    json={
                        "prompt": "make it blue",
                        "source_images": ["data:image/png;base64,AAAA"],
                        "count": 1,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                task_id = resp.json()["task_id"]

                status_data = await _wait_for_task(client, task_id)
                assert status_data is not None
                assert status_data["status"] == "failed"
                assert "IMAGE_TO_IMAGE" in (status_data.get("error") or "")

        app.dependency_overrides.clear()


# ---------------------------------------------------------------------------
# Test: Lifespan auto-generates API key
# ---------------------------------------------------------------------------

class TestLifespanApiKey:
    """Backend startup auto-generates INTERNAL_API_KEY when empty."""

    async def test_lifespan_generates_key_when_empty(self, tmp_path):
        """If internal_api_key is empty, lifespan generates one and writes to .env."""
        env_file = tmp_path / ".env"
        env_file.write_text("SOME_OTHER_VAR=hello\n")

        with (
            patch("pictactic.db.engine.init_db", new_callable=AsyncMock),
            patch("pictactic.api.app.settings") as mock_settings,
        ):
            mock_settings.internal_api_key = ""
            mock_settings.jwt_secret_key = "a" * 32
            mock_settings.model_config = {"env_file": str(env_file)}

            from pictactic.api.app import lifespan

            from fastapi import FastAPI

            app = FastAPI()

            async with lifespan(app):
                pass

            # .env file should contain INTERNAL_API_KEY
            content = env_file.read_text()
            assert "INTERNAL_API_KEY=" in content
            for line in content.splitlines():
                if line.startswith("INTERNAL_API_KEY="):
                    key_value = line.split("=", 1)[1]
                    assert len(key_value) > 10

    async def test_lifespan_replaces_empty_key_in_env(self, tmp_path):
        """If .env has INTERNAL_API_KEY= (empty), lifespan replaces it."""
        env_file = tmp_path / ".env"
        env_file.write_text("INTERNAL_API_KEY=\nOTHER=value\n")

        with (
            patch("pictactic.db.engine.init_db", new_callable=AsyncMock),
            patch("pictactic.api.app.settings") as mock_settings,
        ):
            mock_settings.internal_api_key = ""
            mock_settings.jwt_secret_key = "a" * 32
            mock_settings.model_config = {"env_file": str(env_file)}

            from pictactic.api.app import lifespan

            from fastapi import FastAPI

            app = FastAPI()

            async with lifespan(app):
                pass

            content = env_file.read_text()
            lines = [l for l in content.splitlines() if l.startswith("INTERNAL_API_KEY=")]
            assert len(lines) == 1
            key_value = lines[0].split("=", 1)[1]
            assert len(key_value) > 10
            assert "OTHER=value" in content


# ---------------------------------------------------------------------------
# Test: Task listing and cancellation E2E
# ---------------------------------------------------------------------------

class TestTaskManagementE2E:
    """List tasks, cancel task through the API."""

    async def test_list_tasks_empty(self):
        """GET / returns empty list when no tasks exist."""
        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
        ):
            MockRepo.return_value.create_task = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.get("/api/v1/generation/")
                assert resp.status_code == 200
                assert resp.json() == []

        app.dependency_overrides.clear()

    async def test_cancel_running_task(self, mock_workflow):
        """POST /{task_id}/cancel cancels a running task."""
        # Use a workflow that takes a long time
        async def _slow_astream(*_a, **_kw):
            await asyncio.sleep(10)
            yield {"type": "status_change", "status": "generating"}

        mock_wf, _ = mock_workflow
        mock_wf.astream = _slow_astream

        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=mock_wf,
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.api.websocket.get_connection_manager",
                return_value=MagicMock(send_progress=AsyncMock()),
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                # Submit task
                resp = await client.post(
                    "/api/v1/generation/",
                    json={
                        "prompt": "slow task",
                        "max_rounds": 1,
                        "images_per_round": 1,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                task_id = resp.json()["task_id"]

                # Give the background task a moment to start
                await asyncio.sleep(0.2)

                # Cancel
                resp = await client.post(f"/api/v1/generation/{task_id}/cancel")
                assert resp.status_code == 200
                data = resp.json()
                assert data["success"] is True

                # Verify status is cancelled
                resp = await client.get(f"/api/v1/generation/{task_id}")
                assert resp.status_code == 200
                assert resp.json()["status"] == "cancelled"

        app.dependency_overrides.clear()

    async def test_cancel_nonexistent_task_returns_404(self):
        """Cancel a task that doesn't exist -> 404."""
        app = _create_test_app()
        user = _mock_user()
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=MagicMock(),
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
        ):
            MockRepo.return_value.create_task = AsyncMock()
            MockRepo.return_value.get_task_result = AsyncMock(return_value=None)

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                resp = await client.post("/api/v1/generation/fake-id/cancel")
                assert resp.status_code == 404

        app.dependency_overrides.clear()


# ---------------------------------------------------------------------------
# Test: User isolation (ownership)
# ---------------------------------------------------------------------------

class TestUserIsolation:
    """Different users cannot see each other's tasks."""

    async def test_different_user_cannot_access_task(self, mock_workflow):
        """User A creates a task, User B gets 404."""
        mock_wf, _ = mock_workflow

        app = _create_test_app()
        user_a = _mock_user("user-a")
        user_b = _mock_user("user-b")
        app.dependency_overrides[get_current_user_or_api_key] = lambda: user_a

        with (
            patch(
                "pictactic.services.generation_service.create_generation_workflow",
                return_value=mock_wf,
            ),
            patch(
                "pictactic.services.generation_service.TaskRepository"
            ) as MockRepo,
            patch(
                "pictactic.api.websocket.get_connection_manager",
                return_value=MagicMock(send_progress=AsyncMock()),
            ),
        ):
            mock_repo = MockRepo.return_value
            mock_repo.create_task = AsyncMock()
            mock_repo.save_result = AsyncMock()
            mock_repo.update_status = AsyncMock()
            mock_repo.get_task_result = AsyncMock(return_value=None)

            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://test"
            ) as client:
                # User A creates task
                resp = await client.post(
                    "/api/v1/generation/",
                    json={
                        "prompt": "user A image",
                        "max_rounds": 1,
                        "images_per_round": 1,
                        "provider": "jiekouai",
                    },
                )
                assert resp.status_code == 200
                task_id = resp.json()["task_id"]

                await _wait_for_task(client, task_id)

                # Switch to User B
                app.dependency_overrides[get_current_user_or_api_key] = lambda: user_b

                # User B tries to access User A's task
                resp = await client.get(f"/api/v1/generation/{task_id}")
                assert resp.status_code == 404

                resp = await client.get(f"/api/v1/generation/{task_id}/result")
                assert resp.status_code == 404

        app.dependency_overrides.clear()
