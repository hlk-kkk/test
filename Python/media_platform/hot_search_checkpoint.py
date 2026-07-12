# -*- coding: utf-8 -*-

from typing import Dict, List, Optional

import config
from model.m_checkpoint import Checkpoint
from pkg.tools import utils
from pkg.tools.user_privacy import sanitize_raw_payload


def get_hot_search_max_items_count() -> int:
    return max(int(getattr(config, "HOT_SEARCH_MAX_ITEMS_COUNT", 50) or 0), 1)


def get_hot_search_per_item_content_count() -> int:
    return max(int(getattr(config, "HOT_SEARCH_PER_ITEM_CONTENT_COUNT", 4) or 0), 1)


def _is_resumeable_hot_search_checkpoint(checkpoint: Optional[Checkpoint]) -> bool:
    if not checkpoint or checkpoint.is_completed:
        return False

    snapshot_ts = checkpoint.current_hot_search_snapshot_ts
    snapshot_items = checkpoint.current_hot_search_snapshot_items or []
    if not snapshot_ts or not snapshot_items:
        return False

    max_age_seconds = int(
        getattr(config, "HOT_SEARCH_CHECKPOINT_MAX_AGE_SECONDS", 6 * 60 * 60) or 0
    )
    if max_age_seconds <= 0:
        return True

    return utils.get_current_timestamp() - snapshot_ts <= max_age_seconds


async def load_or_create_hot_search_checkpoint(
    checkpoint_manager,
    *,
    platform: str,
    mode: str,
    log_prefix: str,
    defaults: Optional[Dict] = None,
) -> Checkpoint:
    checkpoint = None
    if config.ENABLE_CHECKPOINT:
        checkpoint = await checkpoint_manager.load_checkpoint(
            platform=platform,
            mode=mode,
            checkpoint_id=config.SPECIFIED_CHECKPOINT_ID or None,
        )
        if config.SPECIFIED_CHECKPOINT_ID and checkpoint:
            utils.logger.info(
                f"{log_prefix} Resume specified checkpoint: {checkpoint.id}, "
                f"hot index: {checkpoint.current_homefeed_note_index}"
            )
        elif checkpoint and _is_resumeable_hot_search_checkpoint(checkpoint):
            utils.logger.info(
                f"{log_prefix} Auto resume unfinished hot-search snapshot: "
                f"{checkpoint.id}, hot index: {checkpoint.current_homefeed_note_index}"
            )
        elif not config.SPECIFIED_CHECKPOINT_ID:
            checkpoint = None

    if not checkpoint:
        checkpoint = Checkpoint(
            platform=platform,
            mode=mode,
            current_homefeed_note_index=0,
            is_completed=False,
            **(defaults or {}),
        )

    return await checkpoint_manager.save_checkpoint(checkpoint)


async def update_hot_search_checkpoint(
    checkpoint_manager,
    checkpoint: Checkpoint,
    **fields,
) -> Checkpoint:
    latest_checkpoint = None
    if checkpoint.id and hasattr(checkpoint_manager, "load_checkpoint_by_id"):
        latest_checkpoint = await checkpoint_manager.load_checkpoint_by_id(checkpoint.id)
    checkpoint_to_update = latest_checkpoint or checkpoint
    for field_name, field_value in fields.items():
        setattr(checkpoint_to_update, field_name, field_value)
    return await checkpoint_manager.update_checkpoint(checkpoint_to_update)


async def ensure_hot_search_snapshot(
    checkpoint_manager,
    checkpoint: Checkpoint,
    hot_search_items: List[Dict],
) -> Checkpoint:
    if checkpoint.current_hot_search_snapshot_items:
        return checkpoint

    snapshot_items = sanitize_raw_payload(
        hot_search_items[: get_hot_search_max_items_count()]
    )
    return await update_hot_search_checkpoint(
        checkpoint_manager,
        checkpoint,
        current_hot_search_snapshot_ts=utils.get_current_timestamp(),
        current_hot_search_snapshot_items=snapshot_items,
        current_homefeed_note_index=checkpoint.current_homefeed_note_index or 0,
        is_completed=False,
    )


def get_hot_search_content_ids(checkpoint: Checkpoint, hot_item_id: str) -> List[str]:
    content_ids = (checkpoint.current_hot_search_content_ids or {}).get(hot_item_id, [])
    return [str(content_id) for content_id in content_ids if content_id]


async def save_hot_search_content_ids(
    checkpoint_manager,
    checkpoint: Checkpoint,
    hot_item_id: str,
    content_ids: List[str],
) -> Checkpoint:
    latest_checkpoint = checkpoint
    if checkpoint.id and hasattr(checkpoint_manager, "load_checkpoint_by_id"):
        latest_checkpoint = (
            await checkpoint_manager.load_checkpoint_by_id(checkpoint.id)
        ) or checkpoint

    content_ids_map = dict(latest_checkpoint.current_hot_search_content_ids or {})
    content_ids_map[hot_item_id] = [str(content_id) for content_id in content_ids if content_id]
    return await update_hot_search_checkpoint(
        checkpoint_manager,
        latest_checkpoint,
        current_hot_search_content_ids=content_ids_map,
    )


async def mark_hot_search_checkpoint_completed(
    checkpoint_manager,
    checkpoint: Checkpoint,
) -> Checkpoint:
    return await update_hot_search_checkpoint(
        checkpoint_manager,
        checkpoint,
        is_completed=True,
    )
