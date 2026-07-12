"""Video generation providers."""

from .base import (
    VideoGenerationMode,
    VideoGenerationParams,
    VideoGenerationResult,
    VideoProvider,
    VideoProviderCapabilities,
)
from .kling import KlingVideoProvider
from .seedance import SeedanceVideoProvider
from .sora import SoraVideoProvider
from .veo import VeoVideoProvider

__all__ = [
    "VideoProvider",
    "VideoGenerationParams",
    "VideoGenerationResult",
    "VideoGenerationMode",
    "VideoProviderCapabilities",
    "KlingVideoProvider",
    "SeedanceVideoProvider",
    "SoraVideoProvider",
    "VeoVideoProvider",
    "get_video_provider",
    "get_video_provider_capabilities",
]

_VIDEO_PROVIDERS: dict[str, type[VideoProvider]] = {
    "kling": KlingVideoProvider,
    "seedance": SeedanceVideoProvider,
    "sora": SoraVideoProvider,
    "veo": VeoVideoProvider,
}


def get_video_provider(provider_name: str) -> VideoProvider:
    """Get a video provider by name."""
    cls = _VIDEO_PROVIDERS.get(provider_name)
    if not cls:
        from ...core.exceptions import ProviderError

        raise ProviderError(
            message=(
                f"Unknown video provider: {provider_name}. "
                f"Available: {list(_VIDEO_PROVIDERS.keys())}"
            ),
            provider=provider_name,
        )
    return cls()


def get_video_provider_capabilities(
    provider_name: str | None = None,
) -> list[VideoProviderCapabilities]:
    """Get capabilities for one or all video providers."""
    if provider_name:
        provider = get_video_provider(provider_name)
        return [provider.capabilities]
    return [cls().capabilities for cls in _VIDEO_PROVIDERS.values()]
