# -*- coding: utf-8 -*-

import asyncio
from typing import TYPE_CHECKING, List, Dict

import config
import constant
from model.m_bilibili import VideoIdInfo
from model.m_checkpoint import Checkpoint
from media_platform.hot_search_checkpoint import (
    ensure_hot_search_snapshot,
    get_hot_search_max_items_count,
    load_or_create_hot_search_checkpoint,
    mark_hot_search_checkpoint_completed,
    update_hot_search_checkpoint,
)
from pkg.tools import utils
from repo.platform_save_data.hot_search import (
    batch_update_hot_search_content_relations,
    build_bilibili_hot_search_item,
    update_hot_search_item,
)
from var import source_keyword_var

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import BilibiliClient
    from ..processors.comment_processor import CommentProcessor
    from ..processors.video_processor import VideoProcessor


class HotSearchHandler(BaseHandler):
    """Handles Bilibili hot search video lists."""

    def __init__(
        self,
        bili_client: "BilibiliClient",
        checkpoint_manager: "CheckpointRepoManager",
        video_processor: "VideoProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            bili_client, checkpoint_manager, video_processor, comment_processor
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
            platform=constant.BILIBILI_PLATFORM_NAME,
            mode=constant.CRALER_TYPE_HOT_SEARCH,
            log_prefix="[HotSearchHandler.get_hot_search_videos]",
            defaults={"current_search_page": 1},
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
    def _build_rank_no(page: int, page_size: int, page_index: int) -> int:
        return max(page - 1, 0) * page_size + page_index + 1

    @staticmethod
    def _build_relation_content_id(video_info: VideoIdInfo) -> str:
        return str(
            video_info.aid or getattr(video_info, "video_id", "") or video_info.bvid
        )

    async def get_hot_search_videos(self) -> None:
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_videos] Begin get bilibili hot search"
        )
        checkpoint = await self._load_or_create_checkpoint()

        page = checkpoint.current_search_page or 1
        page_size = get_hot_search_max_items_count()
        saved_video_count = self._count_success_crawled_videos(checkpoint)
        source_keyword_var.set("bilibili_hot_search")
        checkpoint.current_search_keyword = "bilibili_hot_search"

        if checkpoint.current_hot_search_snapshot_items:
            full_video_list: List[Dict] = checkpoint.current_hot_search_snapshot_items
            hot_search_res = {"no_more": True}
        else:
            hot_search_res = await self.bili_client.get_hot_search_items(
                page=page,
                page_size=page_size,
            )
            full_video_list: List[Dict] = hot_search_res.get("list") or []
            checkpoint = await ensure_hot_search_snapshot(
                self.checkpoint_manager,
                checkpoint,
                full_video_list,
            )
            full_video_list = checkpoint.current_hot_search_snapshot_items or []
        if not full_video_list:
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        hot_search_items_by_video_key = {}
        snapshot_ts = checkpoint.current_hot_search_snapshot_ts
        for page_index, video_item in enumerate(full_video_list):
            hot_item = build_bilibili_hot_search_item(
                video_item,
                rank_no=self._build_rank_no(page, page_size, page_index),
                snapshot_ts=snapshot_ts,
            )
            await update_hot_search_item(hot_item)
            bvid = str(video_item.get("bvid") or "")
            aid = str(video_item.get("aid", "") or video_item.get("id", ""))
            if bvid:
                hot_search_items_by_video_key[f"bvid:{bvid}"] = hot_item
            if aid:
                hot_search_items_by_video_key[f"aid:{aid}"] = hot_item

        page_start_index = checkpoint.current_homefeed_note_index or 0
        video_list = full_video_list[page_start_index:]
        if not video_list:
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        selected_video_list: List[Dict] = []
        new_video_count = 0
        for video_item in video_list:
            bvid = video_item.get("bvid", "")
            selected_video_list.append(video_item)
            if not self._is_success_crawled(checkpoint, bvid):
                new_video_count += 1

        video_list = selected_video_list
        video_infos: List[VideoIdInfo] = (
            await self.video_processor.batch_get_video_list(
                video_list=video_list,
                checkpoint_id=checkpoint.id,
            )
        )
        for video_info in video_infos:
            hot_item = hot_search_items_by_video_key.get(
                f"bvid:{video_info.bvid}"
            ) or hot_search_items_by_video_key.get(f"aid:{video_info.aid}")
            content_id = self._build_relation_content_id(video_info)
            if not hot_item or not content_id:
                continue
            await batch_update_hot_search_content_relations(
                hot_item=hot_item,
                content_ids=[content_id],
                content_type="video",
                content_table="bilibili_video",
                source_keyword="bilibili_hot_search",
            )
        await self.comment_processor.batch_get_video_comments(
            video_infos,
            checkpoint_id=checkpoint.id,
        )

        saved_video_count += new_video_count

        checkpoint = await self._update_checkpoint_position(
            checkpoint,
            current_search_keyword="bilibili_hot_search",
            current_search_page=page,
            current_homefeed_note_index=len(full_video_list),
        )
        await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)

        utils.logger.info(
            f"[HotSearchHandler.get_hot_search_videos] Saved {saved_video_count} "
            "bilibili videos from hot search"
        )
