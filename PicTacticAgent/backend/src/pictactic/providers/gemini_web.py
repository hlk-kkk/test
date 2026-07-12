"""Gemini Web reverse-proxy image generation provider.

Uses the gemini-webapi library to interact with gemini.google.com via cookies,
bypassing the official API and using the consumer web interface directly.
"""

import asyncio
import logging
import uuid
from pathlib import Path

from ..core.config import settings
from ..core.constants import MAX_CONCURRENT_GENERATIONS
from ..core.exceptions import ProviderError
from ..i18n import get_text
from .base import GenerationMode, GenerationParams, GenerationResult, ImageProvider

logger = logging.getLogger(__name__)


class GeminiWebImageProvider(ImageProvider):
    """Gemini Web image generation provider (reverse-proxy via cookies).

    Uses the gemini-webapi library to interact with gemini.google.com.
    Authentication is done via Google account cookies (__Secure-1PSID).
    Supports both text-to-image and image-to-image generation.
    """

    def __init__(
        self,
        secure_1psid: str | None = None,
        secure_1psidts: str | None = None,
        proxy: str | None = None,
    ):
        self._secure_1psid = secure_1psid or settings.gemini_web_secure_1psid
        self._secure_1psidts = secure_1psidts or settings.gemini_web_secure_1psidts
        self._proxy = proxy or settings.gemini_web_proxy or None
        self._client = None
        self._init_lock = asyncio.Lock()

    @property
    def name(self) -> str:
        return "gemini_web"

    @property
    def supports_batch(self) -> bool:
        return True

    @property
    def max_concurrent(self) -> int:
        # Gemini Web has stricter rate limits than API; keep concurrency low
        return min(MAX_CONCURRENT_GENERATIONS, 2)

    @property
    def supported_modes(self) -> list[GenerationMode]:
        return [GenerationMode.TEXT_TO_IMAGE, GenerationMode.IMAGE_TO_IMAGE]

    async def _get_client(self):
        """Get or create and initialize GeminiClient (lazy, thread-safe)."""
        if self._client is not None:
            return self._client

        async with self._init_lock:
            # Double-check after acquiring lock
            if self._client is not None:
                return self._client

            from gemini_webapi import GeminiClient

            if not self._secure_1psid:
                raise ProviderError(
                    message=get_text("provider.gemini_web.no_cookie"),
                    provider="gemini_web",
                )

            client = GeminiClient(
                secure_1psid=self._secure_1psid,
                secure_1psidts=self._secure_1psidts,
                proxy=self._proxy,
            )
            await client.init(
                timeout=450,
                auto_close=False,
                auto_refresh=True,
                verbose=False,
            )
            self._client = client
            logger.info("GeminiClient initialized successfully")
            return self._client

    def _build_prompt(self, params: GenerationParams) -> str:
        """Build prompt with aspect ratio and size hints embedded.

        The Gemini Web interface doesn't have dedicated params for aspect ratio
        and size, so we embed them as natural language hints in the prompt.
        """
        prompt = params.prompt

        hints = []
        if params.aspect_ratio and params.aspect_ratio != "16:9":
            hints.append(f"aspect ratio {params.aspect_ratio}")
        if params.size:
            size_map = {"1K": "1024px", "2K": "2048px", "4K": "4096px"}
            hints.append(f"resolution {size_map.get(params.size, params.size)}")

        if hints:
            prompt = f"{prompt} ({', '.join(hints)})"

        return prompt

    async def _prepare_reference_files(self, reference_images: list[str]) -> list[bytes]:
        """Convert reference image URLs/data-URIs to bytes for the files parameter."""
        import base64

        import httpx

        files: list[bytes] = []
        for img_data in reference_images:
            try:
                if img_data.startswith("data:"):
                    # data URI -> bytes
                    base64_data = img_data.split(",", 1)[1] if "," in img_data else img_data
                    files.append(base64.b64decode(base64_data))
                elif img_data.startswith(("http://", "https://")):
                    # URL -> download
                    async with httpx.AsyncClient(timeout=30.0) as http:
                        resp = await http.get(img_data)
                        resp.raise_for_status()
                        files.append(resp.content)
                else:
                    # Local file path
                    path = Path(img_data)
                    if path.exists():
                        files.append(path.read_bytes())
                    else:
                        logger.warning("Reference image not found: %s", img_data)
            except Exception as e:
                logger.warning("Failed to load reference image %s: %s", img_data, e)

        return files

    async def generate(
        self,
        params: GenerationParams,
        index: int = 0,
    ) -> GenerationResult:
        """Generate an image using the Gemini Web interface.

        Args:
            params: Generation parameters
            index: Index for batch operations

        Returns:
            GenerationResult with local path and metadata
        """
        from gemini_webapi import GeneratedImage as GeminiGeneratedImage

        client = await self._get_client()
        mode = params.get_mode()
        prompt = self._build_prompt(params)

        # Prepare files for image-to-image mode
        files = None
        if mode == GenerationMode.IMAGE_TO_IMAGE and params.reference_images:
            files = await self._prepare_reference_files(params.reference_images)
            if not files:
                raise ProviderError(
                    message=get_text("provider.gemini.image_load_error", error="No valid reference images"),
                    provider="gemini_web",
                )

        # Determine model
        model_name = settings.gemini_web_model
        model = self._resolve_model(model_name)

        # Retry logic
        max_retries = 3
        last_error = None

        for attempt in range(max_retries):
            try:
                response = await client.generate_content(
                    prompt=prompt,
                    files=files,
                    model=model,
                    temporary=True,
                )

                # Find generated images (not web images)
                generated_images = [
                    img for img in (response.images or [])
                    if isinstance(img, GeminiGeneratedImage)
                ]

                if not generated_images:
                    raise ProviderError(
                        message=get_text("provider.gemini.no_images"),
                        provider="gemini_web",
                        details={"text": response.text or ""},
                    )

                # Take the first generated image
                image = generated_images[0]

                # Save locally
                local_path = await self._save_image(image, index)

                # Extract dimensions
                width, height = None, None
                if local_path:
                    width, height = self._extract_dimensions(local_path)

                metadata = {
                    "provider": "gemini-web",
                    "model": model_name,
                    "aspect_ratio": params.aspect_ratio,
                    "size": params.size,
                    "index": index,
                    "alt": image.alt or "",
                }
                if width and height:
                    metadata["width"] = width
                    metadata["height"] = height

                url = f"/static/images/{Path(local_path).name}" if local_path else image.url

                return GenerationResult(
                    url=url,
                    local_path=local_path,
                    revised_prompt=response.text,
                    metadata=metadata,
                )

            except ProviderError:
                raise
            except Exception as e:
                last_error = e
                error_str = str(e).lower()
                logger.warning(
                    "Gemini Web generation attempt %d/%d failed: %s",
                    attempt + 1, max_retries, e,
                )

                retryable = any(
                    keyword in error_str
                    for keyword in ["timeout", "connection", "rate", "temporarily", "blocked"]
                )

                if retryable and attempt < max_retries - 1:
                    await asyncio.sleep(2 ** attempt)
                    continue

                raise ProviderError(
                    message=get_text("provider.gemini.generation_error", error=str(e)),
                    provider="gemini_web",
                    details={"error": str(e)},
                ) from e

        raise ProviderError(
            message=get_text(
                "provider.gemini.network_error",
                retries=max_retries,
                error=str(last_error),
            ),
            provider="gemini_web",
            details={"error": str(last_error)},
        )

    def _resolve_model(self, model_name: str):
        """Resolve model name string to Model enum."""
        from gemini_webapi.constants import Model

        model_map = {
            "gemini-3-pro": Model.BASIC_PRO,
            "gemini-3-flash": Model.BASIC_FLASH,
            "gemini-3-flash-thinking": Model.BASIC_THINKING,
            "gemini-3-pro-plus": Model.PLUS_PRO,
            "gemini-3-flash-plus": Model.PLUS_FLASH,
        }
        return model_map.get(model_name, Model.BASIC_PRO)

    async def _save_image(self, image, index: int) -> str | None:
        """Save a GeneratedImage to local storage."""
        if settings.storage_type != "local":
            return None

        storage_path = Path(settings.storage_path)
        storage_path.mkdir(parents=True, exist_ok=True)

        image_id = str(uuid.uuid4())[:8]
        filename = f"gemini_web_{image_id}_{index}"

        try:
            # Use the image's built-in save method (fetches full-size)
            saved_path = await image.save(
                path=str(storage_path),
                filename=filename,
                verbose=False,
                full_size=True,
            )

            # Re-encode to JPEG for consistency with other providers
            from PIL import Image as PILImage

            img = PILImage.open(saved_path)
            if img.mode in ("RGBA", "LA", "P"):
                img = img.convert("RGB")

            jpeg_path = storage_path / f"gemini_web_{image_id}_{index}.jpg"
            img.save(str(jpeg_path), format="JPEG", quality=92, optimize=True)

            # Remove original if different from JPEG
            original = Path(saved_path)
            if original != jpeg_path and original.exists():
                original.unlink()

            from ..services.thumbnail import generate_thumbnail

            generate_thumbnail(jpeg_path)

            return str(jpeg_path)
        except Exception as e:
            logger.error("Failed to save Gemini Web image: %s", e)
            return None

    @staticmethod
    def _extract_dimensions(path: str) -> tuple[int | None, int | None]:
        """Extract width and height from saved image file."""
        from PIL import Image as PILImage

        try:
            with PILImage.open(path) as img:
                return int(img.width), int(img.height)
        except Exception:
            return None, None

    async def health_check(self) -> bool:
        """Check if the Gemini Web client is properly configured."""
        if not self._secure_1psid:
            return False
        try:
            await self._get_client()
            return True
        except Exception:
            return False

    async def close(self):
        """Close the Gemini Web client."""
        if self._client:
            await self._client.close()
            self._client = None
