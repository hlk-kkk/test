# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/cmd_arg/arg.py
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

import sys
import re
from typing import Optional
from enum import Enum

import typer
from typing_extensions import Annotated

import config
import constant


def _normalize_tieba_note_id(value: str) -> str:
    """Accept a raw Tieba thread id or a /p/<id> URL."""
    value = value.strip()
    match = re.search(r"/p/(\d+)", value)
    return match.group(1) if match else value


def _normalize_tieba_creator_url(value: str) -> str:
    """Accept a Tieba creator homepage URL or a portrait id."""
    value = value.strip()
    if value.startswith("http://") or value.startswith("https://"):
        return value
    return f"https://tieba.baidu.com/home/main?id={value}"


class PlatformEnum(str, Enum):
    """支持的媒体平台枚举"""
    XHS = constant.XHS_PLATFORM_NAME
    DOUYIN = constant.DOUYIN_PLATFORM_NAME
    KUAISHOU = constant.KUAISHOU_PLATFORM_NAME
    WEIBO = constant.WEIBO_PLATFORM_NAME
    BILIBILI = constant.BILIBILI_PLATFORM_NAME
    TIEBA = constant.TIEBA_PLATFORM_NAME
    ZHIHU = constant.ZHIHU_PLATFORM_NAME


class CrawlerTypeEnum(str, Enum):
    """爬虫类型枚举"""
    SEARCH = constant.CRALER_TYPE_SEARCH
    DETAIL = constant.CRALER_TYPE_DETAIL
    CREATOR = constant.CRALER_TYPE_CREATOR
    HOMEFEED = constant.CRALER_TYPE_HOMEFEED
    HOT_SEARCH = constant.CRALER_TYPE_HOT_SEARCH


class SaveDataOptionEnum(str, Enum):
    """数据保存选项枚举"""
    CSV = "csv"
    DB = "db"
    JSON = "json"


def parse_cmd():
    """
    解析命令行参数并更新配置

    这个函数保持与原有 argparse 版本的完全兼容性，
    同时提供更好的用户体验和错误处理。
    """
    def main(
        platform: Annotated[
            PlatformEnum,
            typer.Option(
                "--platform",
                help="🎯 选择媒体平台 (xhs=小红书, dy=抖音, ks=快手, bili=B站, wb=微博, tieba=贴吧, zhihu=知乎)"
            )
        ] = config.PLATFORM,

        crawler_type: Annotated[
            CrawlerTypeEnum,
            typer.Option(
                "--type",
                help="🔍 爬虫类型 (search=关键词搜索, detail=帖子详情, creator=创作者主页, homefeed=首页推荐, hot_search=热搜列表)"
            )
        ] = config.CRAWLER_TYPE,

        enable_checkpoint: Annotated[
            bool,
            typer.Option(
                "--enable_checkpoint/--no-enable_checkpoint",
                help="💾 是否启用断点续爬功能"
            )
        ] = config.ENABLE_CHECKPOINT,

        checkpoint_id: Annotated[
            str,
            typer.Option(
                "--checkpoint_id",
                help="🔖 指定断点续爬的检查点ID，如果为空则加载最新的检查点"
            )
        ] = config.SPECIFIED_CHECKPOINT_ID,

        keywords: Annotated[
            str,
            typer.Option(
                "--keywords",
                help="🔤 搜索关键词，多个关键词用逗号分隔"
            )
        ] = config.KEYWORDS,

        urls: Annotated[
            str,
            typer.Option(
                "--urls",
                help="🔗 detail/creator模式的URL或ID列表，多个用逗号分隔"
            )
        ] = "",

        enable_comments: Annotated[
            bool,
            typer.Option(
                "--enable_comments/--no-enable_comments",
                help="💬 是否爬取评论（默认开启，不需要评论时用 --no-enable_comments 可加速爬取）"
            )
        ] = config.ENABLE_GET_COMMENTS,

        enable_sub_comments: Annotated[
            bool,
            typer.Option(
                "--enable_sub_comments/--no-enable_sub_comments",
                help="💬 是否爬取二级评论（默认关闭）"
            )
        ] = config.ENABLE_GET_SUB_COMMENTS,

        hot_search_max_items: Annotated[
            int,
            typer.Option(
                "--hot_search_max_items",
                help="🔥 hot_search模式最多处理多少个热搜榜单项"
            )
        ] = config.HOT_SEARCH_MAX_ITEMS_COUNT,

        hot_search_per_item_count: Annotated[
            int,
            typer.Option(
                "--hot_search_per_item_count",
                help="🔥 hot_search模式每个热搜项最多抓取多少条内容"
            )
        ] = config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT,

    ):
        """
        🚀 MediaCrawlerPro - 多平台媒体爬虫工具

        支持小红书、抖音、快手、B站、微博、贴吧、知乎等平台的数据爬取。

        [bold green]示例用法:[/bold green]

        • 爬取小红书搜索结果：
          python main.py --platform xhs --type search --keywords "深度学习,AI"

        • 爬取小红书笔记详情（通过CLI传入URL）：
          python main.py --platform xhs --type detail --urls "https://www.xiaohongshu.com/explore/xxx?xsec_token=xxx"

        • 爬取创作者主页：
          python main.py --platform dy --type creator --urls "sec_uid1,sec_uid2"

        • 启用断点续爬：
          python main.py --platform dy --type creator --enable_checkpoint

        """
        # 更新全局配置，保持与原有逻辑的兼容性
        config.PLATFORM = platform.value
        config.CRAWLER_TYPE = crawler_type.value
        config.KEYWORDS = keywords
        config.ENABLE_CHECKPOINT = enable_checkpoint
        config.SPECIFIED_CHECKPOINT_ID = checkpoint_id
        config.ENABLE_GET_COMMENTS = enable_comments
        config.ENABLE_GET_SUB_COMMENTS = enable_sub_comments
        config.HOT_SEARCH_MAX_ITEMS_COUNT = hot_search_max_items
        config.HOT_SEARCH_PER_ITEM_CONTENT_COUNT = hot_search_per_item_count

        # 如果通过CLI传入了urls，覆盖对应平台的配置
        if urls:
            url_list = [u.strip() for u in urls.split(",") if u.strip()]
            _override_urls_config(platform.value, crawler_type.value, url_list)


    # 检查是否是帮助命令
    import sys
    if '--help' in sys.argv or '-h' in sys.argv:
        # 如果是帮助命令，直接运行 typer 并退出
        typer.run(main)
        return

    # 使用 typer.run 但捕获 SystemExit 以避免程序提前退出
    try:
        typer.run(main)
    except SystemExit as e:
        # 如果是参数错误导致的退出，重新抛出
        if e.code != 0:
            raise
        # 如果是正常的参数解析完成，继续执行后续代码
        pass


