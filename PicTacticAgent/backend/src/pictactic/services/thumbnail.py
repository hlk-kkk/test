"""Thumbnail generation utilities for image optimization."""

import logging
from pathlib import Path

from PIL import Image

from ..core.config import settings

logger = logging.getLogger(__name__)


def generate_thumbnail(
    source_path: str | Path,
    max_size: int | None = None,
    quality: int | None = None,
) -> str | None:
    """Generate a compressed JPEG thumbnail for the given image.

    Args:
        source_path: Path to the source image file.
        max_size: Max dimension (long edge) in pixels. Defaults to settings.thumbnail_max_size.
        quality: JPEG quality (1-100). Defaults to settings.thumbnail_quality.

    Returns:
        Path to the generated thumbnail, or None on failure.
    """
    if max_size is None:
        max_size = settings.thumbnail_max_size
    if quality is None:
        quality = settings.thumbnail_quality

    source = Path(source_path)
    if not source.exists():
        return None

    thumb_path = thumbnail_path_from_original(source)
    try:
        with Image.open(source) as img:
            # Convert to RGB for JPEG compatibility
            if img.mode != "RGB":
                img = img.convert("RGB")

            # Only resize if larger than max_size
            w, h = img.size
            if max(w, h) > max_size:
                img.thumbnail((max_size, max_size), Image.LANCZOS)

            img.save(str(thumb_path), format="JPEG", quality=quality, optimize=True)

        return str(thumb_path)
    except Exception:
        logger.warning("Failed to generate thumbnail for %s", source_path, exc_info=True)
        return None


def get_thumbnail_url(original_url: str) -> str:
    """Convert an original image URL to its thumbnail URL.

    Only transforms /static/images/ URLs. External URLs are returned as-is.

    Examples:
        /static/images/foo.jpg -> /static/images/foo_thumb.jpg
        /static/images/bar.png -> /static/images/bar_thumb.jpg
        https://cdn.example.com/img.jpg -> https://cdn.example.com/img.jpg
    """
    if not original_url or not original_url.startswith("/static/images/"):
        return original_url

    p = Path(original_url)
    return str(p.with_name(f"{p.stem}_thumb.jpg"))


def thumbnail_path_from_original(original_path: str | Path) -> Path:
    """Compute the thumbnail filesystem path from the original image path."""
    p = Path(original_path)
    return p.with_name(f"{p.stem}_thumb.jpg")
