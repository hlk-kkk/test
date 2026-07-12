# -*- coding: utf-8 -*-

import hashlib
import json
from typing import Dict, Iterable, Optional
from urllib.parse import quote

import config
from model.m_hot_search import HotSearchContentRelation, HotSearchItem
from pkg.tools import utils
from pkg.tools.user_privacy import sanitize_raw_payload
from var import media_crawler_db_var

from .hot_search_store_sql import (
    add_new_hot_search_content_relation,
    add_new_hot_search_item,
    query_hot_search_content_relation_by_rel_id,
    query_hot_search_item_by_item_id,
    update_hot_search_content_relation_by_rel_id,
    update_hot_search_item_by_item_id,
)


def _make_id(*parts: object) -> str:
    raw = "|".join(str(part or "") for part in parts)
    return hashlib.md5(raw.encode("utf-8")).hexdigest()


def _json_dumps(raw_item: Dict) -> str:
    return json.dumps(sanitize_raw_payload(raw_item or {}), ensure_ascii=False, sort_keys=True)


def build_hot_search_item(
    platform: str,
    title: str,
    rank_no: int,
    raw_item: Dict,
    *,
    board_name: str = "hot_search",
    board_type: str = "",
    keyword: str = "",
    hot_value: object = "",
    badge: str = "",
    topic_id: str = "",
    topic_url: str = "",
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    current_ts = snapshot_ts or utils.get_current_timestamp()
    snapshot_date = utils.get_date_str_from_unix_time(current_ts)
    normalized_title = str(title or "").strip()
    normalized_keyword = str(keyword or normalized_title).strip()
    hot_item_id = _make_id(
        platform,
        board_name,
        snapshot_date,
        current_ts,
        rank_no,
        normalized_title,
    )
    return HotSearchItem(
        hot_item_id=hot_item_id,
        platform=platform,
        board_name=board_name,
        board_type=board_type,
        snapshot_date=snapshot_date,
        snapshot_ts=current_ts,
        rank_no=int(rank_no or 0),
        title=normalized_title,
        keyword=normalized_keyword,
        hot_value=str(hot_value or ""),
        badge=str(badge or ""),
        topic_id=str(topic_id or ""),
        topic_url=str(topic_url or ""),
        raw_json=_json_dumps(raw_item),
    )


def build_weibo_hot_search_item(
    raw_item: Dict,
    *,
    rank_no: Optional[int] = None,
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    title = str(raw_item.get("word") or raw_item.get("note") or "").strip()
    word_scheme = str(raw_item.get("word_scheme") or title)
    return build_hot_search_item(
        platform="wb",
        board_name="hot_search",
        board_type="topic",
        title=title,
        rank_no=rank_no or raw_item.get("realpos") or raw_item.get("rank") or 0,
        keyword=title,
        hot_value=raw_item.get("num") or raw_item.get("raw_hot") or "",
        badge=raw_item.get("label_name") or raw_item.get("icon_desc") or "",
        topic_id=raw_item.get("mid") or raw_item.get("word_scheme") or "",
        topic_url=f"https://s.weibo.com/weibo?q={quote(word_scheme)}",
        raw_item=raw_item,
        snapshot_ts=snapshot_ts,
    )


def build_douyin_hot_search_item(
    raw_item: Dict,
    *,
    rank_no: Optional[int] = None,
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    title = str(raw_item.get("word") or "").strip()
    return build_hot_search_item(
        platform="dy",
        board_name="hot_search",
        board_type="topic",
        title=title,
        rank_no=rank_no or raw_item.get("position") or 0,
        keyword=title,
        hot_value=raw_item.get("hot_value") or raw_item.get("sentence_id") or "",
        badge=raw_item.get("label") or raw_item.get("tag") or "",
        topic_id=raw_item.get("group_id") or raw_item.get("sentence_id") or "",
        topic_url=f"https://www.douyin.com/search/{quote(title)}",
        raw_item=raw_item,
        snapshot_ts=snapshot_ts,
    )


def build_zhihu_hot_search_item(
    raw_item: Dict,
    *,
    rank_no: int,
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    question = raw_item.get("question") or {}
    question_id = str(question.get("id") or "")
    title = str(question.get("title") or "").strip()
    return build_hot_search_item(
        platform="zhihu",
        board_name="hot_search",
        board_type="question",
        title=title,
        rank_no=rank_no,
        keyword=title,
        hot_value=raw_item.get("score") or raw_item.get("heat") or "",
        badge=raw_item.get("label") or "",
        topic_id=question_id,
        topic_url=f"https://www.zhihu.com/question/{question_id}" if question_id else "",
        raw_item=raw_item,
        snapshot_ts=snapshot_ts,
    )


def build_bilibili_hot_search_item(
    raw_item: Dict,
    *,
    rank_no: int,
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    bvid = str(raw_item.get("bvid") or "")
    aid = str(raw_item.get("aid") or "")
    title = str(raw_item.get("title") or "").strip()
    return build_hot_search_item(
        platform="bili",
        board_name="popular",
        board_type="video",
        title=title,
        rank_no=rank_no,
        keyword="bilibili_hot_search",
        hot_value=raw_item.get("stat", {}).get("view") or raw_item.get("video_play_count") or "",
        badge="",
        topic_id=bvid or aid,
        topic_url=f"https://www.bilibili.com/video/{bvid}" if bvid else "",
        raw_item=raw_item,
        snapshot_ts=snapshot_ts,
    )


def build_kuaishou_hot_search_item(
    raw_item: Dict,
    *,
    rank_no: int,
    snapshot_ts: Optional[int] = None,
) -> HotSearchItem:
    photo = raw_item.get("photo") or {}
    video_id = str(photo.get("id") or "")
    title = str(
        photo.get("caption")
        or photo.get("title")
        or raw_item.get("caption")
        or "kuaishou_hot_search"
    ).strip()
    return build_hot_search_item(
        platform="ks",
        board_name="hot_feed",
        board_type="video",
        title=title,
        rank_no=rank_no,
        keyword="kuaishou_hot_search",
        hot_value=photo.get("realLikeCount") or photo.get("viewCount") or "",
        badge="",
        topic_id=video_id,
        topic_url=f"https://www.kuaishou.com/short-video/{video_id}" if video_id else "",
        raw_item=raw_item,
        snapshot_ts=snapshot_ts,
    )


def build_hot_search_content_relation(
    hot_item: HotSearchItem,
    content_id: str,
    content_type: str,
    content_table: str,
    *,
    source_keyword: str = "",
) -> HotSearchContentRelation:
    rel_id = _make_id(
        hot_item.hot_item_id,
        hot_item.platform,
        content_table,
        content_id,
        content_type,
    )
    return HotSearchContentRelation(
        rel_id=rel_id,
        hot_item_id=hot_item.hot_item_id,
        platform=hot_item.platform,
        content_id=str(content_id or ""),
        content_type=content_type,
        content_table=content_table,
        rank_no=hot_item.rank_no,
        source_keyword=source_keyword or hot_item.keyword or hot_item.title,
    )


async def update_hot_search_item(hot_item: HotSearchItem) -> str:
    if config.SAVE_DATA_OPTION != "db" or media_crawler_db_var.get(None) is None:
        return hot_item.hot_item_id

    save_item = hot_item.model_dump()
    save_item["last_modify_ts"] = utils.get_current_timestamp()
    existing_item = await query_hot_search_item_by_item_id(hot_item.hot_item_id)
    if not existing_item:
        save_item["add_ts"] = utils.get_current_timestamp()
        await add_new_hot_search_item(save_item)
    else:
        save_item.pop("add_ts", None)
        await update_hot_search_item_by_item_id(hot_item.hot_item_id, save_item)
    return hot_item.hot_item_id


async def update_hot_search_content_relation(
    relation: HotSearchContentRelation,
) -> str:
    if (
        config.SAVE_DATA_OPTION != "db"
        or media_crawler_db_var.get(None) is None
        or not relation.content_id
    ):
        return relation.rel_id

    save_item = relation.model_dump()
    save_item["last_modify_ts"] = utils.get_current_timestamp()
    existing_item = await query_hot_search_content_relation_by_rel_id(relation.rel_id)
    if not existing_item:
        save_item["add_ts"] = utils.get_current_timestamp()
        await add_new_hot_search_content_relation(save_item)
    else:
        save_item.pop("add_ts", None)
        await update_hot_search_content_relation_by_rel_id(relation.rel_id, save_item)
    return relation.rel_id


async def batch_update_hot_search_content_relations(
    hot_item: HotSearchItem,
    content_ids: Iterable[str],
    content_type: str,
    content_table: str,
    *,
    source_keyword: str = "",
) -> None:
    for content_id in content_ids:
        relation = build_hot_search_content_relation(
            hot_item=hot_item,
            content_id=content_id,
            content_type=content_type,
            content_table=content_table,
            source_keyword=source_keyword,
        )
        await update_hot_search_content_relation(relation)
