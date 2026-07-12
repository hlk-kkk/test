#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# Copyright (c) 2025 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-CookieBridge project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge/blob/main/test_e2e.py
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

"""CookieBridge V2 End-to-End Test Suite"""

import asyncio
import json
import sys
import time
import traceback

import requests
import websockets

BASE_URL = "http://127.0.0.1:8274"
WS_URL = "ws://127.0.0.1:8274/ws"

# Bypass any system proxy for localhost testing
SESSION = requests.Session()
SESSION.trust_env = False

results = []


def record(name, passed, detail=""):
    status = "PASS" if passed else "FAIL"
    results.append({"name": name, "status": status, "detail": detail})
    print(f"  [{status}] {name}" + (f" -- {detail}" if detail else ""))


def assert_ok(resp, test_name):
    data = resp.json()
    if resp.status_code != 200:
        record(test_name, False, f"HTTP {resp.status_code}")
        return None
    if data.get("biz_code") != 0 or not data.get("isok"):
        record(test_name, False, f"biz_code={data.get('biz_code')}, msg={data.get('msg')}")
        return None
    return data


def assert_error(resp, test_name, expected_biz_code=None):
    data = resp.json()
    if data.get("isok"):
        record(test_name, False, f"Expected error but got isok=True")
        return None
    if expected_biz_code is not None and data.get("biz_code") != expected_biz_code:
        record(test_name, False, f"Expected biz_code={expected_biz_code}, got {data.get('biz_code')}")
        return None
    return data


# ============================================================
# Test 1: Health Check
# ============================================================
def test_ping():
    print("\n=== Test 1: Health Check ===")
    resp = SESSION.get(f"{BASE_URL}/ping")
    data = assert_ok(resp, "GET /ping returns OK")
    if data:
        pong = data.get("data", {}).get("message")
        record("GET /ping returns pong", pong == "pong", f"message={pong}")

    resp2 = SESSION.get(f"{BASE_URL}/")
    data2 = assert_ok(resp2, "GET / returns OK")
    if data2:
        record("GET / returns pong", data2.get("data", {}).get("message") == "pong")


# ============================================================
# Test 2: Manual Set Multi-Account Cookies
# ============================================================
def test_set_cookies():
    print("\n=== Test 2: Set Multi-Account Cookies ===")

    # client-a xhs
    resp = SESSION.post(f"{BASE_URL}/api/cookies/xhs/set", json={
        "cookies": "web_session=aaa; a1=bbb",
        "client_id": "client-a"
    })
    data = assert_ok(resp, "Set xhs cookies for client-a")
    if data:
        d = data.get("data", {})
        record("client-a xhs set response correct",
               d.get("platform") == "xhs" and d.get("client_id") == "client-a" and d.get("source") == "manual",
               f"data={d}")

    # client-b xhs
    resp = SESSION.post(f"{BASE_URL}/api/cookies/xhs/set", json={
        "cookies": "web_session=ccc; a1=ddd",
        "client_id": "client-b"
    })
    data = assert_ok(resp, "Set xhs cookies for client-b")
    if data:
        d = data.get("data", {})
        record("client-b xhs set response correct",
               d.get("platform") == "xhs" and d.get("client_id") == "client-b",
               f"data={d}")

    # client-a bili
    resp = SESSION.post(f"{BASE_URL}/api/cookies/bili/set", json={
        "cookies": "bili_ticket=eee",
        "client_id": "client-a"
    })
    data = assert_ok(resp, "Set bili cookies for client-a")
    if data:
        d = data.get("data", {})
        record("client-a bili set response correct",
               d.get("platform") == "bili" and d.get("client_id") == "client-a",
               f"data={d}")


