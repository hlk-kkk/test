# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/async_db.py
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
# @Time    : 2024/4/6 14:21
# @Desc    : 异步数据库增删改查封装（支持 MySQL 和 SQLite）
import asyncio
import hashlib
import json
from abc import ABC, abstractmethod
from typing import Any, Dict, List, Optional, Set, Union

import aiomysql  # type: ignore

from pkg.tools.user_privacy import (
    FORBIDDEN_USER_FIELDS,
    USER_ID_SOURCE_KEYS,
    sanitize_user_item,
)


LEGACY_PRIVACY_SOURCE_COLUMNS = set(USER_ID_SOURCE_KEYS)
LEGACY_PRIVACY_COLUMNS = FORBIDDEN_USER_FIELDS | LEGACY_PRIVACY_SOURCE_COLUMNS


def _legacy_privacy_placeholder(column_name: str, item: Dict[str, Any], table_name: str) -> str:
    if column_name in LEGACY_PRIVACY_SOURCE_COLUMNS:
        basis = item.get("creator_hash") or json.dumps(
            item, ensure_ascii=False, sort_keys=True, default=str
        )
        digest = hashlib.sha256(f"{table_name}:{basis}".encode("utf-8")).hexdigest()[:16]
        return f"redacted_{digest}"
    return ""


class AbstractAsyncDB(ABC):
    """数据库抽象基类，提供统一的 CRUD 接口"""

    @abstractmethod
    async def query(self, sql: str, *args: Union[str, int]) -> List[Dict[str, Any]]:
        """查询多条记录，返回列表"""
        pass

    @abstractmethod
    async def get_first(self, sql: str, *args: Union[str, int]) -> Optional[Dict[str, Any]]:
        """查询第一条记录"""
        pass

    @abstractmethod
    async def item_to_table(self, table_name: str, item: Dict[str, Any]) -> int:
        """插入一条记录，返回 lastrowid"""
        pass

    @abstractmethod
    async def update_table(self, table_name: str, updates: Dict[str, Any], field_where: str,
                           value_where: Union[str, int, float]) -> int:
        """更新指定记录，返回影响行数"""
        pass

    @abstractmethod
    async def execute(self, sql: str, *args: Union[str, int]) -> int:
        """执行写操作（DDL/DML），返回影响行数"""
        pass


