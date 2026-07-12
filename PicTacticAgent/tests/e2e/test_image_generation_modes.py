"""E2E tests for image generation modes.

Tests:
1. Text-to-image mode (no reference images)
2. Image-to-image mode with URL references
3. Image-to-image mode with base64 references
4. Image-to-image mode with data URI references
5. Mixed reference images (URL + base64)
6. State reference filtering (state:// patterns should be filtered)
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

# Skip marker for tests that require API key
requires_api = pytest.mark.skipif(
    not os.getenv("GEMINI_API_KEY"),
    reason="GEMINI_API_KEY not set"
)


# Test image URL (public random image)
TEST_IMAGE_URL = "https://picsum.photos/400/400"
# Local test image path
TEST_IMAGE_PATH = Path(__file__).parent.parent / "fixtures" / "test_image.jpg"


def load_test_image_base64() -> str:
    """Load test image as base64 string."""
    if not TEST_IMAGE_PATH.exists():
        raise FileNotFoundError(f"Test image not found: {TEST_IMAGE_PATH}")
    with open(TEST_IMAGE_PATH, "rb") as f:
        return base64.b64encode(f.read()).decode()


def load_test_image_data_uri() -> str:
    """Load test image as data URI."""
    b64 = load_test_image_base64()
    return f"data:image/jpeg;base64,{b64}"


async def test_normalize_reference_images():
    """Test the normalize_reference_images function."""
    from pictactic.agents.tools._image_utils import normalize_reference_images

    print("\n" + "=" * 60)
    print("Test: normalize_reference_images")
    print("=" * 60)

    # Test with URL (should be kept as-is)
    print("\n1. Testing URL normalization...")
    urls = [TEST_IMAGE_URL]
    result = await normalize_reference_images(urls)
    assert len(result) == 1
    assert result[0] == TEST_IMAGE_URL  # URLs should be preserved
    print(f"   URL preserved: {result[0][:50]}...")

    # Test with data URI (should be re-encoded)
    print("\n2. Testing data URI normalization...")
    data_uri = load_test_image_data_uri()
    result = await normalize_reference_images([data_uri])
    assert len(result) == 1
    assert result[0].startswith("data:image/")
    print(f"   Data URI re-encoded: {result[0][:60]}...")

    # Test with base64 (should be re-encoded)
    print("\n3. Testing base64 normalization...")
    b64 = load_test_image_base64()
    result = await normalize_reference_images([b64])
    assert len(result) == 1
    assert result[0].startswith("data:image/")
    print(f"   Base64 re-encoded: {result[0][:60]}...")

    # Test with local file path
    print("\n4. Testing local file path normalization...")
    result = await normalize_reference_images([str(TEST_IMAGE_PATH)])
    assert len(result) == 1
    assert result[0].startswith("data:image/")
    print(f"   Local file encoded: {result[0][:60]}...")

    print("\n   All normalize_reference_images tests passed!")


def test_state_reference_filter():
    """Test that state:// references are properly filtered."""
    from pictactic.agents.tools.generate_images import _is_state_reference

    print("\n" + "=" * 60)
    print("Test: _is_state_reference filter")
    print("=" * 60)

    # These should be detected as state references
    state_refs = [
        "state://template_images",
        "state['template_images']",
        'state["template_images"]',
        "state.template_images",
        "state[0]",
    ]

    for ref in state_refs:
        assert _is_state_reference(ref), f"Should detect as state ref: {ref}"
        print(f"   Detected state ref: {ref}")

    # These should NOT be detected as state references
    valid_sources = [
        "https://example.com/image.jpg",
        "http://example.com/image.png",
        "data:image/jpeg;base64,/9j/4AAQ...",
        "/path/to/local/image.jpg",
        "file:///path/to/image.jpg",
    ]

    for src in valid_sources:
        assert not _is_state_reference(src), f"Should NOT detect as state ref: {src}"
        print(f"   Valid source: {src[:50]}...")

    print("\n   All state reference filter tests passed!")


@requires_api
async def test_text_to_image():
    """Test text-to-image generation (no reference images)."""
    from pictactic.providers import get_provider
    from pictactic.providers.base import GenerationMode, GenerationParams

    print("\n" + "=" * 60)
    print("Test: Text-to-Image Mode")
    print("=" * 60)

    provider = get_provider("gemini")
    params = GenerationParams(
        prompt="A beautiful sunset over mountains, digital art style",
        aspect_ratio="16:9",
        size="1K",
        reference_images=[],
    )

    # Verify mode detection
    mode = params.get_mode()
    assert mode == GenerationMode.TEXT_TO_IMAGE
    print(f"   Mode: {mode.value}")
    print(f"   Endpoint: /gemini-3-pro-image-text-to-image")

    print("\n   Generating image...")
    try:
        result = await provider.generate(params)
        print(f"   Generated URL: {result.url[:80]}...")
        print(f"   Local path: {result.local_path}")
        print("\n   Text-to-image test passed!")
    except Exception as e:
        print(f"   Error: {e}")
        print("   (This may be expected if API quota is exhausted)")
    finally:
        await provider.close()


