# -*- coding: utf-8 -*-

import asyncio
from typing import TYPE_CHECKING, List

import config
import constant
from model.m_checkpoint import Checkpoint
from model.m_zhihu import ZhihuContent
from media_platform.hot_search_checkpoint import (
    ensure_hot_search_snapshot,
    get_hot_search_content_ids,
    get_hot_search_max_items_count,
    get_hot_search_per_item_content_count,
    load_or_create_hot_search_checkpoint,
    mark_hot_search_checkpoint_completed,
    save_hot_search_content_ids,
    update_hot_search_checkpoint,
)
from pkg.tools import utils
from repo.platform_save_data.hot_search import (
    batch_update_hot_search_content_relations,
    build_zhihu_hot_search_item,
    update_hot_search_item,
)
from var import source_keyword_var

from .base_handler import BaseHandler

if TYPE_CHECKING:
    from repo.checkpoint.checkpoint_store import CheckpointRepoManager

    from ..client import ZhiHuClient
    from ..processors.comment_processor import CommentProcessor
    from ..processors.content_processor import ContentProcessor


class HotSearchHandler(BaseHandler):
    """Handles Zhihu hot search content lists."""

    def __init__(
        self,
        zhihu_client: "ZhiHuClient",
        checkpoint_manager: "CheckpointRepoManager",
        content_processor: "ContentProcessor",
        comment_processor: "CommentProcessor",
    ):
        super().__init__(
            zhihu_client, checkpoint_manager, content_processor, comment_processor
        )

    async def handle(self) -> None:
        await self.get_hot_search_contents()

    @staticmethod
    def _count_success_crawled_contents(checkpoint: Checkpoint) -> int:
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
            platform=constant.ZHIHU_PLATFORM_NAME,
            mode=constant.CRALER_TYPE_HOT_SEARCH,
            log_prefix="[HotSearchHandler.get_hot_search_contents]",
            defaults={"current_search_page": 0},
        )

    async def _update_checkpoint_position(
        self, checkpoint: Checkpoint, **fields
    ) -> Checkpoint:
        return await update_hot_search_checkpoint(
            self.checkpoint_manager,
            checkpoint,
            **fields,
        )

    async def get_hot_search_contents(self) -> None:
        utils.logger.info(
            "[HotSearchHandler.get_hot_search_contents] Begin get zhihu hot search"
        )
        checkpoint = await self._load_or_create_checkpoint()

        offset = checkpoint.current_search_page or 0
        saved_content_count = self._count_success_crawled_contents(checkpoint)

        if checkpoint.current_hot_search_snapshot_items:
            hot_search_items = checkpoint.current_hot_search_snapshot_items
        else:
            hot_search_res = await self.zhihu_client.get_hot_search_items(
                offset=0,
                limit=get_hot_search_max_items_count(),
            )
            checkpoint = await ensure_hot_search_snapshot(
                self.checkpoint_manager,
                checkpoint,
                hot_search_res.get("data") or [],
            )
            hot_search_items = checkpoint.current_hot_search_snapshot_items or []
        if not hot_search_items:
            utils.logger.info("[HotSearchHandler.get_hot_search_contents] No hot search contents")
            await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
            return

        hot_search_items_by_index = {}
        snapshot_ts = checkpoint.current_hot_search_snapshot_ts
        for question_index, hot_item in enumerate(hot_search_items):
            question = hot_item.get("question") or {}
            question_id = str(question.get("id") or "")
            title = question.get("title") or ""
            if not question_id or not title:
                continue
            hot_search_item = build_zhihu_hot_search_item(
                hot_item,
                rank_no=question_index + 1,
                snapshot_ts=snapshot_ts,
            )
            await update_hot_search_item(hot_search_item)
            hot_search_items_by_index[question_index] = hot_search_item

        start_question_index = checkpoint.current_homefeed_note_index or 0
        for question_index, hot_item in enumerate(
            hot_search_items[start_question_index:], start=start_question_index
        ):
            question = hot_item.get("question") or {}
            question_id = str(question.get("id") or "")
            title = question.get("title") or ""
            if not question_id or not title:
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_search_page=offset,
                    current_homefeed_note_index=question_index + 1,
                )
                continue

            source_keyword_var.set(title)
            checkpoint.current_search_keyword = title
            checkpoint.current_search_id = question_id
            hot_search_item = hot_search_items_by_index.get(question_index)
            utils.logger.info(
                f"[HotSearchHandler.get_hot_search_contents] Get hot question answers: "
                f"{title}, question_id: {question_id}"
            )

            stored_content_ids = get_hot_search_content_ids(
                checkpoint,
                hot_search_item.hot_item_id if hot_search_item else "",
            )
            content_list: List[ZhihuContent] = []
            if stored_content_ids:
                for content_id in stored_content_ids:
                    await self.checkpoint_manager.add_note_to_checkpoint(
                        checkpoint_id=checkpoint.id,
                        note_id=content_id,
                        extra_params_info={
                            "content_type": constant.ANSWER_NAME,
                            "question_id": question_id,
                        },
                    )
                    content_item = await self.content_processor.process_content_by_id(
                        content_id,
                        constant.ANSWER_NAME,
                        checkpoint_id=checkpoint.id,
                        question_id=question_id,
                    )
                    if content_item:
                        content_list.append(content_item)
            else:
                content_list = await self.zhihu_client.get_all_answers_by_question_id(
                    question_id,
                    crawl_interval=config.CRAWLER_TIME_SLEEP,
                    max_answers=get_hot_search_per_item_content_count(),
                )
                content_list = content_list[: get_hot_search_per_item_content_count()]
                stored_content_ids = [
                    content_item.content_id
                    for content_item in content_list
                    if content_item.content_id
                ]
                if hot_search_item:
                    checkpoint = await save_hot_search_content_ids(
                        self.checkpoint_manager,
                        checkpoint,
                        hot_search_item.hot_item_id,
                        stored_content_ids,
                    )
            if not content_list:
                checkpoint = await self._update_checkpoint_position(
                    checkpoint,
                    current_search_keyword=title,
                    current_search_id=question_id,
                    current_search_page=offset,
                    current_homefeed_note_index=question_index + 1,
                )
                continue

            new_content_count = 0
            for content_id in stored_content_ids:
                if not self._is_success_crawled(checkpoint, content_id):
                    new_content_count += 1

            content_ids = await self.content_processor.batch_process_contents(
                content_list=content_list,
                checkpoint_id=checkpoint.id,
            )
            content_id_set = set(content_ids)
            content_ids_by_type = {}
            for content_item in content_list:
                if (
                    not content_item.content_id
                    or content_item.content_id not in content_id_set
                ):
                    continue
                content_type = content_item.content_type or constant.ANSWER_NAME
                content_ids_by_type.setdefault(content_type, []).append(
                    content_item.content_id
                )
            for content_type, related_content_ids in content_ids_by_type.items():
                await batch_update_hot_search_content_relations(
                    hot_item=hot_search_item,
                    content_ids=related_content_ids,
                    content_type=content_type,
                    content_table="zhihu_content",
                    source_keyword=title,
                )
            await self.comment_processor.batch_get_content_comments(
                content_list=content_list,
                checkpoint_id=checkpoint.id,
            )
            saved_content_count += new_content_count
            checkpoint = await self._update_checkpoint_position(
                checkpoint,
                current_search_keyword=title,
                current_search_id=question_id,
                current_search_page=offset,
                current_homefeed_note_index=question_index + 1,
            )
            await asyncio.sleep(config.CRAWLER_TIME_SLEEP)

        await mark_hot_search_checkpoint_completed(self.checkpoint_manager, checkpoint)
        utils.logger.info(
            f"[HotSearchHandler.get_hot_search_contents] Saved {saved_content_count} "
            "zhihu contents from hot search questions"
        )
