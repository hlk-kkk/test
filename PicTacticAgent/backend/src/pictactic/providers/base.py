"""Base classes for image generation providers."""

from abc import ABC, abstractmethod
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field

from ..core.constants import MAX_CONCURRENT_GENERATIONS


class GenerationMode(str, Enum):
    """Image generation mode."""

    TEXT_TO_IMAGE = "text_to_image"
    IMAGE_TO_IMAGE = "image_to_image"


class GenerationParams(BaseModel):
    """Parameters for image generation."""

    prompt: str = Field(..., description="Text prompt for generation")
    negative_prompt: Optional[str] = Field(default=None, description="What to avoid")
    aspect_ratio: str = Field(default="16:9", description="Image aspect ratio")
    size: str = Field(default="2K", description="Image size (1K, 2K, 4K)")
    reference_images: list[str] = Field(
        default_factory=list, description="Reference image URLs for style"
    )
    style: Optional[str] = Field(default=None, description="Style preset")
    seed: Optional[int] = Field(default=None, description="Random seed for reproducibility")

    def get_mode(self) -> GenerationMode:
        """Detect generation mode based on whether reference images are provided."""
        return GenerationMode.IMAGE_TO_IMAGE if self.reference_images else GenerationMode.TEXT_TO_IMAGE

    def validate_mode(self, supported_modes: list["GenerationMode"]) -> None:
        """Validate that the requested mode is supported by the provider."""
        mode = self.get_mode()
        if mode not in supported_modes:
            from ..core.exceptions import ProviderError
            raise ProviderError(
                message=f"Mode '{mode.value}' is not supported by this provider. Supported: {[m.value for m in supported_modes]}",
                provider="unknown",
            )


class GenerationResult(BaseModel):
    """Result from image generation."""

    url: str = Field(description="URL of the generated image")
    local_path: Optional[str] = Field(default=None, description="Local file path if saved")
    revised_prompt: Optional[str] = Field(default=None, description="Revised prompt if modified")
    metadata: dict = Field(default_factory=dict, description="Additional metadata from provider")


class ImageProvider(ABC):
    """Abstract base class for image generation providers."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Provider name identifier."""
        pass

    @property
    @abstractmethod
    def supports_batch(self) -> bool:
        """Whether this provider supports batch generation."""
        pass

    @property
    def supported_modes(self) -> list[GenerationMode]:
        """Supported generation modes. Override to restrict."""
        return [GenerationMode.TEXT_TO_IMAGE, GenerationMode.IMAGE_TO_IMAGE]

    @property
    def max_concurrent(self) -> int:
        """Maximum concurrent requests allowed."""
        return MAX_CONCURRENT_GENERATIONS

    @abstractmethod
    async def generate(
        self,
        params: GenerationParams,
        index: int = 0,
    ) -> GenerationResult:
        """
        Generate a single image.

        Args:
            params: Generation parameters
            index: Index for batch operations (used in naming/tracking)

        Returns:
            GenerationResult with image URL and metadata
        """
        pass

    async def generate_batch(
        self,
        params: GenerationParams,
        count: int,
    ) -> list[GenerationResult]:
        """
        Generate multiple images.

        Default implementation calls generate() multiple times.
        Override for providers with native batch support.

        Args:
            params: Generation parameters
            count: Number of images to generate

        Returns:
            List of GenerationResult
        """
        import asyncio
        import logging

        logger = logging.getLogger(__name__)

        semaphore = asyncio.Semaphore(self.max_concurrent)

        async def limited_generate(idx: int) -> GenerationResult:
            async with semaphore:
                return await self.generate(params, idx)

        tasks = [limited_generate(i) for i in range(count)]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Filter out exceptions and return valid results
        valid_results = []
        errors = []
        for i, r in enumerate(results):
            if isinstance(r, GenerationResult):
                valid_results.append(r)
            elif isinstance(r, Exception):
                logger.error(f"Image generation {i} failed: {type(r).__name__}: {r}")
                errors.append(str(r))

        if not valid_results and results:
            logger.error(f"All {len(results)} image generations failed")
            # Raise exception when all generations fail
            error_msg = errors[0] if errors else "All image generations failed"
            raise RuntimeError(f"图片生成全部失败: {error_msg}")

        return valid_results

    async def generate_batch_stream(
        self,
        params: GenerationParams,
        count: int,
        on_image_ready=None,
    ) -> list[GenerationResult]:
        """
        Generate multiple images, calling on_image_ready as each completes.

        Args:
            params: Generation parameters
            count: Number of images to generate
            on_image_ready: Optional async callback(idx, result) called per image

        Returns:
            List of GenerationResult
        """
        import asyncio
        import logging

        logger = logging.getLogger(__name__)

        semaphore = asyncio.Semaphore(self.max_concurrent)

        async def limited_generate(idx: int):
            async with semaphore:
                return idx, await self.generate(params, idx)

        tasks = [asyncio.ensure_future(limited_generate(i)) for i in range(count)]
        valid_results = []
        for coro in asyncio.as_completed(tasks):
            try:
                idx, result = await coro
                valid_results.append(result)
                if on_image_ready:
                    await on_image_ready(idx, result)
            except Exception as e:
                logger.error(f"Image generation failed: {e}")

        if not valid_results:
            raise RuntimeError("图片生成全部失败")
        return valid_results

    async def health_check(self) -> bool:
        """Check if the provider is available."""
        return True
