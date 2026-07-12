"""Node: prepare state for the next video generation round."""

import logging

from ..video_state import VideoGenerationState

logger = logging.getLogger(__name__)


async def video_prepare_next_node(state: VideoGenerationState) -> dict:
    """Prepare the state for the next video generation round.

    This node is called when should_continue_video returns "continue".
    It logs the transition and updates the status. The video workflow
    does not re-enhance the prompt between rounds.

    Args:
        state: Current video workflow state.

    Returns:
        Dict with status update.
    """
    current_round = state.get("current_round", 0)
    max_rounds = state.get("max_rounds", 2)
    generated_videos = state.get("generated_videos", [])

    current_round_videos = [v for v in generated_videos if v.get("round_number") == current_round]

    logger.info(
        "Preparing next video round: current=%d/%d, videos_this_round=%d, total=%d",
        current_round,
        max_rounds,
        len(current_round_videos),
        len(generated_videos),
    )

    return {
        "status": "preparing_next_round",
    }
