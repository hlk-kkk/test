# -*- coding: utf-8 -*-

from typing import TYPE_CHECKING

import asyncio
import config
import constant
from model.m_checkpoint import Checkpoint
from media_platform.hot_search_checkpoint import (
    ensure_hot_search_snapshot,
    get_hot_search_content_ids,
    get_hot_search_per_item_content_count,
    load_or_create_hot_search_checkpoint,
    mark_hot_search_checkpoint_completed,
    save_hot_search_content_ids,
    update_hot_search_checkpoint,
)
from pkg.tools import utils
from repo.platform_save_data.hot_search import (
    batch_update_hot_search_content_relations,
    build_weibo_hot_search_item,
    update_hot_search_item,
)
from var import source_keyword_var

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import WeiboClient
    from ..processors.comment_processor import CommentProcessor
    from ..processors.note_processor import NoteProcessor


class HotSearchHandler(BaseHandler):
    """Placeholder handler for Weibo hot search note lists."""

    def __init__(
        self,
        wb_client: "WeiboClient",
        checkpoint_manager: "CheckpointRepoManager",
        note_processor: "NoteProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            wb_client, checkpoint_manager, note_processor, comment_processor
        )

    async def handle(self) -> None:
        await self.get_hot_search_notes()

    @staticmethod
    def _count_success_crawled_notes(checkpoint: Checkpoint) -> int:
        return sum(
            1
            for checkpoint_note in checkpoint.crawled_note_list or []
            if checkpoint_note.is_success_crawled
        )

    @staticmethod
    def _is_success_crawled(checkpoint: Checkpoint, note_id: str) -> bool:
        return any(
            checkpoint_note.note_id == note_id and checkpoint_note.is_success_crawled
            for checkpoint_note in checkpoint.crawled_note_list or []
        )

    async def _load_or_create_checkpoint(self) -> Checkpoint:
        return await load_or_create_hot_search_checkpoint(
            self.checkpoint_manager,
            platform=constant.WEIBO_PLATFORM_NAME,
            mode=constant.CRALER_TYPE_HOT_SEARCH,
            log_prefix="[HotSearchHandler.get_hot_search_notes]",
        )

    async def _update_checkpoint_position(
        self, checkpoint: Checkpoint, **fields
    ) -> Checkpoint:
        return await update_hot_search_checkpoint(
            self.checkpoint_manager,
            checkpoint,
            **fields,
        )

    async def get_hot_search_notes(self) -> None:
        utils.logger.info("[HotSearchHandler.get_hot_search_notes] Begin get weibo hot search")
        checkpoint = await self._load_or_create_checkpoint()

        if checkpoint.current_hot_search_snapshot_items:
            hot_search_items = checkpoint.current_hot_search_snapshot_items
        else:
            hot_search_items = await self.wb_client.get_hot_search_items()
            checkpoint = await ensure_hot_search_snapshot(
                self.checkpoint_manager,
                checkpoint,
                hot_search_items,
            )
            hot_search_items = checkpoint.current_hot_search_snapshot_items or []
        if not hot_search_items:
            utils.logger.info("[HotSearchHandler.get_hot_search_notes] No hot search topics")
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        hot_search_items_by_index = {}
        snapshot_ts = checkpoint.current_hot_search_snapshot_ts
        for hot_index, hot_item in enumerate(hot_search_items):
            if not hot_item.get("realpos") or hot_item.get("is_ad"):
                continue
            hot_word = hot_item.get("word", "")
            if not hot_word:
                continue
            hot_search_item = build_weibo_hot_search_item(
                hot_item,
                snapshot_ts=snapshot_ts,
            )
            await update_hot_search_item(hot_search_item)
            hot_search_items_by_index[hot_index] = hot_search_item

        saved_note_count = self._count_success_crawled_notes(checkpoint)
        start_index = checkpoint.current_homefeed_note_index or 0
        for hot_index, hot_item in enumerate(
            hot_search_items[start_index:], start=start_index
        ):
            if not hot_item.get("realpos") or hot_item.get("is_ad"):
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_homefeed_note_index=hot_index + 1,
                )
                continue

            hot_word = hot_item.get("word", "")
            if not hot_word:
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_homefeed_note_index=hot_index + 1,
                )
                continue

            hot_search_item = hot_search_items_by_index.get(hot_index)
            source_keyword_var.set(hot_word)
            checkpoint.current_search_keyword = hot_word
            utils.logger.info(
                f"[HotSearchHandler.get_hot_search_notes] Search hot topic: {hot_word}"
            )
            note_list, _ = await self.wb_client.get_note_by_keyword(keyword=hot_word)
            stored_note_ids = get_hot_search_content_ids(
                checkpoint,
                hot_search_item.hot_item_id if hot_search_item else "",
            )
            if stored_note_ids:
                note_map = {note.note_id: note for note in note_list or [] if note.note_id}
                selected_note_list = [
                    note_map[note_id]
                    for note_id in stored_note_ids
                    if note_id in note_map
                ]
            else:
                selected_note_list = (note_list or [])[
                    : get_hot_search_per_item_content_count()
                ]
                stored_note_ids = [
                    note_item.note_id for note_item in selected_note_list if note_item.note_id
                ]
                if hot_search_item:
                    checkpoint = await save_hot_search_content_ids(
                        self.checkpoint_manager,
                        checkpoint,
                        hot_search_item.hot_item_id,
                        stored_note_ids,
                    )
            if not stored_note_ids:
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_search_keyword=hot_word,
                    current_homefeed_note_index=hot_index + 1,
                )
                continue

            new_note_count = 0
            for note_id in stored_note_ids:
                if not self._is_success_crawled(checkpoint, note_id):
                    new_note_count += 1

            note_id_list = await self.note_processor.batch_get_note_list(
                note_list=selected_note_list,
                checkpoint_id=checkpoint.id,
            )
            missing_note_ids = [
                note_id for note_id in stored_note_ids if note_id not in set(note_id_list)
            ]
            if missing_note_ids and hasattr(self.note_processor, "batch_get_specified_notes"):
                note_id_list.extend(
                    await self.note_processor.batch_get_specified_notes(
                        missing_note_ids,
                        checkpoint_id=checkpoint.id,
                    )
                )
            if note_id_list:
                await batch_update_hot_search_content_relations(
                    hot_item=hot_search_item,
                    content_ids=note_id_list,
                    content_type="note",
                    content_table="weibo_note",
                    source_keyword=hot_word,
                )
            await self.comment_processor.batch_get_note_comments(
                note_id_list,
                checkpoint_id=checkpoint.id,
            )
            saved_note_count += new_note_count
            checkpoint = await self._update_checkpoint_position(
                checkpoint,
                current_search_keyword=hot_word,
                current_homefeed_note_index=hot_index + 1,
            )
            await asyncio.sleep(config.CRAWLER_TIME_SLEEP)

        await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
        utils.logger.info(
            f"[HotSearchHandler.get_hot_search_notes] Saved {saved_note_count} weibo notes from hot search topics"
        )
