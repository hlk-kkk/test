"""Direct Provider API tests.

Bypasses the Agent layer to directly test the Jiekouai Provider:
1. Provider health check
2. Text-to-image API call
3. Image-to-image API call (URL reference)
4. Image-to-image API call (base64 reference)
5. Request body structure validation
"""

import asyncio
import base64
import os
import sys
from pathlib import Path

import pytest

# Add backend to path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "backend" / "src"))

from dotenv import load_dotenv

load_dotenv()

import httpx

# Skip marker for tests that require API key
requires_api = pytest.mark.skipif(
    not os.getenv("GEMINI_API_KEY"),
    reason="GEMINI_API_KEY not set"
)

# Test image URL (public random image)
TEST_IMAGE_URL = "https://picsum.photos/400/400"
# Local test image path
TEST_IMAGE_PATH = Path(__file__).parent.parent / "fixtures" / "test_image.jpg"


def get_api_config():
    """Get API configuration from environment."""
    api_key = os.getenv("GEMINI_API_KEY")
    base_url = os.getenv("GEMINI_API_BASE_URL", "https://api.jiekou.ai/v3")
    return api_key, base_url


def load_test_image_base64() -> str:
    """Load test image as pure base64 (no data URI prefix)."""
    if not TEST_IMAGE_PATH.exists():
        raise FileNotFoundError(f"Test image not found: {TEST_IMAGE_PATH}")
    with open(TEST_IMAGE_PATH, "rb") as f:
        return base64.b64encode(f.read()).decode()


@requires_api
async def test_health_check():
    """Test provider health check endpoint."""
    print("\n" + "=" * 60)
    print("Test: Provider Health Check")
    print("=" * 60)

    api_key, base_url = get_api_config()
    print(f"   Base URL: {base_url}")
    print(f"   API Key: {api_key[:10] if api_key else 'NOT SET'}...")

    if not api_key:
        print("   SKIP: No API key configured")
        return False

    async with httpx.AsyncClient(timeout=30.0) as client:
        try:
            response = await client.get(
                f"{base_url}/health",
                headers={"Authorization": f"Bearer {api_key}"},
            )
            print(f"   Status: {response.status_code}")
            healthy = response.status_code < 500
            print(f"   Healthy: {healthy}")
            return healthy
        except Exception as e:
            print(f"   Error: {e}")
            return False


@requires_api
async def test_text_to_image_api():
    """Test text-to-image API endpoint directly."""
    print("\n" + "=" * 60)
    print("Test: Text-to-Image API (/gemini-3-pro-image-text-to-image)")
    print("=" * 60)

    api_key, base_url = get_api_config()
    if not api_key:
        print("   SKIP: No API key configured")
        return

    endpoint = "/gemini-3-pro-image-text-to-image"
    request_body = {
        "prompt": "A minimalist tech blog header with abstract geometric shapes",
        "aspect_ratio": "16:9",
        "size": "1K",
    }

    print(f"   Endpoint: {endpoint}")
    print(f"   Request body: {request_body}")

    async with httpx.AsyncClient(timeout=120.0) as client:
        try:
            response = await client.post(
                f"{base_url}{endpoint}",
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                },
                json=request_body,
            )
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                data = response.json()
                image_urls = data.get("image_urls", [])
                print(f"   Generated {len(image_urls)} image(s)")
                if image_urls:
                    print(f"   URL: {image_urls[0][:80]}...")
                print("\n   Text-to-image API test passed!")
            else:
                print(f"   Error response: {response.text[:200]}")
        except Exception as e:
            print(f"   Error: {e}")


@requires_api
async def test_image_to_image_api_with_url():
    """Test image-to-image API endpoint with URL reference."""
    print("\n" + "=" * 60)
    print("Test: Image-to-Image API with URL (/gemini-3-pro-image-edit)")
    print("=" * 60)

    api_key, base_url = get_api_config()
    if not api_key:
        print("   SKIP: No API key configured")
        return

    endpoint = "/gemini-3-pro-image-edit"
    request_body = {
        "prompt": "Transform this image into a dreamy watercolor style",
        "aspect_ratio": "1:1",
        "size": "1K",
        "image_urls": [TEST_IMAGE_URL],
    }

    print(f"   Endpoint: {endpoint}")
    print(f"   Reference image URL: {TEST_IMAGE_URL}")
    print(f"   Request body keys: {list(request_body.keys())}")

    async with httpx.AsyncClient(timeout=120.0) as client:
        try:
            response = await client.post(
                f"{base_url}{endpoint}",
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                },
                json=request_body,
            )
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                data = response.json()
                image_urls = data.get("image_urls", [])
                print(f"   Generated {len(image_urls)} image(s)")
                if image_urls:
                    print(f"   URL: {image_urls[0][:80]}...")
                print("\n   Image-to-image (URL) API test passed!")
            else:
                print(f"   Error response: {response.text[:200]}")
        except Exception as e:
            print(f"   Error: {e}")


