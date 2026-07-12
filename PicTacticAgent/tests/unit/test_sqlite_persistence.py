"""Tests for SQLite persistence behavior and task history queries."""

from datetime import datetime

import pytest
from pictactic.db import engine as db_engine
from pictactic.db.engine import get_session
from pictactic.db.models import ImageRecord, RoundRecord
from pictactic.db.repository import TaskRepository
from pictactic.models.enums import GenerationStatus, ImageProvider
from pictactic.models.schemas import GeneratedImage, GenerationResult, RoundResult
from sqlalchemy import func, select, text


@pytest.fixture
async def isolated_db(tmp_path, monkeypatch):
    """Initialize a temporary SQLite database for repository tests."""
    db_path = tmp_path / "pictactic-test.db"
    original_db_path = db_engine.settings.db_path
    original_engine = db_engine._engine
    original_factory = db_engine._session_factory

    monkeypatch.setattr(db_engine.settings, "db_path", str(db_path), raising=False)
    db_engine._engine = None
    db_engine._session_factory = None
    await db_engine.init_db()

    try:
        yield
    finally:
        if db_engine._engine is not None:
            await db_engine._engine.dispose()
        db_engine._engine = original_engine
        db_engine._session_factory = original_factory
        monkeypatch.setattr(db_engine.settings, "db_path", original_db_path, raising=False)


def _build_generation_result(task_id: str) -> GenerationResult:
    created_at = datetime.utcnow()
    img_1 = GeneratedImage(
        id="img-1",
        url="https://example.com/1.png",
        local_path="/tmp/1.png",
        prompt="prompt",
        provider=ImageProvider.GEMINI,
        round_number=1,
        score=0.72,
        metadata={"seed": 1},
        created_at=created_at,
    )
    img_2 = GeneratedImage(
        id="img-2",
        url="https://example.com/2.png",
        local_path="/tmp/2.png",
        prompt="prompt",
        provider=ImageProvider.GEMINI,
        round_number=1,
        score=0.91,
        metadata={"seed": 2},
        created_at=created_at,
    )
    return GenerationResult(
        task_id=task_id,
        status=GenerationStatus.COMPLETED,
        original_prompt="prompt",
        enhanced_prompt="enhanced prompt",
        template_images=[],
        config={"images_per_round": 2, "aspect_ratio": "16:9", "size": "2K"},
        rounds=[
            RoundResult(
                round_number=1,
                images=[img_1, img_2],
                best_score=0.91,
                prompt_used="enhanced prompt",
                feedback="good",
            )
        ],
        total_rounds=1,
        total_images=2,
        best_images=[img_2],
        started_at=created_at,
        completed_at=created_at,
    )


@pytest.mark.asyncio
async def test_init_db_enables_sqlite_concurrency_pragmas_and_indexes(isolated_db):
    async with get_session() as session:
        journal_mode = (await session.execute(text("PRAGMA journal_mode"))).scalar_one()
        busy_timeout = (await session.execute(text("PRAGMA busy_timeout"))).scalar_one()
        image_indexes = await session.execute(text("PRAGMA index_list('images')"))
        image_index_names = {row[1] for row in image_indexes}

    assert str(journal_mode).lower() == "wal"
    assert busy_timeout == 30000
    assert "ix_images_task_best_created_at" in image_index_names


@pytest.mark.asyncio
async def test_save_result_is_idempotent_and_history_returns_thumbnail(isolated_db):
    repo = TaskRepository()
    task_id = "task-1"
    result = _build_generation_result(task_id)

    await repo.create_task(
        task_id=task_id,
        original_prompt="prompt",
        template_images=[],
        config={"images_per_round": 2, "aspect_ratio": "16:9", "size": "2K"},
    )

    await repo.save_result(result)
    await repo.save_result(result)

    async with get_session() as session:
        round_count = (
            await session.execute(
                select(func.count()).select_from(RoundRecord).where(RoundRecord.task_id == task_id)
            )
        ).scalar_one()
        image_count = (
            await session.execute(
                select(func.count()).select_from(ImageRecord).where(ImageRecord.task_id == task_id)
            )
        ).scalar_one()

    history = await repo.list_tasks(limit=10)

    assert round_count == 1
    assert image_count == 2
    assert history[0]["thumbnail_url"] == "https://example.com/2.png"
