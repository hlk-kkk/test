"""Tests for image providers."""

import pytest
from unittest.mock import AsyncMock, patch, MagicMock

from pictactic.providers.base import ImageProvider, GenerationParams, GenerationResult
from pictactic.providers.jiekouai import JiekouaiImageProvider


class TestGenerationParams:
    """Tests for GenerationParams."""

    def test_default_params(self):
        """Test default parameter values."""
        params = GenerationParams(prompt="test")
        assert params.prompt == "test"
        assert params.aspect_ratio == "16:9"
        assert params.size == "2K"
        assert params.reference_images == []

    def test_full_params(self):
        """Test with all parameters."""
        params = GenerationParams(
            prompt="detailed prompt",
            negative_prompt="blurry, low quality",
            aspect_ratio="1:1",
            size="4K",
            reference_images=["ref1.jpg"],
            style="vivid",
            seed=42,
        )
        assert params.negative_prompt == "blurry, low quality"
        assert params.seed == 42


class TestGenerationResult:
    """Tests for GenerationResult."""

    def test_minimal_result(self):
        """Test result with minimal fields."""
        result = GenerationResult(url="https://example.com/image.png")
        assert result.url == "https://example.com/image.png"
        assert result.local_path is None
        assert result.metadata == {}

    def test_full_result(self):
        """Test result with all fields."""
        result = GenerationResult(
            url="https://example.com/image.png",
            local_path="/tmp/image.png",
            revised_prompt="revised prompt",
            metadata={"provider": "test"},
        )
        assert result.local_path == "/tmp/image.png"
        assert result.metadata["provider"] == "test"


class TestJiekouaiProvider:
    """Tests for JiekouaiImageProvider."""

    def test_provider_properties(self):
        """Test provider properties."""
        provider = JiekouaiImageProvider(api_key="test-key")
        assert provider.name == "jiekouai"
        assert provider.supports_batch is True
        assert provider.max_concurrent == 5

    @pytest.mark.asyncio
    async def test_generate_success(self):
        """Test successful image generation."""
        provider = JiekouaiImageProvider(api_key="test-key")

        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "image_urls": ["https://example.com/generated.png"],
        }
        mock_response.raise_for_status = MagicMock()

        with patch.object(provider, "_get_client") as mock_client:
            mock_client.return_value.post = AsyncMock(return_value=mock_response)
            mock_client.return_value.get = AsyncMock(return_value=MagicMock(content=b"image"))

            params = GenerationParams(prompt="test prompt")
            result = await provider.generate(params)

            assert result.url == "https://example.com/generated.png"
            assert result.metadata["provider"] == "gemini-3-pro-image"

    @pytest.mark.asyncio
    async def test_generate_batch(self):
        """Test batch generation."""
        provider = JiekouaiImageProvider(api_key="test-key")

        mock_response = MagicMock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "image_urls": ["https://example.com/generated.png"],
        }
        mock_response.raise_for_status = MagicMock()

        with patch.object(provider, "_get_client") as mock_client:
            mock_client.return_value.post = AsyncMock(return_value=mock_response)
            mock_client.return_value.get = AsyncMock(return_value=MagicMock(content=b"image"))

            params = GenerationParams(prompt="test prompt")
            results = await provider.generate_batch(params, count=3)

            assert len(results) == 3
            for result in results:
                assert result.url == "https://example.com/generated.png"
