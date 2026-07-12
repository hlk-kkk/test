"""Video generation workflow using LangGraph StateGraph.

This module defines a deterministic workflow for video generation,
simpler than the image workflow since video has no AI evaluation step.

Workflow:
    START
      |
      v
    [enhance] ---- enhance prompt for video generation
      |
      v
    [generate_videos] <---------+
      |                         |
      v                         |
    [should_continue_video]     |
      |           |             |
      | finalize  | continue    |
      v           v             |
    [finalize]  [prepare_next] -+
      |
      v
     END
"""

import logging

from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, START, StateGraph

from .nodes.video_enhance_node import video_enhance_node
from .nodes.video_finalize_node import video_finalize_node
from .nodes.video_generate_node import video_generate_node
from .nodes.video_prepare_next_node import video_prepare_next_node
from .video_conditions import should_continue_video
from .video_state import VideoGenerationState

logger = logging.getLogger(__name__)


def create_video_generation_workflow(checkpointer=None):
    """Create and return the video generation workflow graph.

    The video workflow is simpler than the image workflow:
    - No template analysis step
    - No AI evaluation/scoring step
    - Prompt is only enhanced once (first round)
    - Runs for max_rounds then finalizes all videos

    Args:
        checkpointer: LangGraph checkpointer for state persistence.
                     If None, uses in-memory storage.

    Returns:
        Compiled StateGraph ready for execution.
    """
    builder = StateGraph(VideoGenerationState)

    # Add nodes
    builder.add_node("enhance", video_enhance_node)
    builder.add_node("generate_videos", video_generate_node)
    builder.add_node("prepare_next", video_prepare_next_node)
    builder.add_node("finalize", video_finalize_node)

    # Edges
    # START -> enhance (always start with prompt enhancement)
    builder.add_edge(START, "enhance")

    # enhance -> generate_videos
    builder.add_edge("enhance", "generate_videos")

    # generate_videos -> should_continue_video (conditional)
    builder.add_conditional_edges(
        "generate_videos",
        should_continue_video,
        {
            "continue": "prepare_next",
            "finalize": "finalize",
        },
    )

    # prepare_next -> generate_videos (loop back, skip re-enhancement)
    builder.add_edge("prepare_next", "generate_videos")

    # finalize -> END
    builder.add_edge("finalize", END)

    # Compile with checkpointer
    workflow = builder.compile(checkpointer=checkpointer or MemorySaver())

    logger.info("Video generation workflow compiled successfully")
    return workflow
