"""Node: prepare state for the next generation round."""

import logging

from ..state import GenerationState

logger = logging.getLogger(__name__)


async def prepare_next_node(state: GenerationState) -> dict:
    """Prepare the state for the next generation round.

    This node is called when should_continue returns "continue".
    It prepares the state for another iteration by:
    1. Keeping the evaluation feedback for prompt enhancement
    2. Preserving existing generated images
    3. Logging the transition

    The actual prompt enhancement happens in enhance_node,
    which will use the evaluation_feedback.

    Args:
        state: Current workflow state.

    Returns:
        Dict with status update.
    """
    current_round = state.get("current_round", 0)
    max_rounds = state.get("max_rounds", 3)
    quality_threshold = state.get("quality_threshold", 0.7)
    generated_images = state.get("generated_images", [])

    # Get best score from current round
    current_round_images = [
        img for img in generated_images
        if img.get("round_number") == current_round
    ]
    best_score = max(
        (img.get("score", 0) for img in current_round_images),
        default=0
    )

    logger.info(
        "Preparing next round: current=%d/%d, best_score=%.2f (threshold=%.2f)",
        current_round,
        max_rounds,
        best_score,
        quality_threshold
    )

    return {
        "status": "preparing_next_round",
    }