class AsyncMysqlDB(AbstractAsyncDB):
    def __init__(self, pool: aiomysql.Pool) -> None:
        self.__pool = pool
        self._legacy_privacy_columns_cache: Dict[str, Set[str]] = {}

    async def query(self, sql: str, *args: Union[str, int]) -> List[Dict[str, Any]]:
        """
        从给定的 SQL 中查询记录，返回的是一个列表
        :param sql: 查询的sql
        :param args: sql中传递动态参数列表
        :return:
        """
        async with self.__pool.acquire() as conn:
            async with conn.cursor(aiomysql.DictCursor) as cur:
                await cur.execute(sql, args)
                data = await cur.fetchall()
                return data or []

    async def get_first(self, sql: str, *args: Union[str, int]) -> Optional[Dict[str, Any]]:
        """
        从给定的 SQL 中查询记录，返回的是符合条件的第一个结果
        :param sql: 查询的sql
        :param args:sql中传递动态参数列表
        :return:
        """
        async with self.__pool.acquire() as conn:
            async with conn.cursor(aiomysql.DictCursor) as cur:
                await cur.execute(sql, args)
                data = await cur.fetchone()
                if isinstance(data, dict):
                    return data
                return None

    async def item_to_table(self, table_name: str, item: Dict[str, Any]) -> int:
        """
        表中插入数据
        :param table_name: 表名
        :param item: 一条记录的字典信息
        :return:
        """
        item = sanitize_user_item(item)
        item = await self._fill_legacy_privacy_placeholders(table_name, item)
        fields = list(item.keys())
        values = list(item.values())
        fields = [f'`{field}`' for field in fields]
        fieldstr = ','.join(fields)
        valstr = ','.join(['%s'] * len(item))
        sql = "INSERT INTO %s (%s) VALUES(%s)" % (table_name, fieldstr, valstr)
        async with self.__pool.acquire() as conn:
            async with conn.cursor(aiomysql.DictCursor) as cur:
                await cur.execute(sql, values)
                lastrowid: int = cur.lastrowid
                return lastrowid

    async def update_table(self, table_name: str, updates: Dict[str, Any], field_where: str,
                           value_where: Union[str, int, float]) -> int:
        """
        更新指定表的记录
        :param table_name: 表名
        :param updates: 需要更新的字段和值的 key - value 映射
        :param field_where: update 语句 where 条件中的字段名
        :param value_where: update 语句 where 条件中的字段值
        :return:
        """
        updates = sanitize_user_item(updates)
        upsets = []
        values = []
        for k, v in updates.items():
            s = '`%s`=%%s' % k
            upsets.append(s)
            values.append(v)
        upsets_str = ','.join(upsets)
        sql = 'UPDATE %s SET %s WHERE %s="%s"' % (
            table_name,
            upsets_str,
            field_where, value_where,
        )
        async with self.__pool.acquire() as conn:
            async with conn.cursor() as cur:
                rows: int = await cur.execute(sql, values)
                return rows

    async def _fill_legacy_privacy_placeholders(
        self, table_name: str, item: Dict[str, Any]
    ) -> Dict[str, Any]:
        legacy_columns = await self._get_legacy_privacy_columns(table_name)
        if not legacy_columns:
            return item
        compatible_item = dict(item)
        for column_name in sorted(legacy_columns):
            compatible_item.setdefault(
                column_name,
                _legacy_privacy_placeholder(column_name, compatible_item, table_name),
            )
        return compatible_item

    async def _get_legacy_privacy_columns(self, table_name: str) -> Set[str]:
        if table_name not in self._legacy_privacy_columns_cache:
            async with self.__pool.acquire() as conn:
                async with conn.cursor(aiomysql.DictCursor) as cur:
                    await cur.execute(
                        """
                        SELECT COLUMN_NAME AS column_name
                        FROM information_schema.COLUMNS
                        WHERE TABLE_SCHEMA = DATABASE()
                          AND TABLE_NAME = %s
                        """,
                        (table_name,),
                    )
                    rows = await cur.fetchall() or []
            column_names = {row.get("column_name") for row in rows}
            self._legacy_privacy_columns_cache[table_name] = (
                column_names & LEGACY_PRIVACY_COLUMNS
            )
        return self._legacy_privacy_columns_cache[table_name]

    async def execute(self, sql: str, *args: Union[str, int]) -> int:
        """
        需要更新、写入等操作的 excute 执行语句
        :param sql:
        :param args:
        :return:
        """
        async with self.__pool.acquire() as conn:
            async with conn.cursor() as cur:
                rows: int = await cur.execute(sql, args)
                return rows


