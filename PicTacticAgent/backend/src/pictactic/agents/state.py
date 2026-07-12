"""State definition for the generation agent."""

from dataclasses import dataclass, field
from typing import Annotated, Optional

from langgraph.graph.message import add_messages
from typing_extensions import NotRequired, TypedDict


class GeneratedImageInfo(TypedDict):
    """Information about a generated image."""

    id: str
    url: str
    local_path: str | None
    score: float
    review: str
    round_number: int
    provider: str
    metadata: NotRequired[dict]


class GenerationState(TypedDict):
    """Agent state for the generation workflow."""

    # Messages (kept for compatibility, used by nodes that need LLM)
    messages: Annotated[list, add_messages]

    # User input
    user_prompt: NotRequired[str]  # Original user prompt
    template_images: list[str]  # Template images (base64 or URLs) from user upload

    # Analysis / prompt
    template_analysis: Optional[dict]
    enhanced_prompt: Optional[str]

    # Generation results
    generated_images: list[GeneratedImageInfo]
    best_images: list[GeneratedImageInfo]
    current_round: int

    # Evaluation feedback for iterative enhancement
    evaluation_feedback: NotRequired[str]

    # Status
    status: Optional[str]
    final_result: Optional[dict]
    error: Optional[str]

    # Config (copied from request)
    max_rounds: NotRequired[int]
    images_per_round: NotRequired[int]
    top_k: NotRequired[int]  # Number of top-scored images to output
    aspect_ratio: NotRequired[str]
    image_size: NotRequired[str]
    provider: NotRequired[str]
    enhance_prompt_enabled: NotRequired[bool]
    enable_ai_scoring: NotRequired[bool]
    quality_threshold: NotRequired[float]


@dataclass
class GenerationContext:
    """Runtime context passed to the agent via context_schema."""

    task_id: str
    user_prompt: str
    template_images: list[str] = field(default_factory=list)
    max_rounds: int = 3
    images_per_round: int = 5
    aspect_ratio: str = "16:9"
    image_size: str = "2K"
    provider: str = "gemini"
    enhance_prompt_enabled: bool = True


def create_initial_state(
    user_prompt: str,
    template_images: list[str] | None = None,
    max_rounds: int = 3,
    images_per_round: int = 5,
    top_k: int = 3,
    quality_threshold: float = 0.7,
    aspect_ratio: str = "16:9",
    image_size: str = "2K",
    provider: str = "gemini",
    enhance_prompt_enabled: bool = True,
    enable_ai_scoring: bool = True,
) -> dict:
    """Create initial state for the generation workflow.

    This is a convenience function to create a properly initialized
    state dictionary for starting a new generation workflow.

    Args:
        user_prompt: The user's original prompt.
        template_images: Optional list of template image URLs/base64.
        max_rounds: Maximum number of generation rounds.
        images_per_round: Number of images to generate per round.
        quality_threshold: Score threshold to stop early.
        aspect_ratio: Image aspect ratio.
        image_size: Image size (1K, 2K, 4K).
        provider: Image generation provider name.
        enhance_prompt_enabled: Whether to enhance the prompt.
        enable_ai_scoring: Whether to enable AI scoring during evaluation.

    Returns:
        A dictionary suitable for initializing GenerationState.
    """
    return {
        "messages": [],
        "user_prompt": user_prompt,
        "template_images": template_images or [],
        "template_analysis": None,
        "enhanced_prompt": None,
        "generated_images": [],
        "best_images": [],
        "current_round": 0,
        "evaluation_feedback": "",
        "status": "pending",
        "final_result": None,
        "error": None,
        "max_rounds": max_rounds,
        "images_per_round": images_per_round,
        "top_k": top_k,
        "aspect_ratio": aspect_ratio,
        "image_size": image_size,
        "provider": provider,
        "enhance_prompt_enabled": enhance_prompt_enabled,
        "enable_ai_scoring": enable_ai_scoring,
        "quality_threshold": quality_threshold,
    }
