"""State definition for the video generation workflow."""

from typing import Annotated, NotRequired

from langgraph.graph.message import add_messages
from typing_extensions import TypedDict


class GeneratedVideoInfo(TypedDict):
    """Information about a generated video."""

    id: str  # UUID
    url: str  # Video URL (CDN or local serving)
    local_path: str | None  # Local storage path
    video_type: str  # "mp4", "gif"
    duration: int | None  # Duration in seconds
    round_number: int  # Which generation round
    provider: str  # "kling", "sora", "veo"
    provider_task_id: str | None  # Provider's async task ID
    metadata: NotRequired[dict]


class VideoGenerationState(TypedDict):
    """Agent state for the video generation workflow."""

    # Messages (kept for LLM-based nodes like enhance)
    messages: Annotated[list, add_messages]

    # User inputs
    user_prompt: NotRequired[str]
    start_image: NotRequired[str | None]  # Start frame image URL/data URI
    end_image: NotRequired[str | None]  # End frame image URL/data URI
    reference_images: list[str]  # Additional reference images (Veo)

    # Enhanced prompt
    enhanced_prompt: str | None

    # Generated videos
    generated_videos: list[GeneratedVideoInfo]
    current_round: int

    # Status
    status: str | None  # pending, enhancing, generating, polling, completed, failed
    final_result: dict | None
    error: str | None

    # Configuration
    max_rounds: NotRequired[int]  # Default: 2
    videos_per_round: NotRequired[int]  # Default: 2
    duration: NotRequired[int]  # Seconds
    resolution: NotRequired[str | None]
    aspect_ratio: NotRequired[str]
    generate_audio: NotRequired[bool]
    provider: NotRequired[str]  # "kling", "sora", "veo"
    enhance_prompt_enabled: NotRequired[bool]
    negative_prompt: NotRequired[str | None]
    # Provider-specific
    cfg_scale: NotRequired[float | None]
    professional: NotRequired[bool | None]
    seed: NotRequired[int | None]
    sample_count: NotRequired[int]


def create_initial_video_state(
    user_prompt: str,
    start_image: str | None = None,
    end_image: str | None = None,
    reference_images: list[str] | None = None,
    max_rounds: int = 2,
    videos_per_round: int = 2,
    duration: int = 5,
    resolution: str | None = None,
    aspect_ratio: str = "16:9",
    generate_audio: bool = False,
    provider: str = "kling",
    enhance_prompt_enabled: bool = True,
    negative_prompt: str | None = None,
    cfg_scale: float | None = None,
    professional: bool | None = None,
    seed: int | None = None,
    sample_count: int = 1,
) -> dict:
    """Create initial state for the video generation workflow.

    Args:
        user_prompt: The user's original prompt.
        start_image: Optional start frame image URL/data URI.
        end_image: Optional end frame image URL/data URI.
        reference_images: Optional list of reference image URLs.
        max_rounds: Maximum number of generation rounds.
        videos_per_round: Number of videos to generate per round.
        duration: Video duration in seconds.
        resolution: Video resolution (e.g., "720p", "1080p").
        aspect_ratio: Video aspect ratio.
        generate_audio: Whether to generate audio.
        provider: Video generation provider name.
        enhance_prompt_enabled: Whether to enhance the prompt.
        negative_prompt: Negative prompt for generation.
        cfg_scale: CFG scale (provider-specific).
        professional: Professional mode (provider-specific).
        seed: Random seed for reproducibility.
        sample_count: Number of samples per generation call.

    Returns:
        A dictionary suitable for initializing VideoGenerationState.
    """
    return {
        "messages": [],
        "user_prompt": user_prompt,
        "start_image": start_image,
        "end_image": end_image,
        "reference_images": reference_images or [],
        "enhanced_prompt": None,
        "generated_videos": [],
        "current_round": 0,
        "status": "pending",
        "final_result": None,
        "error": None,
        "max_rounds": max_rounds,
        "videos_per_round": videos_per_round,
        "duration": duration,
        "resolution": resolution,
        "aspect_ratio": aspect_ratio,
        "generate_audio": generate_audio,
        "provider": provider,
        "enhance_prompt_enabled": enhance_prompt_enabled,
        "negative_prompt": negative_prompt,
        "cfg_scale": cfg_scale,
        "professional": professional,
        "seed": seed,
        "sample_count": sample_count,
    }