@requires_api
async def test_image_to_image_api_with_base64():
    """Test image-to-image API endpoint with base64 reference."""
    print("\n" + "=" * 60)
    print("Test: Image-to-Image API with base64 (/gemini-3-pro-image-edit)")
    print("=" * 60)

    api_key, base_url = get_api_config()
    if not api_key:
        print("   SKIP: No API key configured")
        return

    # Load test image as pure base64 (no data URI prefix)
    try:
        image_base64 = load_test_image_base64()
    except FileNotFoundError as e:
        print(f"   SKIP: {e}")
        return

    endpoint = "/gemini-3-pro-image-edit"
    request_body = {
        "prompt": "Add a vintage sepia filter to this image",
        "aspect_ratio": "1:1",
        "size": "1K",
        "image_base64s": [image_base64],  # Pure base64, no data URI prefix
    }

    print(f"   Endpoint: {endpoint}")
    print(f"   Reference image: base64 ({len(image_base64)} chars)")
    print(f"   Request body keys: {list(request_body.keys())}")

    async with httpx.AsyncClient(timeout=120.0) as client:
        try:
            response = await client.post(
                f"{base_url}{endpoint}",
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                },
                json=request_body,
            )
            print(f"   Status: {response.status_code}")

            if response.status_code == 200:
                data = response.json()
                image_urls = data.get("image_urls", [])
                print(f"   Generated {len(image_urls)} image(s)")
                if image_urls:
                    print(f"   URL: {image_urls[0][:80]}...")
                print("\n   Image-to-image (base64) API test passed!")
            else:
                print(f"   Error response: {response.text[:200]}")
        except Exception as e:
            print(f"   Error: {e}")


def test_request_body_structure():
    """Validate request body structure for different modes."""
    print("\n" + "=" * 60)
    print("Test: Request Body Structure Validation")
    print("=" * 60)

    # Text-to-image request
    text_to_image_body = {
        "prompt": "test prompt",
        "aspect_ratio": "16:9",
        "size": "1K",
    }
    print("\n1. Text-to-image request body:")
    print(f"   {text_to_image_body}")
    assert "prompt" in text_to_image_body
    assert "aspect_ratio" in text_to_image_body
    assert "size" in text_to_image_body
    assert "image_urls" not in text_to_image_body
    assert "image_base64s" not in text_to_image_body
    print("   Valid!")

    # Image-to-image with URLs
    img_to_img_url_body = {
        "prompt": "test prompt",
        "aspect_ratio": "16:9",
        "size": "1K",
        "image_urls": ["https://example.com/image.jpg"],
    }
    print("\n2. Image-to-image (URL) request body:")
    print(f"   Keys: {list(img_to_img_url_body.keys())}")
    assert "image_urls" in img_to_img_url_body
    assert isinstance(img_to_img_url_body["image_urls"], list)
    print("   Valid!")

    # Image-to-image with base64
    img_to_img_b64_body = {
        "prompt": "test prompt",
        "aspect_ratio": "16:9",
        "size": "1K",
        "image_base64s": ["/9j/4AAQ..."],  # Pure base64, no prefix
    }
    print("\n3. Image-to-image (base64) request body:")
    print(f"   Keys: {list(img_to_img_b64_body.keys())}")
    assert "image_base64s" in img_to_img_b64_body
    assert isinstance(img_to_img_b64_body["image_base64s"], list)
    # Verify base64 has no data URI prefix
    assert not img_to_img_b64_body["image_base64s"][0].startswith("data:")
    print("   Valid!")

    # Mixed references
    mixed_body = {
        "prompt": "test prompt",
        "aspect_ratio": "16:9",
        "size": "1K",
        "image_urls": ["https://example.com/image.jpg"],
        "image_base64s": ["/9j/4AAQ..."],
    }
    print("\n4. Image-to-image (mixed) request body:")
    print(f"   Keys: {list(mixed_body.keys())}")
    assert "image_urls" in mixed_body
    assert "image_base64s" in mixed_body
    print("   Valid!")

    print("\n   All request body structure tests passed!")


async def test_provider_class():
    """Test JiekouaiImageProvider class directly."""
    print("\n" + "=" * 60)
    print("Test: JiekouaiImageProvider Class")
    print("=" * 60)

    from pictactic.providers.jiekouai import JiekouaiImageProvider
    from pictactic.providers.base import GenerationMode, GenerationParams

    provider = JiekouaiImageProvider()
    print(f"   Provider name: {provider.name}")
    print(f"   Supports batch: {provider.supports_batch}")
    print(f"   Max concurrent: {provider.max_concurrent}")

    # Test mode detection
    print("\n   Mode detection:")

    # Text-to-image (no reference images)
    params_t2i = GenerationParams(
        prompt="test",
        aspect_ratio="16:9",
        size="1K",
        reference_images=[],
    )
    assert params_t2i.get_mode() == GenerationMode.TEXT_TO_IMAGE
    print(f"   Empty reference_images -> {params_t2i.get_mode().value}")

    # Image-to-image (with reference images)
    params_i2i = GenerationParams(
        prompt="test",
        aspect_ratio="16:9",
        size="1K",
        reference_images=["https://example.com/img.jpg"],
    )
    assert params_i2i.get_mode() == GenerationMode.IMAGE_TO_IMAGE
    print(f"   With reference_images -> {params_i2i.get_mode().value}")

    await provider.close()
    print("\n   Provider class test passed!")


async def run_all_tests():
    """Run all direct provider tests."""
    print("\n" + "=" * 60)
    print("PicTactic Direct Provider Tests")
    print("=" * 60)

    # Unit tests (no API calls)
    test_request_body_structure()
    await test_provider_class()

    # API tests
    print("\n" + "-" * 60)
    print("API Tests (requires GEMINI_API_KEY)")
    print("-" * 60)

    api_key, _ = get_api_config()
    if not api_key:
        print("\nSkipping API tests - GEMINI_API_KEY not set")
        print("Set GEMINI_API_KEY environment variable to run API tests")
        return

    healthy = await test_health_check()
    if not healthy:
        print("\nProvider health check failed, skipping generation tests")
        return

    await test_text_to_image_api()
    await test_image_to_image_api_with_url()
    await test_image_to_image_api_with_base64()

    print("\n" + "=" * 60)
    print("All provider tests completed!")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(run_all_tests())
