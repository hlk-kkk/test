# -*- coding: utf-8 -*-
import asyncio
import json
import pathlib

import pytest


ROOT = pathlib.Path(__file__).resolve().parent.parent

FORBIDDEN_KEYS = {
    "user_id",
    "sec_uid",
    "short_user_id",
    "user_unique_id",
    "user_signature",
    "avatar",
    "user_avatar",
    "ip_location",
    "gender",
    "profile_url",
    "user_link",
    "user_url_token",
    "url_token",
    "description",
}

RAW_SENTINELS = {
    "PII_USER",
    "PII_SEC",
    "PII_SHORT",
    "PII_UNIQUE",
    "PII_NICK",
    "PII_AVATAR",
    "PII_IP",
    "PII_PROFILE",
    "PII_LINK",
    "PII_TOKEN",
    "PII_SIGNATURE",
}


class SpyStore:
    def __init__(self):
        self.calls = []

    async def store_content(self, content_item):
        self.calls.append(("content", dict(content_item)))

    async def store_comment(self, comment_item):
        self.calls.append(("comment", dict(comment_item)))

    async def store_creator(self, creator):
        self.calls.append(("creator", dict(creator)))


def _assert_sanitized(item):
    assert "creator_hash" in item
    assert not (set(item) & FORBIDDEN_KEYS)
    payload = json.dumps(item, ensure_ascii=False)
    for raw_value in RAW_SENTINELS:
        assert raw_value not in payload


def _patch_factory(monkeypatch, factory, spy):
    monkeypatch.setattr(factory, "create_store", staticmethod(lambda: spy))


