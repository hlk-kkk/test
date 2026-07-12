# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/apis/cookies_handler.py
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

import json
import logging

import config
import constant
from apis.base_handler import TornadoBaseReqHandler
from constant.error_code import ApiCode
from logic.ws_bridge import ExtensionWebSocketHandler, cookie_store
from models.cookies_model import CookiesResponseData

logger = logging.getLogger(__name__)


class GetCookiesHandler(TornadoBaseReqHandler):
    async def get(self, platform: str):
        if platform not in constant.VALID_PLATFORMS:
            self.return_error(ApiCode.INVALID_PARAMETER)
            return

        client_id = self.get_argument("client_id", None)

        if client_id:
            # Specific client requested
            await self._get_for_client(platform, client_id)
        else:
            # Backward compat: any available client
            await self._get_any(platform)

    async def _get_for_client(self, platform: str, client_id: str):
        # 1. Check cache for this client
        if cookie_store.is_fresh(client_id, platform, ttl=config.CACHE_TTL):
            entry = cookie_store.get(client_id, platform)
            data = CookiesResponseData(
                platform=platform,
                cookies=entry["cookies"],
                source="cache",
                client_id=client_id,
            )
            self.return_ok(data=data)
            return

        # 2. Request from extension if connected
        if ExtensionWebSocketHandler.is_client_connected(client_id):
            result = await ExtensionWebSocketHandler.request_cookies(
                platform, client_id=client_id, timeout=config.WS_REQUEST_TIMEOUT
            )
            if result and result.get("cookies"):
                data = CookiesResponseData(
                    platform=platform,
                    cookies=result["cookies"],
                    source="extension",
                    client_id=client_id,
                )
                self.return_ok(data=data)
                return

        # 3. Stale cache
        stale_entry = cookie_store.get(client_id, platform)
        if stale_entry and stale_entry.get("cookies"):
            data = CookiesResponseData(
                platform=platform,
                cookies=stale_entry["cookies"],
                source="stale_cache",
                client_id=client_id,
            )
            self.return_ok(data=data)
            return

        # 4. Nothing available
        if not ExtensionWebSocketHandler.is_client_connected(client_id):
            self.return_error(ApiCode.EXTENSION_NOT_CONNECTED)
        else:
            self.return_error(ApiCode.COOKIES_NOT_FOUND)

    async def _get_any(self, platform: str):
        # 1. Check fresh cache from any client
        cid, entry = cookie_store.get_fresh_for_platform(platform, ttl=config.CACHE_TTL)
        if cid and entry:
            data = CookiesResponseData(
                platform=platform,
                cookies=entry["cookies"],
                source="cache",
                client_id=cid,
            )
            self.return_ok(data=data)
            return

        # 2. Request from any connected extension
        if ExtensionWebSocketHandler.is_connected():
            result = await ExtensionWebSocketHandler.request_cookies(
                platform, timeout=config.WS_REQUEST_TIMEOUT
            )
            if result and result.get("cookies"):
                data = CookiesResponseData(
                    platform=platform,
                    cookies=result["cookies"],
                    source="extension",
                    client_id=result.get("_client_id"),
                )
                self.return_ok(data=data)
                return

        # 3. Stale cache from any client
        cid, stale_entry = cookie_store.get_any_for_platform(platform)
        if cid and stale_entry and stale_entry.get("cookies"):
            data = CookiesResponseData(
                platform=platform,
                cookies=stale_entry["cookies"],
                source="stale_cache",
                client_id=cid,
            )
            self.return_ok(data=data)
            return

        # 4. Nothing
        if not ExtensionWebSocketHandler.is_connected():
            self.return_error(ApiCode.EXTENSION_NOT_CONNECTED)
        else:
            self.return_error(ApiCode.COOKIES_NOT_FOUND)


class GetAllCookiesHandler(TornadoBaseReqHandler):
    async def get(self):
        client_id = self.get_argument("client_id", None)
        platforms_data = {}

        if client_id:
            # Specific client
            if ExtensionWebSocketHandler.is_client_connected(client_id):
                result = await ExtensionWebSocketHandler.request_all_cookies(
                    client_id=client_id, timeout=config.WS_REQUEST_TIMEOUT + 5
                )
                if result and result.get("type") == "cookies_response_complete":
                    for plat, cookies in result.get("platforms", {}).items():
                        if plat in constant.VALID_PLATFORMS:
                            cookie_str = cookies if isinstance(cookies, str) else ""
                            platforms_data[plat] = CookiesResponseData(
                                platform=plat,
                                cookies=cookie_str,
                                source="extension",
                                client_id=client_id,
                            ).model_dump()

            # Fill from cache for this client
            for plat in constant.VALID_PLATFORMS:
                if plat not in platforms_data:
                    entry = cookie_store.get(client_id, plat)
                    if entry and entry.get("cookies"):
                        source = "cache" if cookie_store.is_fresh(client_id, plat, ttl=config.CACHE_TTL) else "stale_cache"
                        platforms_data[plat] = CookiesResponseData(
                            platform=plat,
                            cookies=entry["cookies"],
                            source=source,
                            client_id=client_id,
                        ).model_dump()
        else:
            # Backward compat: any connected client
            if ExtensionWebSocketHandler.is_connected():
                result = await ExtensionWebSocketHandler.request_all_cookies(
                    timeout=config.WS_REQUEST_TIMEOUT + 5
                )
                if result and result.get("type") == "cookies_response_complete":
                    for plat, cookies in result.get("platforms", {}).items():
                        if plat in constant.VALID_PLATFORMS:
                            cookie_str = cookies if isinstance(cookies, str) else ""
                            platforms_data[plat] = CookiesResponseData(
                                platform=plat,
                                cookies=cookie_str,
                                source="extension",
                            ).model_dump()

            # Fill from any client's cache
            for plat in constant.VALID_PLATFORMS:
                if plat not in platforms_data:
                    cid, entry = cookie_store.get_any_for_platform(plat)
                    if cid and entry and entry.get("cookies"):
                        source = "cache" if cookie_store.is_fresh(cid, plat, ttl=config.CACHE_TTL) else "stale_cache"
                        platforms_data[plat] = CookiesResponseData(
                            platform=plat,
                            cookies=entry["cookies"],
                            source=source,
                            client_id=cid,
                        ).model_dump()

        self.return_ok(data={"platforms": platforms_data})


class SetCookiesHandler(TornadoBaseReqHandler):
    async def post(self, platform: str):
        if platform not in constant.VALID_PLATFORMS:
            self.return_error(ApiCode.INVALID_PARAMETER)
            return

        cookies = self.request_data.get("cookies")
        if not cookies:
            self.return_error(ApiCode.MISSING_PARAMETER)
            return

        client_id = self.request_data.get("client_id", "manual")
        if len(client_id) > 64 or not client_id.replace("-", "").replace("_", "").isalnum():
            self.return_error(ApiCode.INVALID_PARAMETER)
            return
        cookie_store.set(client_id, platform, cookies, source="manual")
        logger.info("Cookies manually set for client=%s platform=%s", client_id, platform)
        self.return_ok(data={"platform": platform, "client_id": client_id, "source": "manual"})