class AsyncSQLiteDB(AbstractAsyncDB):
    """
    SQLite 异步数据库实现，使用 aiosqlite。
    支持 WAL 模式以允许并发读操作，使用写锁序列化写操作。
    """

    def __init__(self, db_path: str) -> None:
        self._db_path = db_path
        self._conn: Optional[Any] = None  # aiosqlite.Connection
        self._write_lock = asyncio.Lock()
        self._legacy_privacy_columns_cache: Dict[str, Set[str]] = {}

    async def connect(self) -> None:
        """建立 SQLite 连接并开启 WAL 模式"""
        import aiosqlite
        self._conn = await aiosqlite.connect(self._db_path)
        await self._conn.execute("PRAGMA journal_mode=WAL")
        await self._conn.commit()

    async def close(self) -> None:
        """关闭 SQLite 连接"""
        if self._conn is not None:
            await self._conn.close()
            self._conn = None

    @staticmethod
    def _convert_params(sql: str, args: tuple) -> tuple:
        """将 MySQL 的 %s 占位符转换为 SQLite 的 ? 占位符"""
        return sql.replace('%s', '?'), args

    async def query(self, sql: str, *args: Union[str, int]) -> List[Dict[str, Any]]:
        """查询多条记录，返回字典列表"""
        sql, args = self._convert_params(sql, args)
        async with self._conn.execute(sql, args) as cursor:
            rows = await cursor.fetchall()
            if not rows:
                return []
            columns = [desc[0] for desc in cursor.description]
            return [dict(zip(columns, row)) for row in rows]

    async def get_first(self, sql: str, *args: Union[str, int]) -> Optional[Dict[str, Any]]:
        """查询第一条记录，返回字典或 None"""
        sql, args = self._convert_params(sql, args)
        async with self._conn.execute(sql, args) as cursor:
            row = await cursor.fetchone()
            if row is None:
                return None
            columns = [desc[0] for desc in cursor.description]
            return dict(zip(columns, row))

    async def item_to_table(self, table_name: str, item: Dict[str, Any]) -> int:
        """插入一条记录，返回 lastrowid"""
        item = sanitize_user_item(item)
        item = await self._fill_legacy_privacy_placeholders(table_name, item)
        fields = list(item.keys())
        values = list(item.values())
        fields = [f'`{field}`' for field in fields]
        fieldstr = ','.join(fields)
        valstr = ','.join(['?'] * len(item))
        sql = "INSERT INTO %s (%s) VALUES(%s)" % (table_name, fieldstr, valstr)
        async with self._write_lock:
            async with self._conn.execute(sql, values) as cursor:
                await self._conn.commit()
                return cursor.lastrowid

    async def _fill_legacy_privacy_placeholders(
        self, table_name: str, item: Dict[str, Any]
    ) -> Dict[str, Any]:
        legacy_columns = await self._get_legacy_privacy_columns(table_name)
        if not legacy_columns:
            return item
        compatible_item = dict(item)
        for column_name in sorted(legacy_columns):
            compatible_item.setdefault(
                column_name,
                _legacy_privacy_placeholder(column_name, compatible_item, table_name),
            )
        return compatible_item

    async def _get_legacy_privacy_columns(self, table_name: str) -> Set[str]:
        if table_name not in self._legacy_privacy_columns_cache:
            async with self._conn.execute(f"PRAGMA table_info(`{table_name}`)") as cursor:
                rows = await cursor.fetchall()
            column_names = {row[1] for row in rows}
            self._legacy_privacy_columns_cache[table_name] = (
                column_names & LEGACY_PRIVACY_COLUMNS
            )
        return self._legacy_privacy_columns_cache[table_name]

    async def update_table(self, table_name: str, updates: Dict[str, Any], field_where: str,
                           value_where: Union[str, int, float]) -> int:
        """更新指定记录，返回影响行数"""
        updates = sanitize_user_item(updates)
        upsets = []
        values = []
        for k, v in updates.items():
            s = '`%s`=?' % k
            upsets.append(s)
            values.append(v)
        upsets_str = ','.join(upsets)
        sql = 'UPDATE %s SET %s WHERE %s="%s"' % (
            table_name,
            upsets_str,
            field_where, value_where,
        )
        async with self._write_lock:
            async with self._conn.execute(sql, values) as cursor:
                await self._conn.commit()
                return cursor.rowcount

    async def execute(self, sql: str, *args: Union[str, int]) -> int:
        """
        执行写操作。
        - 有参数时：执行单条参数化 DML 语句
        - 无参数时：使用 executescript 执行多条 DDL（如建表 SQL 文件）
        """
        async with self._write_lock:
            if args:
                sql, args = self._convert_params(sql, args)
                async with self._conn.execute(sql, args) as cursor:
                    await self._conn.commit()
                    return cursor.rowcount
            else:
                # executescript 会自动提交前序事务，适合执行整个 schema 文件
                await self._conn.executescript(sql)
                return 0
