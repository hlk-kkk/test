# -*- coding: utf-8 -*-

import asyncio
import sys

import pytest

import cmd_arg
import config
import constant
import media_platform.bilibili.handlers.hot_search_handler as bilibili_hot_search_handler
import media_platform.douyin.handlers.hot_search_handler as douyin_hot_search_handler
import media_platform.kuaishou.handlers.hot_search_handler as kuaishou_hot_search_handler
import media_platform.weibo.handlers.hot_search_handler as weibo_hot_search_handler
import media_platform.zhihu.handlers.hot_search_handler as zhihu_hot_search_handler
from repo.platform_save_data.hot_search import (
    build_douyin_hot_search_item,
    build_hot_search_content_relation,
    build_weibo_hot_search_item,
)
from model.m_bilibili import VideoIdInfo
from model.m_checkpoint import Checkpoint, CheckpointNote
from model.m_zhihu import ZhihuContent
from model.m_weibo import WeiboNote
from pkg.tools import utils
from media_platform.bilibili import BilibiliCrawler
from media_platform.bilibili.handlers.hot_search_handler import (
    HotSearchHandler as BilibiliHotSearchHandler,
)
from media_platform.douyin import DouYinCrawler
from media_platform.douyin.handlers.hot_search_handler import (
    HotSearchHandler as DouyinHotSearchHandler,
)
from media_platform.kuaishou import KuaiShouCrawler
from media_platform.kuaishou.handlers.hot_search_handler import (
    HotSearchHandler as KuaishouHotSearchHandler,
)
from media_platform.tieba import TieBaCrawler
from media_platform.weibo import WeiboCrawler
from media_platform.weibo.extractor import WeiboExtractor
from media_platform.weibo.handlers.hot_search_handler import HotSearchHandler as WeiboHotSearchHandler
from media_platform.xhs import XiaoHongShuCrawler
from media_platform.zhihu import ZhihuCrawler
from media_platform.zhihu.handlers.hot_search_handler import (
    HotSearchHandler as ZhihuHotSearchHandler,
)
from var import request_keyword_var, source_keyword_var


class StubHotSearchHandler:
    def __init__(self):
        self.called = False

    async def handle(self):
        self.called = True


class FakeCheckpointManager:
    def __init__(self, checkpoint=None, checkpoint_id="hot_search_test"):
        self.checkpoint = checkpoint
        self.checkpoint_id = checkpoint_id
        self.load_calls = []
        self.updated_checkpoints = []

    async def load_checkpoint(
        self,
        platform=None,
        mode=None,
        checkpoint_id=None,
    ):
        self.load_calls.append((platform, mode, checkpoint_id))
        return self.checkpoint

    async def save_checkpoint(self, checkpoint):
        if checkpoint.id is None:
            checkpoint.id = self.checkpoint_id
        self.checkpoint = checkpoint
        return checkpoint

    async def load_checkpoint_by_id(self, checkpoint_id):
        if self.checkpoint and self.checkpoint.id == checkpoint_id:
            return self.checkpoint
        return None

    async def update_checkpoint(self, checkpoint):
        self.checkpoint = checkpoint
        self.updated_checkpoints.append(checkpoint.model_copy(deep=True))
        return checkpoint


def install_hot_search_store_spy(monkeypatch, handler_module):
    calls = {"items": [], "relations": []}

    async def fake_update_hot_search_item(hot_item):
        calls["items"].append(hot_item)
        return hot_item.hot_item_id

    async def fake_batch_update_hot_search_content_relations(
        hot_item,
        content_ids,
        content_type,
        content_table,
        source_keyword="",
    ):
        calls["relations"].append(
            {
                "hot_item": hot_item,
                "content_ids": list(content_ids),
                "content_type": content_type,
                "content_table": content_table,
                "source_keyword": source_keyword,
            }
        )

    monkeypatch.setattr(
        handler_module,
        "update_hot_search_item",
        fake_update_hot_search_item,
    )
    monkeypatch.setattr(
        handler_module,
        "batch_update_hot_search_content_relations",
        fake_batch_update_hot_search_content_relations,
    )
    return calls


def test_parse_cmd_accepts_hot_search(monkeypatch):
    original_platform = config.PLATFORM
    original_crawler_type = config.CRAWLER_TYPE
    original_argv = sys.argv[:]
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "main.py",
            "--platform",
            constant.XHS_PLATFORM_NAME,
            "--type",
            constant.CRALER_TYPE_HOT_SEARCH,
        ],
    )

    try:
        cmd_arg.parse_cmd()
        assert config.PLATFORM == constant.XHS_PLATFORM_NAME
        assert config.CRAWLER_TYPE == constant.CRALER_TYPE_HOT_SEARCH
    finally:
        config.PLATFORM = original_platform
        config.CRAWLER_TYPE = original_crawler_type
        sys.argv = original_argv


