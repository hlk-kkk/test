"""Node: finalize the workflow by selecting the best images."""

import logging

from ..state import GenerationState

logger = logging.getLogger(__name__)


async def finalize_node(state: GenerationState) -> dict:
    """Finalize the workflow by selecting the best images.

    This node:
    1. Collects all generated images from all rounds
    2. Sorts them by score
    3. Selects the top 3 as best_images
    4. Creates the final_result summary

    Args:
        state: Current workflow state.

    Returns:
        Dict with best_images, final_result, and status to update state.
    """
    generated_images = state.get("generated_images", [])
    current_round = state.get("current_round", 1)
    quality_threshold = state.get("quality_threshold", 0.7)

    # Sort all images by score (descending)
    sorted_images = sorted(
        generated_images,
        key=lambda x: x.get("score", 0),
        reverse=True
    )

    # Select top N as best images (based on top_k config, default 3)
    num_best = state.get("top_k", 3)
    best_images = sorted_images[:num_best] if sorted_images else []

    # Calculate summary stats
    best_score = best_images[0].get("score", 0) if best_images else 0
    quality_met = best_score >= quality_threshold

    # Success requires at least one generated image and one best image
    success = len(generated_images) > 0 and len(best_images) > 0

    final_result = {
        "success": success,
        "best_images": best_images,
        "total_rounds": current_round,
        "total_images": len(generated_images),
        "best_score": best_score,
        "quality_met": quality_met,
    }

    logger.info(
        "Workflow finalized: rounds=%d, total_images=%d, best_score=%.2f, quality_met=%s",
        current_round,
        len(generated_images),
        best_score,
        quality_met
    )

    return {
        "best_images": best_images,
        "status": "completed" if success else "failed",
        "final_result": final_result,
    }
