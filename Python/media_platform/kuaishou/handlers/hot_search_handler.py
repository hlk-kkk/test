# -*- coding: utf-8 -*-

import asyncio
from typing import TYPE_CHECKING, Dict, List

import config
import constant
from model.m_checkpoint import Checkpoint
from media_platform.hot_search_checkpoint import (
    ensure_hot_search_snapshot,
    load_or_create_hot_search_checkpoint,
    mark_hot_search_checkpoint_completed,
    update_hot_search_checkpoint,
)
from pkg.tools import utils
from repo.platform_save_data.hot_search import (
    batch_update_hot_search_content_relations,
    build_kuaishou_hot_search_item,
    update_hot_search_item,
)
from var import source_keyword_var

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import KuaiShouApiClient
    from ..processors.comment_processor import CommentProcessor
    from ..processors.video_processor import VideoProcessor


class HotSearchHandler(BaseHandler):
    """Handles Kuaishou hot search video lists."""

    def __init__(
        self,
        ks_client: "KuaiShouApiClient",
        checkpoint_manager: "CheckpointRepoManager",
        video_processor: "VideoProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            ks_client, checkpoint_manager, video_processor, comment_processor
        )

    async def handle(self) -> None:
        await self.get_hot_search_videos()

    @staticmethod
    def _count_success_crawled_videos(checkpoint: Checkpoint) -> int:
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
            platform=constant.KUAISHOU_PLATFORM_NAME,
            mode=constant.CRALER_TYPE_HOT_SEARCH,
            log_prefix="[HotSearchHandler.get_hot_search_videos]",
            defaults={"current_homefeed_cursor": ""},
        )

    async def _update_checkpoint_position(
        self, checkpoint: Checkpoint, **fields
    ) -> Checkpoint:
        return await update_hot_search_checkpoint(
            self.checkpoint_manager,
            checkpoint,
            **fields,
        )

    @staticmethod
    def _build_rank_no(page_index: int) -> int:
        return page_index + 1

    @staticmethod
    def _extract_video_id(video_item: Dict) -> str:
        return str((video_item.get("photo") or {}).get("id") or "")

    @staticmethod
    def _extract_video_title(video_item: Dict) -> str:
        photo = video_item.get("photo") or {}
        return (
            photo.get("caption")
            or photo.get("title")
            or video_item.get("caption")
            or "kuaishou_hot_search"
        )

    async def get_hot_search_videos(self) -> None:
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_videos] Begin get kuaishou hot search"
        )
        checkpoint = await self._load_or_create_checkpoint()

        pcursor = checkpoint.current_homefeed_cursor or ""
        saved_video_count = self._count_success_crawled_videos(checkpoint)
        source_keyword_var.set("kuaishou_hot_search")

        if checkpoint.current_hot_search_snapshot_items:
            full_video_list: List[Dict] = checkpoint.current_hot_search_snapshot_items
            hot_search_res = {"pcursor": "no_more"}
        else:
            hot_search_res = await self.ks_client.get_hot_search_items(pcursor=pcursor)
            full_video_list: List[Dict] = hot_search_res.get("feeds") or []
            checkpoint = await ensure_hot_search_snapshot(
                self.checkpoint_manager,
                checkpoint,
                full_video_list,
            )
            full_video_list = checkpoint.current_hot_search_snapshot_items or []
        if not full_video_list:
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        hot_search_items_by_video_id = {}
        snapshot_ts = checkpoint.current_hot_search_snapshot_ts
        for video_index, video_item in enumerate(full_video_list):
            hot_item = build_kuaishou_hot_search_item(
                video_item,
                rank_no=self._build_rank_no(video_index),
                snapshot_ts=snapshot_ts,
            )
            await update_hot_search_item(hot_item)
            video_id = self._extract_video_id(video_item)
            if video_id:
                hot_search_items_by_video_id[video_id] = hot_item

        start_video_index = checkpoint.current_homefeed_note_index or 0
        video_list = full_video_list[start_video_index:]
        if not video_list:
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        selected_video_ids: List[str] = []
        new_video_count = 0
        last_title = "kuaishou_hot_search"
        for video_item in video_list:
            video_id = self._extract_video_id(video_item)
            if not video_id:
                continue

            selected_video_ids.append(video_id)
            last_title = self._extract_video_title(video_item)
            if not self._is_success_crawled(checkpoint, video_id):
                new_video_count += 1

        if not selected_video_ids:
            checkpoint = await self._update_checkpoint_position(
                checkpoint,
                current_search_keyword="kuaishou_hot_search",
                current_homefeed_note_index=start_video_index + len(video_list),
            )
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        source_keyword_var.set("kuaishou_hot_search")
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_videos] Get hot videos, "
            f"cursor: {pcursor}, start index: {start_video_index}, "
            f"video count: {len(selected_video_ids)}"
        )
        video_ids = await self.video_processor.batch_get_video_list(
            selected_video_ids,
            checkpoint_id=checkpoint.id,
        )
        for video_id in video_ids:
            hot_item = hot_search_items_by_video_id.get(video_id)
            if not hot_item:
                continue
            await batch_update_hot_search_content_relations(
                hot_item=hot_item,
                content_ids=[video_id],
                content_type="video",
                content_table="kuaishou_video",
                source_keyword="kuaishou_hot_search",
            )
        await self.comment_processor.batch_get_video_comments(
            video_ids,
            checkpoint_id=checkpoint.id,
        )

        saved_video_count += new_video_count

        checkpoint = await self._update_checkpoint_position(
            checkpoint,
            current_search_keyword=last_title,
            current_homefeed_cursor=hot_search_res.get("pcursor") or "no_more",
            current_homefeed_note_index=len(full_video_list),
        )
        await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)

        utils.logger.info(
            f"[HotSearchHandler.get_hot_search_videos] Saved {saved_video_count} "
            "kuaishou videos from hot search"
        )
