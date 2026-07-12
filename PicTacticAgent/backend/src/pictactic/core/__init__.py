"""Core module for configuration, exceptions, and constants."""

from .config import settings
from .exceptions import (
    PicTacticError,
    GenerationError,
    EvaluationError,
    ProviderError,
)
from .constants import (
    DEFAULT_MAX_ROUNDS,
    DEFAULT_IMAGES_PER_ROUND,
    DEFAULT_QUALITY_THRESHOLD,
)

__all__ = [
    "settings",
    "PicTacticError",
    "GenerationError",
    "EvaluationError",
    "ProviderError",
    "DEFAULT_MAX_ROUNDS",
    "DEFAULT_IMAGES_PER_ROUND",
    "DEFAULT_QUALITY_THRESHOLD",
]
