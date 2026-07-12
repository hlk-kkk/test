"""Shared utilities for image tools.

Uses WebP encoding with smart downscaling to achieve the best
quality-to-size ratio for vision LLM evaluation.
"""

import asyncio
import base64
import io
import logging
import os
import re
from pathlib import Path
from urllib.parse import urlparse

import httpx
from PIL import Image, ImageFilter

from ...core.config import settings

logger = logging.getLogger(__name__)

# Vision LLM sweet spot: 1024-1536px is enough for comparative evaluation.
# Going higher wastes tokens/bandwidth with negligible quality gain.
_MAX_EVAL_LONG_EDGE = 1536

# WebP quality 85 ≈ JPEG quality 92 visually, but ~30% smaller file size.
_WEBP_QUALITY = 85
_REF_JPEG_QUALITY = 88

def _is_provider_usable(source: str) -> bool:
    """Check whether *source* can be sent to a remote image provider.

    Only http(s) URLs and data-URIs are usable; local filesystem paths are not.
    """
    return source.startswith(("http://", "https://", "data:"))


_DATA_URI_RE = re.compile(
    r"^data:(?P<mime>image/[^;]+)?;base64,(?P<data>[A-Za-z0-9+/=\s]+)$",
    re.IGNORECASE | re.DOTALL,
)
_BASE64_RE = re.compile(r"^[A-Za-z0-9+/=\s]+$")


def _smart_resize(img: Image.Image, max_long_edge: int) -> Image.Image:
    """Downscale only if the image exceeds max_long_edge.

    Uses LANCZOS resampling + a mild sharpen pass to compensate for
    softness introduced by downscaling, keeping edges crisp for the
    vision model.
    """
    w, h = img.size
    long_edge = max(w, h)
    if long_edge <= max_long_edge:
        return img

    scale = max_long_edge / long_edge
    new_w, new_h = round(w * scale), round(h * scale)
    img = img.resize((new_w, new_h), Image.LANCZOS)
    # Light sharpen to recover detail lost during downscale
    img = img.filter(ImageFilter.SHARPEN)
    return img


def _to_webp_bytes(img: Image.Image) -> bytes:
    """Encode image to WebP with optimal settings."""
    if img.mode in ("RGBA", "P", "LA"):
        img = img.convert("RGB")

    buf = io.BytesIO()
    img.save(
        buf,
        format="WEBP",
        quality=_WEBP_QUALITY,
        method=6,  # slowest but best compression
    )
    return buf.getvalue()


def _decode_data_uri(data_uri: str) -> bytes:
    match = _DATA_URI_RE.match(data_uri)
    if not match:
        raise ValueError("Invalid data URI")
    raw = re.sub(r"\s+", "", match.group("data"))
    return base64.b64decode(raw)


def _looks_like_base64(value: str) -> bool:
    if len(value) < 200:
        return False
    head = value[:200]
    return bool(_BASE64_RE.fullmatch(head)) and bool(_BASE64_RE.fullmatch(value.strip()))


def _resolve_local_path(source: str) -> Path | None:
    if source.startswith("file://"):
        parsed = urlparse(source)
        candidate = Path(parsed.path)
        return candidate if candidate.exists() else None

    if source.startswith("/static/images/") and settings.storage_type == "local":
        candidate = Path(settings.storage_path) / Path(source).name
        return candidate if candidate.exists() else None

    candidate = Path(source)
    if candidate.is_absolute() and candidate.exists():
        return candidate

    if source.startswith("./") or source.startswith("../"):
        candidate = (Path.cwd() / source).resolve()
        return candidate if candidate.exists() else None

    # Plain relative path (e.g. "storage/images/foo.png")
    candidate = (Path.cwd() / source).resolve()
    return candidate if candidate.exists() else None


def _image_bytes_to_data_uri(image_bytes: bytes) -> str:
    original_size = len(image_bytes)
    img = Image.open(io.BytesIO(image_bytes))
    img = _smart_resize(img, _MAX_EVAL_LONG_EDGE)
    webp_bytes = _to_webp_bytes(img)

    b64 = base64.b64encode(webp_bytes).decode()
    logger.info(
        "Image compressed: %d KB → %d KB (%.0f%% reduction, %dx%d)",
        original_size // 1024,
        len(webp_bytes) // 1024,
        (1 - len(webp_bytes) / original_size) * 100,
        img.size[0],
        img.size[1],
    )
    return f"data:image/webp;base64,{b64}"


