"""Node: generate videos using the configured video provider."""

import logging
import uuid
from pathlib import Path

from langgraph.config import get_stream_writer

from ...providers.video.base import VideoGenerationParams, VideoGenerationResult
from ..video_state import GeneratedVideoInfo, VideoGenerationState

logger = logging.getLogger(__name__)


async def video_generate_node(state: VideoGenerationState) -> dict:
    """Generate videos concurrently using the configured video provider.

    This node:
    1. Reads the enhanced prompt from state
    2. Creates VideoGenerationParams from state config
    3. Uses the provider's async batch generation (submit -> poll -> download)
    4. Streams progress and completion events via get_stream_writer
    5. Updates state with all generated videos

    Args:
        state: Current video workflow state.

    Returns:
        Dict with generated_videos, current_round, and status to update state.
    """
    writer = get_stream_writer()

    # Get prompt (prefer enhanced, fallback to user prompt)
    prompt = state.get("enhanced_prompt") or state.get("user_prompt", "")
    if not prompt:
        logger.error("No prompt available for video generation")
        return {
            "status": "error",
            "error": "No prompt provided",
        }

    # Get configuration from state
    num_videos = state.get("videos_per_round", 2)
    provider_name = state.get("provider", "kling")
    current_round = state.get("current_round", 0) + 1

    # Lazy import to avoid circular dependencies
    from ...providers.video import get_video_provider

    provider = get_video_provider(provider_name)

    params = VideoGenerationParams(
        prompt=prompt,
        start_image=state.get("start_image"),
        end_image=state.get("end_image"),
        duration=state.get("duration", 5),
        resolution=state.get("resolution"),
        aspect_ratio=state.get("aspect_ratio", "16:9"),
        generate_audio=state.get("generate_audio", False),
        negative_prompt=state.get("negative_prompt"),
        seed=state.get("seed"),
        cfg_scale=state.get("cfg_scale"),
        professional=state.get("professional"),
        reference_images=state.get("reference_images", []),
        sample_count=state.get("sample_count", 1),
    )

    # Stream status
    if writer:
        writer(
            {
                "type": "status_change",
                "status": "generating",
                "current_round": current_round,
            }
        )

    logger.info(
        "Starting video generation: round=%d, num_videos=%d, provider=%s",
        current_round,
        num_videos,
        provider_name,
    )

    # Collect generated videos (merge with existing)
    generated_videos: list[GeneratedVideoInfo] = list(state.get("generated_videos", []))

    async def on_progress(index: int, progress_percent: int, eta: int, status: str):
        """Callback for video generation progress updates."""
        if writer:
            writer(
                {
                    "type": "video_progress",
                    "video_index": index,
                    "progress_percent": progress_percent,
                    "eta": eta,
                    "status": status,
                    "current_round": current_round,
                }
            )

    async def on_video_ready(idx: int, result: VideoGenerationResult):
        """Callback called when each video generation completes."""
        local_name = Path(result.local_path).name if result.local_path else None
        video_info: GeneratedVideoInfo = {
            "id": str(uuid.uuid4()),
            "url": f"/static/videos/{local_name}" if local_name else result.url,
            "local_path": result.local_path,
            "video_type": result.video_type,
            "duration": result.duration,
            "round_number": current_round,
            "provider": provider_name,
            "provider_task_id": result.provider_task_id,
            "metadata": result.metadata if isinstance(result.metadata, dict) else {},
        }
        generated_videos.append(video_info)

        if writer:
            writer({"type": "video_ready", "video": video_info})

    try:
        await provider.generate_batch(
            params,
            num_videos,
            on_video_ready=on_video_ready,
            on_progress=on_progress,
        )
    except Exception as e:
        logger.error("Video generation failed: %s", e)
        # If no videos were generated at all, return error
        new_videos = [v for v in generated_videos if v.get("round_number") == current_round]
        if not new_videos:
            return {
                "generated_videos": generated_videos,
                "current_round": current_round,
                "status": "error",
                "error": str(e),
            }

    new_count = sum(1 for v in generated_videos if v.get("round_number") == current_round)
    logger.info(
        "Video generation complete: round=%d, generated=%d, total=%d",
        current_round,
        new_count,
        len(generated_videos),
    )

    return {
        "generated_videos": generated_videos,
        "current_round": current_round,
        "status": "generated" if new_count > 0 else "generation_failed",
    }
