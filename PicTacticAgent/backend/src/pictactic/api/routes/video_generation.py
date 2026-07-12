"""Video generation API endpoints."""

import logging

from fastapi import APIRouter, BackgroundTasks, Depends, HTTPException, Request

from ...core.exceptions import TaskNotFoundError
from ...i18n import get_text
from ...models.schemas import ErrorResponse
from ...models.video_schemas import (
    VideoGenerationRequest,
    VideoGenerationResult,
    VideoProviderInfo,
    VideoTaskCreate,
    VideoTaskProgress,
)
from ...services.auth_service import User
from ...services.video_generation_service import get_video_generation_service
from ..dependencies import effective_user_id, get_current_user_or_api_key

logger = logging.getLogger(__name__)


def _lang(request: Request) -> str:
    return getattr(request.state, "lang", "en")


router = APIRouter(prefix="/video", tags=["Video Generation"])


@router.post(
    "/",
    response_model=VideoTaskCreate,
    responses={400: {"model": ErrorResponse}},
)
async def create_video_task(
    request: VideoGenerationRequest,
    background_tasks: BackgroundTasks,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Create a new video generation task.

    This endpoint:
    1. Creates a new task with the provided parameters
    2. Starts the video generation workflow in the background
    3. Returns immediately with the task ID

    Use GET /api/v1/video/{task_id} to check progress.
    """
    service = get_video_generation_service()

    try:
        task_id = await service.create_task(request, user_id=effective_user_id(user))
        background_tasks.add_task(service.start_task, task_id)

        lang = _lang(req) if req else "en"
        return VideoTaskCreate(
            task_id=task_id,
            status="pending",
            message=get_text("task.created", lang),
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e)) from e


@router.get(
    "/providers/list",
    response_model=list[VideoProviderInfo],
)
async def list_video_providers():
    """
    List available video providers and their capabilities.

    Returns information about each provider including:
    - Supported generation modes
    - Duration options
    - Resolution and aspect ratio support
    """
    providers = _get_video_provider_infos()
    return providers


@router.get(
    "/{task_id}",
    response_model=VideoTaskProgress,
    responses={404: {"model": ErrorResponse}},
)
async def get_video_task_status(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Get the current status of a video generation task.

    Returns progress information including:
    - Current round number
    - Number of videos generated
    - Video generation progress percentage
    - Recent generated videos
    """
    service = get_video_generation_service()
    lang = _lang(req) if req else "en"

    try:
        return await service.get_task_status(task_id, lang=lang, user_id=effective_user_id(user))
    except TaskNotFoundError as err:
        raise HTTPException(
            status_code=404,
            detail=get_text("task.not_found", lang, task_id=task_id),
        ) from err


@router.get(
    "/{task_id}/result",
    response_model=VideoGenerationResult,
    responses={404: {"model": ErrorResponse}},
)
async def get_video_task_result(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Get the complete result of a video generation task.

    Returns full details including:
    - All rounds with generated videos
    - Enhanced prompt
    - Configuration used
    """
    service = get_video_generation_service()
    lang = _lang(req) if req else "en"

    try:
        return await service.get_task_result(task_id, user_id=effective_user_id(user))
    except TaskNotFoundError as err:
        raise HTTPException(
            status_code=404,
            detail=get_text("task.not_found", lang, task_id=task_id),
        ) from err


@router.post(
    "/{task_id}/cancel",
    responses={404: {"model": ErrorResponse}},
)
async def cancel_video_task(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Cancel a running video generation task.

    Returns:
    - success: True if task was cancelled
    - message: Status message
    """
    service = get_video_generation_service()
    lang = _lang(req) if req else "en"

    try:
        cancelled = await service.cancel_task(task_id, user_id=effective_user_id(user))
        return {
            "success": cancelled,
            "message": get_text("task.cancelled", lang)
            if cancelled
            else get_text("task.not_running", lang),
        }
    except TaskNotFoundError as err:
        raise HTTPException(
            status_code=404,
            detail=get_text("task.not_found", lang, task_id=task_id),
        ) from err


@router.delete(
    "/{task_id}",
    responses={404: {"model": ErrorResponse}},
)
async def delete_video_task(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """Delete a video task and all related records."""
    service = get_video_generation_service()
    lang = _lang(req) if req else "en"

    try:
        await service.delete_task(task_id, user_id=effective_user_id(user))
        return {"success": True, "message": get_text("task.deleted", lang)}
    except TaskNotFoundError as err:
        raise HTTPException(
            status_code=404,
            detail=get_text("task.not_found", lang, task_id=task_id),
        ) from err


@router.get("/", response_model=list[VideoTaskProgress])
async def list_video_tasks(
    limit: int = 20,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    List recent video generation tasks.

    Args:
        limit: Maximum number of tasks to return (default: 20)
    """
    service = get_video_generation_service()
    return service.list_tasks(limit=limit, user_id=effective_user_id(user))


def _get_video_provider_infos() -> list[VideoProviderInfo]:
    """Collect capabilities from all registered video providers."""
    provider_infos = []

    # Try to import each provider and get its capabilities
    _provider_classes = []

    try:
        from ...providers.video.kling import KlingVideoProvider

        _provider_classes.append(KlingVideoProvider)
    except ImportError:
        logger.debug("Kling video provider not available")

    try:
        from ...providers.video.sora import SoraVideoProvider

        _provider_classes.append(SoraVideoProvider)
    except ImportError:
        logger.debug("Sora video provider not available")

    try:
        from ...providers.video.veo import VeoVideoProvider

        _provider_classes.append(VeoVideoProvider)
    except ImportError:
        logger.debug("Veo video provider not available")

    try:
        from ...providers.video.seedance import SeedanceVideoProvider

        _provider_classes.append(SeedanceVideoProvider)
    except ImportError:
        logger.debug("Seedance video provider not available")

    for provider_cls in _provider_classes:
        try:
            provider = provider_cls()
            caps = provider.capabilities
            provider_infos.append(
                VideoProviderInfo(
                    name=caps.name,
                    display_name=caps.display_name,
                    supported_modes=[m.value for m in caps.supported_modes],
                    supports_end_frame=caps.supports_end_frame,
                    supports_audio=caps.supports_audio,
                    duration_type=caps.duration_type,
                    duration_min=caps.duration_min,
                    duration_max=caps.duration_max,
                    duration_values=caps.duration_values,
                    duration_default=caps.duration_default,
                    resolutions=caps.resolutions,
                    aspect_ratios=caps.aspect_ratios,
                    max_sample_count=caps.max_sample_count,
                )
            )
        except Exception:
            logger.warning(
                "Failed to get capabilities for %s",
                provider_cls.__name__,
                exc_info=True,
            )

    return provider_infos
