"""Official Google Gemini SDK image generation provider."""

import asyncio
import base64
import io
import uuid
from pathlib import Path

from ..core.config import settings
from ..core.constants import MAX_CONCURRENT_GENERATIONS
from ..core.exceptions import ProviderError
from ..i18n import get_text
from .base import GenerationMode, GenerationParams, GenerationResult, ImageProvider


class GeminiImageProvider(ImageProvider):
    """Official Google Gemini SDK image generation provider.

    Uses the google-genai SDK to interact with the Gemini API.
    Supports both text-to-image and image-to-image generation.

    Models:
    - gemini-2.5-flash-image: Fast generation, 1024px
    - gemini-3-pro-image-preview: High quality, supports 1K/2K/4K and thinking mode
    """

    def __init__(
        self,
        api_key: str | None = None,
        model: str | None = None,
    ):
        self.api_key = api_key or settings.gemini_api_key
        self.model = model or settings.gemini_model
        self._client = None

    @property
    def name(self) -> str:
        return "gemini"

    @property
    def supports_batch(self) -> bool:
        return True

    @property
    def max_concurrent(self) -> int:
        return MAX_CONCURRENT_GENERATIONS

    @property
    def supported_modes(self) -> list[GenerationMode]:
        return [GenerationMode.TEXT_TO_IMAGE, GenerationMode.IMAGE_TO_IMAGE]

    def _get_client(self):
        """Get or create Google GenAI client (lazy init)."""
        if self._client is None:
            from google import genai

            self._client = genai.Client(api_key=self.api_key)
        return self._client

    def _load_pil_image(self, image_data: str):
        """Load image from URL or base64 data URI into PIL Image.

        Args:
            image_data: Either a URL or a data URI (data:image/...;base64,...)

        Returns:
            PIL.Image object
        """
        from PIL import Image

        if image_data.startswith("data:"):
            # Extract base64 from data URI
            base64_data = image_data.split(",", 1)[1] if "," in image_data else image_data
            image_bytes = base64.b64decode(base64_data)
            return Image.open(io.BytesIO(image_bytes))
        else:
            # It's a URL, download it
            import httpx

            response = httpx.get(image_data, timeout=30.0)
            response.raise_for_status()
            return Image.open(io.BytesIO(response.content))

    @staticmethod
    def _extract_dimensions(image_bytes: bytes) -> tuple[int | None, int | None]:
        """Extract width and height from raw image bytes."""
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
        """Generate an image using the official Gemini SDK.

        Args:
            params: Generation parameters
            index: Index for batch operations

        Returns:
            GenerationResult with local path and metadata
        """
        from google.genai import types

        mode = params.get_mode()
        client = self._get_client()

        # Build contents
        contents: list = [params.prompt]

        # For image-to-image, add reference images
        if mode == GenerationMode.IMAGE_TO_IMAGE and params.reference_images:
            for img_data in params.reference_images:
                try:
                    pil_image = await asyncio.to_thread(self._load_pil_image, img_data)
                    contents.append(pil_image)
                except Exception as e:
                    raise ProviderError(
                        message=get_text("provider.gemini.image_load_error", error=str(e)),
                        provider="gemini",
                        details={"error": str(e)},
                    )

        # Build image config
        image_config_kwargs = {}
        if params.aspect_ratio:
            image_config_kwargs["aspect_ratio"] = params.aspect_ratio

        # Map size to image_size (only for gemini-3-pro-image-preview)
        if params.size and "pro" in self.model.lower():
            image_config_kwargs["image_size"] = params.size

        # Build generation config
        config = types.GenerateContentConfig(
            response_modalities=["IMAGE"],
            image_config=types.ImageConfig(**image_config_kwargs) if image_config_kwargs else None,
        )

        # Retry logic
        max_retries = 3
        last_error = None

        for attempt in range(max_retries):
            try:
                # Call API in thread pool (SDK is sync)
                response = await asyncio.to_thread(
                    client.models.generate_content,
                    model=self.model,
                    contents=contents,
                    config=config,
                )

                # Parse response
                if not response.parts:
                    raise ProviderError(
                        message=get_text("provider.gemini.no_images"),
                        provider="gemini",
                        details={"response": str(response)},
                    )

                # Find image in response parts
                for part in response.parts:
                    if hasattr(part, "inline_data") and part.inline_data is not None:
                        # Get raw image bytes from inline_data
                        image_bytes = part.inline_data.data

                        # Save locally
                        local_path = await asyncio.to_thread(
                            self._save_image_bytes, image_bytes, index
                        )
                        width, height = await asyncio.to_thread(
                            self._extract_dimensions, image_bytes
                        )

                        metadata = {
                            "provider": "gemini-official",
                            "model": self.model,
                            "aspect_ratio": params.aspect_ratio,
                            "size": params.size,
                            "index": index,
                        }
                        if width and height:
                            metadata["width"] = width
                            metadata["height"] = height

                        # Build URL for serving
                        url = f"/static/images/{Path(local_path).name}" if local_path else ""

                        return GenerationResult(
                            url=url,
                            local_path=local_path,
                            revised_prompt=None,
                            metadata=metadata,
                        )

                # No image found in response
                raise ProviderError(
                    message=get_text("provider.gemini.no_images"),
                    provider="gemini",
                    details={"response": str(response)},
                )

            except ProviderError:
                raise
            except Exception as e:
                last_error = e
                error_str = str(e).lower()

                # Check for retryable errors
                retryable = any(
                    keyword in error_str
                    for keyword in ["timeout", "connection", "rate", "429", "500", "502", "503", "504"]
                )

                if retryable and attempt < max_retries - 1:
                    await asyncio.sleep(2 ** attempt)
                    continue

                # Non-retryable or last attempt
                raise ProviderError(
                    message=get_text(
                        "provider.gemini.generation_error",
                        error=str(e),
                    ),
                    provider="gemini",
                    details={"error": str(e)},
                )

        raise ProviderError(
            message=get_text(
                "provider.gemini.network_error",
                retries=max_retries,
                error=str(last_error),
            ),
            provider="gemini",
            details={"error": str(last_error)},
        )

    def _save_image_bytes(self, image_bytes: bytes, index: int) -> str | None:
        """Save raw image bytes to local storage.

        Args:
            image_bytes: Raw image bytes
            index: Image index for filename

        Returns:
            Local file path or None if storage is not local
        """
        if settings.storage_type != "local":
            return None

        storage_path = Path(settings.storage_path)
        storage_path.mkdir(parents=True, exist_ok=True)

        # Detect image format from magic bytes
        ext = "png"  # default
        if image_bytes[:2] == b'\xff\xd8':
            ext = "jpg"
        elif image_bytes[:4] == b'\x89PNG':
            ext = "png"
        elif image_bytes[:4] == b'RIFF' and image_bytes[8:12] == b'WEBP':
            ext = "webp"

        image_id = str(uuid.uuid4())[:8]
        filename = f"gemini_official_{image_id}_{index}.{ext}"
        file_path = storage_path / filename

        # Write raw bytes directly
        with open(file_path, "wb") as f:
            f.write(image_bytes)

        from ..services.thumbnail import generate_thumbnail

        generate_thumbnail(file_path)

        return str(file_path)

    async def health_check(self) -> bool:
        """Check if the Gemini SDK is properly configured."""
        if not self.api_key:
            return False

        try:
            # Try to create client
            self._get_client()
            return True
        except Exception:
            return False

    async def close(self):
        """Clean up resources."""
        self._client = None
