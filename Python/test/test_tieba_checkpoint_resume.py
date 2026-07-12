# -*- coding: utf-8 -*-

import asyncio
from types import SimpleNamespace

import config
import constant
from media_platform.tieba.handlers.creator_handler import CreatorHandler
from media_platform.tieba.handlers.search_handler import SearchHandler
from media_platform.tieba.processors.comment_processor import CommentProcessor
from model.m_baidu_tieba import TiebaComment, TiebaNote
from model.m_checkpoint import Checkpoint


class MemoryCheckpointManager:
    def __init__(self, checkpoint: Checkpoint):
        self.checkpoint = checkpoint
        self.saved = []

    async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
        if checkpoint_id and checkpoint_id != self.checkpoint.id:
            return None
        if platform and platform != self.checkpoint.platform:
            return None
        if mode and mode != self.checkpoint.mode:
            return None
        return self.checkpoint

    async def load_checkpoint_by_id(self, checkpoint_id):
        return self.checkpoint if checkpoint_id == self.checkpoint.id else None

    async def save_checkpoint(self, checkpoint):
        if checkpoint.id is None:
            checkpoint.id = self.checkpoint.id
        self.checkpoint = checkpoint
        self.saved.append(checkpoint.model_dump())
        return checkpoint

    async def update_checkpoint(self, checkpoint):
        self.checkpoint = checkpoint
        self.saved.append(checkpoint.model_dump())
        return checkpoint


class NoopNoteProcessor:
    def __init__(self):
        self.note_id_batches = []

    async def batch_get_note_list(self, note_id_list, checkpoint_id=""):
        self.note_id_batches.append(list(note_id_list))
        return [
            TiebaNote(
                note_id=note_id,
                title="title",
                note_url=f"https://tieba.baidu.com/p/{note_id}",
                tieba_name="编程吧",
                tieba_link="https://tieba.baidu.com/f?kw=%E7%BC%96%E7%A8%8B",
                total_replay_page=1,
            )
            for note_id in note_id_list
        ]


class NoopCommentProcessor:
    def __init__(self):
        self.note_batches = []

    async def batch_get_note_comments(self, note_detail_list, checkpoint_id=""):
        self.note_batches.append([note.note_id for note in note_detail_list])


def _patch_config(monkeypatch, **values):
    originals = {name: getattr(config, name) for name in values}
    for name, value in values.items():
        monkeypatch.setattr(config, name, value)
    return originals


def test_tieba_search_resume_uses_checkpoint_page(monkeypatch):
    _patch_config(
        monkeypatch,
        KEYWORDS="编程",
        CRAWLER_MAX_NOTES_COUNT=10,
        ENABLE_CHECKPOINT=True,
    )
    checkpoint = Checkpoint(
        id="tieba_search_resume",
        platform=constant.TIEBA_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_SEARCH,
        current_search_keyword="编程",
        current_search_page=2,
    )
    checkpoint_manager = MemoryCheckpointManager(checkpoint)
    note_processor = NoopNoteProcessor()
    comment_processor = NoopCommentProcessor()
    calls = []

    class Client:
        async def get_notes_by_keyword(self, keyword, page, page_size, sort, note_type):
            calls.append(page)
            return []

    handler = SearchHandler(
        Client(), checkpoint_manager, note_processor, comment_processor
    )

    asyncio.run(handler.search())

    assert calls == [2]
    assert checkpoint_manager.checkpoint.current_search_page == 2


