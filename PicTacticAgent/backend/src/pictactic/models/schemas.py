"""Pydantic schemas for PicTacticAgent."""

from datetime import datetime, timezone

from pydantic import BaseModel, Field, field_validator

from .enums import GenerationStatus, ImageProvider


class TemplateAnalysis(BaseModel):
    """Analysis result of template images."""

    style: str = Field(description="Identified visual style")
    colors: list[str] = Field(default_factory=list, description="Main color palette")
    layout: str = Field(description="Layout and composition description")
    mood: str = Field(description="Overall mood/atmosphere")
    elements: list[str] = Field(default_factory=list, description="Key visual elements")
    suggested_keywords: list[str] = Field(
        default_factory=list, description="Suggested keywords for prompt"
    )


class GenerationRequest(BaseModel):
    """Request to start an image generation task."""

    prompt: str = Field(..., min_length=1, description="User's original prompt")
    template_images: list[str] = Field(
        default_factory=list, description="URLs or paths to template images"
    )
    max_rounds: int = Field(default=3, ge=1, le=10, description="Maximum generation rounds")
    images_per_round: int = Field(default=5, ge=1, le=10, description="Images per round")
    quality_threshold: float = Field(
        default=0.7, ge=0.0, le=1.0, description="Quality threshold to stop"
    )
    enable_ai_scoring: bool = Field(
        default=True, description="Whether to enable AI scoring in evaluation"
    )
    aspect_ratio: str = Field(default="16:9", description="Image aspect ratio")
    size: str = Field(default="2K", description="Image size")
    enhance_prompt: bool = Field(default=True, description="Whether to enhance prompt with LLM")
    provider: ImageProvider = Field(default=None, description="Image provider")

    def __init__(self, **data):
        if data.get("provider") is None:
            from ..core.config import settings

            data["provider"] = settings.default_provider
        super().__init__(**data)


class ScoreDetails(BaseModel):
    """Detailed scoring breakdown."""

    model_config = {"extra": "allow"}

    prompt_alignment: float = Field(default=0.0, ge=0.0, le=1.0, description="提示词匹配度")
    text_quality: float = Field(default=0.0, ge=0.0, le=1.0, description="文字质量")
    layout_rationality: float = Field(default=0.0, ge=0.0, le=1.0, description="布局合理性")
    visual_attraction: float = Field(default=0.0, ge=0.0, le=1.0, description="视觉吸引力")
    technical_integrity: float = Field(default=0.0, ge=0.0, le=1.0, description="技术完整性")
    # I2I mode additional field
    identity_preservation: float | None = Field(
        default=None, ge=0.0, le=1.0, description="人像一致性"
    )


class GeneratedImage(BaseModel):
    """A generated image with metadata."""

    id: str = Field(description="Unique image ID")
    url: str = Field(description="Image URL")
    thumbnail_url: str | None = Field(default=None, description="Thumbnail image URL")
    local_path: str | None = Field(default=None, description="Local file path")
    prompt: str = Field(description="Prompt used for generation")
    provider: ImageProvider = Field(description="Provider used")
    round_number: int = Field(ge=1, description="Which round this was generated in")
    score: float = Field(default=0.0, ge=0.0, le=1.0, description="Overall score")
    score_details: ScoreDetails | None = Field(default=None, description="Detailed scores")
    metadata: dict = Field(default_factory=dict, description="Additional metadata")
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc))


class RoundResult(BaseModel):
    """Results from a single generation round."""

    round_number: int
    images: list[GeneratedImage]
    best_score: float
    prompt_used: str
    feedback: str | None = None


class GenerationResult(BaseModel):
    """Complete result of a generation task."""

    task_id: str
    status: GenerationStatus
    original_prompt: str
    enhanced_prompt: str | None = None
    template_images: list[str] = Field(default_factory=list)
    template_analysis: TemplateAnalysis | None = None
    config: dict | None = None
    rounds: list[RoundResult] = Field(default_factory=list)
    total_rounds: int = 0
    total_images: int = 0
    best_images: list[GeneratedImage] = Field(default_factory=list)
    started_at: datetime
    completed_at: datetime | None = None
    error: str | None = None


class TaskProgress(BaseModel):
    """Progress update for a generation task."""

    task_id: str
    status: GenerationStatus
    current_round: int
    max_rounds: int
    images_per_round: int | None = None
    aspect_ratio: str | None = None
    size: str | None = None
    template_images: list[str] = Field(default_factory=list)
    images_generated: int
    images_in_current_round: int
    current_best_score: float
    quality_threshold: float
    message: str
    original_prompt: str | None = None
    enhanced_prompt: str | None = None
    recent_images: list[GeneratedImage] = Field(default_factory=list)
    error: str | None = None


class TaskCreate(BaseModel):
    """Response when a task is created."""

    task_id: str
    status: GenerationStatus
    message: str


