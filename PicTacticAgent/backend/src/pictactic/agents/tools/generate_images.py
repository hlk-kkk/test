"""Tool: generate images using the configured provider."""

import logging
import uuid
from typing import Annotated

from langchain_core.messages import ToolMessage
from langchain_core.tools import InjectedToolCallId, tool
from langgraph.config import get_stream_writer
from langgraph.prebuilt import InjectedState
from langgraph.types import Command

from ...providers.base import GenerationParams
from ..state import GeneratedImageInfo, GenerationState
from ._image_utils import normalize_reference_images

logger = logging.getLogger(__name__)


def _is_state_reference(value: str) -> bool:
    """Check if a value is a state reference placeholder (not an actual image source)."""
    v = value.strip()
    return (
        v.startswith("state://")
        or v.startswith("state[")
        or v.startswith("state.")
        or v.startswith("state['")
        or v.startswith('state["')
    )


@tool
async def generate_images(
    prompt: str,
    num_images: int,
    tool_call_id: Annotated[str, InjectedToolCallId],
    state: Annotated[GenerationState, InjectedState()],
    aspect_ratio: str = "16:9",
    image_size: str = "2K",
    reference_images: list[str] | None = None,
) -> Command:
    """Generate images using the configured provider.
    Args:
        prompt: The prompt to use for generation (use enhanced prompt)
        num_images: Number of images to generate
        aspect_ratio: Image aspect ratio (default "16:9")
        image_size: Image size (default "2K")
        reference_images: Optional list of reference image URLs
    Returns a Command that updates state with generated images. Also streams each image via stream_writer."""
    from ...core.config import settings
    from ...providers import get_provider

    provider = state.get("provider") or settings.default_provider

    current_round = state.get("current_round", 0) + 1
    writer = get_stream_writer()

    # Fallback: if reference_images not provided, read from state template_images
    actual_reference_images = reference_images
    if not actual_reference_images:
        actual_reference_images = state.get("template_images", [])

    # Filter out invalid state references (e.g., "state://template_images")
    # These are placeholder strings that LLM may mistakenly pass instead of actual images
    if actual_reference_images:
        actual_reference_images = [
            img for img in actual_reference_images
            if isinstance(img, str) and img.strip() and not _is_state_reference(img)
        ]
        # If all references were filtered out, try reading from state again
        if not actual_reference_images:
            actual_reference_images = state.get("template_images", [])

    # Enforce config from state if present
    state_aspect_ratio = state.get("aspect_ratio")
    if state_aspect_ratio:
        aspect_ratio = state_aspect_ratio

    state_image_size = state.get("image_size")
    if state_image_size:
        image_size = state_image_size

    state_images_per_round = state.get("images_per_round")
    if isinstance(state_images_per_round, int) and state_images_per_round > 0:
        num_images = state_images_per_round

    if actual_reference_images:
        logger.info(f"Using {len(actual_reference_images)} reference images for generation")
        try:
            actual_reference_images = await normalize_reference_images(actual_reference_images)
        except Exception as e:
            logger.warning("Failed to normalize reference images, using originals: %s", e)

    provider_instance = get_provider(provider)
    params = GenerationParams(
        prompt=prompt,
        aspect_ratio=aspect_ratio,
        size=image_size,
        reference_images=actual_reference_images or [],
    )

    images: list[GeneratedImageInfo] = []

    async def _on_ready(idx, result):
        image_info: GeneratedImageInfo = {
            "id": str(uuid.uuid4()),
            "url": result.url,
            "local_path": result.local_path,
            "score": 0.0,
            "review": "",
            "round_number": current_round,
            "provider": provider,
        }
        images.append(image_info)
        if writer:
            writer({"type": "image_ready", "image": image_info})

    await provider_instance.generate_batch_stream(params, num_images, on_image_ready=_on_ready)

    all_images = state.get("generated_images", []) + images

    summary = f"Generated {len(images)} images in round {current_round}."
    return Command(update={
        "generated_images": all_images,
        "current_round": current_round,
        "status": "generated",
        "messages": [ToolMessage(content=summary, tool_call_id=tool_call_id)],
    })
