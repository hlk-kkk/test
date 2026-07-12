"""Condition functions for the generation workflow."""

from typing import Literal

from .state import GenerationState


def check_template(state: GenerationState) -> Literal["analyze", "enhance"]:
    """Check if template images are provided.

    Returns:
        "analyze" if template images exist, "enhance" otherwise.
    """
    template_images = state.get("template_images", [])
    return "analyze" if template_images else "enhance"


def should_continue(state: GenerationState) -> Literal["continue", "finalize"]:
    """Determine whether to continue generation or finalize.

    Continues if:
    - Current round < max_rounds AND
    - Best score < quality_threshold

    Returns:
        "continue" to generate another round, "finalize" to end.
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

    # Check termination conditions
    quality_met = best_score >= quality_threshold
    rounds_exhausted = current_round >= max_rounds

    if quality_met or rounds_exhausted:
        return "finalize"
    return "continue"
