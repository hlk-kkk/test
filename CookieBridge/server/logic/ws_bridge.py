# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/server/logic/ws_bridge.py
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

import asyncio
import json
import logging
import time
import uuid

import tornado.websocket

logger = logging.getLogger(__name__)


class CookieStore:
    """In-memory cookie cache with (client_id, platform) two-level keys"""

    def __init__(self):
        # (client_id, platform) -> {"cookies": str, "timestamp": float, "source": str, "valid": bool}
        self._store: dict[tuple[str, str], dict] = {}

    def get(self, client_id: str, platform: str) -> dict | None:
        return self._store.get((client_id, platform))

    def set(self, client_id: str, platform: str, cookies: str, source: str = "extension"):
        self._store[(client_id, platform)] = {
            "cookies": cookies,
            "timestamp": time.time(),
            "source": source,
            "valid": bool(cookies),
        }

    def is_fresh(self, client_id: str, platform: str, ttl: int = 300) -> bool:
        entry = self._store.get((client_id, platform))
        if not entry:
            return False
        return (time.time() - entry["timestamp"]) < ttl

    def get_all_for_client(self, client_id: str) -> dict:
        """Return platform -> data for a specific client"""
        result = {}
        for (cid, plat), data in self._store.items():
            if cid == client_id:
                result[plat] = data
        return result

    def get_all(self) -> dict:
        """Return client_id -> {platform -> data}"""
        result: dict[str, dict] = {}
        for (cid, plat), data in self._store.items():
            if cid not in result:
                result[cid] = {}
            result[cid][plat] = data
        return result

    def get_any_for_platform(self, platform: str) -> tuple[str | None, dict | None]:
        """Get cookies from any client for a platform (backward compat). Prefers newest."""
        best_cid = None
        best_entry = None
        for (cid, plat), data in self._store.items():
            if plat == platform and data.get("cookies"):
                if best_entry is None or data["timestamp"] > best_entry["timestamp"]:
                    best_cid = cid
                    best_entry = data
        return best_cid, best_entry

    def get_fresh_for_platform(self, platform: str, ttl: int = 300) -> tuple[str | None, dict | None]:
        """Get fresh cookies from any client for a platform. Prefers newest."""
        best_cid = None
        best_entry = None
        now = time.time()
        for (cid, plat), data in self._store.items():
            if plat == platform and data.get("cookies") and (now - data["timestamp"]) < ttl:
                if best_entry is None or data["timestamp"] > best_entry["timestamp"]:
                    best_cid = cid
                    best_entry = data
        return best_cid, best_entry


class ClientRegistry:
    """Track registered extension clients"""

    def __init__(self):
        # client_id -> {"platforms": {...}, "nicknames": {...}, "connected": bool, "connected_at": float, "last_seen": float}
        self._clients: dict[str, dict] = {}

    def register(self, client_id: str, platforms: dict | None = None, nicknames: dict | None = None):
        if client_id in self._clients:
            self._clients[client_id]["platforms"] = platforms or {}
            self._clients[client_id]["nicknames"] = nicknames or {}
            self._clients[client_id]["last_seen"] = time.time()
        else:
            now = time.time()
            self._clients[client_id] = {
                "platforms": platforms or {},
                "nicknames": nicknames or {},
                "connected": True,
                "connected_at": now,
                "last_seen": now,
            }

    def set_connected(self, client_id: str, connected: bool):
        if client_id in self._clients:
            self._clients[client_id]["connected"] = connected
            if connected:
                self._clients[client_id]["connected_at"] = time.time()

    def set_last_seen(self, client_id: str):
        if client_id in self._clients:
            self._clients[client_id]["last_seen"] = time.time()

    def get(self, client_id: str) -> dict | None:
        return self._clients.get(client_id)

    def get_all(self) -> dict:
        return dict(self._clients)

    def get_connected_client_ids(self) -> list[str]:
        return [cid for cid, info in self._clients.items() if info.get("connected")]


# Singletons
cookie_store = CookieStore()
client_registry = ClientRegistry()


