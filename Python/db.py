# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/db.py
# GitHub: https://github.com/NanmiCoder
# Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
#

# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。


# -*- coding: utf-8 -*-
# @Author  : relakkes@gmail.com
# @Time    : 2024/4/6 14:54
# @Desc    : mediacrawler db 管理（支持 MySQL 和 SQLite）
import asyncio

import aiofiles
import aiomysql

import config
from async_db import AbstractAsyncDB, AsyncMysqlDB, AsyncSQLiteDB
from pkg.tools import utils
from pkg.tools.user_privacy import anonymize_user_id, mask_nickname
from var import db_conn_pool_var, media_crawler_db_var


async def init_mediacrawler_db():
    """
    初始化 MySQL 数据库链接池对象，并将该对象塞给 media_crawler_db_var 上下文变量
    """
    pool = await aiomysql.create_pool(
        host=config.RELATION_DB_HOST,
        port=config.RELATION_DB_PORT,
        user=config.RELATION_DB_USER,
        password=config.RELATION_DB_PWD,
        db=config.RELATION_DB_NAME,
        autocommit=True,
    )
    async_db_obj = AsyncMysqlDB(pool)

    # 将连接池对象和封装的CRUD sql接口对象放到上下文变量中
    db_conn_pool_var.set(pool)
    media_crawler_db_var.set(async_db_obj)


async def init_sqlite_db():
    """
    初始化 SQLite 数据库连接对象，并将该对象塞给 media_crawler_db_var 上下文变量
    """
    async_db_obj = AsyncSQLiteDB(config.SQLITE_DB_PATH)
    await async_db_obj.connect()
    media_crawler_db_var.set(async_db_obj)


async def init_db():
    """
    初始化 db 连接，根据 DB_TYPE 配置选择 MySQL 或 SQLite
    """
    utils.logger.info(f"[init_db] start init mediacrawler db connect object, DB_TYPE={config.DB_TYPE}")
    if config.DB_TYPE == "sqlite":
        await init_sqlite_db()
    else:
        await init_mediacrawler_db()
    await init_table_schema()
    await ensure_privacy_schema_compatibility()
    await ensure_mysql_schema_compatibility()
    await ensure_hot_search_schema_compatibility()
    utils.logger.info("[init_db] end init mediacrawler db connect object")


async def close():
    """
    关闭数据库连接/连接池
    """
    utils.logger.info("[close] close mediacrawler db connection")
    if config.DB_TYPE == "sqlite":
        async_db_obj: AbstractAsyncDB = media_crawler_db_var.get(None)
        if async_db_obj is not None and isinstance(async_db_obj, AsyncSQLiteDB):
            await async_db_obj.close()
    else:
        db_pool = db_conn_pool_var.get(None)
        if db_pool is not None:
            db_pool.close()
            await db_pool.wait_closed()


async def init_table_schema():
    """
    初始化数据库表结构。首次运行时自动建表，已有表则跳过。
    支持 MySQL 和 SQLite 两种模式。
    调用前需已通过 init_db()/init_sqlite_db()/init_mediacrawler_db() 建立连接。
    """
    utils.init_logging_config()
    utils.logger.info(f"[init_table_schema] begin init db table schema, DB_TYPE={config.DB_TYPE} ...")

    # 使用已有连接，不重复初始化（避免连接泄漏）
    async_db_obj: AbstractAsyncDB = media_crawler_db_var.get()

    # 检查是否已经初始化过表结构（不同数据库使用不同的 check 语句）
    if config.DB_TYPE == "sqlite":
        check_sql = "SELECT name FROM sqlite_master WHERE type='table'"
    else:
        check_sql = "show tables"

    tables = await async_db_obj.query(check_sql)
    if len(tables) > 0:
        utils.logger.info(
            "[init_table_schema] db table schema already init, skip init table schema"
        )
        return

    # 根据数据库类型选择对应的 schema 文件
    if config.DB_TYPE == "sqlite":
        schema_file = "schema/sqlite_tables.sql"
    else:
        schema_file = "schema/tables.sql"

    async with aiofiles.open(schema_file, mode="r", encoding="utf-8") as f:
        schema_sql = await f.read()
        await async_db_obj.execute(schema_sql)
        utils.logger.info(
            "[init_table_schema] db table schema init successful"
        )


