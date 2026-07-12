"""Tests for prompt template API routes."""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient

from pictactic.api.routes.templates import router
from pictactic.api.dependencies import get_current_user
from pictactic.core.exceptions import TemplateNotFoundError
from pictactic.models.schemas import (
    TaskCreate,
    TemplateListResponse,
    TemplateResponse,
)
from pictactic.services.auth_service import User, UserStatus


def _create_test_app():
    """Create test FastAPI app with templates router."""
    app = FastAPI()
    app.include_router(router, prefix="/api/v1/templates")
    return app


app = _create_test_app()


def _mock_user():
    return User(
        user_id="user-001",
        email="test@example.com",
        display_name="Test User",
        status=UserStatus.ACTIVE,
    )


def _mock_template_response(**overrides):
    defaults = dict(
        id="tpl-001",
        user_id="user-001",
        title="Test Template",
        description="A test template",
        prompt_text="Generate a beautiful landscape",
        category="landscape",
        tags=["nature"],
        preview_image_url=None,
        config=None,
        usage_count=0,
        is_public=False,
        created_at="2025-01-01T00:00:00",
        updated_at="2025-01-02T00:00:00",
    )
    defaults.update(overrides)
    return TemplateResponse(**defaults)


class TestCreateTemplate:
    async def test_create_success(self):
        mock_resp = _mock_template_response()
        mock_service = MagicMock()
        mock_service.create_template = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/templates/",
                    json={
                        "title": "Test Template",
                        "prompt_text": "Generate a beautiful landscape",
                        "category": "landscape",
                        "tags": ["nature"],
                    },
                )

        app.dependency_overrides.clear()

        assert response.status_code == 201
        data = response.json()
        assert data["id"] == "tpl-001"
        assert data["title"] == "Test Template"
        mock_service.create_template.assert_awaited_once()

    async def test_create_unauthenticated(self):
        # No dependency override: get_current_user will require a token
        app.dependency_overrides.clear()

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.post(
                "/api/v1/templates/",
                json={"title": "T", "prompt_text": "P"},
            )

        assert response.status_code in (401, 403)

    async def test_create_invalid_body(self):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.post(
                "/api/v1/templates/",
                json={"title": "", "prompt_text": ""},
            )

        app.dependency_overrides.clear()
        assert response.status_code == 422


class TestListTemplates:
    async def test_list_success(self):
        mock_resp = TemplateListResponse(
            items=[_mock_template_response()],
            total=1,
            page=1,
            page_size=20,
        )
        mock_service = MagicMock()
        mock_service.list_templates = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/templates/")

        app.dependency_overrides.clear()

        assert response.status_code == 200
        data = response.json()
        assert data["total"] == 1
        assert len(data["items"]) == 1
        assert data["page"] == 1

    async def test_list_with_search_params(self):
        mock_resp = TemplateListResponse(items=[], total=0, page=1, page_size=10)
        mock_service = MagicMock()
        mock_service.list_templates = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get(
                    "/api/v1/templates/",
                    params={"search": "landscape", "category": "landscape", "page_size": 10},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 200
        mock_service.list_templates.assert_awaited_once()
        call_kwargs = mock_service.list_templates.call_args[1]
        assert call_kwargs["search"] == "landscape"
        assert call_kwargs["category"] == "landscape"
        assert call_kwargs["page_size"] == 10


class TestGetTemplate:
    async def test_get_success(self):
        mock_resp = _mock_template_response()
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/templates/tpl-001")

        app.dependency_overrides.clear()

        assert response.status_code == 200
        data = response.json()
        assert data["id"] == "tpl-001"

    async def test_get_not_found(self):
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(
            side_effect=TemplateNotFoundError("nonexistent")
        )

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/templates/nonexistent")

        app.dependency_overrides.clear()

        assert response.status_code == 404
        assert "not found" in response.json()["detail"].lower()


class TestUpdateTemplate:
    async def test_update_success(self):
        mock_resp = _mock_template_response(title="Updated Title")
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(return_value=mock_resp)
        mock_service.update_template = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/templates/tpl-001",
                    json={"title": "Updated Title"},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 200
        assert response.json()["title"] == "Updated Title"

    async def test_update_system_template_forbidden(self):
        mock_resp = _mock_template_response(is_system=True)
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/templates/tpl-001",
                    json={"title": "X"},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 403

    async def test_update_not_found(self):
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(
            side_effect=TemplateNotFoundError("nonexistent")
        )

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/templates/nonexistent",
                    json={"title": "X"},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 404


class TestDeleteTemplate:
    async def test_delete_success(self):
        mock_resp = _mock_template_response()
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(return_value=mock_resp)
        mock_service.delete_template = AsyncMock(return_value=None)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.delete("/api/v1/templates/tpl-001")

        app.dependency_overrides.clear()

        assert response.status_code == 200
        assert response.json()["success"] is True

    async def test_delete_system_template_forbidden(self):
        mock_resp = _mock_template_response(is_system=True)
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(return_value=mock_resp)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.delete("/api/v1/templates/tpl-001")

        app.dependency_overrides.clear()

        assert response.status_code == 403

    async def test_delete_not_found(self):
        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(
            side_effect=TemplateNotFoundError("nonexistent")
        )

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.delete("/api/v1/templates/nonexistent")

        app.dependency_overrides.clear()

        assert response.status_code == 404


class TestGenerateFromTemplate:
    async def test_generate_success(self):
        mock_task = TaskCreate(task_id="task-001", status="pending", message="Task created")
        mock_service = MagicMock()
        mock_service.generate_from_template = AsyncMock(return_value=mock_task)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/templates/tpl-001/generate",
                    json={},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 201
        data = response.json()
        assert data["task_id"] == "task-001"
        assert data["status"] == "pending"

    async def test_generate_not_found(self):
        mock_service = MagicMock()
        mock_service.generate_from_template = AsyncMock(
            side_effect=TemplateNotFoundError("nonexistent")
        )

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/templates/nonexistent/generate",
                    json={},
                )

        app.dependency_overrides.clear()

        assert response.status_code == 404

    async def test_generate_with_overrides(self):
        mock_task = TaskCreate(task_id="task-002", status="pending", message="Task created")
        mock_service = MagicMock()
        mock_service.generate_from_template = AsyncMock(return_value=mock_task)

        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.post(
                    "/api/v1/templates/tpl-001/generate",
                    json={
                        "config_overrides": {"max_rounds": 2},
                        "template_images": ["img.jpg"],
                    },
                )

        app.dependency_overrides.clear()

        assert response.status_code == 201
        assert response.json()["task_id"] == "task-002"
