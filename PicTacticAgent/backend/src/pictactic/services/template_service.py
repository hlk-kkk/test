"""Service layer for prompt template operations."""

import logging

from fastapi import BackgroundTasks

from ..core.config import settings
from ..core.exceptions import TemplateNotFoundError
from ..db.models import PromptTemplate
from ..db.template_repository import TemplateRepository
from ..models.schemas import (
    GenerationRequest,
    TaskCreate,
    TemplateCreate,
    TemplateGenerateRequest,
    TemplateListResponse,
    TemplateResponse,
    TemplateUpdate,
)

logger = logging.getLogger(__name__)


def _to_response(template: PromptTemplate) -> TemplateResponse:
    return TemplateResponse(
        id=template.id,
        user_id=template.user_id,
        title=template.title,
        description=template.description,
        prompt_text=template.prompt_text,
        category=template.category,
        tags=template.tags or [],
        preview_image_url=template.preview_image_url,
        config=template.config,
        usage_count=template.usage_count,
        is_public=template.is_public,
        is_system=getattr(template, "is_system", False),
        source_url=getattr(template, "source_url", None),
        source_author=getattr(template, "source_author", None),
        created_at=template.created_at.isoformat(),
        updated_at=template.updated_at.isoformat(),
    )


class TemplateService:
    def __init__(self):
        self.repo = TemplateRepository()

    async def create_template(self, user_id: str, data: TemplateCreate) -> TemplateResponse:
        template = await self.repo.create(user_id, data)
        return _to_response(template)

    async def get_template(self, template_id: str, user_id: str) -> TemplateResponse:
        template = await self.repo.get_by_id(template_id, user_id)
        if not template:
            raise TemplateNotFoundError(template_id)
        return _to_response(template)

    async def list_templates(
        self,
        user_id: str,
        search: str | None,
        category: str | None,
        page: int,
        page_size: int,
    ) -> TemplateListResponse:
        items, total = await self.repo.list_templates(
            user_id=user_id,
            search=search,
            category=category,
            page=page,
            page_size=page_size,
        )
        return TemplateListResponse(
            items=[_to_response(t) for t in items],
            total=total,
            page=page,
            page_size=page_size,
        )

    async def update_template(
        self, template_id: str, user_id: str, data: TemplateUpdate
    ) -> TemplateResponse:
        template = await self.repo.update(template_id, user_id, data)
        if not template:
            raise TemplateNotFoundError(template_id)
        return _to_response(template)

    async def list_categories(self) -> list[str]:
        return await self.repo.list_categories()

    async def delete_template(self, template_id: str, user_id: str) -> None:
        deleted = await self.repo.delete(template_id, user_id)
        if not deleted:
            raise TemplateNotFoundError(template_id)

    async def generate_from_template(
        self,
        template_id: str,
        user_id: str,
        request: TemplateGenerateRequest,
        background_tasks: BackgroundTasks,
    ) -> TaskCreate:
        template = await self.repo.get_by_id(template_id, user_id)
        if not template:
            raise TemplateNotFoundError(template_id)

        # Build merged config: global defaults -> template config -> request overrides
        base_config = {
            "max_rounds": settings.default_max_rounds,
            "images_per_round": settings.default_images_per_round,
            "quality_threshold": settings.default_quality_threshold,
            "aspect_ratio": settings.default_aspect_ratio,
            "size": settings.default_image_size,
            "enhance_prompt": True,
            "enable_ai_scoring": True,
            "provider": settings.default_provider,
        }

        if template.config:
            for k, v in template.config.items():
                if v is not None:
                    base_config[k] = v

        if request.config_overrides:
            overrides = request.config_overrides.model_dump(exclude_none=True)
            base_config.update(overrides)

        gen_request = GenerationRequest(
            prompt=template.prompt_text,
            template_images=request.template_images,
            **base_config,
        )

        from .generation_service import get_generation_service

        service = get_generation_service()
        task_id = await service.create_task(gen_request, user_id=user_id)
        background_tasks.add_task(service.start_task, task_id)

        await self.repo.increment_usage(template_id)

        return TaskCreate(
            task_id=task_id,
            status="pending",
            message="Task created",
        )


_template_service: TemplateService | None = None


def get_template_service() -> TemplateService:
    global _template_service
    if _template_service is None:
        _template_service = TemplateService()
    return _template_service
