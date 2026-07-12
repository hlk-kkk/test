# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/apis/status_handler.py
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

import config
import constant
from apis.base_handler import BaseAPIHandler
from logic.ws_bridge import ExtensionWebSocketHandler, client_registry, cookie_store


class StatusHandler(BaseAPIHandler):
    async def get(self):
        all_clients = client_registry.get_all()
        connected_ids = ExtensionWebSocketHandler.get_connected_client_ids()

        clients = {}
        for cid, info in all_clients.items():
            client_platforms = {}
            for plat in constant.VALID_PLATFORMS:
                entry = cookie_store.get(cid, plat)
                if entry:
                    client_platforms[plat] = {
                        "has_cookies": bool(entry.get("cookies")),
                        "source": entry.get("source", "unknown"),
                        "fresh": cookie_store.is_fresh(cid, plat, ttl=config.CACHE_TTL),
                        "age_seconds": round(time.time() - entry["timestamp"], 1) if entry.get("timestamp") else None,
                    }
                else:
                    client_platforms[plat] = None

            clients[cid] = {
                "connected": cid in connected_ids,
                "nicknames": info.get("nicknames", {}),
                "last_seen": info.get("last_seen"),
                "platforms": client_platforms,
            }

        self.return_ok(data={
            "extension_connected": ExtensionWebSocketHandler.is_connected(),
            "connected_count": len(connected_ids),
            "total_clients": len(all_clients),
            "clients": clients,
        })
