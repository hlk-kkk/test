"""Async SQLAlchemy engine and session factory."""

import asyncio
import logging
import os

from sqlalchemy import event
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.pool import StaticPool

from ..core.config import settings
from .models import Base

_engine = None
_session_factory = None
_db_write_lock: asyncio.Lock | None = None
logger = logging.getLogger(__name__)
_SQLITE_BUSY_TIMEOUT_MS = 30_000


def get_write_lock() -> asyncio.Lock:
    """Get the global database write lock (singleton per event loop)."""
    global _db_write_lock
    if _db_write_lock is None:
        _db_write_lock = asyncio.Lock()
    return _db_write_lock


def _get_db_url() -> str:
    db_path = settings.db_path
    os.makedirs(os.path.dirname(os.path.abspath(db_path)), exist_ok=True)
    return f"sqlite+aiosqlite:///{db_path}"


async def init_db() -> None:
    """Initialize database engine and create tables."""
    global _engine, _session_factory
    _engine = create_async_engine(
        _get_db_url(),
        echo=False,
        poolclass=StaticPool,
        connect_args={"timeout": _SQLITE_BUSY_TIMEOUT_MS / 1000, "check_same_thread": False},
    )
    _configure_sqlite(_engine)
    _session_factory = async_sessionmaker(_engine, class_=AsyncSession, expire_on_commit=False)
    async with _engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
        await _apply_migrations(conn)


def get_session() -> AsyncSession:
    """Get a new async session."""
    if _session_factory is None:
        raise RuntimeError("Database not initialized. Call init_db() first.")
    return _session_factory()


async def _apply_migrations(conn) -> None:
    """Apply lightweight SQLite migrations for new columns."""
    try:
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_tasks_created_at ON tasks (created_at DESC)"
        )
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_tasks_user_created_at ON tasks (user_id, created_at DESC)"
        )
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_images_task_best_created_at "
            "ON images (task_id, is_best, created_at DESC)"
        )
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_rounds_task_round_number "
            "ON rounds (task_id, round_number)"
        )

        result = await conn.exec_driver_sql("PRAGMA table_info(tasks)")
        columns = {row[1] for row in result.fetchall()}
        if "user_id" not in columns:
            await conn.exec_driver_sql("ALTER TABLE tasks ADD COLUMN user_id VARCHAR(36)")
            await conn.exec_driver_sql(
                "CREATE INDEX IF NOT EXISTS ix_tasks_user_id ON tasks (user_id)"
            )

        # Video task indexes
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_video_tasks_created_at "
            "ON video_tasks (created_at DESC)"
        )
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_video_tasks_user_created_at "
            "ON video_tasks (user_id, created_at DESC)"
        )
        await conn.exec_driver_sql(
            "CREATE INDEX IF NOT EXISTS ix_video_records_task_created_at "
            "ON video_records (task_id, created_at DESC)"
        )

        # Fix incomplete prompt_templates table: if it exists but lacks expected
        # columns, drop and let create_all rebuild it on next startup.
        result = await conn.exec_driver_sql("PRAGMA table_info(prompt_templates)")
        pt_columns = {row[1] for row in result.fetchall()}
        if pt_columns and "title" not in pt_columns:
            logger.info("Dropping incomplete prompt_templates table for rebuild")
            await conn.exec_driver_sql("DROP TABLE prompt_templates")
            await conn.run_sync(Base.metadata.create_all)

        # Ensure prompt_templates has is_system column and nullable user_id
        # SQLite cannot ALTER COLUMN, so rebuild if user_id is NOT NULL
        result = await conn.exec_driver_sql("PRAGMA table_info(prompt_templates)")
        pt_rows = result.fetchall()
        pt_cols = {r[1] for r in pt_rows}
        user_id_not_null = any(r[1] == "user_id" and r[3] == 1 for r in pt_rows)
        if pt_cols and user_id_not_null:
            logger.info("Rebuilding prompt_templates for nullable user_id + new columns")
            await conn.exec_driver_sql("DROP TABLE prompt_templates")
            await conn.run_sync(Base.metadata.create_all)
        elif pt_cols and "is_system" not in pt_cols:
            await conn.exec_driver_sql(
                "ALTER TABLE prompt_templates ADD COLUMN is_system BOOLEAN DEFAULT 0"
            )
            await conn.exec_driver_sql(
                "ALTER TABLE prompt_templates ADD COLUMN source_url VARCHAR(512)"
            )
            await conn.exec_driver_sql(
                "ALTER TABLE prompt_templates ADD COLUMN source_author VARCHAR(100)"
            )
            logger.info("Added is_system, source_url, source_author to prompt_templates")
    except Exception:
        logger.exception("Database migration failed — aborting startup")
        raise


async def dispose_engine() -> None:
    """Dispose the async engine, flushing WAL and releasing connections."""
    global _engine, _session_factory, _db_write_lock
    if _engine is not None:
        await _engine.dispose()
        _engine = None
    _session_factory = None
    _db_write_lock = None


def _configure_sqlite(engine) -> None:
    """Apply SQLite pragmas on every connection to improve concurrency."""

    @event.listens_for(engine.sync_engine, "connect")
    def _set_sqlite_pragmas(dbapi_connection, _connection_record) -> None:
        cursor = dbapi_connection.cursor()
        cursor.execute("PRAGMA journal_mode=WAL")
        cursor.execute(f"PRAGMA busy_timeout={_SQLITE_BUSY_TIMEOUT_MS}")
        cursor.execute("PRAGMA synchronous=NORMAL")
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()
