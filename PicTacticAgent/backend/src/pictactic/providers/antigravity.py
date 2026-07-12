"""Antigravity reverse proxy provider (OpenAI-compatible API)."""

import asyncio
import base64
import io
import re
import uuid
from pathlib import Path

from ..core.config import settings
from ..core.constants import MAX_CONCURRENT_GENERATIONS
from ..core.exceptions import ProviderError
from .base import GenerationMode, GenerationParams, GenerationResult, ImageProvider

# aspect_ratio -> OpenAI size string
ASPECT_RATIO_TO_SIZE = {
    "1:1": "1024x1024",
    "16:9": "1280x720",
    "9:16": "720x1280",
    "4:3": "1216x896",
    "3:4": "896x1216",
}


class AntigravityImageProvider(ImageProvider):
    """Image generation via Antigravity OpenAI-compatible reverse proxy.

    Only supports text-to-image generation (no image editing).
    """

    def __init__(
        self,
        api_key: str | None = None,
        base_url: str | None = None,
        model: str | None = None,
    ):
        self.api_key = api_key or settings.antigravity_api_key
        self.base_url = base_url or settings.antigravity_api_base_url
        self.model = model or settings.antigravity_model
        self._client = None

    @property
    def name(self) -> str:
        return "antigravity"

    @property
    def supports_batch(self) -> bool:
        return True

    @property
    def max_concurrent(self) -> int:
        return MAX_CONCURRENT_GENERATIONS

    @property
    def supported_modes(self) -> list[GenerationMode]:
        return [GenerationMode.TEXT_TO_IMAGE]

    def _get_client(self):
        """Get or create OpenAI client (sync, thread-safe lazy init)."""
        if self._client is None:
            from openai import OpenAI
            self._client = OpenAI(
                base_url=self.base_url,
                api_key=self.api_key,
            )
        return self._client

    @staticmethod
    def _parse_image_data(content: str) -> bytes:
        """Extract image bytes from response content.

        Handles formats:
        - ![image](data:image/jpeg;base64,...)  (markdown)
        - data:image/jpeg;base64,...             (data URI)
        - raw base64 string
        """
        # Try markdown format: ![...](data:image/...;base64,...)
        md_match = re.search(r'!\[.*?\]\(data:image/[^;]+;base64,([A-Za-z0-9+/=\s]+)\)', content)
        if md_match:
            return base64.b64decode(md_match.group(1))

        # Try data URI
        uri_match = re.search(r'data:image/[^;]+;base64,([A-Za-z0-9+/=\s]+)', content)
        if uri_match:
            return base64.b64decode(uri_match.group(1))

        # Try raw base64 (if content is mostly base64 chars)
        stripped = content.strip()
        if len(stripped) > 1000 and re.fullmatch(r'[A-Za-z0-9+/=\s]+', stripped[:200]):
            return base64.b64decode(stripped)

        raise ValueError("Could not extract image data from response")

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
        """Generate an image using the Antigravity API."""
        params.validate_mode(self.supported_modes)

        size_str = ASPECT_RATIO_TO_SIZE.get(params.aspect_ratio, "1024x1024")

        max_retries = 3
        last_error = None

        for attempt in range(max_retries):
            try:
                client = self._get_client()

                # Run sync OpenAI call in thread pool
                response = await asyncio.to_thread(
                    client.chat.completions.create,
                    model=self.model,
                    messages=[{"role": "user", "content": params.prompt}],
                    extra_body={"size": size_str},
                )

                content = response.choices[0].message.content
                if not content:
                    raise ProviderError(
                        message="Antigravity API returned empty response",
                        provider="antigravity",
                    )

                # Parse image data and save locally
                image_bytes = self._parse_image_data(content)
                local_path = self._save_image_bytes(image_bytes, index)
                width, height = self._extract_dimensions(image_bytes)

                metadata = {
                    "provider": "antigravity",
                    "model": self.model,
                    "aspect_ratio": params.aspect_ratio,
                    "size": size_str,
                    "index": index,
                }
                if width and height:
                    metadata["width"] = width
                    metadata["height"] = height

                # Serve via local static path
                url = f"/static/images/{Path(local_path).name}" if local_path else ""

                return GenerationResult(
                    url=url,
                    local_path=local_path,
                    revised_prompt=None,
                    metadata=metadata,
                )

            except ProviderError:
                raise
            except Exception as e:
                last_error = e
                if attempt < max_retries - 1:
                    await asyncio.sleep(2 ** attempt)
                continue

        raise ProviderError(
            message=f"Antigravity generation failed after {max_retries} retries: {last_error}",
            provider="antigravity",
            details={"error": str(last_error)},
        )

    def _save_image_bytes(self, data: bytes, index: int) -> str | None:
        """Save image bytes to local storage."""
        if settings.storage_type != "local":
            return None

        storage_path = Path(settings.storage_path)
        storage_path.mkdir(parents=True, exist_ok=True)

        image_id = str(uuid.uuid4())[:8]
        filename = f"antigravity_{image_id}_{index}.jpg"
        file_path = storage_path / filename

        with open(file_path, "wb") as f:
            f.write(data)

        from ..services.thumbnail import generate_thumbnail

        generate_thumbnail(file_path)

        return str(file_path)

    async def health_check(self) -> bool:
        """Check if the API is accessible."""
        if not self.api_key:
            return False
        try:
            self._get_client()
            return True
        except Exception:
            return False

    async def close(self):
        """Clean up client resources."""
        self._client = None
