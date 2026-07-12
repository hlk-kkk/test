"""Prompt template API endpoints."""

from fastapi import APIRouter, BackgroundTasks, Depends, HTTPException, Query

from ...core.exceptions import TemplateNotFoundError
from ...models.schemas import (
    TaskCreate,
    TemplateCreate,
    TemplateGenerateRequest,
    TemplateListResponse,
    TemplateResponse,
    TemplateUpdate,
)
from ...services.auth_service import User
from ...services.template_service import get_template_service
from ..dependencies import get_current_user

router = APIRouter()


@router.post("/", response_model=TemplateResponse, status_code=201)
async def create_template(
    data: TemplateCreate,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    return await service.create_template(user.user_id, data)


@router.get("/", response_model=TemplateListResponse)
async def list_templates(
    search: str | None = None,
    category: str | None = None,
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=20, ge=1, le=100),
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    return await service.list_templates(
        user_id=user.user_id,
        search=search,
        category=category,
        page=page,
        page_size=page_size,
    )


@router.get("/categories", response_model=list[str])
async def list_categories(
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    return await service.list_categories()


@router.get("/{template_id}", response_model=TemplateResponse)
async def get_template(
    template_id: str,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        return await service.get_template(template_id, user.user_id)
    except TemplateNotFoundError as e:
        raise HTTPException(status_code=404, detail="Template not found") from e


@router.put("/{template_id}", response_model=TemplateResponse)
async def update_template(
    template_id: str,
    data: TemplateUpdate,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        template = await service.get_template(template_id, user.user_id)
        if template.is_system:
            raise HTTPException(status_code=403, detail="System templates cannot be modified")
        return await service.update_template(template_id, user.user_id, data)
    except TemplateNotFoundError as e:
        raise HTTPException(status_code=404, detail="Template not found") from e


@router.delete("/{template_id}")
async def delete_template(
    template_id: str,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        template = await service.get_template(template_id, user.user_id)
        if template.is_system:
            raise HTTPException(status_code=403, detail="System templates cannot be deleted")
        await service.delete_template(template_id, user.user_id)
        return {"success": True}
    except TemplateNotFoundError as e:
        raise HTTPException(status_code=404, detail="Template not found") from e


@router.post("/{template_id}/generate", response_model=TaskCreate, status_code=201)
async def generate_from_template(
    template_id: str,
    request: TemplateGenerateRequest,
    background_tasks: BackgroundTasks,
    user: User = Depends(get_current_user),
):
    service = get_template_service()
    try:
        return await service.generate_from_template(
            template_id,
            user.user_id,
            request,
            background_tasks,
        )
    except TemplateNotFoundError as e:
        raise HTTPException(status_code=404, detail="Template not found") from e
