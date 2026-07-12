"""Conditional edge functions for the video generation workflow."""

from typing import Literal

from .video_state import VideoGenerationState


def check_video_input(state: VideoGenerationState) -> Literal["enhance"]:
    """Check video input and route to enhance.

    Unlike the image workflow, video generation has no template analysis step.
    Always routes directly to the enhance node.

    Returns:
        "enhance" always.
    """
    return "enhance"


def should_continue_video(state: VideoGenerationState) -> Literal["continue", "finalize"]:
    """Determine whether to continue video generation or finalize.

    Video generation has no quality threshold evaluation.
    Continues generating until max_rounds is reached.

    Returns:
        "continue" if more rounds remain, "finalize" otherwise.
    """
    current_round = state.get("current_round", 0)
    max_rounds = state.get("max_rounds", 2)

    if current_round >= max_rounds:
        return "finalize"
    return "continue"
