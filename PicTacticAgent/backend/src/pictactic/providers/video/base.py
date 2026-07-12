"""Video generation provider abstraction."""

import logging
from abc import ABC, abstractmethod
from enum import Enum

from pydantic import BaseModel

logger = logging.getLogger(__name__)


class VideoGenerationMode(str, Enum):
    """Video generation mode."""

    TEXT_TO_VIDEO = "text_to_video"
    IMAGE_TO_VIDEO = "image_to_video"


class VideoGenerationParams(BaseModel):
    """Parameters for video generation."""

    prompt: str
    start_image: str | None = None  # URL or base64 data URI for first frame
    end_image: str | None = None  # URL or base64 data URI for last frame
    duration: int = 5  # seconds
    resolution: str | None = None  # "720p", "1080p"
    aspect_ratio: str = "16:9"  # "16:9", "9:16"
    generate_audio: bool = False
    negative_prompt: str | None = None
    seed: int | None = None
    enhance_prompt: bool = True
    # Provider-specific
    cfg_scale: float | None = None  # Kling: 0-1
    professional: bool | None = None  # Sora: pro mode
    reference_images: list[str] = []  # Veo: reference images
    sample_count: int = 1  # Veo: 1-4

    def get_mode(self) -> VideoGenerationMode:
        """Detect generation mode based on whether a start image is provided."""
        if self.start_image:
            return VideoGenerationMode.IMAGE_TO_VIDEO
        return VideoGenerationMode.TEXT_TO_VIDEO

    def validate_mode(self, supported_modes: list[VideoGenerationMode]) -> None:
        """Validate that the requested mode is supported by the provider."""
        mode = self.get_mode()
        if mode not in supported_modes:
            from ...core.exceptions import ProviderError

            raise ProviderError(
                message=(
                    f"Mode '{mode.value}' is not supported by this provider. "
                    f"Supported: {[m.value for m in supported_modes]}"
                ),
                provider="unknown",
            )


class VideoGenerationResult(BaseModel):
    """Result of a video generation."""

    url: str  # Video URL (CDN or local serving path)
    local_path: str | None = None  # Local storage path
    video_type: str = "mp4"  # mp4, gif
    duration: int | None = None  # Duration in seconds
    provider_task_id: str | None = None  # Provider's async task ID
    metadata: dict = {}


class VideoProviderCapabilities(BaseModel):
    """Describes a video provider's capabilities for frontend adaptation."""

    name: str
    display_name: str
    supported_modes: list[VideoGenerationMode]
    supports_end_frame: bool = False
    supports_audio: bool = False
    duration_type: str = "enum"  # "range" or "enum"
    duration_min: int | None = None  # For range type
    duration_max: int | None = None  # For range type
    duration_values: list[int] = []  # For enum type
    duration_default: int = 5
    resolutions: list[str] = []  # e.g., ["720p", "1080p"]
    aspect_ratios: list[str] = []  # e.g., ["16:9", "9:16"]
    max_sample_count: int = 1
    max_concurrent: int = 2


# Constants
MAX_CONCURRENT_VIDEO_GENERATIONS = 2
POLL_INITIAL_INTERVAL = 5  # seconds
POLL_MAX_INTERVAL = 30  # seconds
POLL_MAX_ATTEMPTS = 360  # ~30 minutes max wait


