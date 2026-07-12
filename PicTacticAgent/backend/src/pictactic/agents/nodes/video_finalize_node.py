"""Node: finalize the video generation workflow."""

import logging

from langgraph.config import get_stream_writer

from ..video_state import VideoGenerationState

logger = logging.getLogger(__name__)


async def video_finalize_node(state: VideoGenerationState) -> dict:
    """Finalize the video generation workflow.

    This node:
    1. Collects all generated videos from all rounds
    2. Creates the final_result summary
    3. Streams a completion status event

    Unlike image finalization, there is no scoring or ranking;
    all generated videos are included in the result.

    Args:
        state: Current video workflow state.

    Returns:
        Dict with final_result and status to update state.
    """
    writer = get_stream_writer()

    generated_videos = state.get("generated_videos", [])
    current_round = state.get("current_round", 1)

    success = len(generated_videos) > 0

    final_result = {
        "success": success,
        "videos": generated_videos,
        "total_rounds": current_round,
        "total_videos": len(generated_videos),
    }

    logger.info(
        "Video workflow finalized: rounds=%d, total_videos=%d, success=%s",
        current_round,
        len(generated_videos),
        success,
    )

    # Stream completion event
    if writer:
        writer(
            {
                "type": "status_change",
                "status": "completed" if success else "failed",
                "total_rounds": current_round,
                "total_videos": len(generated_videos),
            }
        )

    return {
        "status": "completed" if success else "failed",
        "final_result": final_result,
    }