def test_tieba_forum_resume_keeps_current_pc_offset(monkeypatch):
    _patch_config(
        monkeypatch,
        TIEBA_NAME_LIST=["编程"],
        CRAWLER_MAX_NOTES_COUNT=60,
        ENABLE_CHECKPOINT=True,
    )
    checkpoint = Checkpoint(
        id="tieba_forum_resume",
        platform=constant.TIEBA_PLATFORM_NAME,
        mode="tieba_search",
        current_search_keyword="编程",
        current_search_page=30,
    )
    checkpoint_manager = MemoryCheckpointManager(checkpoint)
    note_processor = NoopNoteProcessor()
    comment_processor = NoopCommentProcessor()
    calls = []

    class Client:
        async def get_notes_by_tieba_name(self, tieba_name, page_num):
            calls.append(page_num)
            if page_num == 30:
                return [
                    TiebaNote(
                        note_id="1001",
                        title="title",
                        note_url="https://tieba.baidu.com/p/1001",
                        tieba_name="编程吧",
                        tieba_link="https://tieba.baidu.com/f?kw=%E7%BC%96%E7%A8%8B",
                    )
                ]
            return []

    handler = SearchHandler(
        Client(), checkpoint_manager, note_processor, comment_processor
    )

    asyncio.run(handler.get_specified_tieba_notes())

    assert calls == [30, 60]
    assert checkpoint_manager.checkpoint.current_search_page == 60


def test_tieba_creator_resume_uses_checkpoint_page(monkeypatch):
    _patch_config(monkeypatch, CRAWLER_MAX_NOTES_COUNT=20)
    checkpoint = Checkpoint(
        id="tieba_creator_resume",
        platform=constant.TIEBA_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_CREATOR,
        current_creator_id="tb.1.creator",
        current_creator_page="2",
    )
    checkpoint_manager = MemoryCheckpointManager(checkpoint)
    note_processor = NoopNoteProcessor()
    comment_processor = NoopCommentProcessor()
    calls = []

    class Extractor:
        @staticmethod
        def extract_creator_thread_id_list_from_api(_):
            return []

    class Client:
        page_extractor = Extractor()

        @staticmethod
        def _extract_creator_portrait(creator_url):
            return creator_url

        async def get_notes_by_creator_portrait(self, portrait, page_number, page_size=20):
            calls.append(page_number)
            return {"error_code": 0, "data": {"has_more": 0, "list": []}}

    handler = CreatorHandler(
        Client(), checkpoint_manager, note_processor, comment_processor
    )

    asyncio.run(
        handler.get_all_notes_by_creator(
            creator_url="tb.1.creator", checkpoint_id="tieba_creator_resume"
        )
    )

    assert calls == [2]
    assert checkpoint_manager.checkpoint.current_creator_page == "2"


def test_tieba_comment_resume_uses_comment_cursor(monkeypatch):
    _patch_config(
        monkeypatch,
        ENABLE_GET_SUB_COMMENTS=False,
        PER_NOTE_MAX_COMMENTS_COUNT=0,
        CRAWLER_TIME_SLEEP=0,
    )
    pages = []
    cursor_updates = []

    class CheckpointManager:
        async def get_note_comment_cursor(self, checkpoint_id, note_id):
            return "2"

        async def update_note_comment_cursor(
            self,
            checkpoint_id,
            note_id,
            comment_cursor,
            is_success_crawled_comments=False,
        ):
            cursor_updates.append((comment_cursor, is_success_crawled_comments))

    class Extractor:
        @staticmethod
        def extract_tieba_note_parent_comments_from_api(response_data, note_detail):
            if response_data["page"] == 2:
                return [
                    TiebaComment(
                        comment_id="2001",
                        content="comment",
                        note_id=note_detail.note_id,
                        note_url=note_detail.note_url,
                        tieba_id="1",
                        tieba_name=note_detail.tieba_name,
                        tieba_link=note_detail.tieba_link,
                    )
                ]
            return []

    class Client:
        page_extractor = Extractor()

        async def get_note_comments(self, note_id, page=1):
            pages.append(page)
            return {"page": page}

    processor = CommentProcessor(
        Client(), CheckpointManager(), asyncio.Semaphore(1)
    )
    note = TiebaNote(
        note_id="1001",
        title="title",
        note_url="https://tieba.baidu.com/p/1001",
        tieba_name="编程吧",
        tieba_link="https://tieba.baidu.com/f?kw=%E7%BC%96%E7%A8%8B",
        total_replay_page=2,
    )

    asyncio.run(
        processor.get_note_all_comments(
            note_detail=note,
            callback=None,
            checkpoint_id="tieba_comment_resume",
        )
    )

    assert pages == [2]
    assert cursor_updates[-1] == ("3", True)
