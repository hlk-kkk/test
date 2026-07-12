"""Task repository for database CRUD operations."""

import uuid
from datetime import datetime, timezone

from sqlalchemy import delete, desc, select
from sqlalchemy.orm import selectinload

from ..models.schemas import (
    GeneratedImage,
    GenerationResult,
    RoundResult,
    ScoreDetails,
    TemplateAnalysis,
)
from .engine import get_session, get_write_lock
from .models import ImageRecord, RoundRecord, TaskRecord


class TaskRepository:
    """CRUD operations for task persistence."""

    def __init__(self) -> None:
        self._write_lock = get_write_lock()

    async def create_task(
        self,
        task_id: str,
        original_prompt: str,
        template_images: list[str],
        config: dict,
        user_id: str | None = None,
    ) -> None:
        async with self._write_lock, get_session() as session:
            record = TaskRecord(
                id=task_id,
                user_id=user_id,
                status="pending",
                original_prompt=original_prompt,
                template_images=template_images,
                config=config,
                created_at=datetime.now(timezone.utc),
                updated_at=datetime.now(timezone.utc),
            )
            session.add(record)
            await session.commit()

    async def save_result(self, result: GenerationResult) -> None:
        """Save a completed/failed task result to DB."""
        async with self._write_lock, get_session() as session:
            stmt = select(TaskRecord).where(TaskRecord.id == result.task_id)
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if task is None:
                task = TaskRecord(id=result.task_id, created_at=result.started_at)
                session.add(task)

            task.status = (
                result.status.value if hasattr(result.status, "value") else str(result.status)
            )
            task.original_prompt = result.original_prompt
            task.enhanced_prompt = result.enhanced_prompt
            task.total_rounds = result.total_rounds
            task.total_images = result.total_images
            task.best_score = max((img.score for img in result.best_images), default=0.0)
            task.error = result.error
            task.updated_at = datetime.now(timezone.utc)
            task.completed_at = result.completed_at or datetime.now(timezone.utc)

            if result.template_analysis:
                task.template_analysis = result.template_analysis.model_dump()

            # Replace previous round/image records so final saves are idempotent.
            # Use batch DELETE + bulk INSERT to minimize DB round-trips.
            await session.execute(delete(ImageRecord).where(ImageRecord.task_id == result.task_id))
            await session.execute(delete(RoundRecord).where(RoundRecord.task_id == result.task_id))
            await session.flush()

            best_image_ids = {img.id for img in result.best_images if img.id}

            # Batch insert rounds, flush once to get all auto-generated IDs
            round_recs = []
            for round_data in result.rounds:
                round_rec = RoundRecord(
                    task_id=result.task_id,
                    round_number=round_data.round_number,
                    best_score=round_data.best_score,
                    prompt_used=round_data.prompt_used,
                    feedback=round_data.feedback,
                )
                session.add(round_rec)
                round_recs.append((round_rec, round_data))

            await session.flush()  # Single flush for all rounds

            # Batch insert images (now round_rec.id is available)
            for round_rec, round_data in round_recs:
                for img in round_data.images:
                    img_rec = ImageRecord(
                        id=img.id or str(uuid.uuid4()),
                        round_id=round_rec.id,
                        task_id=result.task_id,
                        url=img.url,
                        local_path=img.local_path,
                        prompt=img.prompt,
                        provider=img.provider.value
                        if hasattr(img.provider, "value")
                        else str(img.provider),
                        round_number=img.round_number,
                        score=img.score,
                        score_details=img.score_details.model_dump() if img.score_details else None,
                        metadata_=img.metadata if img.metadata else None,
                        is_best=img.id in best_image_ids,
                        created_at=img.created_at,
                    )
                    session.add(img_rec)

            await session.commit()

    async def update_status(self, task_id: str, status: str, error: str | None = None) -> None:
        async with self._write_lock, get_session() as session:
            stmt = select(TaskRecord).where(TaskRecord.id == task_id)
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()
            if task:
                task.status = status
                task.updated_at = datetime.now(timezone.utc)
                if error:
                    task.error = error
                if status in ("completed", "failed", "cancelled"):
                    task.completed_at = datetime.now(timezone.utc)
                await session.commit()

    async def list_tasks(
        self,
        limit: int = 50,
        offset: int = 0,
        user_id: str | None = None,
    ) -> list[dict]:
        """Return task summaries for sidebar display."""
        async with get_session() as session:
            thumbnail_subquery = (
                select(ImageRecord.url)
                .where(
                    ImageRecord.task_id == TaskRecord.id,
                    ImageRecord.is_best.is_(True),
                )
                .order_by(desc(ImageRecord.created_at))
                .limit(1)
                .scalar_subquery()
            )

            stmt = select(TaskRecord, thumbnail_subquery.label("thumbnail_url"))
            if user_id:
                stmt = stmt.where(TaskRecord.user_id == user_id)
            stmt = stmt.order_by(desc(TaskRecord.created_at)).offset(offset).limit(limit)
            db_result = await session.execute(stmt)
            rows = db_result.all()

            from ..services.thumbnail import get_thumbnail_url

            summaries = []
            for t, thumbnail in rows:
                config = t.config or {}

                summaries.append(
                    {
                        "task_id": t.id,
                        "status": t.status,
                        "original_prompt": t.original_prompt,
                        "enhanced_prompt": t.enhanced_prompt,
                        "images_per_round": config.get("images_per_round"),
                        "aspect_ratio": config.get("aspect_ratio"),
                        "size": config.get("size"),
                        "template_images": [],
                        "best_score": t.best_score,
                        "total_rounds": t.total_rounds,
                        "total_images": t.total_images,
                        "thumbnail_url": get_thumbnail_url(thumbnail) if thumbnail else None,
                        "thumbnail_original_url": thumbnail,
                        "created_at": t.created_at.isoformat() if t.created_at else None,
                        "completed_at": t.completed_at.isoformat() if t.completed_at else None,
                        "task_type": "image",
                    }
                )
            return summaries

    async def get_task_result(
        self, task_id: str, user_id: str | None = None
    ) -> GenerationResult | None:
        """Load full task result from DB."""
        async with get_session() as session:
            stmt = select(TaskRecord).where(TaskRecord.id == task_id)
            if user_id:
                stmt = stmt.where(TaskRecord.user_id == user_id)
            stmt = stmt.options(
                selectinload(TaskRecord.rounds).selectinload(RoundRecord.images),
                selectinload(TaskRecord.images),
            )
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if not task:
                return None

            def _build_image(img: ImageRecord) -> GeneratedImage:
                score_details = None
                if img.score_details:
                    score_details = ScoreDetails(**img.score_details)
                from ..models.enums import ImageProvider
                from ..services.thumbnail import get_thumbnail_url

                return GeneratedImage(
                    id=img.id,
                    url=img.url,
                    thumbnail_url=get_thumbnail_url(img.url) if img.url else None,
                    local_path=img.local_path,
                    prompt=img.prompt,
                    provider=ImageProvider(img.provider),
                    round_number=img.round_number,
                    score=img.score,
                    score_details=score_details,
                    metadata=img.metadata_ or {},
                    created_at=img.created_at,
                )

            rounds = []
            for r in task.rounds:
                round_images = [_build_image(img) for img in r.images]
                rounds.append(
                    RoundResult(
                        round_number=r.round_number,
                        images=round_images,
                        best_score=r.best_score,
                        prompt_used=r.prompt_used,
                        feedback=r.feedback,
                    )
                )

            best_images = [_build_image(img) for img in task.images if img.is_best]
            best_images.sort(key=lambda x: x.score, reverse=True)

            template_analysis = None
            if task.template_analysis:
                template_analysis = TemplateAnalysis(**task.template_analysis)

            from ..models.enums import GenerationStatus

            return GenerationResult(
                task_id=task.id,
                status=GenerationStatus(task.status),
                original_prompt=task.original_prompt,
                enhanced_prompt=task.enhanced_prompt,
                template_images=task.template_images or [],
                template_analysis=template_analysis,
                config=task.config or {},
                rounds=rounds,
                total_rounds=task.total_rounds,
                total_images=task.total_images,
                best_images=best_images,
                started_at=task.created_at,
                completed_at=task.completed_at,
                error=task.error,
            )

    async def get_task_owner(self, task_id: str) -> str | None:
        """Return the user_id for a task if it exists."""
        async with get_session() as session:
            stmt = select(TaskRecord.user_id).where(TaskRecord.id == task_id)
            result = await session.execute(stmt)
            return result.scalar_one_or_none()

    async def delete_task(self, task_id: str, user_id: str) -> list[str]:
        """Delete a task and all related records. Returns local image paths for cleanup."""
        async with self._write_lock, get_session() as session:
            stmt = (
                select(TaskRecord)
                .where(
                    TaskRecord.id == task_id,
                    TaskRecord.user_id == user_id,
                )
                .options(selectinload(TaskRecord.images))
            )
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if not task:
                return []

            # Collect local paths before deletion
            local_paths = [img.local_path for img in task.images if img.local_path]

            await session.delete(task)
            await session.commit()

            return local_paths
