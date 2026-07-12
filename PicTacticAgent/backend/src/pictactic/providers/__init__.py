"""Image generation providers."""

from .antigravity import AntigravityImageProvider
from .base import GenerationMode, GenerationParams, GenerationResult, ImageProvider
from .gemini import GeminiImageProvider
from .gemini_web import GeminiWebImageProvider
from .jiekouai import JiekouaiImageProvider

__all__ = [
    "ImageProvider",
    "GenerationParams",
    "GenerationResult",
    "GenerationMode",
    "GeminiImageProvider",
    "JiekouaiImageProvider",
    "AntigravityImageProvider",
    "GeminiWebImageProvider",
    "get_provider",
    "get_provider_capabilities",
]


_PROVIDERS: dict[str, type[ImageProvider]] = {
    "gemini": GeminiImageProvider,
    "jiekouai": JiekouaiImageProvider,
    "antigravity": AntigravityImageProvider,
    "gemini_web": GeminiWebImageProvider,
}


def get_provider(provider_name: str) -> ImageProvider:
    """Get an image provider by name."""
    if provider_name not in _PROVIDERS:
        raise ValueError(f"Unknown provider: {provider_name}. Available: {list(_PROVIDERS.keys())}")
    return _PROVIDERS[provider_name]()


def get_provider_capabilities(provider_name: str) -> dict:
    """Get the capabilities of a provider without instantiating it fully."""
    provider = get_provider(provider_name)
    return {
        "name": provider.name,
        "supported_modes": [m.value for m in provider.supported_modes],
        "supports_batch": provider.supports_batch,
    }