def _override_urls_config(platform: str, crawler_type: str, url_list: list):
    """根据平台和爬虫类型，将CLI传入的URL/ID列表覆盖到对应的config变量"""
    if platform == constant.TIEBA_PLATFORM_NAME:
        if crawler_type == constant.CRALER_TYPE_DETAIL:
            url_list = [_normalize_tieba_note_id(item) for item in url_list]
        elif crawler_type == constant.CRALER_TYPE_CREATOR:
            url_list = [_normalize_tieba_creator_url(item) for item in url_list]

    detail_mapping = {
        constant.XHS_PLATFORM_NAME: "XHS_SPECIFIED_NOTE_URL_LIST",
        constant.DOUYIN_PLATFORM_NAME: "DY_SPECIFIED_ID_LIST",
        constant.KUAISHOU_PLATFORM_NAME: "KS_SPECIFIED_ID_LIST",
        constant.BILIBILI_PLATFORM_NAME: "BILI_SPECIFIED_ID_LIST",
        constant.WEIBO_PLATFORM_NAME: "WEIBO_SPECIFIED_ID_LIST",
        constant.TIEBA_PLATFORM_NAME: "TIEBA_SPECIFIED_ID_LIST",
        constant.ZHIHU_PLATFORM_NAME: "ZHIHU_SPECIFIED_ID_LIST",
    }
    creator_mapping = {
        constant.XHS_PLATFORM_NAME: "XHS_CREATOR_URL_LIST",
        constant.DOUYIN_PLATFORM_NAME: "DY_CREATOR_ID_LIST",
        constant.KUAISHOU_PLATFORM_NAME: "KS_CREATOR_ID_LIST",
        constant.BILIBILI_PLATFORM_NAME: "BILI_CREATOR_ID_LIST",
        constant.WEIBO_PLATFORM_NAME: "WEIBO_CREATOR_ID_LIST",
        constant.TIEBA_PLATFORM_NAME: "TIEBA_CREATOR_URL_LIST",
        constant.ZHIHU_PLATFORM_NAME: "ZHIHU_CREATOR_URL_LIST",
    }

    if crawler_type == constant.CRALER_TYPE_DETAIL:
        attr_name = detail_mapping.get(platform)
    elif crawler_type == constant.CRALER_TYPE_CREATOR:
        attr_name = creator_mapping.get(platform)
    else:
        return

    if attr_name and hasattr(config, attr_name):
        setattr(config, attr_name, url_list)