async def ensure_mysql_schema_compatibility():
    """
    为存量 MySQL 库补齐必要的兼容修复。
    老库中的 douyin_aweme.title 可能仍是 varchar(255)，
    但当前抓取逻辑会将长 desc 回退写入 title。
    老库中的 xhs_note_comment.pictures 可能仍是 varchar(512)，
    但小红书评论图片会存多个长 URL。
    """
    if config.DB_TYPE == "sqlite":
        return

    async_db_obj: AbstractAsyncDB = media_crawler_db_var.get()
    column_info = await async_db_obj.get_first(
        """
        SELECT DATA_TYPE AS data_type, CHARACTER_MAXIMUM_LENGTH AS char_len
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = %s
          AND TABLE_NAME = 'douyin_aweme'
          AND COLUMN_NAME = 'title'
        """,
        config.RELATION_DB_NAME,
    )
    if column_info:
        data_type = str(column_info.get("data_type", "")).lower()
        if data_type not in {"text", "mediumtext", "longtext"}:
            await async_db_obj.execute(
                "alter table douyin_aweme modify column `title` text comment '视频标题'"
            )
            utils.logger.info(
                "[ensure_mysql_schema_compatibility] upgraded douyin_aweme.title to TEXT"
            )

    column_info = await async_db_obj.get_first(
        """
        SELECT DATA_TYPE AS data_type, CHARACTER_MAXIMUM_LENGTH AS char_len
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = %s
          AND TABLE_NAME = 'xhs_note_comment'
          AND COLUMN_NAME = 'pictures'
        """,
        config.RELATION_DB_NAME,
    )
    if column_info:
        data_type = str(column_info.get("data_type", "")).lower()
        if data_type != "longtext":
            await async_db_obj.execute(
                "alter table xhs_note_comment modify column `pictures` longtext "
                "DEFAULT NULL COMMENT '图片链接列表，逗号分隔'"
            )
            utils.logger.info(
                "[ensure_mysql_schema_compatibility] upgraded xhs_note_comment.pictures to LONGTEXT"
            )


PRIVACY_CONTENT_TABLES = (
    "bilibili_video",
    "bilibili_video_comment",
    "douyin_aweme",
    "douyin_aweme_comment",
    "kuaishou_video",
    "kuaishou_video_comment",
    "weibo_note",
    "weibo_note_comment",
    "xhs_note",
    "xhs_note_comment",
    "tieba_note",
    "tieba_comment",
    "zhihu_content",
    "zhihu_comment",
)

PRIVACY_CREATOR_TABLES = (
    "bilibili_up_info",
    "dy_creator",
    "kuaishou_creator",
    "xhs_creator",
    "weibo_creator",
    "tieba_creator",
    "zhihu_creator",
)

PRIVACY_USER_ID_SOURCE_COLUMNS = (
    "user_id",
    "sec_uid",
    "short_user_id",
    "user_unique_id",
    "profile_url",
    "user_link",
    "user_url_token",
    "url_token",
    "mid",
    "uid",
    "author_id",
)

PRIVACY_NICKNAME_COLUMNS = ("nickname", "user_nickname")

LEGACY_PRIVACY_COLUMNS = PRIVACY_USER_ID_SOURCE_COLUMNS + (
    "user_signature",
    "user_name",
    "avatar",
    "user_avatar",
    "ip_location",
    "ip_address",
    "gender",
    "sex",
    "description",
)


