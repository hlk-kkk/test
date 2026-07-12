"""Video task repository for database CRUD operations."""

import uuid
from datetime import datetime, timezone

from sqlalchemy import delete, desc, select
from sqlalchemy.orm import selectinload

from ..models.video_schemas import (
    GeneratedVideo,
    VideoGenerationResult,
    VideoRoundResult,
)
from .engine import get_session, get_write_lock
from .models import VideoRecord, VideoRoundRecord, VideoTaskRecord


class VideoTaskRepository:
    """CRUD operations for video task persistence."""

    def __init__(self) -> None:
        self._write_lock = get_write_lock()

    async def create_task(
        self,
        task_id: str,
        original_prompt: str,
        start_image: str | None,
        end_image: str | None,
        config: dict,
        user_id: str | None = None,
    ) -> None:
        async with self._write_lock, get_session() as session:
            record = VideoTaskRecord(
                id=task_id,
                user_id=user_id,
                status="pending",
                original_prompt=original_prompt,
                start_image=start_image,
                end_image=end_image,
                config=config,
                created_at=datetime.now(timezone.utc),
                updated_at=datetime.now(timezone.utc),
            )
            session.add(record)
            await session.commit()

    async def save_result(self, result: VideoGenerationResult) -> None:
        """Save a completed/failed video task result to DB."""
        async with self._write_lock, get_session() as session:
            stmt = select(VideoTaskRecord).where(VideoTaskRecord.id == result.task_id)
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if task is None:
                task = VideoTaskRecord(id=result.task_id, created_at=result.started_at)
                session.add(task)

            task.status = result.status
            task.original_prompt = result.original_prompt
            task.enhanced_prompt = result.enhanced_prompt
            task.start_image = result.start_image
            task.end_image = result.end_image
            task.total_rounds = result.total_rounds
            task.total_videos = result.total_videos
            task.error = result.error
            task.updated_at = datetime.now(timezone.utc)
            task.completed_at = result.completed_at or datetime.now(timezone.utc)

            if result.config:
                task.config = result.config

            # Replace previous round/video records for idempotency.
            await session.execute(
                delete(VideoRecord).where(VideoRecord.task_id == result.task_id)
            )
            await session.execute(
                delete(VideoRoundRecord).where(VideoRoundRecord.task_id == result.task_id)
            )

            for round_data in result.rounds:
                round_rec = VideoRoundRecord(
                    task_id=result.task_id,
                    round_number=round_data.round_number,
                )
                session.add(round_rec)
                await session.flush()

                for vid in round_data.videos:
                    vid_rec = VideoRecord(
                        id=vid.id or str(uuid.uuid4()),
                        round_id=round_rec.id,
                        task_id=result.task_id,
                        url=vid.url,
                        local_path=vid.local_path,
                        video_type=vid.video_type,
                        duration=vid.duration,
                        round_number=vid.round_number,
                        provider=vid.provider,
                        provider_task_id=vid.provider_task_id,
                        metadata_=vid.metadata if vid.metadata else None,
                        created_at=vid.created_at or datetime.now(timezone.utc),
                    )
                    session.add(vid_rec)

            await session.commit()

    async def update_status(
        self, task_id: str, status: str, error: str | None = None
    ) -> None:
        async with self._write_lock, get_session() as session:
            stmt = select(VideoTaskRecord).where(VideoTaskRecord.id == task_id)
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
        """Return video task summaries for sidebar display."""
        async with get_session() as session:
            # Get first video URL as thumbnail
            thumbnail_subquery = (
                select(VideoRecord.url)
                .where(VideoRecord.task_id == VideoTaskRecord.id)
                .order_by(desc(VideoRecord.created_at))
                .limit(1)
                .scalar_subquery()
            )

            stmt = select(VideoTaskRecord, thumbnail_subquery.label("thumbnail_url"))
            if user_id:
                stmt = stmt.where(VideoTaskRecord.user_id == user_id)
            stmt = stmt.order_by(desc(VideoTaskRecord.created_at)).offset(offset).limit(limit)
            db_result = await session.execute(stmt)
            rows = db_result.all()

            summaries = []
            for t, thumbnail in rows:
                summaries.append(
                    {
                        "task_id": t.id,
                        "status": t.status,
                        "original_prompt": t.original_prompt,
                        "enhanced_prompt": t.enhanced_prompt,
                        "best_score": 0.0,
                        "total_rounds": t.total_rounds,
                        "total_videos": t.total_videos,
                        "thumbnail_url": thumbnail,
                        "created_at": t.created_at.isoformat() if t.created_at else None,
                        "completed_at": t.completed_at.isoformat() if t.completed_at else None,
                        "task_type": "video",
                    }
                )
            return summaries

    async def get_task_result(
        self, task_id: str, user_id: str | None = None
    ) -> VideoGenerationResult | None:
        """Load full video task result from DB."""
        async with get_session() as session:
            stmt = select(VideoTaskRecord).where(VideoTaskRecord.id == task_id)
            if user_id:
                stmt = stmt.where(VideoTaskRecord.user_id == user_id)
            stmt = stmt.options(
                selectinload(VideoTaskRecord.rounds).selectinload(VideoRoundRecord.videos),
                selectinload(VideoTaskRecord.videos),
            )
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if not task:
                return None

            def _build_video(vid: VideoRecord) -> GeneratedVideo:
                return GeneratedVideo(
                    id=vid.id,
                    url=vid.url,
                    local_path=vid.local_path,
                    video_type=vid.video_type,
                    duration=vid.duration,
                    round_number=vid.round_number,
                    provider=vid.provider,
                    provider_task_id=vid.provider_task_id,
                    metadata=vid.metadata_ or {},
                    created_at=vid.created_at,
                )

            rounds = []
            all_videos = []
            for r in task.rounds:
                round_videos = [_build_video(vid) for vid in r.videos]
                rounds.append(
                    VideoRoundResult(
                        round_number=r.round_number,
                        videos=round_videos,
                    )
                )
                all_videos.extend(round_videos)

            return VideoGenerationResult(
                task_id=task.id,
                status=task.status,
                original_prompt=task.original_prompt,
                enhanced_prompt=task.enhanced_prompt,
                start_image=task.start_image,
                end_image=task.end_image,
                config=task.config or {},
                rounds=rounds,
                total_rounds=task.total_rounds,
                total_videos=task.total_videos,
                videos=all_videos,
                started_at=task.created_at,
                completed_at=task.completed_at,
                error=task.error,
            )

    async def get_task_owner(self, task_id: str) -> str | None:
        """Return the user_id for a video task if it exists."""
        async with get_session() as session:
            stmt = select(VideoTaskRecord.user_id).where(VideoTaskRecord.id == task_id)
            result = await session.execute(stmt)
            return result.scalar_one_or_none()

    async def delete_task(self, task_id: str, user_id: str) -> list[str]:
        """Delete a video task and all related records. Returns local video paths."""
        async with self._write_lock, get_session() as session:
            stmt = (
                select(VideoTaskRecord)
                .where(
                    VideoTaskRecord.id == task_id,
                    VideoTaskRecord.user_id == user_id,
                )
                .options(selectinload(VideoTaskRecord.videos))
            )
            db_result = await session.execute(stmt)
            task = db_result.scalar_one_or_none()

            if not task:
                return []

            local_paths = [vid.local_path for vid in task.videos if vid.local_path]

            await session.delete(task)
            await session.commit()

            return local_paths
