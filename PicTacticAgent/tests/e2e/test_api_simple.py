"""Simple API test with logging."""

import asyncio
import os
import uuid

import httpx
import json
import pytest

API_URL = "http://localhost:8019/api/v1/generation"
AUTH_URL = "http://localhost:8019/api/v1/auth"

# Skip marker for tests that require running server
requires_server = pytest.mark.skipif(
    os.getenv("SKIP_SERVER_TESTS", "true").lower() == "true",
    reason="Server tests skipped (set SKIP_SERVER_TESTS=false to run)"
)


@requires_server
async def test_simple():
    """Simple API test."""
    async with httpx.AsyncClient(timeout=600.0) as client:
        # Register and get auth token
        register_response = await client.post(
            f"{AUTH_URL}/register",
            json={
                "email": f"test_{uuid.uuid4().hex}@example.com",
                "password": "secure_password_123",
                "display_name": "E2E User",
            },
        )
        register_response.raise_for_status()
        access_token = register_response.json()["access_token"]
        headers = {"Authorization": f"Bearer {access_token}"}

        print("Creating task...")
        response = await client.post(
            f"{API_URL}/",
            json={
                "prompt": "A simple blue square",
                "max_rounds": 1,
                "images_per_round": 1,
                "quality_threshold": 0.3,
                "aspect_ratio": "1:1",
                "size": "1K",
            },
            headers=headers,
        )
        data = response.json()
        task_id = data["task_id"]
        print(f"Task ID: {task_id}")

        # Poll for completion
        for i in range(120):  # 4 minutes max
            await asyncio.sleep(2)
            try:
                response = await client.get(f"{API_URL}/{task_id}", headers=headers)
                if response.status_code != 200:
                    print(f"[{i+1}] HTTP {response.status_code}")
                    continue
                status = response.json()
                print(f"[{i+1}] {status['status']}: images={status.get('images_generated', 0)}")
            except Exception as e:
                print(f"[{i+1}] Error: {e}")
                continue

            if status["status"] in ["completed", "failed"]:
                print(f"\nFinal status: {json.dumps(status, indent=2)}")
                break

        # Get result
        print("\nGetting result...")
        response = await client.get(f"{API_URL}/{task_id}/result", headers=headers)
        result = response.json()
        print(f"Result: {json.dumps(result, indent=2)}")


if __name__ == "__main__":
    asyncio.run(test_simple())
