# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/repo/platform_save_data/bilibili/__init__.py
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
# @Time    : 2024/1/14 19:34
# @Desc    :

from typing import List

import config
from model.m_bilibili import BilibiliVideo, BilibiliComment, BilibiliUpInfo, CreatorQueryResponse
from var import source_keyword_var

from .bilibili_store_impl import *


class BiliStoreFactory:
    STORES = {
        "csv": BiliCsvStoreImplement,
        "db": BiliDbStoreImplement,
        "json": BiliJsonStoreImplement,
    }

    @staticmethod
    def create_store() -> AbstractStore:
        store_class = BiliStoreFactory.STORES.get(config.SAVE_DATA_OPTION)
        if not store_class:
            raise ValueError(
                "[BiliStoreFactory.create_store] Invalid save option only supported csv or db or json ..."
            )
        return store_class()


async def update_bilibili_creator(creator_info: CreatorQueryResponse):
    return


async def update_bilibili_up_info(up_info: BilibiliUpInfo):
    return


async def update_bilibili_video(video_item: BilibiliVideo):
    utils.logger.info(
        f"[store.bilibili.update_bilibili_video] bilibili bvid: {video_item.bvid}, title:{video_item.title[:100]}"
    )

    save_content_item = video_item.model_dump()
    save_content_item["last_modify_ts"] = utils.get_current_timestamp()
    await BiliStoreFactory.create_store().store_content(content_item=save_content_item)


async def update_up_info(up_info: BilibiliUpInfo):
    return


async def batch_update_bilibili_video_comments(video_id: str, comments: List[BilibiliComment]):
    if not comments:
        return
    for comment_item in comments:
        await update_bilibili_video_comment(comment_item)


async def update_bilibili_video_comment(comment_item: BilibiliComment):
    utils.logger.info(
        f"[store.bilibili.update_bilibili_video_comment] Bilibili video comment: {comment_item.comment_id}, content: {comment_item.content[:100]}"
    )

    save_comment_item = comment_item.model_dump()
    save_comment_item["last_modify_ts"] = utils.get_current_timestamp()
    await BiliStoreFactory.create_store().store_comment(comment_item=save_comment_item)
