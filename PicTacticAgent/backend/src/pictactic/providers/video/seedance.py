"""Seedance 2.0 video provider implementation (t2v + i2v)."""

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

# Seedance-specific constants
SEEDANCE_ENDPOINT = "/async/seedance-2.0"
SEEDANCE_POLL_ENDPOINT = "/async/task-result"
SEEDANCE_HTTP_TIMEOUT = 300.0
SEEDANCE_MAX_RETRIES = 3

# Task status mapping from jiekou.ai API to our internal status
_STATUS_MAP = {
    "TASK_STATUS_QUEUED": "queued",
    "TASK_STATUS_PROCESSING": "processing",
    "TASK_STATUS_SUCCEED": "completed",
    "TASK_STATUS_FAILED": "failed",
}


class SeedanceVideoProvider(VideoProvider):
    """Seedance 2.0 video provider (via jiekou.ai proxy)."""

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
        return "seedance"

    @property
    def display_name(self) -> str:
        return "Seedance 2.0"

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
            duration_type="range",
            duration_min=4,
            duration_max=15,
            duration_default=5,
            resolutions=["480p", "720p"],
            aspect_ratios=["16:9", "4:3", "1:1", "3:4", "9:16", "21:9"],
            max_concurrent=self.max_concurrent,
        )

    async def _get_client(self) -> httpx.AsyncClient:
        """Get or create HTTP client with lazy initialization."""
        if self._client is None:
            proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
            self._client = httpx.AsyncClient(
                timeout=SEEDANCE_HTTP_TIMEOUT,
                proxy=proxy,
            )
        return self._client

    async def submit_generation(self, params: VideoGenerationParams) -> str:
        """Submit Seedance video generation request (t2v or i2v via single endpoint)."""
        params.validate_mode(self.supported_modes)

        duration = max(4, min(15, params.duration))

        request_body: dict = {
            "prompt": params.prompt,
            "ratio": params.aspect_ratio,
            "duration": duration,
            "generate_audio": params.generate_audio,
            "resolution": params.resolution or "720p",
        }

        if params.start_image:
            request_body["image"] = params.start_image
        if params.end_image:
            request_body["last_image"] = params.end_image

        if params.reference_images:
            request_body["reference_images"] = [
                {"image": img} for img in params.reference_images[:9]
            ]

        if params.seed is not None:
            request_body["seed"] = params.seed

        # Retry logic with exponential backoff
        last_error: Exception | None = None
        for attempt in range(SEEDANCE_MAX_RETRIES):
            try:
                client = await self._get_client()
                response = await client.post(
                    f"{self.base_url}{SEEDANCE_ENDPOINT}",
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
                        message="Seedance API did not return a task_id",
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
                if retryable and attempt < SEEDANCE_MAX_RETRIES - 1:
                    last_error = e
                    await asyncio.sleep(2**attempt)
                    continue

                raise ProviderError(
                    message=f"Seedance API error (HTTP {status_code}): {error_detail}",
                    provider=self.name,
                    details={"status_code": status_code, "response": e.response.text},
                ) from e

            except httpx.RequestError as e:
                last_error = e
                if self._client:
                    await self._client.aclose()
                    self._client = None
                if attempt < SEEDANCE_MAX_RETRIES - 1:
                    await asyncio.sleep(2**attempt)
                    continue

        raise ProviderError(
            message=f"Seedance API request failed after {SEEDANCE_MAX_RETRIES} retries: {last_error}",
            provider=self.name,
            details={"error": str(last_error)},
        )

    async def poll_result(self, provider_task_id: str) -> dict:
        """Poll Seedance async task result."""
        last_error: Exception | None = None
        for attempt in range(SEEDANCE_MAX_RETRIES):
            try:
                client = await self._get_client()
                response = await client.get(
                    f"{self.base_url}{SEEDANCE_POLL_ENDPOINT}",
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
                    result["extra"] = data.get("extra", {})
                elif status == "failed":
                    result["error"] = task_info.get("reason", "Unknown error")

                return result

            except httpx.HTTPStatusError as e:
                status_code = e.response.status_code
                retryable = status_code in {429, 500, 502, 503, 504}
                if retryable and attempt < SEEDANCE_MAX_RETRIES - 1:
                    last_error = e
                    await asyncio.sleep(2**attempt)
                    continue
                raise ProviderError(
                    message=f"Seedance poll error (HTTP {status_code})",
                    provider=self.name,
                    details={"status_code": status_code},
                ) from e

            except httpx.RequestError as e:
                last_error = e
                if self._client:
                    await self._client.aclose()
                    self._client = None
                if attempt < SEEDANCE_MAX_RETRIES - 1:
                    await asyncio.sleep(2**attempt)
                    continue

        raise ProviderError(
            message=f"Seedance poll request failed after {SEEDANCE_MAX_RETRIES} retries: {last_error}",
            provider=self.name,
            details={"error": str(last_error)},
        )

    async def health_check(self) -> bool:
        """Check if the Seedance API is accessible."""
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
