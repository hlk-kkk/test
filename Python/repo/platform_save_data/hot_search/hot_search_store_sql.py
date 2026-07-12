# -*- coding: utf-8 -*-

from typing import Dict, List

from async_db import AbstractAsyncDB
from var import media_crawler_db_var


async def query_hot_search_item_by_item_id(hot_item_id: str) -> Dict:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    rows: List[Dict] = await async_db_conn.query(
        "select * from platform_hot_search_item where hot_item_id = %s",
        hot_item_id,
    )
    if rows:
        return rows[0]
    return dict()


async def add_new_hot_search_item(hot_item: Dict) -> int:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    return await async_db_conn.item_to_table("platform_hot_search_item", hot_item)


async def update_hot_search_item_by_item_id(hot_item_id: str, hot_item: Dict) -> int:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    return await async_db_conn.update_table(
        "platform_hot_search_item",
        hot_item,
        "hot_item_id",
        hot_item_id,
    )


async def query_hot_search_content_relation_by_rel_id(rel_id: str) -> Dict:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    rows: List[Dict] = await async_db_conn.query(
        "select * from platform_hot_search_content_rel where rel_id = %s",
        rel_id,
    )
    if rows:
        return rows[0]
    return dict()


async def add_new_hot_search_content_relation(relation: Dict) -> int:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    return await async_db_conn.item_to_table("platform_hot_search_content_rel", relation)


async def update_hot_search_content_relation_by_rel_id(rel_id: str, relation: Dict) -> int:
    async_db_conn: AbstractAsyncDB = media_crawler_db_var.get()
    return await async_db_conn.update_table(
        "platform_hot_search_content_rel",
        relation,
        "rel_id",
        rel_id,
    )
