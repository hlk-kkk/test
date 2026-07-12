# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/router.py
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

from apis.accounts_handler import AccountDetailHandler, AccountsHandler
from apis.base_handler import PongHandler
from apis.cookies_handler import (GetAllCookiesHandler, GetCookiesHandler,
                                  SetCookiesHandler)
from apis.status_handler import StatusHandler
from logic.ws_bridge import ExtensionWebSocketHandler

all_router = [
    (r"/", PongHandler),
    (r"/ping", PongHandler),
    (r"/api/cookies/all", GetAllCookiesHandler),
    (r"/api/cookies/([a-z]+)", GetCookiesHandler),
    (r"/api/cookies/([a-z]+)/set", SetCookiesHandler),
    (r"/api/accounts", AccountsHandler),
    (r"/api/accounts/([a-zA-Z0-9_-]+)", AccountDetailHandler),
    (r"/api/status", StatusHandler),
    (r"/ws", ExtensionWebSocketHandler),
]