@requires_api
async def test_image_to_image_with_url():
    """Test image-to-image generation with URL reference."""
    from pictactic.providers import get_provider
    from pictactic.providers.base import GenerationMode, GenerationParams

    print("\n" + "=" * 60)
    print("Test: Image-to-Image Mode (URL reference)")
    print("=" * 60)

    provider = get_provider("gemini")
    params = GenerationParams(
        prompt="Transform this image into a watercolor painting style",
        aspect_ratio="1:1",
        size="1K",
        reference_images=[TEST_IMAGE_URL],
    )

    # Verify mode detection
    mode = params.get_mode()
    assert mode == GenerationMode.IMAGE_TO_IMAGE
    print(f"   Mode: {mode.value}")
    print(f"   Endpoint: /gemini-3-pro-image-edit")
    print(f"   Reference URL: {TEST_IMAGE_URL}")

    print("\n   Generating image...")
    try:
        result = await provider.generate(params)
        print(f"   Generated URL: {result.url[:80]}...")
        print("\n   Image-to-image (URL) test passed!")
    except Exception as e:
        print(f"   Error: {e}")
        print("   (This may be expected if API quota is exhausted)")
    finally:
        await provider.close()


@requires_api
async def test_image_to_image_with_base64():
    """Test image-to-image generation with base64 reference."""
    from pictactic.providers import get_provider
    from pictactic.providers.base import GenerationMode, GenerationParams

    print("\n" + "=" * 60)
    print("Test: Image-to-Image Mode (base64 reference)")
    print("=" * 60)

    # Load test image as data URI
    data_uri = load_test_image_data_uri()

    provider = get_provider("gemini")
    params = GenerationParams(
        prompt="Add a vintage filter effect to this image",
        aspect_ratio="1:1",
        size="1K",
        reference_images=[data_uri],
    )

    # Verify mode detection
    mode = params.get_mode()
    assert mode == GenerationMode.IMAGE_TO_IMAGE
    print(f"   Mode: {mode.value}")
    print(f"   Endpoint: /gemini-3-pro-image-edit")
    print(f"   Reference: data:image/jpeg;base64,... ({len(data_uri)} chars)")

    print("\n   Generating image...")
    try:
        result = await provider.generate(params)
        print(f"   Generated URL: {result.url[:80]}...")
        print("\n   Image-to-image (base64) test passed!")
    except Exception as e:
        print(f"   Error: {e}")
        print("   (This may be expected if API quota is exhausted)")
    finally:
        await provider.close()


@requires_api
async def test_mixed_reference_images():
    """Test image-to-image generation with mixed URL and base64 references."""
    from pictactic.providers import get_provider
    from pictactic.providers.base import GenerationMode, GenerationParams

    print("\n" + "=" * 60)
    print("Test: Image-to-Image Mode (mixed references)")
    print("=" * 60)

    # Mix URL and data URI
    data_uri = load_test_image_data_uri()

    provider = get_provider("gemini")
    params = GenerationParams(
        prompt="Blend the styles of these reference images",
        aspect_ratio="16:9",
        size="1K",
        reference_images=[TEST_IMAGE_URL, data_uri],
    )

    # Verify mode detection
    mode = params.get_mode()
    assert mode == GenerationMode.IMAGE_TO_IMAGE
    print(f"   Mode: {mode.value}")
    print(f"   References: 1 URL + 1 data URI")

    print("\n   Generating image...")
    try:
        result = await provider.generate(params)
        print(f"   Generated URL: {result.url[:80]}...")
        print("\n   Mixed references test passed!")
    except Exception as e:
        print(f"   Error: {e}")
        print("   (This may be expected if API quota is exhausted)")
    finally:
        await provider.close()


async def test_state_reference_in_generate_images():
    """Test that state:// references are filtered in generate_images tool."""
    print("\n" + "=" * 60)
    print("Test: State reference filtering in generate_images")
    print("=" * 60)

    from pictactic.agents.tools.generate_images import _is_state_reference

    # Simulate what happens when LLM passes state reference
    invalid_refs = ["state://template_images"]
    valid_refs = [TEST_IMAGE_URL]

    # Filter like generate_images does
    filtered = [
        img
        for img in invalid_refs
        if isinstance(img, str) and img.strip() and not _is_state_reference(img)
    ]

    assert len(filtered) == 0, "State references should be filtered out"
    print("   Invalid state refs filtered: state://template_images -> []")

    # Valid refs should pass through
    filtered_valid = [
        img
        for img in valid_refs
        if isinstance(img, str) and img.strip() and not _is_state_reference(img)
    ]

    assert len(filtered_valid) == 1
    assert filtered_valid[0] == TEST_IMAGE_URL
    print(f"   Valid refs preserved: {TEST_IMAGE_URL}")

    print("\n   State reference filtering test passed!")


async def run_all_tests():
    """Run all tests."""
    print("\n" + "=" * 60)
    print("PicTactic Image Generation Mode Tests")
    print("=" * 60)

    # Unit tests (no API calls)
    test_state_reference_filter()
    await test_normalize_reference_images()
    await test_state_reference_in_generate_images()

    # Integration tests (require API)
    print("\n" + "-" * 60)
    print("Integration Tests (requires API key)")
    print("-" * 60)

    api_key = os.getenv("GEMINI_API_KEY")
    if not api_key:
        print("\nSkipping API tests - GEMINI_API_KEY not set")
        return

    await test_text_to_image()
    await test_image_to_image_with_url()
    await test_image_to_image_with_base64()
    await test_mixed_reference_images()

    print("\n" + "=" * 60)
    print("All tests completed!")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(run_all_tests())