class TaskSummary(BaseModel):
    """Summary of a task for sidebar/history display."""

    task_id: str
    status: str
    original_prompt: str
    enhanced_prompt: str | None = None
    images_per_round: int | None = None
    aspect_ratio: str | None = None
    size: str | None = None
    template_images: list[str] = Field(default_factory=list)
    best_score: float = 0.0
    total_rounds: int = 0
    total_images: int = 0
    thumbnail_url: str | None = None
    created_at: str | None = None
    completed_at: str | None = None
    task_type: str = Field(default="image", description="Task type: 'image' or 'video'")


class ImageEditRequest(BaseModel):
    """Request to edit an existing generated image."""

    edit_prompt: str = Field(..., min_length=1, description="Edit instruction prompt")
    enhance_prompt: bool = Field(
        default=True, description="Whether to enhance the edit prompt with LLM"
    )


class ImageEditResult(BaseModel):
    """Result of an image edit operation."""

    original_image_id: str = Field(description="ID of the original image")
    new_image: GeneratedImage = Field(description="The newly generated edited image")
    edit_prompt: str = Field(description="Edit prompt used")
    enhanced_edit_prompt: str | None = Field(
        default=None, description="Enhanced edit prompt if enhancement was enabled"
    )


class StandaloneEditRequest(BaseModel):
    """Request to start a standalone image edit task (via CLI submit)."""

    prompt: str = Field(..., min_length=1, description="Edit instruction prompt")
    source_images: list[str] = Field(
        ..., min_length=1, description="Source image data URIs or URLs"
    )
    count: int = Field(default=1, ge=1, le=10, description="Number of outputs")
    aspect_ratio: str = Field(default="16:9", description="Image aspect ratio")
    size: str = Field(default="2K", description="Image size")
    provider: ImageProvider = Field(default=None, description="Image provider")
    enhance_prompt: bool = Field(default=False, description="Whether to enhance edit prompt")

    def __init__(self, **data):
        if data.get("provider") is None:
            from ..core.config import settings

            data["provider"] = settings.default_provider
        super().__init__(**data)


class ErrorResponse(BaseModel):
    """Error response format."""

    error: str
    detail: str | None = None
    code: str | None = None


# ========== Prompt Template Schemas ==========


class TemplateConfigOverride(BaseModel):
    """Optional generation config to store with a template."""

    max_rounds: int | None = Field(default=None, ge=1, le=10)
    images_per_round: int | None = Field(default=None, ge=1, le=10)
    quality_threshold: float | None = Field(default=None, ge=0.0, le=1.0)
    aspect_ratio: str | None = None
    size: str | None = None
    enhance_prompt: bool | None = None
    enable_ai_scoring: bool | None = None
    provider: str | None = None


class TemplateCreate(BaseModel):
    """Request body for creating a prompt template."""

    title: str = Field(..., min_length=1, max_length=200)
    description: str | None = Field(default=None, max_length=2000)
    prompt_text: str = Field(..., min_length=1)
    category: str = Field(default="other", max_length=50)
    tags: list[str] = Field(default_factory=list, max_length=20)
    preview_image_url: str | None = Field(default=None, max_length=512)
    config: TemplateConfigOverride | None = None

    @field_validator("category", mode="before")
    @classmethod
    def empty_str_to_default(cls, v):
        if v == "":
            return "other"
        return v


class TemplateUpdate(BaseModel):
    """Request body for updating a prompt template. All fields optional."""

    title: str | None = Field(default=None, min_length=1, max_length=200)
    description: str | None = Field(default=None, max_length=2000)
    prompt_text: str | None = Field(default=None, min_length=1)
    category: str | None = Field(default=None, max_length=50)
    tags: list[str] | None = Field(default=None, max_length=20)
    preview_image_url: str | None = Field(default=None, max_length=512)
    config: TemplateConfigOverride | None = None

    @field_validator("category", mode="before")
    @classmethod
    def empty_str_to_none(cls, v):
        if v == "":
            return None
        return v


class TemplateResponse(BaseModel):
    """Single template in API responses."""

    id: str
    user_id: str | None = None
    title: str
    description: str | None = None
    prompt_text: str
    category: str
    tags: list[str] = Field(default_factory=list)
    preview_image_url: str | None = None
    config: dict | None = None
    usage_count: int = 0
    is_public: bool = False
    is_system: bool = False
    source_url: str | None = None
    source_author: str | None = None
    created_at: str
    updated_at: str


class TemplateListResponse(BaseModel):
    """Paginated list of templates."""

    items: list[TemplateResponse]
    total: int
    page: int
    page_size: int


class TemplateGenerateRequest(BaseModel):
    """Request to generate images from a template, with optional overrides."""

    config_overrides: TemplateConfigOverride | None = None
    template_images: list[str] = Field(default_factory=list)
