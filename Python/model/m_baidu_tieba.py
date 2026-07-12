# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-Python project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python/blob/main/model/m_baidu_tieba.py
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
from typing import Optional

from pydantic import Field

from pkg.tools.user_privacy import PrivacyBaseModel


class TiebaNote(PrivacyBaseModel):
    """
    百度贴吧帖子
    """
    note_id: str = Field(..., description="帖子ID")
    title: str = Field(..., description="帖子标题")
    desc: str = Field(default="", description="帖子描述")
    note_url: str = Field(..., description="帖子链接")
    publish_time: str = Field(default="", description="发布时间")
    creator_hash: str = Field(default="", description="匿名创作者哈希")
    user_nickname: str = Field(default="", description="脱敏用户昵称")
    tieba_name: str = Field(..., description="贴吧名称")
    tieba_link: str = Field(..., description="贴吧链接")
    total_replay_num: int = Field(default=0, description="回复总数")
    total_replay_page: int = Field(default=0, description="回复总页数")
    source_keyword: str = Field(default="", description="来源关键词")


class TiebaComment(PrivacyBaseModel):
    """
    百度贴吧评论
    """

    comment_id: str = Field(..., description="评论ID")
    parent_comment_id: str = Field(default="", description="父评论ID")
    content: str = Field(..., description="评论内容")
    creator_hash: str = Field(default="", description="匿名创作者哈希")
    user_nickname: str = Field(default="", description="脱敏用户昵称")
    publish_time: str = Field(default="", description="发布时间")
    sub_comment_count: int = Field(default=0, description="子评论数")
    note_id: str = Field(..., description="帖子ID")
    note_url: str = Field(..., description="帖子链接")
    tieba_id: str = Field(..., description="所属的贴吧ID")
    tieba_name: str = Field(..., description="所属的贴吧名称")
    tieba_link: str = Field(..., description="贴吧链接")




class TiebaCreator(PrivacyBaseModel):
    """
    百度贴吧创作者
    """
    creator_hash: str = Field(default="", description="匿名创作者哈希")
    nickname: str = Field(default="", description="脱敏用户昵称")
    follows: int = Field(default=0, description="关注数")
    fans: int = Field(default=0, description="粉丝数")
    registration_duration: str = Field(default="", description="注册时长")
