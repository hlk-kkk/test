"""Veo 3.1 video provider implementation (t2v + i2v)."""

import asyncio
import logging
import os

import httpx

from ...core.config import settings
from ...core.exceptions import ProviderError
from .base import (
    VideoGenerationMode,
    VideoGenerationParams,
    VideoProvider,
    VideoProviderCapabilities,
)

logger = logging.getLogger(__name__)

# Veo-specific constants
VEO_I2V_ENDPOINT = "/async/veo-3.1-generate-img2video"
VEO_T2V_ENDPOINT = "/async/veo-3.1-generate-text2video"
VEO_POLL_ENDPOINT = "/async/task-result"
VEO_HTTP_TIMEOUT = 300.0
VEO_MAX_RETRIES = 3

# Valid duration values for Veo
VEO_VALID_DURATIONS = [4, 6, 8]

# Task status mapping from Veo API to our internal status
_STATUS_MAP = {
    "TASK_STATUS_QUEUED": "queued",
    "TASK_STATUS_PROCESSING": "processing",
    "TASK_STATUS_SUCCEED": "completed",
    "TASK_STATUS_FAILED": "failed",
}


class VeoVideoProvider(VideoProvider):
    """Veo 3.1 video provider (via jiekou.ai proxy)."""

    def __init__(
        self,
        api_key: str | None = None,
        base_url: str | None = None,
    ):
        self.api_key = api_key or settings.jiekouai_api_key
        self.base_url = base_url or settings.jiekouai_api_base_url
        self._client: httpx.AsyncClient | None = None

    @property
    def name(self) -> str:
        return "veo"

    @property
    def display_name(self) -> str:
        return "Veo 3.1"

    @property
    def supported_modes(self) -> list[VideoGenerationMode]:
        return [VideoGenerationMode.TEXT_TO_VIDEO, VideoGenerationMode.IMAGE_TO_VIDEO]

    @property
    def capabilities(self) -> VideoProviderCapabilities:
        return VideoProviderCapabilities(
            name=self.name,
            display_name=self.display_name,
            supported_modes=self.supported_modes,
            supports_end_frame=True,
            supports_audio=True,
            duration_type="enum",
            duration_values=VEO_VALID_DURATIONS,
            duration_default=8,
            resolutions=["720p", "1080p"],
            aspect_ratios=["16:9", "9:16"],
            max_sample_count=4,
            max_concurrent=self.max_concurrent,
        )

    async def _get_client(self) -> httpx.AsyncClient:
        """Get or create HTTP client with lazy initialization."""
        if self._client is None:
            proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
            self._client = httpx.AsyncClient(
                timeout=VEO_HTTP_TIMEOUT,
                proxy=proxy,
            )
        return self._client

    @staticmethod
    def _snap_duration(duration: int) -> int:
        """Snap the requested duration to the nearest valid Veo duration value."""
        return min(VEO_VALID_DURATIONS, key=lambda v: abs(v - duration))

    async def submit_generation(self, params: VideoGenerationParams) -> str:
        """Submit Veo video generation request (t2v or i2v)."""
        params.validate_mode(self.supported_modes)

        mode = params.get_mode()
        duration = self._snap_duration(params.duration)

        request_body: dict = {
            "prompt": params.prompt,
            "aspect_ratio": params.aspect_ratio,
            "duration_seconds": duration,
            "enhance_prompt": params.enhance_prompt,
            "generate_audio": params.generate_audio,
            "person_generation": "allow_adult",
            "resolution": params.resolution or "720p",
            "sample_count": max(1, min(4, params.sample_count)),
        }

        if mode == VideoGenerationMode.IMAGE_TO_VIDEO:
            endpoint = VEO_I2V_ENDPOINT
            if params.start_image:
                request_body["image"] = params.start_image
            if params.end_image:
                request_body["last_image"] = params.end_image
            if params.reference_images:
                ref_images = [
                    {"image": img, "reference_type": "asset"} for img in params.reference_images[:3]
                ]
                request_body["reference_images"] = ref_images
        else:
            endpoint = VEO_T2V_ENDPOINT

        if params.negative_prompt:
            request_body["negative_prompt"] = params.negative_prompt

        if params.seed is not None:
            request_body["seed"] = params.seed

        # Retry logic with exponential backoff
        last_error: Exception | None = None
        for attempt in range(VEO_MAX_RETRIES):
            try:
                client = await self._get_client()
                response = await client.post(
                    f"{self.base_url}{endpoint}",
                    headers={
                        "Authorization": f"Bearer {self.api_key}",
                        "Content-Type": "application/json",
                    },
                    json=request_body,
                )
                response.raise_for_status()
                data = response.json()

                task_id = data.get("task_id")
                if not task_id:
                    raise ProviderError(
                        message="Veo API did not return a task_id",
                        provider=self.name,
                        details={"response": data},
                    )

                logger.info("[%s] Submitted generation, task_id=%s", self.name, task_id)
                return task_id

            except httpx.HTTPStatusError as e:
                status_code = e.response.status_code
                error_detail = ""
                try:
                    error_data = e.response.json()
                    error_detail = error_data.get("error", error_data.get("message", ""))
                except Exception:
                    error_detail = e.response.text[:200] if e.response.text else ""

                retryable = status_code in {429, 500, 502, 503, 504}
                if retryable and attempt < VEO_MAX_RETRIES - 1:
                    last_error = e
                    await asyncio.sleep(2**attempt)
                    continue

                raise ProviderError(
                    message=f"Veo API error (HTTP {status_code}): {error_detail}",
                    provider=self.name,
                    details={"status_code": status_code, "response": e.response.text},
                ) from e

            except httpx.RequestError as e:
                last_error = e
                if self._client:
                    await self._client.aclose()
                    self._client = None
                if attempt < VEO_MAX_RETRIES - 1:
                    await asyncio.sleep(2**attempt)
                    continue

        raise ProviderError(
            message=(f"Veo API request failed after {VEO_MAX_RETRIES} retries: {last_error}"),
            provider=self.name,
            details={"error": str(last_error)},
        )

    async def poll_result(self, provider_task_id: str) -> dict:
        """Poll Veo async task result."""
        last_error: Exception | None = None
        for attempt in range(VEO_MAX_RETRIES):
            try:
                client = await self._get_client()
                response = await client.get(
                    f"{self.base_url}{VEO_POLL_ENDPOINT}",
                    params={"task_id": provider_task_id},
                    headers={
                        "Authorization": f"Bearer {self.api_key}",
                    },
                )
                response.raise_for_status()
                data = response.json()

                task_info = data.get("task", {})
                raw_status = task_info.get("status", "")
                status = _STATUS_MAP.get(raw_status, "processing")

                result: dict = {
                    "status": status,
                    "progress_percent": task_info.get("progress_percent", 0),
                    "eta": task_info.get("eta", 0),
                }

                if status == "completed":
                    result["videos"] = data.get("videos", [])
                elif status == "failed":
                    result["error"] = task_info.get("reason", "Unknown error")

                return result

            except httpx.HTTPStatusError as e:
                status_code = e.response.status_code
                retryable = status_code in {429, 500, 502, 503, 504}
                if retryable and attempt < VEO_MAX_RETRIES - 1:
                    last_error = e
                    await asyncio.sleep(2**attempt)
                    continue
                raise ProviderError(
                    message=f"Veo poll error (HTTP {status_code})",
                    provider=self.name,
                    details={"status_code": status_code},
                ) from e

            except httpx.RequestError as e:
                last_error = e
                if self._client:
                    await self._client.aclose()
                    self._client = None
                if attempt < VEO_MAX_RETRIES - 1:
                    await asyncio.sleep(2**attempt)
                    continue

        raise ProviderError(
            message=(f"Veo poll request failed after {VEO_MAX_RETRIES} retries: {last_error}"),
            provider=self.name,
            details={"error": str(last_error)},
        )

    async def health_check(self) -> bool:
        """Check if the Veo API is accessible."""
        if not self.api_key:
            return False
        try:
            client = await self._get_client()
            response = await client.get(
                f"{self.base_url}/health",
                headers={"Authorization": f"Bearer {self.api_key}"},
                timeout=10.0,
            )
            return response.status_code < 500
        except Exception:
            return False

    async def close(self):
        """Close the HTTP client."""
        if self._client:
            await self._client.aclose()
            self._client = None
