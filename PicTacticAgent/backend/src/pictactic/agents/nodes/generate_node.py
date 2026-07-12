"""Node: generate images using the configured provider."""

import logging
import uuid

from langgraph.config import get_stream_writer

from ...core.config import settings
from ...providers import get_provider
from ...providers.base import GenerationParams
from ..state import GeneratedImageInfo, GenerationState
from ..tools._image_utils import normalize_reference_images

logger = logging.getLogger(__name__)


async def generate_node(state: GenerationState) -> dict:
    """Generate images concurrently using the configured provider.

    This node:
    1. Reads the enhanced prompt from state
    2. Uses the provider's batch stream generation
    3. Streams each completed image via get_stream_writer
    4. Updates state with all generated images

    Args:
        state: Current workflow state.

    Returns:
        Dict with generated_images, current_round, and status to update state.
    """
    writer = get_stream_writer()

    # Get prompt (prefer enhanced, fallback to user prompt)
    prompt = state.get("enhanced_prompt") or state.get("user_prompt", "")
    if not prompt:
        logger.error("No prompt available for generation")
        return {
            "status": "error",
            "error": "No prompt provided",
        }

    # Get configuration from state
    num_images = state.get("images_per_round", 5)
    provider_name = state.get("provider") or settings.default_provider
    aspect_ratio = state.get("aspect_ratio", "16:9")
    image_size = state.get("image_size", "2K")
    current_round = state.get("current_round", 0) + 1

    # Get reference images
    template_images = state.get("template_images", [])
    reference_images = []
    if template_images:
        try:
            reference_images = await normalize_reference_images(template_images)
            logger.info("Using %d reference images for generation", len(reference_images))
        except Exception as e:
            logger.warning("Failed to normalize reference images: %s", e)
            reference_images = template_images

    # Create provider and params
    provider = get_provider(provider_name)
    params = GenerationParams(
        prompt=prompt,
        aspect_ratio=aspect_ratio,
        size=image_size,
        reference_images=reference_images,
    )

    # Generate images with streaming callback
    images: list[GeneratedImageInfo] = []

    async def on_image_ready(idx: int, result):
        """Callback called when each image completes."""
        image_info: GeneratedImageInfo = {
            "id": str(uuid.uuid4()),
            "url": result.url,
            "local_path": result.local_path,
            "score": 0.0,
            "review": "",
            "round_number": current_round,
            "provider": provider_name,
            "metadata": result.metadata if isinstance(result.metadata, dict) else {},
        }
        images.append(image_info)

        # Stream the image to the client
        if writer:
            writer({"type": "image_ready", "image": image_info})

    logger.info(
        "Starting generation: round=%d, num_images=%d, provider=%s",
        current_round, num_images, provider_name
    )

    if writer:
        writer({
            "type": "status_change",
            "status": "generating",
            "current_round": current_round,
        })

    try:
        await provider.generate_batch_stream(
            params, num_images, on_image_ready=on_image_ready
        )
    except Exception as e:
        logger.error("Image generation failed: %s", e)
        return {
            "current_round": current_round,
            "status": "error",
            "error": str(e),
        }

    # Merge with existing images
    all_images = state.get("generated_images", []) + images

    logger.info(
        "Generation complete: round=%d, generated=%d, total=%d",
        current_round, len(images), len(all_images)
    )

    return {
        "generated_images": all_images,
        "current_round": current_round,
        "status": "generated" if images else "generation_failed",
    }