async def ensure_privacy_schema_compatibility():
    """
    Make existing databases compatible with the privacy-preserving schema.

    New installations use schema/*.sql without raw user identity columns.
    Existing installations may already have those columns and creator tables.
    Do not drop them automatically: preserve the user's old schema/data, add
    creator_hash where needed, backfill it from legacy IDs, and mask stored
    nicknames in place so future reads/writes use anonymized author data.
    """
    async_db_obj: AbstractAsyncDB = media_crawler_db_var.get()
    if config.DB_TYPE == "sqlite":
        await _ensure_sqlite_creator_tables_compatibility(async_db_obj)

        for table_name in PRIVACY_CONTENT_TABLES:
            table_exists = await async_db_obj.get_first(
                "SELECT name FROM sqlite_master WHERE type='table' AND name=%s",
                table_name,
            )
            if not table_exists:
                continue

            column_names = await _get_sqlite_column_names(async_db_obj, table_name)
            await _ensure_privacy_table_compatibility(
                async_db_obj,
                table_name,
                column_names,
                creator_hash_type="TEXT DEFAULT ''",
            )
        return

    await _ensure_mysql_creator_tables_compatibility(async_db_obj)

    for table_name in PRIVACY_CONTENT_TABLES:
        table_exists = await async_db_obj.get_first(
            """
            SELECT TABLE_NAME AS table_name
            FROM information_schema.TABLES
            WHERE TABLE_SCHEMA = %s
              AND TABLE_NAME = %s
            """,
            config.RELATION_DB_NAME,
            table_name,
        )
        if not table_exists:
            continue

        column_names = await _get_mysql_column_names(async_db_obj, table_name)
        await _ensure_privacy_table_compatibility(
            async_db_obj,
            table_name,
            column_names,
            creator_hash_type="varchar(64) DEFAULT '' COMMENT '匿名创作者哈希'",
        )


async def _get_sqlite_column_names(async_db_obj: AbstractAsyncDB, table_name: str) -> set[str]:
    table_columns = await async_db_obj.query(f"PRAGMA table_info(`{table_name}`)")
    return {column.get("name") for column in table_columns}


async def _get_mysql_column_names(async_db_obj: AbstractAsyncDB, table_name: str) -> set[str]:
    existing_columns = await async_db_obj.query(
        """
        SELECT COLUMN_NAME AS column_name
        FROM information_schema.COLUMNS
        WHERE TABLE_SCHEMA = %s
          AND TABLE_NAME = %s
        """,
        config.RELATION_DB_NAME,
        table_name,
    )
    return {column.get("column_name") for column in existing_columns}


async def _ensure_privacy_table_compatibility(
    async_db_obj: AbstractAsyncDB,
    table_name: str,
    column_names: set[str],
    creator_hash_type: str,
):
    if "creator_hash" not in column_names:
        after_clause = ""
        if config.DB_TYPE != "sqlite":
            if "nickname" in column_names:
                after_clause = " AFTER `nickname`"
            elif "user_nickname" in column_names:
                after_clause = " AFTER `user_nickname`"
        await async_db_obj.execute(
            f"ALTER TABLE `{table_name}` ADD COLUMN `creator_hash` {creator_hash_type}{after_clause}"
        )
        column_names = set(column_names)
        column_names.add("creator_hash")
        utils.logger.info(
            f"[ensure_privacy_schema_compatibility] added {table_name}.creator_hash"
        )

    await _backfill_creator_hash(async_db_obj, table_name, column_names)
    await _mask_existing_nicknames(async_db_obj, table_name, column_names)
    await _scrub_legacy_privacy_columns(async_db_obj, table_name, column_names)

    legacy_columns = sorted(set(column_names) & set(LEGACY_PRIVACY_COLUMNS))
    if legacy_columns:
        utils.logger.info(
            f"[ensure_privacy_schema_compatibility] table {table_name} still has "
            f"legacy privacy columns {','.join(legacy_columns)}; values are scrubbed "
            "and columns are preserved for compatibility"
        )