class VideoProvider(ABC):
    """Abstract base class for video generation providers."""

    @property
    @abstractmethod
    def name(self) -> str:
        """Provider identifier."""

    @property
    @abstractmethod
    def display_name(self) -> str:
        """Human-readable provider name."""

    @property
    def supported_modes(self) -> list[VideoGenerationMode]:
        """Supported generation modes. Override to restrict."""
        return [VideoGenerationMode.IMAGE_TO_VIDEO]

    @property
    @abstractmethod
    def capabilities(self) -> VideoProviderCapabilities:
        """Return provider capabilities for frontend adaptation."""

    @property
    def max_concurrent(self) -> int:
        """Maximum concurrent video generations allowed."""
        return MAX_CONCURRENT_VIDEO_GENERATIONS

    @abstractmethod
    async def submit_generation(self, params: VideoGenerationParams) -> str:
        """Submit video generation request, return provider task_id."""

    @abstractmethod
    async def poll_result(self, provider_task_id: str) -> dict:
        """Poll for generation result.

        Returns dict with:
        - status: "queued" | "processing" | "completed" | "failed"
        - progress_percent: int (0-100)
        - eta: int (seconds)
        - videos: list[dict] (when completed, each has video_url, video_type)
        - error: str (when failed)
        """

    async def generate(
        self,
        params: VideoGenerationParams,
        index: int = 0,
        on_progress: callable = None,
    ) -> VideoGenerationResult:
        """Full generation cycle: submit -> poll -> download -> return result."""
        import asyncio

        params.validate_mode(self.supported_modes)

        # Submit
        provider_task_id = await self.submit_generation(params)
        logger.info(
            "[%s] Submitted video generation, task_id=%s",
            self.name,
            provider_task_id,
        )

        # Poll with exponential backoff
        interval = POLL_INITIAL_INTERVAL
        for _attempt in range(POLL_MAX_ATTEMPTS):
            await asyncio.sleep(interval)
            result = await self.poll_result(provider_task_id)
            status = result.get("status", "unknown")

            if on_progress:
                await on_progress(
                    index=index,
                    progress_percent=result.get("progress_percent", 0),
                    eta=result.get("eta", 0),
                    status=status,
                )

            if status == "completed":
                videos = result.get("videos", [])
                if not videos:
                    from ...core.exceptions import ProviderError

                    raise ProviderError(
                        message=(
                            f"[{self.name}] Video generation completed but no videos returned"
                        ),
                        provider=self.name,
                    )

                video_info = videos[0]
                video_url = video_info.get("video_url", "")

                # Download and save locally
                local_path = await self._download_video(video_url, index)

                return VideoGenerationResult(
                    url=video_url,
                    local_path=local_path,
                    video_type=video_info.get("video_type", "mp4"),
                    duration=params.duration,
                    provider_task_id=provider_task_id,
                    metadata=result.get("extra", {}),
                )

            if status == "failed":
                error = result.get("error", "Unknown error")
                from ...core.exceptions import ProviderError

                raise ProviderError(
                    message=f"[{self.name}] Video generation failed: {error}",
                    provider=self.name,
                )

            # Exponential backoff
            interval = min(interval * 1.5, POLL_MAX_INTERVAL)

        from ...core.exceptions import ProviderError

        raise ProviderError(
            message=f"[{self.name}] Video generation timed out after polling",
            provider=self.name,
        )

    async def generate_batch(
        self,
        params: VideoGenerationParams,
        count: int,
        on_video_ready: callable = None,
        on_progress: callable = None,
    ) -> list[VideoGenerationResult]:
        """Generate multiple videos concurrently with semaphore limiting."""
        import asyncio

        semaphore = asyncio.Semaphore(self.max_concurrent)
        results: list[VideoGenerationResult] = []

        async def _generate_one(idx: int) -> VideoGenerationResult:
            async with semaphore:
                result = await self.generate(params, index=idx, on_progress=on_progress)
                if on_video_ready:
                    await on_video_ready(idx, result)
                return result

        tasks = [_generate_one(i) for i in range(count)]
        for coro in asyncio.as_completed(tasks):
            try:
                result = await coro
                results.append(result)
            except Exception as e:
                logger.error("[%s] Video generation error: %s", self.name, e)

        return results

    async def _download_video(self, video_url: str, index: int) -> str | None:
        """Download video from URL and save to local storage."""
        import uuid
        from pathlib import Path

        import os

        import httpx

        from ...core.config import settings

        if settings.storage_type != "local":
            return None

        try:
            video_dir = Path(settings.storage_path).parent / "videos"
            video_dir.mkdir(parents=True, exist_ok=True)

            filename = f"video_{uuid.uuid4().hex[:12]}_{index}.mp4"
            filepath = video_dir / filename

            proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
            async with httpx.AsyncClient(timeout=120.0, proxy=proxy) as client:
                response = await client.get(video_url)
                response.raise_for_status()
                filepath.write_bytes(response.content)

            logger.info("[%s] Video saved to %s", self.name, filepath)
            return str(filepath)
        except Exception as e:
            logger.warning("[%s] Failed to download video: %s", self.name, e)
            return None

    async def health_check(self) -> bool:
        """Check if the provider is available."""
        return True
