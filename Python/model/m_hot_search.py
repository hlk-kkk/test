# -*- coding: utf-8 -*-

from typing import Optional

from pydantic import BaseModel, Field


class HotSearchItem(BaseModel):
    """Cross-platform hot-search board item snapshot."""

    hot_item_id: str = Field(default="", description="热搜榜单项快照ID")
    platform: str = Field(default="", description="平台名称")
    board_name: str = Field(default="hot_search", description="榜单名称")
    board_type: str = Field(default="", description="榜单类型")
    snapshot_date: str = Field(default="", description="采集日期")
    snapshot_ts: int = Field(default=0, description="采集时间戳")
    rank_no: int = Field(default=0, description="榜单排名")
    title: str = Field(default="", description="热搜标题")
    keyword: str = Field(default="", description="检索关键词")
    hot_value: str = Field(default="", description="热度值")
    badge: str = Field(default="", description="热搜标签")
    topic_id: str = Field(default="", description="平台侧话题ID")
    topic_url: str = Field(default="", description="平台侧话题链接")
    raw_json: str = Field(default="", description="平台原始榜单项JSON")
    add_ts: Optional[int] = Field(default=None, description="记录添加时间戳")
    last_modify_ts: Optional[int] = Field(default=None, description="记录最后修改时间戳")


class HotSearchContentRelation(BaseModel):
    """Relation from a hot-search board item to a crawled content item."""

    rel_id: str = Field(default="", description="热搜内容关系ID")
    hot_item_id: str = Field(default="", description="热搜榜单项快照ID")
    platform: str = Field(default="", description="平台名称")
    content_id: str = Field(default="", description="内容ID")
    content_type: str = Field(default="", description="内容类型")
    content_table: str = Field(default="", description="内容表名")
    rank_no: int = Field(default=0, description="榜单排名")
    source_keyword: str = Field(default="", description="来源关键词")
    add_ts: Optional[int] = Field(default=None, description="记录添加时间戳")
    last_modify_ts: Optional[int] = Field(default=None, description="记录最后修改时间戳")