async def _backfill_creator_hash(
    async_db_obj: AbstractAsyncDB, table_name: str, column_names: set[str]
):
    if "id" not in column_names or "creator_hash" not in column_names:
        return

    for source_column in PRIVACY_USER_ID_SOURCE_COLUMNS:
        if source_column not in column_names:
            continue
        rows = await async_db_obj.query(
            f"""
            SELECT `id`, `{source_column}` AS source_value
            FROM `{table_name}`
            WHERE (`creator_hash` IS NULL OR `creator_hash` = '')
              AND `{source_column}` IS NOT NULL
              AND `{source_column}` != ''
              AND `{source_column}` NOT LIKE 'redacted_%'
            """
        )
        if not rows:
            continue
        updated_count = 0
        for row in rows:
            creator_hash = anonymize_user_id(row.get("source_value"))
            if not creator_hash:
                continue
            await async_db_obj.execute(
                f"UPDATE `{table_name}` SET `creator_hash`=%s WHERE `id`=%s",
                creator_hash,
                row.get("id"),
            )
            updated_count += 1
        if updated_count:
            utils.logger.info(
                f"[ensure_privacy_schema_compatibility] backfilled {updated_count} {table_name}.creator_hash values"
            )
        break


async def _mask_existing_nicknames(
    async_db_obj: AbstractAsyncDB, table_name: str, column_names: set[str]
):
    if "id" not in column_names:
        return

    for nickname_column in PRIVACY_NICKNAME_COLUMNS:
        if nickname_column not in column_names:
            continue
        rows = await async_db_obj.query(
            f"""
            SELECT `id`, `{nickname_column}` AS nickname_value
            FROM `{table_name}`
            WHERE `{nickname_column}` IS NOT NULL
              AND `{nickname_column}` != ''
            """
        )
        updated_count = 0
        for row in rows:
            raw_nickname = row.get("nickname_value")
            masked_nickname = mask_nickname(raw_nickname)
            if masked_nickname == raw_nickname:
                continue
            await async_db_obj.execute(
                f"UPDATE `{table_name}` SET `{nickname_column}`=%s WHERE `id`=%s",
                masked_nickname,
                row.get("id"),
            )
            updated_count += 1
        if updated_count:
            utils.logger.info(
                f"[ensure_privacy_schema_compatibility] masked {updated_count} {table_name}.{nickname_column} values"
            )


async def _scrub_legacy_privacy_columns(
    async_db_obj: AbstractAsyncDB, table_name: str, column_names: set[str]
):
    if "id" not in column_names:
        return

    for column_name in PRIVACY_USER_ID_SOURCE_COLUMNS:
        if column_name not in column_names:
            continue
        rows = await async_db_obj.query(
            f"""
            SELECT `id`, `{column_name}` AS privacy_value
            FROM `{table_name}`
            WHERE `{column_name}` IS NOT NULL
              AND `{column_name}` != ''
              AND `{column_name}` NOT LIKE 'redacted_%'
            """
        )
        updated_count = 0
        for row in rows:
            await async_db_obj.execute(
                f"UPDATE `{table_name}` SET `{column_name}`=%s WHERE `id`=%s",
                f"redacted_{row.get('id')}",
                row.get("id"),
            )
            updated_count += 1
        if updated_count:
            utils.logger.info(
                f"[ensure_privacy_schema_compatibility] scrubbed {updated_count} {table_name}.{column_name} values"
            )

    for column_name in set(LEGACY_PRIVACY_COLUMNS) - set(PRIVACY_USER_ID_SOURCE_COLUMNS):
        if column_name not in column_names:
            continue
        updated_count = await async_db_obj.execute(
            f"""
            UPDATE `{table_name}`
            SET `{column_name}`=%s
            WHERE `{column_name}` IS NOT NULL
              AND `{column_name}` != ''
            """,
            "",
        )
        if updated_count:
            utils.logger.info(
                f"[ensure_privacy_schema_compatibility] cleared {updated_count} {table_name}.{column_name} values"
            )


async def _ensure_sqlite_creator_tables_compatibility(async_db_obj: AbstractAsyncDB):
    for table_name in PRIVACY_CREATOR_TABLES:
        table_exists = await async_db_obj.get_first(
            "SELECT name FROM sqlite_master WHERE type='table' AND name=%s",
            table_name,
            )
        if table_exists:
            utils.logger.warning(
                f"[ensure_privacy_schema_compatibility] legacy creator table {table_name} "
                "is preserved; new privacy schema no longer creates creator profile tables"
            )
            column_names = await _get_sqlite_column_names(async_db_obj, table_name)
            await _ensure_privacy_table_compatibility(
                async_db_obj,
                table_name,
                column_names,
                creator_hash_type="TEXT DEFAULT ''",
            )


