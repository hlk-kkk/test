# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/constant/base_constant.py
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

XHS_PLATFORM_NAME = "xhs"
DY_PLATFORM_NAME = "dy"
KS_PLATFORM_NAME = "ks"
BILI_PLATFORM_NAME = "bili"
WB_PLATFORM_NAME = "wb"
TIEBA_PLATFORM_NAME = "tieba"
ZHIHU_PLATFORM_NAME = "zhihu"

VALID_PLATFORMS = [
    XHS_PLATFORM_NAME,
    DY_PLATFORM_NAME,
    KS_PLATFORM_NAME,
    BILI_PLATFORM_NAME,
    WB_PLATFORM_NAME,
    TIEBA_PLATFORM_NAME,
    ZHIHU_PLATFORM_NAME,
]

REQUEST_ID_HEADERS_KEY = "x-request-id"
RESPONSE_TYPE_JSON_HEADERS_KEY = "x-response-type"
RESPONSE_TYPE_JSON_HEADERS_VALUE = "json"

PLATFORM_DOMAINS = {
    XHS_PLATFORM_NAME: [".xiaohongshu.com"],
    DY_PLATFORM_NAME: [".douyin.com"],
    KS_PLATFORM_NAME: [".kuaishou.com"],
    BILI_PLATFORM_NAME: [".bilibili.com"],
    WB_PLATFORM_NAME: [".weibo.com", ".weibo.cn"],
    TIEBA_PLATFORM_NAME: [".tieba.baidu.com"],
    ZHIHU_PLATFORM_NAME: [".zhihu.com"],
}
