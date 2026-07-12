# CookieBridge WebSocket Protocol (V2)

## Overview

CookieBridge uses WebSocket for real-time communication between Chrome Extension(s) and the Python Server.
V2 supports multiple Extension connections simultaneously, each identified by a unique `client_id`.

- WebSocket endpoint: `ws://localhost:8274/ws`
- All messages are JSON-encoded strings
- Each Extension generates a persistent `client_id` (UUID) stored in `chrome.storage.local`

## Extension -> Server Messages

All Extension-to-Server messages include a `client_id` field.

### register
Sent immediately after WebSocket connection is established.

```json
{
  "type": "register",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "nicknames": {
    "bili": "UserName",
    "xhs": "XHS_Nick"
  },
  "platforms": {
    "xhs": {"has_cookies": true, "cookie_count": 15},
    "dy": {"has_cookies": true, "cookie_count": 30},
    "ks": {"has_cookies": false, "cookie_count": 0},
    "bili": {"has_cookies": true, "cookie_count": 8},
    "wb": {"has_cookies": false, "cookie_count": 0},
    "tieba": {"has_cookies": false, "cookie_count": 0},
    "zhihu": {"has_cookies": false, "cookie_count": 0}
  }
}
```

### cookies_update
Sent proactively when cookies change.

```json
{
  "type": "cookies_update",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "platform": "xhs",
  "cookies": "web_session=xxx; a1=yyy; ...",
  "timestamp": 1234567890
}
```

> Note: The `timestamp` field is provided by the Extension but the Server ignores it, using its own `time.time()` when storing in CookieStore. This ensures consistent server-side timestamps regardless of client clock skew.

### cookies_response
Sent in response to a `get_cookies` or `get_all_cookies` request.

```json
{
  "type": "cookies_response",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "uuid-string",
  "platform": "xhs",
  "cookies": "web_session=xxx; a1=yyy; ...",
  "valid": true
}
```

For `get_all_cookies`, multiple `cookies_response` messages are sent (one per platform),
followed by a final completion message:

```json
{
  "type": "cookies_response_complete",
  "client_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "uuid-string"
}
```

### heartbeat
Sent periodically (every 24 seconds via chrome.alarms) to keep the connection alive.

```json
{
  "type": "heartbeat",
  "client_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## Server -> Extension Messages

Server-to-Extension messages are sent through specific WebSocket connections and do not include `client_id`.
The server maintains a `client_id -> WebSocket connection` mapping to route messages to the correct Extension instance.

### get_cookies
Request cookies for a specific platform.

```json
{
  "type": "get_cookies",
  "request_id": "uuid-string",
  "platform": "xhs"
}
```

### get_all_cookies
Request cookies for all platforms.

```json
{
  "type": "get_all_cookies",
  "request_id": "uuid-string"
}
```

### heartbeat_ack
Acknowledgment of heartbeat.

```json
{
  "type": "heartbeat_ack"
}
```

## REST API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/ping` | GET | Health check |
| `/api/cookies/{platform}` | GET | Get cookies. Optional `?client_id=xxx`, without it returns any available account |
| `/api/cookies/all` | GET | Get all platform cookies. Optional `?client_id=xxx` |
| `/api/cookies/{platform}/set` | POST | Set cookies manually (see request body below) |
| `/api/accounts` | GET | List all registered accounts |
| `/api/accounts/{client_id}` | GET | Get specific account details |
| `/api/status` | GET | Multi-client status summary |
| `/ws` | WS | WebSocket endpoint (supports multiple connections) |

### POST /api/cookies/{platform}/set - Request Body

```json
{
  "cookies": "key1=value1; key2=value2; ...",
  "client_id": "optional-client-id"
}
```

- `cookies` (required): Plain cookie string
- `client_id` (optional): Defaults to `"manual"`. Max 64 chars, alphanumeric + hyphens + underscores only.

### Path Parameters

- `{platform}`: One of `xhs`, `dy`, `ks`, `bili`, `wb`, `tieba`, `zhihu` (lowercase only)
- `{client_id}`: Alphanumeric string with hyphens/underscores, max 64 chars

### Error Codes

| Code | Message |
|------|---------|
| 0 | OK! |
| 3 | Invalid Parameter |
| 4 | Missing Parameter |
| 100 | Extension Not Connected |
| 101 | Cookies Not Found |
| 102 | Request Timeout |
| 103 | Account Not Found |

## Nickname Extraction

The Extension attempts to extract logged-in user nicknames via platform APIs (best-effort):

| Platform | API | Field |
|----------|-----|-------|
| bili | `api.bilibili.com/x/web-interface/nav` | `data.uname` |
| zhihu | `zhihu.com/api/v4/me` | `name` |
| wb | `weibo.com/ajax/login/config` | `data.user.screen_name` |
| xhs | `edith.xiaohongshu.com/api/sns/web/v2/user/me` | `data.nickname` |
| tieba | `tieba.baidu.com/f/user/json_userinfo` | `data.user_name_show` |

Note: Nickname extraction from Service Worker `fetch` may not carry website cookies for all platforms. Nicknames are best-effort and do not block the registration flow.

## Platform Domain Mapping

| Platform Key | Domains |
|-------------|---------|
| xhs | .xiaohongshu.com |
| dy | .douyin.com |
| ks | .kuaishou.com |
| bili | .bilibili.com |
| wb | .weibo.com, .weibo.cn |
| tieba | .tieba.baidu.com |
| zhihu | .zhihu.com |

## Login Detection Cookies

| Platform | Key Cookie |
|----------|-----------|
| xhs | web_session |
| dy | ttwid |
| ks | kuaishou.server.web_st |
| bili | bili_ticket |
| wb | SUBP |
| tieba | BDUSS |
| zhihu | z_c0 |
