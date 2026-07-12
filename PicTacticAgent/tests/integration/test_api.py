"""Integration tests for API endpoints."""

import uuid
import pytest
from fastapi.testclient import TestClient

from pictactic.api.app import app


@pytest.fixture
def client():
    """Create test client."""
    return TestClient(app)


@pytest.fixture
def auth_headers(client):
    """Create auth headers for generation endpoints."""
    email = f"test_{uuid.uuid4().hex}@example.com"
    register_response = client.post(
        "/api/v1/auth/register",
        json={
            "email": email,
            "password": "secure_password_123",
            "display_name": "Test User",
        },
    )
    if register_response.status_code != 200:
        raise AssertionError(f"Failed to register test user: {register_response.text}")
    access_token = register_response.json()["access_token"]
    return {"Authorization": f"Bearer {access_token}"}


class TestHealthEndpoints:
    """Tests for health check endpoints."""

    def test_health_check(self, client):
        """Test health endpoint returns healthy status."""
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "pictactic"

    def test_root_endpoint(self, client):
        """Test root endpoint returns service info."""
        response = client.get("/")
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "PicTacticAgent API"
        assert "version" in data
        assert data["status"] == "running"


class TestGenerationEndpoints:
    """Tests for generation endpoints."""

    def test_create_task(self, client, auth_headers):
        """Test creating a generation task."""
        response = client.post(
            "/api/v1/generation/",
            json={
                "prompt": "Test prompt for cover image",
                "max_rounds": 2,
                "images_per_round": 3,
            },
            headers=auth_headers,
        )
        assert response.status_code == 200
        data = response.json()
        assert "task_id" in data
        assert data["status"] == "pending"
        assert "message" in data

    def test_create_task_minimal(self, client, auth_headers):
        """Test creating task with minimal parameters."""
        response = client.post(
            "/api/v1/generation/",
            json={"prompt": "Simple test"},
            headers=auth_headers,
        )
        assert response.status_code == 200
        data = response.json()
        assert "task_id" in data

    def test_create_task_invalid_prompt(self, client, auth_headers):
        """Test creating task with empty prompt fails."""
        response = client.post(
            "/api/v1/generation/",
            json={"prompt": ""},
            headers=auth_headers,
        )
        assert response.status_code == 422  # Validation error

    def test_get_task_status_not_found(self, client, auth_headers):
        """Test getting status of non-existent task."""
        response = client.get(
            "/api/v1/generation/non-existent-id",
            headers=auth_headers,
        )
        assert response.status_code == 404

    def test_get_task_result_not_found(self, client, auth_headers):
        """Test getting result of non-existent task."""
        response = client.get(
            "/api/v1/generation/non-existent-id/result",
            headers=auth_headers,
        )
        assert response.status_code == 404

    def test_cancel_task_not_found(self, client, auth_headers):
        """Test cancelling non-existent task."""
        response = client.post(
            "/api/v1/generation/non-existent-id/cancel",
            headers=auth_headers,
        )
        assert response.status_code == 404

    def test_list_tasks(self, client, auth_headers):
        """Test listing tasks."""
        response = client.get("/api/v1/generation/", headers=auth_headers)
        assert response.status_code == 200
        data = response.json()
        assert isinstance(data, list)

    def test_create_and_get_task(self, client, auth_headers):
        """Test creating a task and getting its status."""
        # Create task
        create_response = client.post(
            "/api/v1/generation/",
            json={"prompt": "Test prompt"},
            headers=auth_headers,
        )
        assert create_response.status_code == 200
        task_id = create_response.json()["task_id"]

        # Get status
        status_response = client.get(
            f"/api/v1/generation/{task_id}",
            headers=auth_headers,
        )
        assert status_response.status_code == 200
        status_data = status_response.json()
        assert status_data["task_id"] == task_id