@pytest.mark.parametrize(
    "module_path,factory_name,update_name,model_path,kwargs",
    [
        (
            "repo.platform_save_data.xhs",
            "XhsStoreFactory",
            "update_xhs_note",
            "model.m_xhs:XhsNote",
            {
                "note_id": "xhs-note",
                "type": "normal",
                "title": "title",
                "time": "1",
                "last_update_time": "1",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
        ),
        (
            "repo.platform_save_data.douyin",
            "DouyinStoreFactory",
            "update_douyin_aweme",
            "model.m_douyin:DouyinAweme",
            {
                "aweme_id": "dy-aweme",
                "aweme_type": "video",
                "title": "title",
                "create_time": "1",
                "user_id": "PII_USER",
                "sec_uid": "PII_SEC",
                "short_user_id": "PII_SHORT",
                "user_unique_id": "PII_UNIQUE",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
                "user_signature": "PII_SIGNATURE",
                "ip_location": "PII_IP",
            },
        ),
        (
            "repo.platform_save_data.kuaishou",
            "KuaishouStoreFactory",
            "update_kuaishou_video",
            "model.m_kuaishou:KuaishouVideo",
            {
                "video_id": "ks-video",
                "video_type": "video",
                "title": "title",
                "create_time": "1",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
            },
        ),
        (
            "repo.platform_save_data.bilibili",
            "BiliStoreFactory",
            "update_bilibili_video",
            "model.m_bilibili:BilibiliVideo",
            {
                "video_id": "100",
                "bvid": "BV100",
                "video_type": "video",
                "title": "title",
                "create_time": "1",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
            },
        ),
        (
            "repo.platform_save_data.weibo",
            "WeibostoreFactory",
            "update_weibo_note",
            "model.m_weibo:WeiboNote",
            {
                "note_id": "wb-note",
                "content": "content",
                "create_time": "1",
                "create_date_time": "2026-01-01",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "gender": "PII_GENDER",
                "profile_url": "PII_PROFILE",
                "avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
        ),
        (
            "repo.platform_save_data.tieba",
            "TieBaStoreFactory",
            "update_tieba_note",
            "model.m_baidu_tieba:TiebaNote",
            {
                "note_id": "tb-note",
                "title": "title",
                "note_url": "https://tieba.baidu.com/p/1",
                "tieba_name": "bar",
                "tieba_link": "https://tieba.baidu.com/f?kw=bar",
                "user_link": "PII_LINK",
                "user_nickname": "PII_NICK",
                "user_avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
        ),
        (
            "repo.platform_save_data.zhihu",
            "ZhihuStoreFactory",
            "update_zhihu_content",
            "model.m_zhihu:ZhihuContent",
            {
                "content_id": "zh-content",
                "content_type": "answer",
                "title": "title",
                "user_id": "PII_USER",
                "user_link": "PII_LINK",
                "user_nickname": "PII_NICK",
                "user_avatar": "PII_AVATAR",
                "user_url_token": "PII_TOKEN",
            },
        ),
    ],
)
def test_platform_content_storage_is_sanitized(
    monkeypatch, module_path, factory_name, update_name, model_path, kwargs
):
    module = __import__(module_path, fromlist=[factory_name, update_name])
    model_module_name, model_name = model_path.split(":")
    model_module = __import__(model_module_name, fromlist=[model_name])
    model = getattr(model_module, model_name)
    spy = SpyStore()
    _patch_factory(monkeypatch, getattr(module, factory_name), spy)

    asyncio.run(getattr(module, update_name)(model(**kwargs)))

    assert len(spy.calls) == 1
    _assert_sanitized(spy.calls[0][1])


@pytest.mark.parametrize(
    "module_path,factory_name,update_name,model_path,kwargs,args_prefix",
    [
        (
            "repo.platform_save_data.xhs",
            "XhsStoreFactory",
            "update_xhs_note_comment",
            "model.m_xhs:XhsComment",
            {
                "comment_id": "xhs-comment",
                "note_id": "xhs-note",
                "content": "comment",
                "create_time": "1",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
            [],
        ),
        (
            "repo.platform_save_data.douyin",
            "DouyinStoreFactory",
            "update_dy_aweme_comment",
            "model.m_douyin:DouyinAwemeComment",
            {
                "comment_id": "dy-comment",
                "aweme_id": "dy-aweme",
                "content": "comment",
                "create_time": "1",
                "sub_comment_count": "0",
                "user_id": "PII_USER",
                "sec_uid": "PII_SEC",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
            [],
        ),
        (
            "repo.platform_save_data.kuaishou",
            "KuaishouStoreFactory",
            "update_ks_video_comment",
            "model.m_kuaishou:KuaishouVideoComment",
            {
                "comment_id": "ks-comment",
                "video_id": "ks-video",
                "content": "comment",
                "create_time": "1",
                "sub_comment_count": "0",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
            },
            [],
        ),
        (
            "repo.platform_save_data.bilibili",
            "BiliStoreFactory",
            "update_bilibili_video_comment",
            "model.m_bilibili:BilibiliComment",
            {
                "comment_id": "bili-comment",
                "video_id": "100",
                "content": "comment",
                "create_time": "1",
                "sub_comment_count": "0",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "avatar": "PII_AVATAR",
            },
            [],
        ),
        (
            "repo.platform_save_data.weibo",
            "WeibostoreFactory",
            "update_weibo_note_comment",
            "model.m_weibo:WeiboComment",
            {
                "comment_id": "wb-comment",
                "note_id": "wb-note",
                "content": "comment",
                "create_time": "1",
                "create_date_time": "2026-01-01",
                "user_id": "PII_USER",
                "nickname": "PII_NICK",
                "profile_url": "PII_PROFILE",
                "avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
            [],
        ),
        (
            "repo.platform_save_data.tieba",
            "TieBaStoreFactory",
            "update_tieba_note_comment",
            "model.m_baidu_tieba:TiebaComment",
            {
                "comment_id": "tb-comment",
                "content": "comment",
                "note_id": "tb-note",
                "note_url": "https://tieba.baidu.com/p/1",
                "tieba_id": "1",
                "tieba_name": "bar",
                "tieba_link": "https://tieba.baidu.com/f?kw=bar",
                "user_link": "PII_LINK",
                "user_nickname": "PII_NICK",
                "user_avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
            ["tb-note"],
        ),
        (
            "repo.platform_save_data.zhihu",
            "ZhihuStoreFactory",
            "update_zhihu_content_comment",
            "model.m_zhihu:ZhihuComment",
            {
                "comment_id": "zh-comment",
                "content_id": "zh-content",
                "content_type": "answer",
                "content": "comment",
                "publish_time": 1,
                "user_id": "PII_USER",
                "user_link": "PII_LINK",
                "user_nickname": "PII_NICK",
                "user_avatar": "PII_AVATAR",
                "ip_location": "PII_IP",
            },
            [],
        ),
    ],
)
def test_platform_comment_storage_is_sanitized(
    monkeypatch, module_path, factory_name, update_name, model_path, kwargs, args_prefix
):
    module = __import__(module_path, fromlist=[factory_name, update_name])
    model_module_name, model_name = model_path.split(":")
    model_module = __import__(model_module_name, fromlist=[model_name])
    model = getattr(model_module, model_name)
    spy = SpyStore()
    _patch_factory(monkeypatch, getattr(module, factory_name), spy)

    asyncio.run(getattr(module, update_name)(*args_prefix, model(**kwargs)))

    assert len(spy.calls) == 1
    _assert_sanitized(spy.calls[0][1])


def test_schema_no_persisted_identity_columns():
    forbidden_schema_tokens = FORBIDDEN_KEYS | {
        "bilibili_up_info",
        "dy_creator",
        "kuaishou_creator",
        "xhs_creator",
        "weibo_creator",
        "tieba_creator",
        "zhihu_creator",
    }
    for path in [ROOT / "schema/tables.sql", ROOT / "schema/sqlite_tables.sql"]:
        text = path.read_text()
        for token in forbidden_schema_tokens:
            assert token not in text
        assert text.count("creator_hash") >= 14


def test_hot_search_raw_json_is_sanitized():
    from repo.platform_save_data.hot_search import build_bilibili_hot_search_item

    item = build_bilibili_hot_search_item(
        {
            "bvid": "BV100",
            "title": "safe title",
            "owner": {"mid": "PII_USER", "name": "PII_NICK", "face": "PII_AVATAR"},
            "stat": {"view": 1},
        },
        rank_no=1,
    )

    assert "safe title" in item.raw_json
    assert "owner" not in item.raw_json
    for raw_value in RAW_SENTINELS:
        assert raw_value not in item.raw_json


def test_checkpoint_dump_does_not_persist_creator_id_or_raw_snapshot():
    from model.m_checkpoint import Checkpoint, CheckpointNote

    checkpoint = Checkpoint(
        platform="xhs",
        mode="creator",
        current_creator_id="PII_USER",
        current_hot_search_snapshot_items=[
            {"title": "safe title", "author": {"id": "PII_USER", "name": "PII_NICK"}}
        ],
        crawled_note_list=[
            CheckpointNote(
                note_id="note",
                extra_params_info={"user": {"id": "PII_USER"}, "safe": "value"},
            )
        ],
    )

    dumped = checkpoint.model_dump()
    assert "current_creator_id" not in dumped
    payload = json.dumps(dumped, ensure_ascii=False)
    assert "safe title" in payload
    assert "value" in payload
    for raw_value in RAW_SENTINELS:
        assert raw_value not in payload


def test_sqlite_legacy_privacy_migration_preserves_and_scrubs_old_tables(monkeypatch, tmp_path):
    async def scenario():
        import config
        import db
        from async_db import AsyncSQLiteDB
        from pkg.tools.user_privacy import anonymize_user_id
        from var import media_crawler_db_var

        monkeypatch.setattr(config, "DB_TYPE", "sqlite")

        sqlite_db = AsyncSQLiteDB(str(tmp_path / "legacy.db"))
        await sqlite_db.connect()
        token = media_crawler_db_var.set(sqlite_db)
        try:
            await sqlite_db.execute(
                """
                CREATE TABLE `xhs_note`
                (
                    `id`               INTEGER PRIMARY KEY AUTOINCREMENT,
                    `user_id`          TEXT NOT NULL,
                    `nickname`         TEXT,
                    `avatar`           TEXT,
                    `ip_location`      TEXT,
                    `add_ts`           INTEGER NOT NULL,
                    `last_modify_ts`   INTEGER NOT NULL,
                    `note_id`          TEXT NOT NULL,
                    `time`             INTEGER NOT NULL,
                    `last_update_time` INTEGER NOT NULL,
                    `title`            TEXT
                );
                CREATE TABLE `xhs_creator`
                (
                    `id`             INTEGER PRIMARY KEY AUTOINCREMENT,
                    `user_id`        TEXT NOT NULL UNIQUE,
                    `nickname`       TEXT,
                    `avatar`         TEXT,
                    `ip_location`    TEXT,
                    `add_ts`         INTEGER NOT NULL,
                    `last_modify_ts` INTEGER NOT NULL
                );
                INSERT INTO `xhs_note`
                    (`user_id`, `nickname`, `avatar`, `ip_location`, `add_ts`, `last_modify_ts`,
                     `note_id`, `time`, `last_update_time`, `title`)
                VALUES
                    ('PII_USER', 'PII_NICK', 'PII_AVATAR', 'PII_IP', 1, 1, 'old-note', 1, 1, 'old title');
                INSERT INTO `xhs_creator`
                    (`user_id`, `nickname`, `avatar`, `ip_location`, `add_ts`, `last_modify_ts`)
                VALUES
                    ('PII_CREATOR', 'PII_CREATOR_NICK', 'PII_CREATOR_AVATAR', 'PII_CREATOR_IP', 1, 1);
                """
            )

            await db.ensure_privacy_schema_compatibility()
            await db.ensure_privacy_schema_compatibility()

            note_columns = {
                row["name"] for row in await sqlite_db.query("PRAGMA table_info(`xhs_note`)")
            }
            creator_columns = {
                row["name"] for row in await sqlite_db.query("PRAGMA table_info(`xhs_creator`)")
            }
            assert {"user_id", "avatar", "ip_location", "creator_hash"} <= note_columns
            assert {"user_id", "avatar", "ip_location", "creator_hash"} <= creator_columns

            note = await sqlite_db.get_first("SELECT * FROM `xhs_note` WHERE `note_id`=%s", "old-note")
            assert note["creator_hash"] == anonymize_user_id("PII_USER")
            assert note["nickname"] == "P***K"
            assert note["user_id"] == "redacted_1"
            assert note["avatar"] == ""
            assert note["ip_location"] == ""

            creator = await sqlite_db.get_first("SELECT * FROM `xhs_creator` WHERE `id`=%s", 1)
            assert creator["creator_hash"] == anonymize_user_id("PII_CREATOR")
            assert creator["nickname"] == "P***K"
            assert creator["user_id"] == "redacted_1"
            assert creator["avatar"] == ""
            assert creator["ip_location"] == ""

            await sqlite_db.item_to_table(
                "xhs_note",
                {
                    "note_id": "new-note",
                    "nickname": "NEW_NICK",
                    "add_ts": 2,
                    "last_modify_ts": 2,
                    "time": 2,
                    "last_update_time": 2,
                    "title": "new title",
                },
            )
            inserted = await sqlite_db.get_first(
                "SELECT * FROM `xhs_note` WHERE `note_id`=%s", "new-note"
            )
            assert inserted["nickname"] == "N***K"
            assert inserted["user_id"].startswith("redacted_")
            assert inserted["avatar"] == ""
            assert inserted["ip_location"] == ""
        finally:
            media_crawler_db_var.reset(token)
            await sqlite_db.close()

    asyncio.run(scenario())


def test_privacy_migration_does_not_drop_old_schema_objects():
    text = (ROOT / "db.py").read_text()
    assert "DROP COLUMN" not in text
    assert "DROP TABLE IF EXISTS `{table_name}`" not in text