async def _ensure_mysql_creator_tables_compatibility(async_db_obj: AbstractAsyncDB):
    for table_name in PRIVACY_CREATOR_TABLES:
        table_exists = await async_db_obj.get_first(
            """
            SELECT TABLE_NAME AS table_name
            FROM information_schema.TABLES
            WHERE TABLE_SCHEMA = %s
              AND TABLE_NAME = %s
            """,
            config.RELATION_DB_NAME,
            table_name,
        )
        if table_exists:
            utils.logger.warning(
                f"[ensure_privacy_schema_compatibility] legacy creator table {table_name} "
                "is preserved; new privacy schema no longer creates creator profile tables"
            )
            column_names = await _get_mysql_column_names(async_db_obj, table_name)
            await _ensure_privacy_table_compatibility(
                async_db_obj,
                table_name,
                column_names,
                creator_hash_type="varchar(64) DEFAULT '' COMMENT '匿名创作者哈希'",
            )


async def ensure_hot_search_schema_compatibility():
    """
    Ensure the shared hot-search tables exist for existing databases.
    """
    async_db_obj: AbstractAsyncDB = media_crawler_db_var.get()
    if config.DB_TYPE == "sqlite":
        statements = [
            """
            CREATE TABLE IF NOT EXISTS `platform_hot_search_item`
            (
                `id`             INTEGER PRIMARY KEY AUTOINCREMENT,
                `hot_item_id`    TEXT    NOT NULL,
                `platform`       TEXT    NOT NULL,
                `board_name`     TEXT    NOT NULL DEFAULT 'hot_search',
                `board_type`     TEXT    NOT NULL DEFAULT '',
                `snapshot_date`  TEXT    NOT NULL,
                `snapshot_ts`    INTEGER NOT NULL,
                `rank_no`        INTEGER NOT NULL DEFAULT 0,
                `title`          TEXT    NOT NULL DEFAULT '',
                `keyword`        TEXT    NOT NULL DEFAULT '',
                `hot_value`      TEXT    NOT NULL DEFAULT '',
                `badge`          TEXT    NOT NULL DEFAULT '',
                `topic_id`       TEXT    NOT NULL DEFAULT '',
                `topic_url`      TEXT    NOT NULL DEFAULT '',
                `raw_json`       TEXT,
                `add_ts`         INTEGER NOT NULL,
                `last_modify_ts` INTEGER NOT NULL
            )
            """,
            "CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_item_id ON platform_hot_search_item (`hot_item_id`)",
            "CREATE INDEX IF NOT EXISTS idx_hot_search_item_board ON platform_hot_search_item (`platform`, `board_name`, `snapshot_date`, `rank_no`)",
            "CREATE INDEX IF NOT EXISTS idx_hot_search_item_title ON platform_hot_search_item (`platform`, `title`)",
            """
            CREATE TABLE IF NOT EXISTS `platform_hot_search_content_rel`
            (
                `id`             INTEGER PRIMARY KEY AUTOINCREMENT,
                `rel_id`         TEXT    NOT NULL,
                `hot_item_id`    TEXT    NOT NULL,
                `platform`       TEXT    NOT NULL,
                `content_id`     TEXT    NOT NULL,
                `content_type`   TEXT    NOT NULL DEFAULT '',
                `content_table`  TEXT    NOT NULL DEFAULT '',
                `rank_no`        INTEGER NOT NULL DEFAULT 0,
                `source_keyword` TEXT    NOT NULL DEFAULT '',
                `add_ts`         INTEGER NOT NULL,
                `last_modify_ts` INTEGER NOT NULL
            )
            """,
            "CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_rel_id ON platform_hot_search_content_rel (`rel_id`)",
            "CREATE INDEX IF NOT EXISTS idx_hot_search_rel_item ON platform_hot_search_content_rel (`hot_item_id`)",
            "CREATE INDEX IF NOT EXISTS idx_hot_search_rel_content ON platform_hot_search_content_rel (`platform`, `content_table`, `content_id`)",
        ]
    else:
        statements = [
            """
            CREATE TABLE IF NOT EXISTS `platform_hot_search_item`
            (
                `id`               bigint       NOT NULL AUTO_INCREMENT COMMENT '自增ID',
                `hot_item_id`      varchar(64)  NOT NULL COMMENT '热搜榜单项快照ID',
                `platform`         varchar(32)  NOT NULL COMMENT '平台名称',
                `board_name`       varchar(64)  NOT NULL DEFAULT 'hot_search' COMMENT '榜单名称',
                `board_type`       varchar(64)  NOT NULL DEFAULT '' COMMENT '榜单类型',
                `snapshot_date`    varchar(16)  NOT NULL COMMENT '采集日期',
                `snapshot_ts`      bigint       NOT NULL COMMENT '采集时间戳',
                `rank_no`          int          NOT NULL DEFAULT 0 COMMENT '榜单排名',
                `title`            varchar(500) NOT NULL DEFAULT '' COMMENT '热搜标题',
                `keyword`          varchar(500) NOT NULL DEFAULT '' COMMENT '检索关键词',
                `hot_value`        varchar(64)  NOT NULL DEFAULT '' COMMENT '热度值',
                `badge`            varchar(64)  NOT NULL DEFAULT '' COMMENT '热搜标签',
                `topic_id`         varchar(128) NOT NULL DEFAULT '' COMMENT '平台侧话题ID',
                `topic_url`        varchar(512) NOT NULL DEFAULT '' COMMENT '平台侧话题链接',
                `raw_json`         longtext COMMENT '平台原始榜单项JSON',
                `add_ts`           bigint       NOT NULL COMMENT '记录添加时间戳',
                `last_modify_ts`   bigint       NOT NULL COMMENT '记录最后修改时间戳',
                PRIMARY KEY (`id`),
                UNIQUE KEY `idx_hot_search_item_id` (`hot_item_id`),
                KEY `idx_hot_search_item_board` (`platform`, `board_name`, `snapshot_date`, `rank_no`),
                KEY `idx_hot_search_item_title` (`platform`, `title`(191))
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='跨平台热搜榜单项快照'
            """,
            """
            CREATE TABLE IF NOT EXISTS `platform_hot_search_content_rel`
            (
                `id`              bigint      NOT NULL AUTO_INCREMENT COMMENT '自增ID',
                `rel_id`          varchar(64) NOT NULL COMMENT '热搜内容关系ID',
                `hot_item_id`     varchar(64) NOT NULL COMMENT '热搜榜单项快照ID',
                `platform`        varchar(32) NOT NULL COMMENT '平台名称',
                `content_id`      varchar(128) NOT NULL COMMENT '内容ID',
                `content_type`    varchar(64) NOT NULL DEFAULT '' COMMENT '内容类型',
                `content_table`   varchar(64) NOT NULL DEFAULT '' COMMENT '内容表名',
                `rank_no`         int         NOT NULL DEFAULT 0 COMMENT '榜单排名',
                `source_keyword`  varchar(500) NOT NULL DEFAULT '' COMMENT '来源关键词',
                `add_ts`          bigint      NOT NULL COMMENT '记录添加时间戳',
                `last_modify_ts`  bigint      NOT NULL COMMENT '记录最后修改时间戳',
                PRIMARY KEY (`id`),
                UNIQUE KEY `idx_hot_search_rel_id` (`rel_id`),
                KEY `idx_hot_search_rel_item` (`hot_item_id`),
                KEY `idx_hot_search_rel_content` (`platform`, `content_table`, `content_id`)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='热搜榜单项与内容关系'
            """,
        ]

    for statement in statements:
        await async_db_obj.execute(statement)


async def _standalone_init_table_schema():
    """仅供命令行直接运行 db.py 时使用，先建连接再建表"""
    if config.DB_TYPE == "sqlite":
        await init_sqlite_db()
    else:
        await init_mediacrawler_db()
    await init_table_schema()
    await ensure_mysql_schema_compatibility()
    await ensure_hot_search_schema_compatibility()


if __name__ == "__main__":
    asyncio.get_event_loop().run_until_complete(_standalone_init_table_schema())