# ============================================================
# Test 3: Query Specific Account Cookies
# ============================================================
def test_get_cookies_by_client():
    print("\n=== Test 3: Query Cookies by Client ID ===")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs", params={"client_id": "client-a"})
    data = assert_ok(resp, "GET /api/cookies/xhs?client_id=client-a")
    if data:
        d = data.get("data", {})
        record("client-a xhs cookies correct",
               d.get("cookies") == "web_session=aaa; a1=bbb" and d.get("client_id") == "client-a",
               f"cookies={d.get('cookies')}, source={d.get('source')}")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs", params={"client_id": "client-b"})
    data = assert_ok(resp, "GET /api/cookies/xhs?client_id=client-b")
    if data:
        d = data.get("data", {})
        record("client-b xhs cookies correct",
               d.get("cookies") == "web_session=ccc; a1=ddd" and d.get("client_id") == "client-b",
               f"cookies={d.get('cookies')}")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/bili", params={"client_id": "client-a"})
    data = assert_ok(resp, "GET /api/cookies/bili?client_id=client-a")
    if data:
        d = data.get("data", {})
        record("client-a bili cookies correct",
               d.get("cookies") == "bili_ticket=eee",
               f"cookies={d.get('cookies')}")


# ============================================================
# Test 4: Backward Compat - No client_id
# ============================================================
def test_get_cookies_no_client_id():
    print("\n=== Test 4: Backward Compat (no client_id) ===")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs")
    data = assert_ok(resp, "GET /api/cookies/xhs (no client_id)")
    if data:
        d = data.get("data", {})
        has_cookies = bool(d.get("cookies"))
        record("Returns cookies from any client", has_cookies, f"cookies={d.get('cookies')}, client_id={d.get('client_id')}")


# ============================================================
# Test 5: Get All Cookies
# ============================================================
def test_get_all_cookies():
    print("\n=== Test 5: Get All Cookies ===")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/all")
    data = assert_ok(resp, "GET /api/cookies/all")
    if data:
        platforms = data.get("data", {}).get("platforms", {})
        has_xhs = "xhs" in platforms
        has_bili = "bili" in platforms
        record("All cookies includes xhs", has_xhs, f"platforms={list(platforms.keys())}")
        record("All cookies includes bili", has_bili)

    resp = SESSION.get(f"{BASE_URL}/api/cookies/all", params={"client_id": "client-a"})
    data = assert_ok(resp, "GET /api/cookies/all?client_id=client-a")
    if data:
        platforms = data.get("data", {}).get("platforms", {})
        record("client-a all cookies has xhs and bili",
               "xhs" in platforms and "bili" in platforms,
               f"platforms={list(platforms.keys())}")
        if "xhs" in platforms:
            record("client-a xhs cookies in all correct",
                   platforms["xhs"].get("cookies") == "web_session=aaa; a1=bbb")


# ============================================================
# Test 6: Accounts API
# ============================================================
def test_accounts_api():
    print("\n=== Test 6: Accounts API ===")

    resp = SESSION.get(f"{BASE_URL}/api/accounts")
    data = assert_ok(resp, "GET /api/accounts")
    if data:
        accounts = data.get("data", {}).get("accounts", [])
        total = data.get("data", {}).get("total", 0)
        record("Accounts API returns list", isinstance(accounts, list), f"total={total}")

    resp = SESSION.get(f"{BASE_URL}/api/accounts/00000000-0000-0000-0000-000000000000")
    data_err = assert_error(resp, "GET /api/accounts/nonexistent returns error", expected_biz_code=103)
    if data_err:
        record("Account not found error code correct", data_err.get("biz_code") == 103, f"msg={data_err.get('msg')}")


# ============================================================
# Test 7: Status API
# ============================================================
def test_status_api():
    print("\n=== Test 7: Status API ===")

    resp = SESSION.get(f"{BASE_URL}/api/status")
    data = assert_ok(resp, "GET /api/status")
    if data:
        d = data.get("data", {})
        record("Status has extension_connected field", "extension_connected" in d)
        record("Status has connected_count field", "connected_count" in d)
        record("Status has total_clients field", "total_clients" in d)
        record("Status has clients field", "clients" in d)


