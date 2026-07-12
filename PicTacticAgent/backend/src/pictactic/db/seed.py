"""Seed system templates on first startup."""

import logging
from datetime import UTC, datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from .models import PromptTemplate

logger = logging.getLogger(__name__)


async def seed_system_templates(session: AsyncSession) -> None:
    """Import or update system templates from seed data."""
    try:
        from .system_templates import SYSTEM_TEMPLATES
    except ImportError:
        logger.info("No system_templates module found, skipping seed")
        return

    # Build lookup of existing system templates
    result = await session.execute(
        select(PromptTemplate).where(PromptTemplate.is_system.is_(True))
    )
    existing = {t.id: t for t in result.scalars().all()}

    now = datetime.now(UTC)
    created = 0
    updated = 0

    for tmpl in SYSTEM_TEMPLATES:
        if tmpl["id"] in existing:
            # Update prompt_text if changed (e.g. after cleaning)
            row = existing[tmpl["id"]]
            if row.prompt_text != tmpl["prompt_text"]:
                row.prompt_text = tmpl["prompt_text"]
                row.updated_at = now
                updated += 1
        else:
            session.add(PromptTemplate(
                id=tmpl["id"],
                user_id=None,
                title=tmpl["title"],
                description=tmpl.get("description"),
                prompt_text=tmpl["prompt_text"],
                category=tmpl.get("category", "other"),
                tags=tmpl.get("tags", []),
                preview_image_url=tmpl.get("preview_image_url"),
                config=None,
                usage_count=0,
                is_public=True,
                is_system=True,
                source_url=tmpl.get("source_url"),
                source_author=tmpl.get("source_author"),
                created_at=now,
                updated_at=now,
            ))
            created += 1

    await session.commit()
    if created or updated:
        logger.info("System templates: %d created, %d updated", created, updated)