@pytest.mark.parametrize(
    "crawler_class",
    [
        XiaoHongShuCrawler,
        DouYinCrawler,
        KuaiShouCrawler,
        BilibiliCrawler,
        WeiboCrawler,
        TieBaCrawler,
        ZhihuCrawler,
    ],
)
def test_hot_search_dispatches_to_platform_handler(crawler_class):
    original_crawler_type = config.CRAWLER_TYPE
    crawler = crawler_class()
    stub_handler = StubHotSearchHandler()
    crawler.hot_search_handler = stub_handler

    try:
        config.CRAWLER_TYPE = constant.CRALER_TYPE_HOT_SEARCH
        asyncio.run(crawler.start())
        assert stub_handler.called is True
    finally:
        config.CRAWLER_TYPE = original_crawler_type


def test_weibo_hot_search_extractor_maps_topics_to_notes():
    hot_search_items = [
        {
            "word": "测试热搜",
            "word_scheme": "#测试热搜#",
            "num": 12345,
            "realpos": 1,
            "label_name": "热",
        }
    ]

    note_list = WeiboExtractor().extract_hot_search_notes(hot_search_items)

    assert len(note_list) == 1
    assert note_list[0].note_id.startswith("weibo_hot_search_")
    assert note_list[0].content == "微博热搜：测试热搜 | 排名：1 | 热度：12345 | 标签：热"
    assert note_list[0].liked_count == "12345"
    assert note_list[0].source_keyword == "weibo_hot_search"
    assert "s.weibo.com/weibo" in note_list[0].note_url