# ============================================================
# Test 8: Error Handling
# ============================================================
def test_error_handling():
    print("\n=== Test 8: Error Handling ===")

    resp = SESSION.get(f"{BASE_URL}/api/cookies/invalidplatform")
    data_err = assert_error(resp, "Invalid platform returns error", expected_biz_code=3)
    if data_err:
        record("Invalid platform biz_code=3", data_err.get("biz_code") == 3)

    resp = SESSION.post(f"{BASE_URL}/api/cookies/xhs/set", json={})
    data_err = assert_error(resp, "Missing cookies param returns error", expected_biz_code=4)
    if data_err:
        record("Missing parameter biz_code=4", data_err.get("biz_code") == 4)

    resp = SESSION.post(f"{BASE_URL}/api/cookies/xhs/set", json={"cookies": ""})
    data_err = assert_error(resp, "Empty cookies string returns error", expected_biz_code=4)
    if data_err:
        record("Empty cookies biz_code=4", data_err.get("biz_code") == 4)


# ============================================================
# Test 9: WebSocket Multi-Connection Tests
# ============================================================
async def test_websocket():
    print("\n=== Test 9: WebSocket Multi-Connection Tests ===")

    # --- 9.1 Connect two Extension clients ---
    try:
        ws1 = await websockets.connect(WS_URL)
        record("WS client-1 connected", True)
    except Exception as e:
        record("WS client-1 connected", False, str(e))
        return

    try:
        ws2 = await websockets.connect(WS_URL)
        record("WS client-2 connected", True)
    except Exception as e:
        record("WS client-2 connected", False, str(e))
        await ws1.close()
        return

    # --- 9.2 Register both clients ---
    await ws1.send(json.dumps({
        "type": "register",
        "client_id": "ext-client-1",
        "platforms": {"xhs": True, "bili": True},
        "nicknames": {"xhs": "User1-XHS"}
    }))
    await asyncio.sleep(0.3)

    await ws2.send(json.dumps({
        "type": "register",
        "client_id": "ext-client-2",
        "platforms": {"xhs": True, "dy": True},
        "nicknames": {"xhs": "User2-XHS"}
    }))
    await asyncio.sleep(0.3)

    # Verify status shows 2 connected WS clients
    resp = SESSION.get(f"{BASE_URL}/api/status")
    data = resp.json().get("data", {})
    connected_count = data.get("connected_count", 0)
    record("Status shows 2 WS clients connected", connected_count >= 2,
           f"connected_count={connected_count}")

    clients = data.get("clients", {})
    record("ext-client-1 in status clients", "ext-client-1" in clients)
    record("ext-client-2 in status clients", "ext-client-2" in clients)

    if "ext-client-1" in clients:
        record("ext-client-1 connected=True", clients["ext-client-1"].get("connected") is True)
    if "ext-client-2" in clients:
        record("ext-client-2 connected=True", clients["ext-client-2"].get("connected") is True)

    # Verify accounts API
    resp = SESSION.get(f"{BASE_URL}/api/accounts")
    accounts_data = resp.json().get("data", {})
    accounts = accounts_data.get("accounts", [])
    ext_ids = [a["client_id"] for a in accounts]
    record("Accounts includes ext-client-1", "ext-client-1" in ext_ids, f"account_ids={ext_ids}")
    record("Accounts includes ext-client-2", "ext-client-2" in ext_ids)

    # --- 9.3 Send cookies_update from ws1 ---
    await ws1.send(json.dumps({
        "type": "cookies_update",
        "client_id": "ext-client-1",
        "platform": "xhs",
        "cookies": "web_session=ws1_xhs_cookie; a1=ws1_a1"
    }))
    await asyncio.sleep(0.3)

    # Verify via HTTP
    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs", params={"client_id": "ext-client-1"})
    data = resp.json()
    if data.get("isok"):
        d = data.get("data", {})
        record("WS cookies_update reflected in HTTP API",
               d.get("cookies") == "web_session=ws1_xhs_cookie; a1=ws1_a1",
               f"cookies={d.get('cookies')}, source={d.get('source')}")
    else:
        record("WS cookies_update reflected in HTTP API", False, f"biz_code={data.get('biz_code')}")

    # --- 9.4 Send cookies_update from ws2 ---
    await ws2.send(json.dumps({
        "type": "cookies_update",
        "client_id": "ext-client-2",
        "platform": "xhs",
        "cookies": "web_session=ws2_xhs_cookie; a1=ws2_a1"
    }))
    await asyncio.sleep(0.3)

    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs", params={"client_id": "ext-client-2"})
    data = resp.json()
    if data.get("isok"):
        d = data.get("data", {})
        record("WS2 cookies_update reflected in HTTP API",
               d.get("cookies") == "web_session=ws2_xhs_cookie; a1=ws2_a1",
               f"cookies={d.get('cookies')}")
    else:
        record("WS2 cookies_update reflected in HTTP API", False)

    # --- 9.5 Test get_cookies request-response flow ---
    # The server sends get_cookies to ws2, ws2 replies with cookies_response.
    # We use a thread for the blocking HTTP call so the async WS recv can happen concurrently.
    from concurrent.futures import ThreadPoolExecutor

    async def simulate_ext2_response(ws):
        try:
            raw = await asyncio.wait_for(ws.recv(), timeout=12.0)
            msg = json.loads(raw)
            if msg.get("type") == "get_cookies":
                await ws.send(json.dumps({
                    "type": "cookies_response",
                    "request_id": msg.get("request_id"),
                    "platform": msg.get("platform", "dy"),
                    "cookies": "douyin_session=fresh_from_ext2",
                    "client_id": "ext-client-2"
                }))
                return True
        except asyncio.TimeoutError:
            return False
        return False

    def blocking_http_get():
        s = requests.Session()
        s.trust_env = False
        return s.get(f"{BASE_URL}/api/cookies/dy", params={"client_id": "ext-client-2"}, timeout=15)

    try:
        loop = asyncio.get_running_loop()
        # Start the WS listener task first
        ws_task = asyncio.create_task(simulate_ext2_response(ws2))
        # Then fire off the blocking HTTP request in a thread
        http_future = loop.run_in_executor(ThreadPoolExecutor(1), blocking_http_get)
        # Wait for both
        resp = await http_future
        ws_replied = await ws_task
        record("Server sent get_cookies to extension", ws_replied)
        if resp.status_code == 200:
            data = resp.json()
            if data.get("isok"):
                d = data.get("data", {})
                record("Extension cookies_response returned via HTTP",
                       d.get("cookies") == "douyin_session=fresh_from_ext2",
                       f"cookies={d.get('cookies')}, source={d.get('source')}")
            else:
                record("Extension cookies_response returned via HTTP", False, f"biz_code={data.get('biz_code')}")
        else:
            record("Extension cookies_response returned via HTTP", False, f"HTTP {resp.status_code}")
    except Exception as e:
        record("get_cookies -> cookies_response flow", False, f"{type(e).__name__}: {e}")

    # --- 9.6 Test heartbeat ---
    await ws1.send(json.dumps({"type": "heartbeat"}))
    try:
        raw = await asyncio.wait_for(ws1.recv(), timeout=3.0)
        msg = json.loads(raw)
        record("Heartbeat returns heartbeat_ack", msg.get("type") == "heartbeat_ack", f"type={msg.get('type')}")
    except asyncio.TimeoutError:
        record("Heartbeat returns heartbeat_ack", False, "timeout")

    # --- 9.7 Disconnect one client ---
    await ws1.close()
    await asyncio.sleep(0.5)

    resp = SESSION.get(f"{BASE_URL}/api/status")
    data = resp.json().get("data", {})
    clients = data.get("clients", {})
    connected_count = data.get("connected_count", 0)

    record("After ws1 disconnect, connected_count decremented",
           connected_count >= 1 and connected_count < 3,
           f"connected_count={connected_count}")

    if "ext-client-1" in clients:
        record("ext-client-1 connected=False after disconnect",
               clients["ext-client-1"].get("connected") is False,
               f"connected={clients['ext-client-1'].get('connected')}")

    if "ext-client-2" in clients:
        record("ext-client-2 still connected=True",
               clients["ext-client-2"].get("connected") is True)

    # ext-client-1's cookies should still be available (stale cache)
    resp = SESSION.get(f"{BASE_URL}/api/cookies/xhs", params={"client_id": "ext-client-1"})
    data = resp.json()
    if data.get("isok"):
        d = data.get("data", {})
        record("ext-client-1 cookies still available after disconnect (cache)",
               bool(d.get("cookies")),
               f"source={d.get('source')}")
    else:
        record("ext-client-1 cookies still available after disconnect (cache)", False)

    # --- 9.8 Accounts still shows both (ext-client-1 as disconnected) ---
    resp = SESSION.get(f"{BASE_URL}/api/accounts")
    accounts = resp.json().get("data", {}).get("accounts", [])
    ext1_found = any(a["client_id"] == "ext-client-1" for a in accounts)
    ext2_connected = any(a["client_id"] == "ext-client-2" and a["connected"] for a in accounts)
    record("Accounts still shows ext-client-1 after disconnect", ext1_found)
    record("Accounts shows ext-client-2 still connected", ext2_connected)

    # Clean up
    await ws2.close()
    await asyncio.sleep(0.3)


