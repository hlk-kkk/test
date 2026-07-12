"""End-to-end test for the complete generation flow."""

import asyncio
import os
import sys
import uuid

# Add backend to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../backend/src'))

import httpx
import pytest
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

BASE_URL = os.getenv("TEST_API_URL", "http://localhost:8019")

# Skip marker for tests that require running server
requires_server = pytest.mark.skipif(
    os.getenv("SKIP_SERVER_TESTS", "true").lower() == "true",
    reason="Server tests skipped (set SKIP_SERVER_TESTS=false to run)"
)


async def get_auth_headers(client):
    """Register a user and return auth headers."""
    register_response = await client.post(
        "/api/v1/auth/register",
        json={
            "email": f"test_{uuid.uuid4().hex}@example.com",
            "password": "secure_password_123",
            "display_name": "E2E User",
        },
    )
    register_response.raise_for_status()
    access_token = register_response.json()["access_token"]
    return {"Authorization": f"Bearer {access_token}"}


@requires_server
class TestGenerationFlow:
    """E2E tests for the complete image generation flow."""

    @pytest.fixture
    def client(self):
        """Create HTTP client."""
        return httpx.AsyncClient(base_url=BASE_URL, timeout=600.0)

    @pytest.mark.asyncio
    async def test_health_check(self, client):
        """Test that the API is running."""
        response = await client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"
        print("✓ Health check passed")

    @pytest.mark.asyncio
    async def test_create_and_poll_task(self, client):
        """Test creating a task and polling until completion."""
        headers = await get_auth_headers(client)
        # Create task
        create_response = await client.post(
            "/api/v1/generation/",
            json={
                "prompt": "A futuristic AI-themed cover image with blue and purple gradient",
                "max_rounds": 1,
                "images_per_round": 2,
                "quality_threshold": 0.5,
                "aspect_ratio": "16:9",
                "size": "1K",
            },
            headers=headers,
        )

        assert create_response.status_code == 200
        task_data = create_response.json()
        task_id = task_data["task_id"]
        print(f"✓ Task created: {task_id}")

        # Poll for status
        max_polls = 60
        poll_interval = 2

        for i in range(max_polls):
            status_response = await client.get(
                f"/api/v1/generation/{task_id}",
                headers=headers,
            )
            assert status_response.status_code == 200

            status_data = status_response.json()
            status = status_data["status"]
            message = status_data["message"]

            print(f"  [{i+1}/{max_polls}] Status: {status} - {message}")

            if status == "completed":
                print("✓ Task completed successfully!")
                break
            elif status == "failed":
                print(f"✗ Task failed: {status_data.get('error', 'Unknown error')}")
                pytest.fail("Task failed")
            elif status == "cancelled":
                print("✗ Task was cancelled")
                pytest.fail("Task cancelled")

            await asyncio.sleep(poll_interval)
        else:
            pytest.fail("Task did not complete within timeout")

        # Get final result
        result_response = await client.get(
            f"/api/v1/generation/{task_id}/result",
            headers=headers,
        )
        assert result_response.status_code == 200

        result_data = result_response.json()
        print(f"\n=== Final Result ===")
        print(f"Total rounds: {result_data['total_rounds']}")
        print(f"Total images: {result_data['total_images']}")
        print(f"Best images: {len(result_data['best_images'])}")

        if result_data.get("enhanced_prompt"):
            print(f"Enhanced prompt: {result_data['enhanced_prompt'][:100]}...")

        for i, img in enumerate(result_data.get("best_images", [])[:3]):
            print(f"\nImage {i+1}:")
            print(f"  URL: {img['url'][:80]}...")
            print(f"  Score: {img['score']:.2f}")

        assert result_data["total_images"] > 0
        print("\n✓ E2E test passed!")


async def run_quick_test():
    """Run a quick test without pytest."""
    print("\n" + "=" * 50)
    print("PicTacticAgent E2E Quick Test")
    print("=" * 50 + "\n")

    async with httpx.AsyncClient(base_url=BASE_URL, timeout=600.0) as client:
        headers = await get_auth_headers(client)
        # Health check
        print("1. Checking API health...")
        try:
            response = await client.get("/health")
            if response.status_code == 200:
                print("   ✓ API is healthy\n")
            else:
                print(f"   ✗ API returned {response.status_code}\n")
                return
        except Exception as e:
            print(f"   ✗ Cannot connect to API: {e}")
            print(f"   Make sure the backend is running at {BASE_URL}\n")
            return

        # Create task
        print("2. Creating generation task...")
        try:
            response = await client.post(
                "/api/v1/generation/",
                json={
                    "prompt": "科技感的 AI 封面图，蓝紫色渐变背景，现代简约风格",
                    "max_rounds": 1,
                    "images_per_round": 2,
                    "quality_threshold": 0.5,
                    "aspect_ratio": "16:9",
                    "size": "1K",
                },
                headers=headers,
            )
            task = response.json()
            task_id = task["task_id"]
            print(f"   ✓ Task created: {task_id}\n")
        except Exception as e:
            print(f"   ✗ Failed to create task: {e}\n")
            return

        # Poll status
        print("3. Polling task status...")
        for i in range(60):
            await asyncio.sleep(2)
            try:
                response = await client.get(
                    f"/api/v1/generation/{task_id}",
                    headers=headers,
                )
                if not response.content:
                    print(f"   [{i+1}] Empty response, retrying...")
                    continue
                status = response.json()
                print(f"   [{i+1}] {status['status']}: {status['message']}")

                if status["status"] in ["completed", "failed", "cancelled"]:
                    break
            except Exception as e:
                print(f"   [{i+1}] Error: {e}, retrying...")
                continue

        print()

        # Get result
        print("4. Getting final result...")
        response = await client.get(
            f"/api/v1/generation/{task_id}/result",
            headers=headers,
        )
        result = response.json()

        print(f"   Status: {result['status']}")
        print(f"   Total rounds: {result['total_rounds']}")
        print(f"   Total images: {result['total_images']}")

        if result.get("best_images"):
            print(f"   Best score: {result['best_images'][0]['score']:.2f}")
            print(f"   Best image URL: {result['best_images'][0]['url'][:60]}...")

        print("\n" + "=" * 50)
        print("E2E Test Complete!")
        print("=" * 50 + "\n")


if __name__ == "__main__":
    asyncio.run(run_quick_test())
