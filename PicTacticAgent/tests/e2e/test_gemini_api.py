"""Test Gemini API directly."""

import asyncio
import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../backend/src'))

from dotenv import load_dotenv
load_dotenv()

import httpx

# Skip marker for tests that require API key
requires_api = pytest.mark.skipif(
    not os.getenv("GEMINI_API_KEY"),
    reason="GEMINI_API_KEY not set"
)


@requires_api
async def test_gemini_api():
    """Test Gemini API directly."""
    api_key = os.getenv("GEMINI_API_KEY")
    base_url = os.getenv("GEMINI_API_BASE_URL", "https://api.jiekou.ai/v3")

    print(f"API Key: {api_key[:10]}..." if api_key else "No API key!")
    print(f"Base URL: {base_url}")

    async with httpx.AsyncClient(timeout=120.0) as client:
        try:
            response = await client.post(
                f"{base_url}/gemini-3-pro-image-edit",
                headers={
                    "Authorization": f"Bearer {api_key}",
                    "Content-Type": "application/json",
                },
                json={
                    "prompt": "A simple blue gradient background",
                    "image_urls": [],
                    "aspect_ratio": "16:9",
                    "size": "1K",
                },
            )
            print(f"Status: {response.status_code}")
            print(f"Response: {response.text[:500]}")

            if response.status_code == 200:
                data = response.json()
                print(f"Image URLs: {data.get('image_urls', [])}")
        except Exception as e:
            print(f"Error: {e}")


if __name__ == "__main__":
    asyncio.run(test_gemini_api())
