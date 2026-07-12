"""Integration tests for prompt template API endpoints.

Uses the real FastAPI app (with full middleware chain) but overrides
authentication and service dependencies to avoid requiring a database.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock, patch

from fastapi.testclient import TestClient

from pictactic.api.app import app
from pictactic.api.dependencies import get_current_user
from pictactic.core.exceptions import TemplateNotFoundError
from pictactic.models.schemas import (
    TaskCreate,
    TemplateListResponse,
    TemplateResponse,
)
from pictactic.services.auth_service import User, UserStatus


def _mock_user(user_id="user-001"):
    return User(
        user_id=user_id,
        email="integration@example.com",
        display_name="Integration User",
        status=UserStatus.ACTIVE,
    )


def _mock_template_response(**overrides):
    defaults = dict(
        id="tpl-int-001",
        user_id="user-001",
        title="Integration Template",
        description="Created during integration test",
        prompt_text="A beautiful cover image",
        category="cover",
        tags=["test", "integration"],
        preview_image_url=None,
        config=None,
        usage_count=0,
        is_public=False,
        created_at="2025-06-01T00:00:00",
        updated_at="2025-06-01T00:00:00",
    )
    defaults.update(overrides)
    return TemplateResponse(**defaults)


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture(autouse=True)
def _cleanup_overrides():
    """Ensure dependency overrides are cleared after each test."""
    yield
    app.dependency_overrides.clear()


class TestTemplatesCRUDFlow:
    """End-to-end CRUD flow through the real app."""

    def test_full_crud_lifecycle(self, client):
        """Create -> Get -> List -> Update -> Delete lifecycle."""
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        created = _mock_template_response()
        updated = _mock_template_response(title="Updated Integration Template")
        list_resp = TemplateListResponse(items=[created], total=1, page=1, page_size=20)

        mock_service = MagicMock()
        mock_service.create_template = AsyncMock(return_value=created)
        mock_service.get_template = AsyncMock(return_value=created)
        mock_service.list_templates = AsyncMock(return_value=list_resp)
        mock_service.update_template = AsyncMock(return_value=updated)
        mock_service.delete_template = AsyncMock(return_value=None)

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            # 1. Create
            resp = client.post(
                "/api/v1/templates/",
                json={
                    "title": "Integration Template",
                    "prompt_text": "A beautiful cover image",
                    "category": "cover",
                    "tags": ["test", "integration"],
                },
            )
            assert resp.status_code == 201
            data = resp.json()
            assert data["id"] == "tpl-int-001"
            assert data["title"] == "Integration Template"
            template_id = data["id"]

            # 2. Get
            resp = client.get(f"/api/v1/templates/{template_id}")
            assert resp.status_code == 200
            assert resp.json()["id"] == template_id

            # 3. List
            resp = client.get("/api/v1/templates/")
            assert resp.status_code == 200
            list_data = resp.json()
            assert list_data["total"] == 1
            assert len(list_data["items"]) == 1

            # 4. Update
            resp = client.put(
                f"/api/v1/templates/{template_id}",
                json={"title": "Updated Integration Template"},
            )
            assert resp.status_code == 200
            assert resp.json()["title"] == "Updated Integration Template"

            # 5. Delete
            resp = client.delete(f"/api/v1/templates/{template_id}")
            assert resp.status_code == 200
            assert resp.json()["success"] is True

    def test_search_and_filter(self, client):
        """Test search and category filter through the real app."""
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        item = _mock_template_response(category="portrait")
        list_resp = TemplateListResponse(items=[item], total=1, page=1, page_size=10)

        mock_service = MagicMock()
        mock_service.list_templates = AsyncMock(return_value=list_resp)

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.get(
                "/api/v1/templates/",
                params={"search": "portrait", "category": "portrait", "page_size": 10},
            )

        assert resp.status_code == 200
        data = resp.json()
        assert data["total"] == 1
        assert data["items"][0]["category"] == "portrait"

        call_kwargs = mock_service.list_templates.call_args[1]
        assert call_kwargs["search"] == "portrait"
        assert call_kwargs["category"] == "portrait"
        assert call_kwargs["page_size"] == 10

    def test_generate_from_template(self, client):
        """Test generate-from-template endpoint through the real app."""
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        mock_task = TaskCreate(task_id="task-int-001", status="pending", message="Task created")
        mock_service = MagicMock()
        mock_service.generate_from_template = AsyncMock(return_value=mock_task)

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.post(
                "/api/v1/templates/tpl-int-001/generate",
                json={
                    "config_overrides": {"max_rounds": 2, "quality_threshold": 0.8},
                    "template_images": ["ref.jpg"],
                },
            )

        assert resp.status_code == 201
        data = resp.json()
        assert data["task_id"] == "task-int-001"
        assert data["status"] == "pending"


class TestTemplatesErrorHandling:
    """Error handling through the real app middleware chain."""

    def test_get_nonexistent_returns_404(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        mock_service = MagicMock()
        mock_service.get_template = AsyncMock(
            side_effect=TemplateNotFoundError("no-such-id")
        )

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.get("/api/v1/templates/no-such-id")

        assert resp.status_code == 404
        assert "not found" in resp.json()["detail"].lower()

    def test_update_nonexistent_returns_404(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        mock_service = MagicMock()
        mock_service.update_template = AsyncMock(
            side_effect=TemplateNotFoundError("no-such-id")
        )

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.put(
                "/api/v1/templates/no-such-id",
                json={"title": "X"},
            )

        assert resp.status_code == 404

    def test_delete_nonexistent_returns_404(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        mock_service = MagicMock()
        mock_service.delete_template = AsyncMock(
            side_effect=TemplateNotFoundError("no-such-id")
        )

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.delete("/api/v1/templates/no-such-id")

        assert resp.status_code == 404

    def test_generate_nonexistent_returns_404(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        mock_service = MagicMock()
        mock_service.generate_from_template = AsyncMock(
            side_effect=TemplateNotFoundError("no-such-id")
        )

        with patch(
            "pictactic.api.routes.templates.get_template_service",
            return_value=mock_service,
        ):
            resp = client.post(
                "/api/v1/templates/no-such-id/generate",
                json={},
            )

        assert resp.status_code == 404


class TestTemplatesAuth:
    """Authentication enforcement through the real app."""

    def test_create_without_auth_returns_401(self, client):
        resp = client.post(
            "/api/v1/templates/",
            json={"title": "T", "prompt_text": "P"},
        )
        assert resp.status_code in (401, 403)

    def test_list_without_auth_returns_401(self, client):
        resp = client.get("/api/v1/templates/")
        assert resp.status_code in (401, 403)

    def test_get_without_auth_returns_401(self, client):
        resp = client.get("/api/v1/templates/some-id")
        assert resp.status_code in (401, 403)

    def test_update_without_auth_returns_401(self, client):
        resp = client.put(
            "/api/v1/templates/some-id",
            json={"title": "X"},
        )
        assert resp.status_code in (401, 403)

    def test_delete_without_auth_returns_401(self, client):
        resp = client.delete("/api/v1/templates/some-id")
        assert resp.status_code in (401, 403)

    def test_generate_without_auth_returns_401(self, client):
        resp = client.post(
            "/api/v1/templates/some-id/generate",
            json={},
        )
        assert resp.status_code in (401, 403)


class TestTemplatesValidation:
    """Request validation through the real app."""

    def test_create_empty_title_returns_422(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        resp = client.post(
            "/api/v1/templates/",
            json={"title": "", "prompt_text": "Valid prompt"},
        )
        assert resp.status_code == 422

    def test_create_empty_prompt_returns_422(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        resp = client.post(
            "/api/v1/templates/",
            json={"title": "Valid title", "prompt_text": ""},
        )
        assert resp.status_code == 422

    def test_create_missing_required_fields_returns_422(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        resp = client.post(
            "/api/v1/templates/",
            json={},
        )
        assert resp.status_code == 422

    def test_list_invalid_page_returns_422(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        resp = client.get("/api/v1/templates/", params={"page": 0})
        assert resp.status_code == 422

    def test_list_invalid_page_size_returns_422(self, client):
        user = _mock_user()
        app.dependency_overrides[get_current_user] = lambda: user

        resp = client.get("/api/v1/templates/", params={"page_size": 101})
        assert resp.status_code == 422