# ============================================================
# Test 10: CORS headers
# ============================================================
def test_cors():
    print("\n=== Test 10: CORS Headers ===")
    resp = SESSION.options(f"{BASE_URL}/api/cookies/xhs")
    record("OPTIONS returns 204", resp.status_code == 204)
    record("CORS Allow-Origin header present",
           "Access-Control-Allow-Origin" in resp.headers,
           f"value={resp.headers.get('Access-Control-Allow-Origin')}")


# ============================================================
# Test 11: Set Cookies with default client_id
# ============================================================
def test_set_cookies_default_client():
    print("\n=== Test 11: Set Cookies with default client_id ===")
    resp = SESSION.post(f"{BASE_URL}/api/cookies/dy/set", json={
        "cookies": "dy_session=test_default"
    })
    data = assert_ok(resp, "Set dy cookies without client_id")
    if data:
        d = data.get("data", {})
        record("Default client_id is 'manual'",
               d.get("client_id") == "manual",
               f"client_id={d.get('client_id')}")


# ============================================================
# Run all tests
# ============================================================
def main():
    print("=" * 60)
    print("CookieBridge V2 End-to-End Test Suite")
    print("=" * 60)

    try:
        test_ping()
        test_set_cookies()
        test_get_cookies_by_client()
        test_get_cookies_no_client_id()
        test_get_all_cookies()
        test_accounts_api()
        test_status_api()
        test_error_handling()
        asyncio.run(test_websocket())
        test_cors()
        test_set_cookies_default_client()
    except Exception as e:
        print(f"\n!!! UNEXPECTED ERROR: {e}")
        traceback.print_exc()
        record("Unexpected test error", False, str(e))

    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    passed = sum(1 for r in results if r["status"] == "PASS")
    failed = sum(1 for r in results if r["status"] == "FAIL")
    total = len(results)
    print(f"Total: {total}  |  PASS: {passed}  |  FAIL: {failed}")
    print()

    if failed > 0:
        print("FAILED TESTS:")
        for r in results:
            if r["status"] == "FAIL":
                print(f"  [FAIL] {r['name']}" + (f" -- {r['detail']}" if r['detail'] else ""))
    else:
        print("ALL TESTS PASSED!")

    print()
    return 1 if failed > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