def _encode_reference_image(img: Image.Image, max_long_edge: int) -> str:
    """Resize and encode reference images for provider upload (JPEG data URI)."""
    img = _smart_resize(img, max_long_edge)
    if img.mode in ("RGBA", "LA"):
        # Composite on white to avoid black background when saving as JPEG.
        background = Image.new("RGB", img.size, (255, 255, 255))
        background.paste(img, mask=img.split()[-1])
        img = background
    elif img.mode != "RGB":
        img = img.convert("RGB")

    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=_REF_JPEG_QUALITY, optimize=True)
    b64 = base64.b64encode(buf.getvalue()).decode()
    return f"data:image/jpeg;base64,{b64}"


async def _source_to_image_bytes(source: str) -> bytes:
    if source.startswith("data:"):
        return _decode_data_uri(source)

    if _looks_like_base64(source):
        return base64.b64decode(source)

    local_path = _resolve_local_path(source)
    if local_path:
        return local_path.read_bytes()

    if source.startswith("http://") or source.startswith("https://"):
        proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
        async with httpx.AsyncClient(timeout=30.0, follow_redirects=True, proxy=proxy) as client:
            resp = await client.get(source)
            resp.raise_for_status()
        return resp.content

    raise ValueError(f"Unsupported image source: {source}")


async def normalize_reference_images(
    sources: list[str],
    max_long_edge: int = _MAX_EVAL_LONG_EDGE,
    keep_http_urls: bool = True,
) -> list[str]:
    """Normalize reference images for generation.

    - http(s) URLs are kept as-is when keep_http_urls is True.
    - data URIs / base64 / local files are resized and re-encoded to JPEG data URIs.
    """
    if not sources:
        return []

    normalized: list[str] = []

    async def _normalize_one(src: str) -> str:
        src = src.strip()
        if keep_http_urls and (src.startswith("http://") or src.startswith("https://")):
            return src
        image_bytes = await _source_to_image_bytes(src)
        img = Image.open(io.BytesIO(image_bytes))
        return _encode_reference_image(img, max_long_edge)

    tasks = []
    src_list: list[str] = []
    for src in sources:
        if not isinstance(src, str) or not src.strip():
            continue
        trimmed = src.strip()
        src_list.append(trimmed)
        tasks.append(_normalize_one(trimmed))

    results = await asyncio.gather(*tasks, return_exceptions=True)
    for original, result in zip(src_list, results):
        if isinstance(result, Exception):
            if _is_provider_usable(original):
                logger.warning("Normalization failed, using original URL: %s", result)
                normalized.append(original)
            else:
                raise ValueError(
                    f"Cannot normalize local image and no usable fallback: {original}"
                ) from result
            continue
        normalized.append(result)

    return normalized


async def url_to_data_uri(url: str) -> str:
    """Normalize an image source and encode as WebP data URI.

    Supports:
    - http(s) URLs
    - data URIs (base64)
    - local file paths / file:// URLs
    - /static/images/* paths (when local storage is enabled)
    """
    if not url or not isinstance(url, str):
        raise ValueError("Empty image source")

    source = url.strip()
    if source.startswith("data:"):
        return _image_bytes_to_data_uri(_decode_data_uri(source))

    if _looks_like_base64(source):
        return _image_bytes_to_data_uri(base64.b64decode(source))

    local_path = _resolve_local_path(source)
    if local_path:
        return _image_bytes_to_data_uri(local_path.read_bytes())

    if source.startswith("http://") or source.startswith("https://"):
        proxy = os.environ.get("https_proxy") or os.environ.get("HTTPS_PROXY")
        async with httpx.AsyncClient(timeout=30.0, follow_redirects=True, proxy=proxy) as client:
            resp = await client.get(source)
            resp.raise_for_status()
        return _image_bytes_to_data_uri(resp.content)

    raise ValueError(f"Unsupported image source: {source}")


async def batch_url_to_data_uri(urls: list[str]) -> list[str]:
    """Download multiple images concurrently and return data URIs.

    Individual failures are logged and the original URL is kept as fallback,
    so one broken image does not crash the entire batch.
    """
    tasks = [url_to_data_uri(url) for url in urls]
    results = await asyncio.gather(*tasks, return_exceptions=True)
    out: list[str] = []
    for i, result in enumerate(results):
        if isinstance(result, BaseException):
            logger.warning("Failed to convert image %d to data URI: %s", i, result)
            out.append(urls[i])  # fallback to original URL
        else:
            out.append(result)
    return out