class ExtensionWebSocketHandler(tornado.websocket.WebSocketHandler):
    """WebSocket handler for Chrome Extension connections (multi-client)"""

    _connections: dict[str, "ExtensionWebSocketHandler"] = {}  # client_id -> handler
    _pending_requests: dict[str, asyncio.Future] = {}
    _bulk_collectors: dict[str, dict] = {}
    _request_client_map: dict[str, str] = {}  # request_id -> client_id

    def check_origin(self, origin):
        return True

    def open(self):
        self.client_id = None
        logger.info("New WebSocket connection opened (awaiting register)")

    def on_message(self, message):
        try:
            data = json.loads(message)
        except json.JSONDecodeError:
            logger.warning("Invalid JSON from extension: %s", message)
            return

        msg_type = data.get("type")
        msg_client_id = data.get("client_id")

        if msg_type == "register":
            self.client_id = msg_client_id
            if not self.client_id:
                logger.warning("Register message missing client_id")
                return

            # Close old connection if exists
            old_conn = self._connections.get(self.client_id)
            if old_conn is not None and old_conn is not self:
                logger.info("Closing old connection for client: %s", self.client_id)
                try:
                    old_conn.close()
                except Exception:
                    logger.debug("Error closing old connection for %s", self.client_id)

            self._connections[self.client_id] = self
            platforms = data.get("platforms", {})
            nicknames = data.get("nicknames", {})
            client_registry.register(self.client_id, platforms, nicknames)
            client_registry.set_connected(self.client_id, True)
            logger.info(
                "Client registered: %s, platforms: %s, nicknames: %s",
                self.client_id, list(platforms.keys()), list(nicknames.keys()),
            )

        elif msg_type == "cookies_update":
            client_id = msg_client_id or self.client_id
            if not client_id:
                return
            platform = data.get("platform")
            cookies = data.get("cookies", "")
            cookie_store.set(client_id, platform, cookies, source="extension")
            client_registry.set_last_seen(client_id)
            logger.info("Cookies updated for client=%s platform=%s", client_id, platform)

        elif msg_type == "cookies_response":
            request_id = data.get("request_id")
            platform = data.get("platform")
            cookies = data.get("cookies", "")
            client_id = msg_client_id or self.client_id
            if platform and client_id:
                cookie_store.set(client_id, platform, cookies, source="extension")
            if request_id in self._bulk_collectors:
                self._bulk_collectors[request_id][platform] = cookies
            elif request_id in self._pending_requests:
                future = self._pending_requests.pop(request_id)
                if not future.done():
                    future.set_result(data)

        elif msg_type == "cookies_response_complete":
            request_id = data.get("request_id")
            collected = self._bulk_collectors.pop(request_id, {})
            if request_id in self._pending_requests:
                future = self._pending_requests.pop(request_id)
                if not future.done():
                    future.set_result({
                        "type": "cookies_response_complete",
                        "request_id": request_id,
                        "platforms": collected,
                    })

        elif msg_type == "heartbeat":
            if self.client_id:
                client_registry.set_last_seen(self.client_id)
            self.write_message(json.dumps({"type": "heartbeat_ack"}))

    def on_close(self):
        if self.client_id and self._connections.get(self.client_id) is self:
            del self._connections[self.client_id]
            client_registry.set_connected(self.client_id, False)
            # Clean up pending requests for this client
            stale_ids = [rid for rid, cid in self._request_client_map.items() if cid == self.client_id]
            for rid in stale_ids:
                self._request_client_map.pop(rid, None)
                self._bulk_collectors.pop(rid, None)
                future = self._pending_requests.pop(rid, None)
                if future and not future.done():
                    future.cancel()
            logger.info("Client disconnected: %s", self.client_id)

    @classmethod
    def is_connected(cls) -> bool:
        """Any connection exists (backward compat)"""
        return len(cls._connections) > 0

    @classmethod
    def is_client_connected(cls, client_id: str) -> bool:
        return client_id in cls._connections

    @classmethod
    def get_connected_client_ids(cls) -> list[str]:
        return list(cls._connections.keys())

    @classmethod
    async def request_cookies(cls, platform: str, client_id: str | None = None, timeout: float = 10.0) -> dict | None:
        """Request cookies from extension for a specific platform.
        If client_id is None, pick any connected client."""
        if client_id:
            conn = cls._connections.get(client_id)
        else:
            # Pick first available connected client
            conn = next(iter(cls._connections.values()), None)
            if conn:
                client_id = conn.client_id

        if not conn:
            return None

        request_id = str(uuid.uuid4())
        future = asyncio.get_running_loop().create_future()
        cls._pending_requests[request_id] = future
        cls._request_client_map[request_id] = client_id

        try:
            conn.write_message(json.dumps({
                "type": "get_cookies",
                "request_id": request_id,
                "platform": platform,
            }))
            result = await asyncio.wait_for(future, timeout=timeout)
            result["_client_id"] = client_id
            return result
        except asyncio.TimeoutError:
            cls._pending_requests.pop(request_id, None)
            cls._request_client_map.pop(request_id, None)
            logger.warning("Timeout requesting cookies for %s from client %s", platform, client_id)
            return None
        except Exception as e:
            cls._pending_requests.pop(request_id, None)
            cls._request_client_map.pop(request_id, None)
            logger.error("Error requesting cookies for %s: %s", platform, e)
            return None

    @classmethod
    async def request_all_cookies(cls, client_id: str | None = None, timeout: float = 15.0) -> dict | None:
        """Request cookies for all platforms from extension.
        If client_id is None, pick any connected client."""
        if client_id:
            conn = cls._connections.get(client_id)
        else:
            conn = next(iter(cls._connections.values()), None)
            if conn:
                client_id = conn.client_id

        if not conn:
            return None

        request_id = str(uuid.uuid4())
        future = asyncio.get_running_loop().create_future()
        cls._pending_requests[request_id] = future
        cls._bulk_collectors[request_id] = {}
        cls._request_client_map[request_id] = client_id

        try:
            conn.write_message(json.dumps({
                "type": "get_all_cookies",
                "request_id": request_id,
            }))
            result = await asyncio.wait_for(future, timeout=timeout)
            return result
        except asyncio.TimeoutError:
            cls._pending_requests.pop(request_id, None)
            cls._bulk_collectors.pop(request_id, None)
            cls._request_client_map.pop(request_id, None)
            logger.warning("Timeout requesting all cookies from client %s", client_id)
            return None
        except Exception as e:
            cls._pending_requests.pop(request_id, None)
            cls._bulk_collectors.pop(request_id, None)
            cls._request_client_map.pop(request_id, None)
            logger.error("Error requesting all cookies: %s", e)
            return None
