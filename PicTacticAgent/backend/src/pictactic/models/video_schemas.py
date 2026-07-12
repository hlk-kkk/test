"""Video generation request/response schemas."""

from datetime import datetime
from enum import Enum

from pydantic import BaseModel, Field, model_validator


class VideoProvider(str, Enum):
    """Supported video generation providers."""

    KLING = "kling"
    SEEDANCE = "seedance"
    SORA = "sora"
    VEO = "veo"


class VideoGenerationRequest(BaseModel):
    """Request to start a video generation task."""

    prompt: str = Field(..., min_length=1, description="Video generation prompt")
    start_image: str | None = Field(None, description="Start frame image URL or data URI")
    end_image: str | None = Field(None, description="End frame image URL or data URI")
    reference_images: list[str] = Field(default_factory=list, description="Reference images (Veo)")
    max_rounds: int = Field(2, ge=1, le=5, description="Number of generation rounds")
    videos_per_round: int = Field(2, ge=1, le=4, description="Videos per round")
    duration: int = Field(5, ge=3, le=15, description="Video duration in seconds")
    resolution: str | None = Field(None, description="720p or 1080p")
    aspect_ratio: str = Field("16:9", description="Video aspect ratio")
    generate_audio: bool = Field(False, description="Generate audio with video")
    enhance_prompt: bool = Field(True, description="Enhance prompt with AI")
    negative_prompt: str | None = Field(None, description="Negative prompt")
    provider: VideoProvider | None = Field(None, description="Video provider")
    # Provider-specific
    cfg_scale: float | None = Field(None, ge=0, le=1, description="Kling cfg_scale")
    professional: bool | None = Field(None, description="Sora professional mode")
    seed: int | None = Field(None, description="Random seed")
    sample_count: int = Field(1, ge=1, le=4, description="Veo sample count")

    @model_validator(mode="before")
    @classmethod
    def set_default_provider(cls, data):
        """Set default video provider from settings when not specified."""
        if isinstance(data, dict) and data.get("provider") is None:
            from ..core.config import settings

            data["provider"] = settings.default_video_provider
        return data


class GeneratedVideo(BaseModel):
    """A generated video with metadata."""

    id: str
    url: str
    local_path: str | None = None
    video_type: str = "mp4"
    duration: int | None = None
    round_number: int
    provider: str
    provider_task_id: str | None = None
    metadata: dict = Field(default_factory=dict)
    created_at: datetime | None = None


class VideoRoundResult(BaseModel):
    """Results from a single video generation round."""

    round_number: int
    videos: list[GeneratedVideo] = Field(default_factory=list)


class VideoGenerationResult(BaseModel):
    """Complete result of a video generation task."""

    task_id: str
    status: str  # pending, enhancing, generating, polling, completed, failed
    original_prompt: str
    enhanced_prompt: str | None = None
    start_image: str | None = None
    end_image: str | None = None
    config: dict | None = None
    rounds: list[VideoRoundResult] = Field(default_factory=list)
    total_rounds: int = 0
    total_videos: int = 0
    videos: list[GeneratedVideo] = Field(default_factory=list)
    started_at: datetime | None = None
    completed_at: datetime | None = None
    error: str | None = None


class VideoTaskProgress(BaseModel):
    """Progress update for a video generation task."""

    task_id: str
    status: str
    current_round: int = 0
    max_rounds: int = 2
    videos_per_round: int | None = None
    videos_generated: int = 0
    videos_in_current_round: int = 0
    message: str = ""
    original_prompt: str | None = None
    enhanced_prompt: str | None = None
    recent_videos: list[GeneratedVideo] = Field(default_factory=list)
    # Video-specific progress
    current_video_progress: int | None = None  # 0-100 percent
    current_video_eta: int | None = None  # seconds
    error: str | None = None


class VideoTaskCreate(BaseModel):
    """Response when a video task is created."""

    task_id: str
    status: str = "pending"
    message: str = ""


class VideoProviderInfo(BaseModel):
    """Information about a video provider's capabilities."""

    name: str
    display_name: str
    supported_modes: list[str]
    supports_end_frame: bool = False
    supports_audio: bool = False
    duration_type: str = "enum"
    duration_min: int | None = None
    duration_max: int | None = None
    duration_values: list[int] = Field(default_factory=list)
    duration_default: int = 5
    resolutions: list[str] = Field(default_factory=list)
    aspect_ratios: list[str] = Field(default_factory=list)
    max_sample_count: int = 1
