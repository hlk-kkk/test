# -*- coding: utf-8 -*-

import asyncio
from typing import TYPE_CHECKING

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
    build_douyin_hot_search_item,
    update_hot_search_item,
)
from var import request_keyword_var, source_keyword_var

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import DouYinApiClient
    from ..processors.aweme_processor import AwemeProcessor
    from ..processors.comment_processor import CommentProcessor


class HotSearchHandler(BaseHandler):
    """Handles Douyin hot search aweme lists."""

    def __init__(
        self,
        dy_client: "DouYinApiClient",
        checkpoint_manager: "CheckpointRepoManager",
        aweme_processor: "AwemeProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            dy_client, checkpoint_manager, aweme_processor, comment_processor
        )

    async def handle(self) -> None:
        await self.get_hot_search_awemes()

    @staticmethod
    def _count_success_crawled_awemes(checkpoint: Checkpoint) -> int:
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
            platform=constant.DOUYIN_PLATFORM_NAME,
            mode=constant.CRALER_TYPE_HOT_SEARCH,
            log_prefix="[HotSearchHandler.get_hot_search_awemes]",
        )

    async def _update_checkpoint_position(
        self, checkpoint: Checkpoint, **fields
    ) -> Checkpoint:
        return await update_hot_search_checkpoint(
            self.checkpoint_manager,
            checkpoint,
            **fields,
        )

    async def get_hot_search_awemes(self) -> None:
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_awemes] Begin get douyin hot search"
        )
        checkpoint = await self._load_or_create_checkpoint()

        if checkpoint.current_hot_search_snapshot_items:
            hot_search_items = checkpoint.current_hot_search_snapshot_items
        else:
            hot_search_items = await self.dy_client.get_hot_search_items()
            checkpoint = await ensure_hot_search_snapshot(
                self.checkpoint_manager,
                checkpoint,
                hot_search_items,
            )
            hot_search_items = checkpoint.current_hot_search_snapshot_items or []
        if not hot_search_items:
            utils.logger.info(
                "[HotSearchHandler.get_hot_search_awemes] No hot search awemes"
            )
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        hot_search_items_by_index = {}
        snapshot_ts = checkpoint.current_hot_search_snapshot_ts
        for hot_index, hot_item in enumerate(hot_search_items):
            hot_word = hot_item.get("word", "")
            if not hot_word or not hot_item.get("position"):
                continue
            hot_search_item = build_douyin_hot_search_item(
                hot_item,
                snapshot_ts=snapshot_ts,
            )
            await update_hot_search_item(hot_search_item)
            hot_search_items_by_index[hot_index] = hot_search_item

        saved_aweme_count = self._count_success_crawled_awemes(checkpoint)
        start_index = checkpoint.current_homefeed_note_index or 0
        for hot_index, hot_item in enumerate(
            hot_search_items[start_index:], start=start_index
        ):
            hot_word = hot_item.get("word", "")
            if not hot_word or not hot_item.get("position"):
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_homefeed_note_index=hot_index + 1,
                )
                continue

            hot_search_item = hot_search_items_by_index.get(hot_index)
            source_keyword_var.set(hot_word)
            request_keyword_var.set(hot_word)
            checkpoint.current_search_keyword = hot_word
            utils.logger.info(
                f"[HotSearchHandler.get_hot_search_awemes] Get hot topic awemes: "
                f"{hot_word}, rank: {hot_item.get('position')}"
            )

            aweme_ids = get_hot_search_content_ids(
                checkpoint,
                hot_search_item.hot_item_id if hot_search_item else "",
            )
            if not aweme_ids:
                aweme_ids = await self.dy_client.get_hot_search_aweme_ids(
                    hot_word,
                    count=get_hot_search_per_item_content_count(),
                )
                aweme_ids = aweme_ids[: get_hot_search_per_item_content_count()]
                if hot_search_item:
                    checkpoint = await save_hot_search_content_ids(
                        self.checkpoint_manager,
                        checkpoint,
                        hot_search_item.hot_item_id,
                        aweme_ids,
                    )
            if not aweme_ids:
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_search_keyword=hot_word,
                    current_homefeed_note_index=hot_index + 1,
                )
                continue

            new_aweme_count = 0
            for aweme_id in aweme_ids:
                if not self._is_success_crawled(checkpoint, aweme_id):
                    new_aweme_count += 1

            aweme_id_list = await self.aweme_processor.batch_get_aweme_list_from_ids(
                aweme_ids,
                checkpoint.id,
            )
            if aweme_id_list:
                await batch_update_hot_search_content_relations(
                    hot_item=hot_search_item,
                    content_ids=aweme_id_list,
                    content_type="aweme",
                    content_table="douyin_aweme",
                    source_keyword=hot_word,
                )
            await self.comment_processor.batch_get_aweme_comments(
                aweme_id_list,
                checkpoint.id,
            )
            saved_aweme_count += new_aweme_count
            checkpoint = await self._update_checkpoint_position(
                checkpoint,
                current_search_keyword=hot_word,
                current_homefeed_note_index=hot_index + 1,
            )
            await asyncio.sleep(config.CRAWLER_TIME_SLEEP)

        await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
        utils.logger.info(
            f"[HotSearchHandler.get_hot_search_awemes] Saved {saved_aweme_count} "
            "douyin awemes from hot search topics"
        )
