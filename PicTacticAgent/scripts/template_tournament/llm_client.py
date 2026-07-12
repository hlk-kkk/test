"""OpenAI-compatible LLM client using project .env config."""

import asyncio
import base64
import logging
import time
from pathlib import Path

from dotenv import load_dotenv
from openai import AsyncOpenAI

logger = logging.getLogger(__name__)

# Load .env from project root
_project_root = Path(__file__).resolve().parent.parent.parent
load_dotenv(_project_root / ".env")

# Sliding window rate limiter: 17 req/60s (safe margin below 20 req/min limit)
_rate_lock = asyncio.Lock()
_request_timestamps: list[float] = []
_MAX_REQUESTS = 17
_WINDOW_SECONDS = 60.0


def get_client() -> AsyncOpenAI:
    """Create AsyncOpenAI client pointed at jiekouai."""
    import os

    return AsyncOpenAI(
        api_key=os.getenv("JIEKOUAI_API_KEY", ""),
        base_url=os.getenv("JIEKOUAI_LLM_BASE_URL", "https://api.jiekou.ai/openai"),
    )


def get_model() -> str:
    """Get the LLM model name from env."""
    import os

    return os.getenv("JIEKOUAI_LLM_MODEL", "gpt-4o")


def encode_image_base64(image_path: str) -> str:
    """Read an image file and return base64 string."""
    with open(image_path, "rb") as f:
        return base64.b64encode(f.read()).decode("utf-8")


async def _rate_limit() -> None:
    """Sliding window: allow up to _MAX_REQUESTS in _WINDOW_SECONDS."""
    async with _rate_lock:
        now = time.monotonic()
        # Prune timestamps outside the window
        while _request_timestamps and now - _request_timestamps[0] >= _WINDOW_SECONDS:
            _request_timestamps.pop(0)
        # If at capacity, wait until the oldest expires
        if len(_request_timestamps) >= _MAX_REQUESTS:
            wait = _WINDOW_SECONDS - (now - _request_timestamps[0]) + 0.1
            if wait > 0:
                logger.debug("Rate limit: waiting %.1fs", wait)
                await asyncio.sleep(wait)
        _request_timestamps.append(time.monotonic())


async def chat(
    messages: list[dict],
    model: str | None = None,
    temperature: float = 0.3,
    max_tokens: int = 2048,
) -> str:
    """Send a chat completion request and return the text response."""
    await _rate_limit()
    client = get_client()
    response = await client.chat.completions.create(
        model=model or get_model(),
        messages=messages,
        temperature=temperature,
        max_tokens=max_tokens,
    )
    return response.choices[0].message.content


async def chat_with_images(
    text_prompt: str,
    image_paths: list[str],
    model: str | None = None,
    temperature: float = 0.3,
    max_tokens: int = 2048,
) -> str:
    """Send a vision request with images and return text response."""
    content: list[dict] = [{"type": "text", "text": text_prompt}]
    for path in image_paths:
        b64 = encode_image_base64(path)
        ext = Path(path).suffix.lstrip(".").lower()
        mime = {"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
                "webp": "image/webp", "gif": "image/gif"}.get(ext, "image/jpeg")
        content.append({
            "type": "image_url",
            "image_url": {"url": f"data:{mime};base64,{b64}"},
        })

    return await chat(
        messages=[{"role": "user", "content": content}],
        model=model,
        temperature=temperature,
        max_tokens=max_tokens,
    )
