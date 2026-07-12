"""Generation API endpoints."""

import asyncio

from fastapi import APIRouter, BackgroundTasks, Depends, HTTPException, Request

from ...core.exceptions import ImageEditConflictError, PicTacticError, TaskNotFoundError
from ...i18n import get_text
from ...models.schemas import (
    ErrorResponse,
    GenerationRequest,
    GenerationResult,
    ImageEditRequest,
    ImageEditResult,
    StandaloneEditRequest,
    TaskCreate,
    TaskProgress,
    TaskSummary,
)
from ...services.auth_service import User
from ...services.generation_service import get_generation_service
from ..dependencies import effective_user_id, get_current_user_or_api_key


def _lang(request: Request) -> str:
    return getattr(request.state, "lang", "en")


router = APIRouter()


@router.get("/provider/capabilities")
async def get_provider_capabilities():
    """Get the capabilities of the current default provider."""
    from ...core.config import settings
    from ...providers import get_provider_capabilities as _get_caps

    return _get_caps(settings.default_provider)


@router.post(
    "/",
    response_model=TaskCreate,
    responses={400: {"model": ErrorResponse}},
)
async def create_generation_task(
    request: GenerationRequest,
    background_tasks: BackgroundTasks,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Create a new image generation task.

    This endpoint:
    1. Creates a new task with the provided parameters
    2. Starts the generation workflow in the background
    3. Returns immediately with the task ID

    Use GET /api/v1/generation/{task_id} to check progress.
    """
    service = get_generation_service()

    try:
        task_id = await service.create_task(request, user_id=effective_user_id(user))
        background_tasks.add_task(service.start_task, task_id)

        lang = _lang(req) if req else "en"
        return TaskCreate(
            task_id=task_id,
            status="pending",
            message=get_text("task.created", lang),
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e)) from e


@router.get(
    "/{task_id}",
    response_model=TaskProgress,
    responses={404: {"model": ErrorResponse}},
)
async def get_task_status(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Get the current status of a generation task.

    Returns progress information including:
    - Current round number
    - Number of images generated
    - Best score so far
    - Recent generated images
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        return await service.get_task_status(
            task_id, lang=lang, user_id=effective_user_id(user)
        )
    except TaskNotFoundError:
        raise HTTPException(
            status_code=404, detail=get_text("task.not_found", lang, task_id=task_id)
        )


@router.get(
    "/{task_id}/result",
    response_model=GenerationResult,
    responses={404: {"model": ErrorResponse}},
)
async def get_task_result(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Get the complete result of a generation task.

    Returns full details including:
    - All rounds with images and scores
    - Best images ranked by score
    - Enhanced prompt and template analysis
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        return await service.get_task_result(task_id, user_id=effective_user_id(user))
    except TaskNotFoundError:
        raise HTTPException(
            status_code=404, detail=get_text("task.not_found", lang, task_id=task_id)
        )


@router.post(
    "/{task_id}/images/{image_id}/edit",
    response_model=ImageEditResult,
    responses={
        404: {"model": ErrorResponse},
        400: {"model": ErrorResponse},
        409: {"model": ErrorResponse},
    },
)
async def edit_image(
    task_id: str,
    image_id: str,
    request: ImageEditRequest,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Edit an existing generated image.

    Uses the original image as a reference and applies the edit prompt
    to generate a new version via image-to-image generation.
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        return await service.edit_image(
            task_id=task_id,
            image_id=image_id,
            edit_request=request,
            user_id=effective_user_id(user),
        )
    except TaskNotFoundError as e:
        raise HTTPException(
            status_code=404,
            detail=get_text("task.not_found", lang, task_id=task_id),
        ) from e
    except ImageEditConflictError as e:
        raise HTTPException(status_code=409, detail=e.message) from e
    except PicTacticError as e:
        raise HTTPException(status_code=400, detail=e.message) from e
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e)) from e


@router.post(
    "/{task_id}/cancel",
    responses={404: {"model": ErrorResponse}},
)
async def cancel_task(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Cancel a running generation task.

    Returns:
    - success: True if task was cancelled
    - message: Status message
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        cancelled = await service.cancel_task(task_id, user_id=effective_user_id(user))
        return {
            "success": cancelled,
            "message": get_text("task.cancelled", lang)
            if cancelled
            else get_text("task.not_running", lang),
        }
    except TaskNotFoundError:
        raise HTTPException(
            status_code=404, detail=get_text("task.not_found", lang, task_id=task_id)
        )


@router.delete(
    "/{task_id}",
    responses={404: {"model": ErrorResponse}},
)
async def delete_task(
    task_id: str,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        await service.delete_task(task_id, user_id=effective_user_id(user))
        return {"success": True, "message": get_text("task.deleted", lang)}
    except TaskNotFoundError:
        raise HTTPException(
            status_code=404, detail=get_text("task.not_found", lang, task_id=task_id)
        )


@router.get("/", response_model=list[TaskProgress])
async def list_tasks(limit: int = 20, user: User = Depends(get_current_user_or_api_key)):
    """
    List recent generation tasks.

    Args:
        limit: Maximum number of tasks to return (default: 20)
    """
    service = get_generation_service()
    return await service.list_tasks(limit=limit, user_id=effective_user_id(user))


@router.get("/history/list", response_model=list[TaskSummary])
async def list_task_history(
    limit: int = 50,
    offset: int = 0,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    List task history from database for sidebar display.
    Includes both image and video tasks, merged and sorted by creation time.
    """
    from ...services.video_generation_service import get_video_generation_service

    user_id = effective_user_id(user)

    # Fetch image and video tasks concurrently
    image_service = get_generation_service()
    video_service = get_video_generation_service()

    results = await asyncio.gather(
        image_service.list_history(limit=limit, offset=offset, user_id=user_id),
        video_service.list_history(limit=limit, offset=offset, user_id=user_id),
        return_exceptions=True,
    )

    all_tasks: list[TaskSummary] = []
    for raw in results:
        if isinstance(raw, BaseException):
            import logging

            logging.getLogger(__name__).warning("list_history partial failure: %s", raw)
            continue
        all_tasks.extend(TaskSummary(**t) for t in raw)

    # Merge and sort by creation time (most recent first)
    all_tasks.sort(
        key=lambda x: x.created_at or "",
        reverse=True,
    )

    return all_tasks[:limit]


@router.post(
    "/edit",
    response_model=TaskCreate,
    responses={400: {"model": ErrorResponse}},
)
async def create_edit_task(
    request: StandaloneEditRequest,
    background_tasks: BackgroundTasks,
    req: Request = None,
    user: User = Depends(get_current_user_or_api_key),
):
    """
    Create a standalone image edit task.

    Accepts source images and edit instructions, runs the edit workflow
    in the background, and returns a task_id for polling.
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"

    try:
        task_id = await service.create_edit_task(request, user_id=effective_user_id(user))
        background_tasks.add_task(service.run_edit_task, task_id)

        return TaskCreate(
            task_id=task_id,
            status="pending",
            message=get_text("task.created", lang),
        )
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e)) from e