def test_weibo_hot_search_handler_saves_note_list(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.keywords = []

        async def get_hot_search_items(self):
            return [
                {"word": "热搜词1", "realpos": 1},
                {"word": "广告词", "is_ad": 1},
                {"word": "无排名词"},
                {"word": "热搜词2", "realpos": 2},
            ]

        async def get_note_by_keyword(self, keyword):
            self.keywords.append(keyword)
            return [
                WeiboNote(note_id=f"{keyword}_note_1", content=f"{keyword} 微博1"),
                WeiboNote(note_id=f"{keyword}_note_2", content=f"{keyword} 微博2"),
            ], {}

    class FakeCheckpointManager:
        async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
            return None

        async def save_checkpoint(self, checkpoint):
            checkpoint.id = "wb_hot_search_test"
            return checkpoint

        async def update_checkpoint(self, checkpoint):
            return checkpoint

    class FakeNoteProcessor:
        def __init__(self):
            self.note_list = []
            self.checkpoint_id = ""

        async def batch_get_note_list(self, note_list, checkpoint_id=""):
            self.note_list.extend(note_list)
            self.checkpoint_id = checkpoint_id
            return [note.note_id for note in note_list]

    class FakeCommentProcessor:
        def __init__(self):
            self.note_id_list = []
            self.checkpoint_id = ""

        async def batch_get_note_comments(self, note_id_list, checkpoint_id=""):
            self.note_id_list.extend(note_id_list)
            self.checkpoint_id = checkpoint_id

    client = FakeClient()
    note_processor = FakeNoteProcessor()
    comment_processor = FakeCommentProcessor()
    hot_search_calls = install_hot_search_store_spy(
        monkeypatch, weibo_hot_search_handler
    )
    handler = WeiboHotSearchHandler(
        client,
        FakeCheckpointManager(),
        note_processor,
        comment_processor,
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 1
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())
        assert client.keywords == ["热搜词1", "热搜词2"]
        assert [note.note_id for note in note_processor.note_list] == [
            "热搜词1_note_1",
            "热搜词2_note_1",
        ]
        assert note_processor.checkpoint_id == "wb_hot_search_test"
        assert comment_processor.note_id_list == ["热搜词1_note_1", "热搜词2_note_1"]
        assert comment_processor.checkpoint_id == "wb_hot_search_test"
        assert [item.title for item in hot_search_calls["items"]] == [
            "热搜词1",
            "热搜词2",
        ]
        assert hot_search_calls["items"][0].rank_no == 1
        assert hot_search_calls["relations"] == [
            {
                "hot_item": hot_search_calls["items"][0],
                "content_ids": ["热搜词1_note_1"],
                "content_type": "note",
                "content_table": "weibo_note",
                "source_keyword": "热搜词1",
            },
            {
                "hot_item": hot_search_calls["items"][1],
                "content_ids": ["热搜词2_note_1"],
                "content_type": "note",
                "content_table": "weibo_note",
                "source_keyword": "热搜词2",
            },
        ]
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_hot_search_starts_fresh_after_completed_latest_checkpoint():
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        async def get_hot_search_items(self):
            return []

    class CompletedLatestCheckpointManager:
        def __init__(self):
            self.saved_checkpoint = None
            self.load_calls = []
            self.completed_checkpoint = Checkpoint(
                id="done_hot_checkpoint",
                platform=constant.WEIBO_PLATFORM_NAME,
                mode=constant.CRALER_TYPE_HOT_SEARCH,
                current_homefeed_note_index=1,
                current_hot_search_snapshot_ts=utils.get_current_timestamp(),
                current_hot_search_snapshot_items=[{"word": "旧热搜", "realpos": 1}],
                is_completed=True,
            )

        async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
            self.load_calls.append((platform, mode, checkpoint_id))
            return self.completed_checkpoint

        async def save_checkpoint(self, checkpoint):
            checkpoint.id = "fresh_hot_checkpoint"
            self.saved_checkpoint = checkpoint
            return checkpoint

        async def update_checkpoint(self, checkpoint):
            self.saved_checkpoint = checkpoint
            return checkpoint

    class FakeNoteProcessor:
        pass

    class FakeCommentProcessor:
        pass

    checkpoint_manager = CompletedLatestCheckpointManager()
    handler = WeiboHotSearchHandler(
        FakeClient(),
        checkpoint_manager,
        FakeNoteProcessor(),
        FakeCommentProcessor(),
    )

    try:
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = ""
        asyncio.run(handler.handle())
        assert checkpoint_manager.load_calls == [
            (constant.WEIBO_PLATFORM_NAME, constant.CRALER_TYPE_HOT_SEARCH, None)
        ]
        assert checkpoint_manager.saved_checkpoint.id == "fresh_hot_checkpoint"
        assert checkpoint_manager.saved_checkpoint.current_homefeed_note_index == 0
    finally:
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_hot_search_auto_resumes_unfinished_snapshot_without_checkpoint_id():
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.keywords = []

        async def get_hot_search_items(self):
            raise AssertionError("resume should use checkpoint snapshot")

        async def get_note_by_keyword(self, keyword):
            self.keywords.append(keyword)
            return [WeiboNote(note_id=f"{keyword}_note", content=keyword)], {}

    class FakeNoteProcessor:
        async def batch_get_note_list(self, note_list, checkpoint_id=""):
            return [note.note_id for note in note_list]

    class FakeCommentProcessor:
        async def batch_get_note_comments(self, note_id_list, checkpoint_id=""):
            pass

    checkpoint = Checkpoint(
        id="auto_resume_wb_hot",
        platform=constant.WEIBO_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_homefeed_note_index=1,
        current_hot_search_snapshot_ts=utils.get_current_timestamp(),
        current_hot_search_snapshot_items=[
            {"word": "已完成热搜", "realpos": 1},
            {"word": "自动续爬", "realpos": 2},
        ],
        is_completed=False,
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    handler = WeiboHotSearchHandler(
        client,
        checkpoint_manager,
        FakeNoteProcessor(),
        FakeCommentProcessor(),
    )

    try:
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = ""
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())

        assert checkpoint_manager.load_calls[0] == (
            constant.WEIBO_PLATFORM_NAME,
            constant.CRALER_TYPE_HOT_SEARCH,
            None,
        )
        assert client.keywords == ["自动续爬"]
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 2
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_bilibili_hot_search_handler_uses_popular_videos(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.pages = []

        async def get_hot_search_items(self, page=1, page_size=20):
            self.pages.append((page, page_size))
            return {
                "list": [
                    {"aid": 1001, "bvid": "BVhot1", "title": "B站热门1"},
                    {"aid": 1002, "bvid": "BVhot2", "title": "B站热门2"},
                ],
                "no_more": False,
            }

    class FakeCheckpointManager:
        async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
            return None

        async def save_checkpoint(self, checkpoint):
            checkpoint.id = "bili_hot_search_test"
            return checkpoint

        async def update_checkpoint(self, checkpoint):
            return checkpoint

    class FakeVideoProcessor:
        def __init__(self):
            self.video_list = []
            self.checkpoint_id = ""
            self.source_keywords = []

        async def batch_get_video_list(self, video_list, checkpoint_id=""):
            self.video_list.extend(video_list)
            self.checkpoint_id = checkpoint_id
            self.source_keywords.append(source_keyword_var.get())
            return [
                VideoIdInfo(aid=str(video["aid"]), bvid=video["bvid"])
                for video in video_list
            ]

    class FakeCommentProcessor:
        def __init__(self):
            self.video_infos = []
            self.checkpoint_id = ""

        async def batch_get_video_comments(self, video_infos, checkpoint_id=""):
            self.video_infos.extend(video_infos)
            self.checkpoint_id = checkpoint_id

    client = FakeClient()
    video_processor = FakeVideoProcessor()
    comment_processor = FakeCommentProcessor()
    hot_search_calls = install_hot_search_store_spy(
        monkeypatch, bilibili_hot_search_handler
    )
    handler = BilibiliHotSearchHandler(
        client,
        FakeCheckpointManager(),
        video_processor,
        comment_processor,
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())
        assert client.pages == [(1, 50)]
        assert video_processor.video_list == [
            {"aid": 1001, "bvid": "BVhot1", "title": "B站热门1"},
            {"aid": 1002, "bvid": "BVhot2", "title": "B站热门2"},
        ]
        assert video_processor.checkpoint_id == "bili_hot_search_test"
        assert video_processor.source_keywords == ["bilibili_hot_search"]
        assert [video.bvid for video in comment_processor.video_infos] == [
            "BVhot1",
            "BVhot2",
        ]
        assert comment_processor.checkpoint_id == "bili_hot_search_test"
        assert [item.title for item in hot_search_calls["items"]] == [
            "B站热门1",
            "B站热门2",
        ]
        assert hot_search_calls["items"][0].rank_no == 1
        assert hot_search_calls["relations"][0]["content_ids"] == ["1001"]
        assert hot_search_calls["relations"][0]["content_type"] == "video"
        assert hot_search_calls["relations"][0]["content_table"] == "bilibili_video"
        assert hot_search_calls["relations"][1]["content_ids"] == ["1002"]
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_kuaishou_hot_search_handler_uses_hot_feed_videos(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.cursors = []

        async def get_hot_search_items(self, pcursor=""):
            self.cursors.append(pcursor)
            return {
                "feeds": [
                    {"photo": {"id": "ks_hot_1", "caption": "快手热点1"}},
                    {"photo": {"id": "ks_hot_2", "caption": "快手热点2"}},
                ],
                "pcursor": "1",
            }

    class FakeVideoProcessor:
        def __init__(self):
            self.video_ids = []
            self.checkpoint_id = ""
            self.source_keywords = []

        async def batch_get_video_list(self, video_id_list, checkpoint_id=""):
            self.video_ids.extend(video_id_list)
            self.checkpoint_id = checkpoint_id
            self.source_keywords.append(source_keyword_var.get())
            return video_id_list

    class FakeCommentProcessor:
        def __init__(self):
            self.video_ids = []
            self.checkpoint_id = ""

        async def batch_get_video_comments(self, video_id_list, checkpoint_id=""):
            self.video_ids.extend(video_id_list)
            self.checkpoint_id = checkpoint_id

    client = FakeClient()
    video_processor = FakeVideoProcessor()
    comment_processor = FakeCommentProcessor()
    hot_search_calls = install_hot_search_store_spy(
        monkeypatch, kuaishou_hot_search_handler
    )
    handler = KuaishouHotSearchHandler(
        client,
        FakeCheckpointManager(checkpoint_id="ks_hot_search_test"),
        video_processor,
        comment_processor,
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())
        assert client.cursors == [""]
        assert video_processor.video_ids == ["ks_hot_1", "ks_hot_2"]
        assert video_processor.checkpoint_id == "ks_hot_search_test"
        assert video_processor.source_keywords == ["kuaishou_hot_search"]
        assert comment_processor.video_ids == ["ks_hot_1", "ks_hot_2"]
        assert comment_processor.checkpoint_id == "ks_hot_search_test"
        assert [item.title for item in hot_search_calls["items"]] == [
            "快手热点1",
            "快手热点2",
        ]
        assert hot_search_calls["items"][0].rank_no == 1
        assert hot_search_calls["relations"][0]["content_ids"] == ["ks_hot_1"]
        assert hot_search_calls["relations"][0]["content_type"] == "video"
        assert hot_search_calls["relations"][0]["content_table"] == "kuaishou_video"
        assert hot_search_calls["relations"][1]["content_ids"] == ["ks_hot_2"]
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_douyin_hot_search_handler_uses_ranked_hotword_aweme_ids(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.hotwords = []

        async def get_hot_search_items(self):
            return [
                {"word": "无代表视频", "position": 1},
                {"word": "无排名", "group_id": "skip_no_rank"},
                {"word": "抖音热搜1", "position": 1, "group_id": "group1"},
                {"word": "抖音热搜2", "position": 2, "group_id": "group2"},
            ]

        async def get_hot_search_aweme_ids(self, hotword, offset=0, count=10):
            self.hotwords.append((hotword, offset, count))
            if hotword == "无代表视频":
                return []
            return {
                "抖音热搜1": ["aweme1", "aweme1_extra"],
                "抖音热搜2": ["aweme2"],
            }.get(hotword, [])

    class FakeCheckpointManager:
        async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
            return None

        async def save_checkpoint(self, checkpoint):
            checkpoint.id = "dy_hot_search_test"
            return checkpoint

        async def update_checkpoint(self, checkpoint):
            return checkpoint

    class FakeAwemeProcessor:
        def __init__(self):
            self.aweme_ids = []
            self.checkpoint_id = ""
            self.source_keywords = []
            self.request_keywords = []

        async def batch_get_aweme_list_from_ids(self, aweme_ids, checkpoint_id):
            self.aweme_ids.extend(aweme_ids)
            self.checkpoint_id = checkpoint_id
            self.source_keywords.append(source_keyword_var.get())
            self.request_keywords.append(request_keyword_var.get())
            return aweme_ids

    class FakeCommentProcessor:
        def __init__(self):
            self.aweme_ids = []
            self.checkpoint_id = ""

        async def batch_get_aweme_comments(self, aweme_ids, checkpoint_id):
            self.aweme_ids.extend(aweme_ids)
            self.checkpoint_id = checkpoint_id

    aweme_processor = FakeAwemeProcessor()
    comment_processor = FakeCommentProcessor()
    client = FakeClient()
    hot_search_calls = install_hot_search_store_spy(
        monkeypatch, douyin_hot_search_handler
    )
    handler = DouyinHotSearchHandler(
        client,
        FakeCheckpointManager(),
        aweme_processor,
        comment_processor,
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 1
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())
        assert client.hotwords == [
            ("无代表视频", 0, 1),
            ("抖音热搜1", 0, 1),
            ("抖音热搜2", 0, 1),
        ]
        assert aweme_processor.aweme_ids == ["aweme1", "aweme2"]
        assert aweme_processor.checkpoint_id == "dy_hot_search_test"
        assert aweme_processor.source_keywords == ["抖音热搜1", "抖音热搜2"]
        assert aweme_processor.request_keywords == ["抖音热搜1", "抖音热搜2"]
        assert comment_processor.aweme_ids == ["aweme1", "aweme2"]
        assert comment_processor.checkpoint_id == "dy_hot_search_test"
        assert [item.title for item in hot_search_calls["items"]] == [
            "无代表视频",
            "抖音热搜1",
            "抖音热搜2",
        ]
        assert [item.rank_no for item in hot_search_calls["items"]] == [1, 1, 2]
        assert hot_search_calls["relations"][0]["content_ids"] == ["aweme1"]
        assert hot_search_calls["relations"][0]["content_type"] == "aweme"
        assert hot_search_calls["relations"][0]["content_table"] == "douyin_aweme"
        assert hot_search_calls["relations"][1]["content_ids"] == ["aweme2"]
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_zhihu_hot_search_handler_uses_hot_questions_answers(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP

    class FakeClient:
        def __init__(self):
            self.question_ids = []

        async def get_hot_search_items(self, offset=0, limit=20):
            return {
                "data": [
                    {"question": {"id": "q_no_answer", "title": "无回答热榜"}},
                    {},
                    {"question": {"id": "q1", "title": "知乎热榜1"}},
                    {"question": {"id": "q2", "title": "知乎热榜2"}},
                ],
                "paging": {"is_end": True},
            }

        async def get_all_answers_by_question_id(
            self,
            question_id,
            crawl_interval=1.0,
            max_answers=0,
            order="default",
            callback=None,
        ):
            self.question_ids.append((question_id, max_answers))
            answers = {
                "q_no_answer": [],
                "q1": [
                    ZhihuContent(
                        content_id="answer1",
                        content_type=constant.ANSWER_NAME,
                        question_id="q1",
                        title="知乎热榜1",
                    ),
                    ZhihuContent(
                        content_id="answer1_extra",
                        content_type=constant.ANSWER_NAME,
                        question_id="q1",
                        title="知乎热榜1",
                    ),
                ],
                "q2": [
                    ZhihuContent(
                        content_id="answer2",
                        content_type=constant.ANSWER_NAME,
                        question_id="q2",
                        title="知乎热榜2",
                    )
                ],
            }.get(question_id, [])
            return answers[:max_answers]

    class FakeCheckpointManager:
        async def load_checkpoint(self, platform=None, mode=None, checkpoint_id=None):
            return None

        async def save_checkpoint(self, checkpoint):
            checkpoint.id = "zhihu_hot_search_test"
            return checkpoint

        async def update_checkpoint(self, checkpoint):
            return checkpoint

    class FakeContentProcessor:
        def __init__(self):
            self.content_ids = []
            self.checkpoint_id = ""
            self.source_keywords = []

        async def batch_process_contents(self, content_list, checkpoint_id=""):
            self.content_ids.extend([content.content_id for content in content_list])
            self.checkpoint_id = checkpoint_id
            self.source_keywords.append(source_keyword_var.get())
            return [content.content_id for content in content_list]

    class FakeCommentProcessor:
        def __init__(self):
            self.content_ids = []
            self.checkpoint_id = ""

        async def batch_get_content_comments(self, content_list, checkpoint_id=""):
            self.content_ids.extend([content.content_id for content in content_list])
            self.checkpoint_id = checkpoint_id

    client = FakeClient()
    content_processor = FakeContentProcessor()
    comment_processor = FakeCommentProcessor()
    hot_search_calls = install_hot_search_store_spy(
        monkeypatch, zhihu_hot_search_handler
    )
    handler = ZhihuHotSearchHandler(
        client,
        FakeCheckpointManager(),
        content_processor,
        comment_processor,
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 1
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        asyncio.run(handler.handle())
        assert client.question_ids == [("q_no_answer", 1), ("q1", 1), ("q2", 1)]
        assert content_processor.content_ids == ["answer1", "answer2"]
        assert content_processor.checkpoint_id == "zhihu_hot_search_test"
        assert content_processor.source_keywords == ["知乎热榜1", "知乎热榜2"]
        assert comment_processor.content_ids == ["answer1", "answer2"]
        assert comment_processor.checkpoint_id == "zhihu_hot_search_test"
        assert [item.title for item in hot_search_calls["items"]] == [
            "无回答热榜",
            "知乎热榜1",
            "知乎热榜2",
        ]
        assert [item.rank_no for item in hot_search_calls["items"]] == [1, 3, 4]
        assert hot_search_calls["relations"][0]["content_ids"] == ["answer1"]
        assert hot_search_calls["relations"][0]["content_type"] == constant.ANSWER_NAME
        assert hot_search_calls["relations"][0]["content_table"] == "zhihu_content"
        assert hot_search_calls["relations"][1]["content_ids"] == ["answer2"]
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep


def test_weibo_hot_search_handler_resumes_from_checkpoint(monkeypatch):
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        def __init__(self):
            self.keywords = []

        async def get_hot_search_items(self):
            return [
                {"word": "已处理热搜", "realpos": 1},
                {"word": "继续热搜", "realpos": 2},
                {"word": "后续热搜", "realpos": 3},
            ]

        async def get_note_by_keyword(self, keyword):
            self.keywords.append(keyword)
            return [
                WeiboNote(note_id=f"{keyword}_note", content=keyword),
            ], {}

    class FakeNoteProcessor:
        async def batch_get_note_list(self, note_list, checkpoint_id=""):
            return [note.note_id for note in note_list]

    class FakeCommentProcessor:
        async def batch_get_note_comments(self, note_id_list, checkpoint_id=""):
            pass

    checkpoint = Checkpoint(
        id="resume_wb_hot",
        platform=constant.WEIBO_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_homefeed_note_index=1,
        crawled_note_list=[
            CheckpointNote(note_id="old_note", is_success_crawled=True)
        ],
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    handler = WeiboHotSearchHandler(
        client,
        checkpoint_manager,
        FakeNoteProcessor(),
        FakeCommentProcessor(),
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 2
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = "resume_wb_hot"
        asyncio.run(handler.handle())

        assert checkpoint_manager.load_calls == [
            (
                constant.WEIBO_PLATFORM_NAME,
                constant.CRALER_TYPE_HOT_SEARCH,
                "resume_wb_hot",
            )
        ]
        assert client.keywords == ["继续热搜", "后续热搜"]
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 3
        assert checkpoint_manager.checkpoint.current_search_keyword == "后续热搜"
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_douyin_hot_search_handler_resumes_from_checkpoint():
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        def __init__(self):
            self.hotwords = []

        async def get_hot_search_items(self):
            return [
                {"word": "已处理抖音", "position": 1},
                {"word": "继续抖音", "position": 2},
                {"word": "后续抖音", "position": 3},
            ]

        async def get_hot_search_aweme_ids(self, hotword, offset=0, count=10):
            self.hotwords.append((hotword, offset, count))
            return [f"{hotword}_aweme"]

    class FakeAwemeProcessor:
        async def batch_get_aweme_list_from_ids(self, aweme_ids, checkpoint_id):
            return aweme_ids

    class FakeCommentProcessor:
        async def batch_get_aweme_comments(self, aweme_ids, checkpoint_id):
            pass

    checkpoint = Checkpoint(
        id="resume_dy_hot",
        platform=constant.DOUYIN_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_homefeed_note_index=1,
        crawled_note_list=[
            CheckpointNote(note_id="old_aweme", is_success_crawled=True)
        ],
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    handler = DouyinHotSearchHandler(
        client,
        checkpoint_manager,
        FakeAwemeProcessor(),
        FakeCommentProcessor(),
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 2
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = "resume_dy_hot"
        asyncio.run(handler.handle())

        assert client.hotwords == [("继续抖音", 0, 1), ("后续抖音", 0, 1)]
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 3
        assert checkpoint_manager.checkpoint.current_search_keyword == "后续抖音"
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_bilibili_hot_search_handler_resumes_page_position_from_checkpoint():
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        def __init__(self):
            self.pages = []

        async def get_hot_search_items(self, page=1, page_size=20):
            self.pages.append((page, page_size))
            return {
                "list": [
                    {"aid": 1001, "bvid": "BVdone", "title": "已处理B站"},
                    {"aid": 1002, "bvid": "BVnext", "title": "继续B站"},
                    {"aid": 1003, "bvid": "BVlater", "title": "后续B站"},
                ],
                "no_more": False,
            }

    class FakeVideoProcessor:
        def __init__(self):
            self.video_list = []

        async def batch_get_video_list(self, video_list, checkpoint_id=""):
            self.video_list.extend(video_list)
            return [
                VideoIdInfo(aid=str(video["aid"]), bvid=video["bvid"])
                for video in video_list
            ]

    class FakeCommentProcessor:
        async def batch_get_video_comments(self, video_infos, checkpoint_id=""):
            pass

    checkpoint = Checkpoint(
        id="resume_bili_hot",
        platform=constant.BILIBILI_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_search_page=1,
        current_homefeed_note_index=1,
        crawled_note_list=[
            CheckpointNote(note_id="BVdone", is_success_crawled=True)
        ],
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    video_processor = FakeVideoProcessor()
    handler = BilibiliHotSearchHandler(
        client,
        checkpoint_manager,
        video_processor,
        FakeCommentProcessor(),
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 2
        config.CRAWLER_TIME_SLEEP = 0
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = "resume_bili_hot"
        asyncio.run(handler.handle())

        assert client.pages == [(1, 50)]
        assert video_processor.video_list == [
            {"aid": 1002, "bvid": "BVnext", "title": "继续B站"},
            {"aid": 1003, "bvid": "BVlater", "title": "后续B站"},
        ]
        assert checkpoint_manager.checkpoint.current_search_page == 1
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 3
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_kuaishou_hot_search_handler_resumes_cursor_position_from_checkpoint():
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        def __init__(self):
            self.cursors = []

        async def get_hot_search_items(self, pcursor=""):
            self.cursors.append(pcursor)
            return {
                "feeds": [
                    {"photo": {"id": "ks_done", "caption": "已处理快手"}},
                    {"photo": {"id": "ks_next", "caption": "继续快手"}},
                    {"photo": {"id": "ks_later", "caption": "后续快手"}},
                ],
                "pcursor": "1",
            }

    class FakeVideoProcessor:
        def __init__(self):
            self.video_ids = []

        async def batch_get_video_list(self, video_id_list, checkpoint_id=""):
            self.video_ids.extend(video_id_list)
            return video_id_list

    class FakeCommentProcessor:
        async def batch_get_video_comments(self, video_id_list, checkpoint_id=""):
            pass

    checkpoint = Checkpoint(
        id="resume_ks_hot",
        platform=constant.KUAISHOU_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_homefeed_cursor="",
        current_homefeed_note_index=1,
        crawled_note_list=[
            CheckpointNote(note_id="ks_done", is_success_crawled=True)
        ],
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    video_processor = FakeVideoProcessor()
    handler = KuaishouHotSearchHandler(
        client,
        checkpoint_manager,
        video_processor,
        FakeCommentProcessor(),
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 2
        config.CRAWLER_TIME_SLEEP = 0
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = "resume_ks_hot"
        asyncio.run(handler.handle())

        assert client.cursors == [""]
        assert video_processor.video_ids == ["ks_next", "ks_later"]
        assert checkpoint_manager.checkpoint.current_homefeed_cursor == "1"
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 3
        assert checkpoint_manager.checkpoint.current_search_keyword == "后续快手"
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_zhihu_hot_search_handler_resumes_offset_and_question_index_from_checkpoint():
    original_crawler_max_notes_count = config.CRAWLER_MAX_NOTES_COUNT
    original_per_item_count = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT
    original_crawler_time_sleep = config.CRAWLER_TIME_SLEEP
    original_enable_checkpoint = config.ENABLE_CHECKPOINT
    original_checkpoint_id = config.SPECIFIED_CHECKPOINT_ID

    class FakeClient:
        def __init__(self):
            self.requests = []

        async def get_hot_search_items(self, offset=0, limit=20):
            self.requests.append((offset, limit))
            return {
                "data": [
                    {"question": {"id": "q_done", "title": "已处理知乎"}},
                    {"question": {"id": "q_next", "title": "继续知乎"}},
                    {"question": {"id": "q_later", "title": "后续知乎"}},
                ],
                "paging": {"is_end": True},
            }

        async def get_all_answers_by_question_id(
            self,
            question_id,
            crawl_interval=1.0,
            max_answers=0,
            order="default",
            callback=None,
        ):
            return [
                ZhihuContent(
                    content_id=f"{question_id}_answer",
                    content_type=constant.ANSWER_NAME,
                    question_id=question_id,
                    title="继续知乎",
                )
            ][:max_answers]

    class FakeContentProcessor:
        def __init__(self):
            self.content_ids = []

        async def batch_process_contents(self, content_list, checkpoint_id=""):
            self.content_ids.extend([content.content_id for content in content_list])
            return [content.content_id for content in content_list]

    class FakeCommentProcessor:
        async def batch_get_content_comments(self, content_list, checkpoint_id=""):
            pass

    checkpoint = Checkpoint(
        id="resume_zhihu_hot",
        platform=constant.ZHIHU_PLATFORM_NAME,
        mode=constant.CRALER_TYPE_HOT_SEARCH,
        current_search_page=0,
        current_homefeed_note_index=1,
        crawled_note_list=[
            CheckpointNote(note_id="old_answer", is_success_crawled=True)
        ],
    )
    checkpoint_manager = FakeCheckpointManager(checkpoint)
    client = FakeClient()
    content_processor = FakeContentProcessor()
    handler = ZhihuHotSearchHandler(
        client,
        checkpoint_manager,
        content_processor,
        FakeCommentProcessor(),
    )

    try:
        config.CRAWLER_MAX_NOTES_COUNT = 2
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = 1
        config.CRAWLER_TIME_SLEEP = 0
        config.ENABLE_CHECKPOINT = True
        config.SPECIFIED_CHECKPOINT_ID = "resume_zhihu_hot"
        asyncio.run(handler.handle())

        assert client.requests == [(0, 50)]
        assert content_processor.content_ids == ["q_next_answer", "q_later_answer"]
        assert checkpoint_manager.checkpoint.current_search_page == 0
        assert checkpoint_manager.checkpoint.current_homefeed_note_index == 3
        assert checkpoint_manager.checkpoint.current_search_id == "q_later"
        assert checkpoint_manager.checkpoint.current_search_keyword == "后续知乎"
        assert checkpoint_manager.checkpoint.is_completed is True
    finally:
        config.CRAWLER_MAX_NOTES_COUNT = original_crawler_max_notes_count
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = original_per_item_count
        config.CRAWLER_TIME_SLEEP = original_crawler_time_sleep
        config.ENABLE_CHECKPOINT = original_enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = original_checkpoint_id


def test_hot_search_builders_create_stable_topic_snapshots():
    weibo_item = build_weibo_hot_search_item(
        {
            "word": "测试热搜",
            "word_scheme": "#测试热搜#",
            "num": 12345,
            "realpos": 3,
            "label_name": "热",
        },
        snapshot_ts=1700000000000,
    )
    douyin_item = build_douyin_hot_search_item(
        {
            "word": "抖音热搜",
            "position": 2,
            "group_id": "group1",
            "hot_value": 888,
        },
        snapshot_ts=1700000000000,
    )

    assert weibo_item.platform == constant.WEIBO_PLATFORM_NAME
    assert weibo_item.board_type == "topic"
    assert weibo_item.rank_no == 3
    assert weibo_item.title == "测试热搜"
    assert weibo_item.hot_value == "12345"
    assert weibo_item.badge == "热"
    assert "s.weibo.com/weibo" in weibo_item.topic_url

    assert douyin_item.platform == constant.DOUYIN_PLATFORM_NAME
    assert douyin_item.rank_no == 2
    assert douyin_item.topic_id == "group1"
    assert douyin_item.hot_value == "888"


def test_hot_search_content_relation_links_topic_to_content():
    hot_item = build_weibo_hot_search_item(
        {"word": "关系热搜", "realpos": 1},
        snapshot_ts=1700000000000,
    )

    relation = build_hot_search_content_relation(
        hot_item=hot_item,
        content_id="note1",
        content_type="note",
        content_table="weibo_note",
        source_keyword="关系热搜",
    )

    assert relation.hot_item_id == hot_item.hot_item_id
    assert relation.platform == constant.WEIBO_PLATFORM_NAME
    assert relation.content_id == "note1"
    assert relation.content_type == "note"
    assert relation.content_table == "weibo_note"
    assert relation.rank_no == 1
    assert relation.source_keyword == "关系热搜"


def test_hot_search_item_id_uses_snapshot_date_not_current_date(monkeypatch):
    snapshot_ts = 1700000000000
    raw_item = {"word": "跨天热搜", "realpos": 1}

    monkeypatch.setattr(utils, "get_current_date", lambda: "2099-01-01")
    first_item = build_weibo_hot_search_item(raw_item, snapshot_ts=snapshot_ts)

    monkeypatch.setattr(utils, "get_current_date", lambda: "1999-01-01")
    second_item = build_weibo_hot_search_item(raw_item, snapshot_ts=snapshot_ts)

    assert first_item.snapshot_date == utils.get_date_str_from_unix_time(snapshot_ts)
    assert second_item.snapshot_date == first_item.snapshot_date
    assert second_item.hot_item_id == first_item.hot_item_id
