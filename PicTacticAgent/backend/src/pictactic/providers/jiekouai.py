"""Jiekouai image provider implementation (Gemini API proxy)."""

import asyncio
import io
import logging
import os
import uuid
from pathlib import Path

import httpx

from ..core.config import settings
from ..core.constants import GENERATION_HTTP_TIMEOUT, IMAGE_DOWNLOAD_TIMEOUT, MAX_CONCURRENT_GENERATIONS
from ..core.exceptions import ProviderError
from ..i18n import get_text
from .base import GenerationMode, GenerationParams, GenerationResult, ImageProvider

logger = logging.getLogger(__name__)


class JiekouaiImageProvider(ImageProvider):
    """Jiekouai image generation provider (proxies Gemini API)."""

    def __init__(self, api_key: str | None = None, base_url: str | None = None):
        self.api_key = api_key or settings.jiekouai_api_key or settings.gemini_api_key
        self.base_url = base_url or settings.jiekouai_api_base_url
        self._client: httpx.AsyncClient | None = None

    @property
    def name(self) -> str:
        return "jiekouai"

    @property
    def supports_batch(self) -> bool:
        return True

    @property
    def max_concurrent(self) -> int:
        return MAX_CONCURRENT_GENERATIONS

    async def _get_client(self) -> httpx.AsyncClient:
        """Get or create HTTP client."""
        if self._client is None:
            # Get proxy from environment
            proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
            self._client = httpx.AsyncClient(
                timeout=GENERATION_HTTP_TIMEOUT,
                proxy=proxy,
            )
        return self._client

    @staticmethod
    def _extract_dimensions(image_bytes: bytes) -> tuple[int | None, int | None]:
        """Extract width and height from downloaded image bytes."""
        from PIL import Image

        try:
            with Image.open(io.BytesIO(image_bytes)) as img:
                return int(img.width), int(img.height)
        except Exception:
            return None, None

    async def generate(
        self,
        params: GenerationParams,
        index: int = 0,
    ) -> GenerationResult:
        """
        Generate an image using Gemini 3 Pro Image API.

        Uses different endpoints based on generation mode:
        - Text-to-Image: POST {base_url}/gemini-3-pro-image-text-to-image
        - Image-to-Image: POST {base_url}/gemini-3-pro-image-edit
        """
        mode = params.get_mode()

        if mode == GenerationMode.TEXT_TO_IMAGE:
            # Text-to-image mode: no reference images needed
            request_body = {
                "prompt": params.prompt,
                "aspect_ratio": params.aspect_ratio,
                "size": params.size,
            }
            endpoint = "/gemini-3-pro-image-text-to-image"
        else:
            # Image-to-image mode: requires reference images
            # 区分 URL 和 base64
            image_urls = []
            image_base64s = []
            for img in params.reference_images:
                if img.startswith("data:"):
                    # 提取纯 base64（去掉 data:image/xxx;base64, 前缀）
                    base64_data = img.split(",", 1)[1] if "," in img else img
                    image_base64s.append(base64_data)
                else:
                    image_urls.append(img)

            request_body = {
                "prompt": params.prompt,
                "aspect_ratio": params.aspect_ratio,
                "size": params.size,
            }
            if image_urls:
                request_body["image_urls"] = image_urls
            if image_base64s:
                request_body["image_base64s"] = image_base64s
            endpoint = "/gemini-3-pro-image-edit"

        if params.negative_prompt:
            request_body["negative_prompt"] = params.negative_prompt

        if params.style:
            request_body["style"] = params.style

        # Retry logic for network errors
        max_retries = 3
        last_error = None

        for attempt in range(max_retries):
            try:
                # Create fresh client for each attempt
                client = await self._get_client()

                # Log payload size for debugging oversized-request issues
                if "image_base64s" in request_body:
                    total_b64_kb = sum(
                        len(b) for b in request_body["image_base64s"]
                    ) // 1024
                    logger.info(
                        "Sending %s request with %d base64 image(s), ~%d KB total",
                        endpoint,
                        len(request_body["image_base64s"]),
                        total_b64_kb,
                    )

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

                image_urls = data.get("image_urls", [])
                if not image_urls:
                    raise ProviderError(
                        message=get_text("provider.gemini.no_images"),
                        provider="gemini",
                        details={"response": data},
                    )

                # Save image locally if storage is configured
                local_path, width, height = await self._save_image(image_urls[0], index)

                metadata = {
                    "provider": "gemini-3-pro-image",
                    "aspect_ratio": params.aspect_ratio,
                    "size": params.size,
                    "index": index,
                }
                if width and height:
                    metadata["width"] = width
                    metadata["height"] = height

                # Prefer local static path over CDN URL
                url = (
                    f"/static/images/{Path(local_path).name}"
                    if local_path
                    else image_urls[0]
                )

                return GenerationResult(
                    url=url,
                    local_path=local_path,
                    revised_prompt=data.get("revised_prompt"),
                    metadata=metadata,
                )

            except httpx.HTTPStatusError as e:
                status_code = e.response.status_code

                # Try to extract error message from response
                error_detail = ""
                try:
                    error_data = e.response.json()
                    error_detail = error_data.get("error", error_data.get("message", ""))
                except Exception:
                    error_detail = e.response.text[:200] if e.response.text else ""

                # Provide user-friendly error messages
                if status_code == 500:
                    message = get_text("provider.gemini.server_error")
                elif status_code == 503:
                    message = get_text("provider.gemini.unavailable")
                elif status_code == 401:
                    message = get_text("provider.gemini.auth_failed")
                elif status_code == 429:
                    message = get_text("provider.gemini.rate_limited")
                elif status_code >= 400 and status_code < 500:
                    message = get_text("provider.gemini.client_error", status_code=status_code)
                else:
                    message = get_text("provider.gemini.generic_error", status_code=status_code)

                if error_detail:
                    message = f"{message}: {error_detail}"

                retryable = status_code in {429, 500, 502, 503, 504}
                if retryable and attempt < max_retries - 1:
                    last_error = e
                    await asyncio.sleep(2 ** attempt)
                    continue

                raise ProviderError(
                    message=message,
                    provider="gemini",
                    details={"status_code": status_code, "response": e.response.text},
                )
            except httpx.RequestError as e:
                last_error = e
                # Reset client on connection error
                if self._client:
                    await self._client.aclose()
                    self._client = None
                if attempt < max_retries - 1:
                    await asyncio.sleep(2 ** attempt)  # Exponential backoff
                continue

        raise ProviderError(
            message=get_text("provider.gemini.network_error", retries=max_retries, error=str(last_error)),
            provider="gemini",
            details={"error": str(last_error)},
        )

    async def _save_image(self, url: str, index: int) -> tuple[str | None, int | None, int | None]:
        """Download image, extract dimensions, and persist as JPEG locally."""
        try:
            from PIL import Image

            client = await self._get_client()
            response = await client.get(url)
            response.raise_for_status()
            image_bytes = response.content
            width, height = self._extract_dimensions(image_bytes)

            if settings.storage_type != "local":
                return None, width, height

            storage_path = Path(settings.storage_path)
            storage_path.mkdir(parents=True, exist_ok=True)

            image_id = str(uuid.uuid4())[:8]
            filename = f"gemini_{image_id}_{index}.jpg"
            file_path = storage_path / filename

            # Re-encode as JPEG to reduce file size for subsequent edit operations
            img = Image.open(io.BytesIO(image_bytes))
            if img.mode in ("RGBA", "LA", "P"):
                img = img.convert("RGB")
            img.save(file_path, format="JPEG", quality=92, optimize=True)

            from ..services.thumbnail import generate_thumbnail

            generate_thumbnail(file_path)

            return str(file_path), width, height
        except Exception:
            return None, None, None

    async def health_check(self) -> bool:
        """Check if Gemini API is accessible."""
        if not self.api_key:
            return False

        try:
            client = await self._get_client()
            # Simple health check - just verify we can reach the API
            response = await client.get(
                f"{self.base_url}/health",
                headers={"Authorization": f"Bearer {self.api_key}"},
                timeout=IMAGE_DOWNLOAD_TIMEOUT,
            )
            return response.status_code < 500
        except Exception:
            return False

    async def close(self):
        """Close the HTTP client."""
        if self._client:
            await self._client.aclose()
            self._client = None
