"""Enumeration types for PicTacticAgent."""

from enum import Enum


class ImageProvider(str, Enum):
    """Supported image generation providers."""

    GEMINI = "gemini"
    DALLE = "dalle"
    STABILITY = "stability"
    ANTIGRAVITY = "antigravity"
    JIEKOUAI = "jiekouai"


class GenerationStatus(str, Enum):
    """Status of a generation task."""

    PENDING = "pending"
    ENHANCING = "enhancing"
    GENERATING = "generating"
    REVIEWING = "reviewing"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class EvaluatorType(str, Enum):
    """Types of image evaluators."""

    VISION_LLM = "vision_llm"
    CLIP = "clip"
    HYBRID = "hybrid"


class AspectRatio(str, Enum):
    """Supported aspect ratios for image generation."""

    SQUARE = "1:1"
    LANDSCAPE_16_9 = "16:9"
    PORTRAIT_9_16 = "9:16"
    LANDSCAPE_4_3 = "4:3"
    PORTRAIT_3_4 = "3:4"


class ImageSize(str, Enum):
    """Supported image sizes."""

    SIZE_1K = "1K"
    SIZE_2K = "2K"
    SIZE_4K = "4K"
