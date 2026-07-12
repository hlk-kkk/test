"""E2E test for WebSocket auth and task ownership."""

import asyncio
import json
import os
import uuid

import httpx
import pytest

try:
    import websockets
except ImportError:  # pragma: no cover - optional dependency
    websockets = None


BASE_URL = os.getenv("TEST_API_URL", "http://localhost:8019")
WS_BASE_URL = BASE_URL.replace("http://", "ws://").replace("https://", "wss://")

# Skip marker for tests that require running server
requires_server = pytest.mark.skipif(
    os.getenv("SKIP_SERVER_TESTS", "true").lower() == "true",
    reason="Server tests skipped (set SKIP_SERVER_TESTS=false to run)",
)
requires_ws = pytest.mark.skipif(websockets is None, reason="websockets not installed")


async def _register_user(client: httpx.AsyncClient) -> str:
    response = await client.post(
        "/api/v1/auth/register",
        json={
            "email": f"test_{uuid.uuid4().hex}@example.com",
            "password": "secure_password_123",
            "display_name": "WS User",
        },
    )
    response.raise_for_status()
    return response.json()["access_token"]


@requires_server
@requires_ws
@pytest.mark.asyncio
async def test_websocket_auth_flow():
    async with httpx.AsyncClient(base_url=BASE_URL, timeout=600.0) as client:
        token = await _register_user(client)
        headers = {"Authorization": f"Bearer {token}"}

        create_response = await client.post(
            "/api/v1/generation/",
            json={
                "prompt": "WS auth test prompt",
                "max_rounds": 1,
                "images_per_round": 1,
                "quality_threshold": 0.2,
                "aspect_ratio": "1:1",
                "size": "1K",
            },
            headers=headers,
        )
        create_response.raise_for_status()
        task_id = create_response.json()["task_id"]

    ws_url = f"{WS_BASE_URL}/ws/progress/{task_id}?lang=en&token={token}"

    async with websockets.connect(ws_url) as websocket:
        raw = await asyncio.wait_for(websocket.recv(), timeout=5)
        data = json.loads(raw)
        assert data["task_id"] == task_id
        assert "status" in data
