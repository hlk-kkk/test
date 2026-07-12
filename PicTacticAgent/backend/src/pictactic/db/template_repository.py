"""Repository for prompt template CRUD operations."""

import uuid
from datetime import datetime, timezone

from sqlalchemy import String as SAString
from sqlalchemy import cast, desc, func, or_, select, update

from ..models.schemas import TemplateCreate, TemplateUpdate
from .engine import get_session, get_write_lock
from .models import PromptTemplate


class TemplateRepository:
    """CRUD operations for prompt templates."""

    def __init__(self) -> None:
        self._write_lock = get_write_lock()

    async def create(self, user_id: str, data: TemplateCreate) -> PromptTemplate:
        async with self._write_lock, get_session() as session:
            template = PromptTemplate(
                id=str(uuid.uuid4()),
                user_id=user_id,
                title=data.title,
                description=data.description,
                prompt_text=data.prompt_text,
                category=data.category,
                tags=data.tags,
                preview_image_url=data.preview_image_url,
                config=data.config.model_dump(exclude_none=True) if data.config else None,
                usage_count=0,
                is_public=False,
                created_at=datetime.now(timezone.utc),
                updated_at=datetime.now(timezone.utc),
            )
            session.add(template)
            await session.commit()
            await session.refresh(template)
            return template

    async def get_by_id(self, template_id: str, user_id: str) -> PromptTemplate | None:
        async with get_session() as session:
            stmt = select(PromptTemplate).where(
                PromptTemplate.id == template_id,
                or_(
                    PromptTemplate.user_id == user_id,
                    PromptTemplate.is_system.is_(True),
                ),
            )
            result = await session.execute(stmt)
            return result.scalar_one_or_none()

    async def list_templates(
        self,
        user_id: str,
        search: str | None = None,
        category: str | None = None,
        page: int = 1,
        page_size: int = 20,
    ) -> tuple[list[PromptTemplate], int]:
        async with get_session() as session:
            stmt = select(PromptTemplate).where(
                or_(
                    PromptTemplate.user_id == user_id,
                    PromptTemplate.is_system.is_(True),
                )
            )

            if category:
                stmt = stmt.where(PromptTemplate.category == category)

            if search:
                pattern = f"%{search}%"
                stmt = stmt.where(
                    or_(
                        PromptTemplate.title.ilike(pattern),
                        PromptTemplate.description.ilike(pattern),
                        PromptTemplate.prompt_text.ilike(pattern),
                        cast(PromptTemplate.tags, SAString).ilike(pattern),
                    )
                )

            total_stmt = select(func.count()).select_from(stmt.subquery())
            total = (await session.execute(total_stmt)).scalar() or 0

            stmt = stmt.order_by(desc(PromptTemplate.updated_at))
            stmt = stmt.offset((page - 1) * page_size).limit(page_size)

            result = await session.execute(stmt)
            items = list(result.scalars().all())
            return items, total

    async def update(
        self, template_id: str, user_id: str, data: TemplateUpdate
    ) -> PromptTemplate | None:
        async with self._write_lock, get_session() as session:
            stmt = select(PromptTemplate).where(
                PromptTemplate.id == template_id,
                PromptTemplate.user_id == user_id,
            )
            result = await session.execute(stmt)
            template = result.scalar_one_or_none()
            if not template:
                return None

            update_data = data.model_dump(exclude_none=True)
            # category is now a plain string, no .value needed
            if "config" in update_data:
                config_obj = data.config
                update_data["config"] = (
                    config_obj.model_dump(exclude_none=True) if config_obj else None
                )

            for key, value in update_data.items():
                setattr(template, key, value)
            template.updated_at = datetime.now(timezone.utc)

            await session.commit()
            await session.refresh(template)
            return template

    async def delete(self, template_id: str, user_id: str) -> bool:
        async with self._write_lock, get_session() as session:
            stmt = select(PromptTemplate).where(
                PromptTemplate.id == template_id,
                PromptTemplate.user_id == user_id,
            )
            result = await session.execute(stmt)
            template = result.scalar_one_or_none()
            if not template:
                return False
            await session.delete(template)
            await session.commit()
            return True

    async def list_categories(self) -> list[str]:
        """Return all distinct category names."""
        async with get_session() as session:
            stmt = select(PromptTemplate.category).distinct().order_by(PromptTemplate.category)
            result = await session.execute(stmt)
            return [row[0] for row in result.fetchall()]

    async def increment_usage(self, template_id: str) -> None:
        async with self._write_lock, get_session() as session:
            stmt = (
                update(PromptTemplate)
                .where(PromptTemplate.id == template_id)
                .values(usage_count=PromptTemplate.usage_count + 1)
            )
            await session.execute(stmt)
            await session.commit()
