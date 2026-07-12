# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/apis/accounts_handler.py
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

import time

from apis.base_handler import BaseAPIHandler
from constant.error_code import ApiCode
from logic.ws_bridge import ExtensionWebSocketHandler, client_registry, cookie_store


class AccountsHandler(BaseAPIHandler):
    async def get(self):
        all_clients = client_registry.get_all()
        connected_ids = ExtensionWebSocketHandler.get_connected_client_ids()

        accounts = []
        for cid, info in all_clients.items():
            accounts.append({
                "client_id": cid,
                "connected": cid in connected_ids,
                "nicknames": info.get("nicknames", {}),
                "platforms": info.get("platforms", {}),
                "connected_at": info.get("connected_at"),
                "last_seen": info.get("last_seen"),
            })

        self.return_ok(data={"accounts": accounts, "total": len(accounts)})


class AccountDetailHandler(BaseAPIHandler):
    async def get(self, client_id: str):
        info = client_registry.get(client_id)
        if not info:
            self.return_error(ApiCode.ACCOUNT_NOT_FOUND)
            return

        connected_ids = ExtensionWebSocketHandler.get_connected_client_ids()
        client_cookies = cookie_store.get_all_for_client(client_id)

        platforms_detail = {}
        for plat, data in client_cookies.items():
            platforms_detail[plat] = {
                "has_cookies": bool(data.get("cookies")),
                "source": data.get("source", "unknown"),
                "timestamp": data.get("timestamp"),
                "age_seconds": round(time.time() - data["timestamp"], 1) if data.get("timestamp") else None,
            }

        self.return_ok(data={
            "client_id": client_id,
            "connected": client_id in connected_ids,
            "nicknames": info.get("nicknames", {}),
            "platforms": info.get("platforms", {}),
            "cookies_detail": platforms_detail,
            "connected_at": info.get("connected_at"),
            "last_seen": info.get("last_seen"),
        })
