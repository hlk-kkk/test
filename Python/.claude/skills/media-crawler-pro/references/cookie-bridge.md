# CookieBridge 参考指南

## 概述

CookieBridge 是 Chrome Extension + Python Tornado Server 的组合，支持多账号同时在线，用于从浏览器自动提取 cookies 并通过 REST API 提供给爬虫和 AI Agent 使用。

```
Chrome Extension (MV3)  <-- WebSocket -->  CookieBridge Server (Python/Tornado)  <-- HTTP -->  爬虫 / AI Agent
```

- **Chrome Extension**: 通过 `chrome.cookies` API 读取浏览器中的 cookies，支持多 Chrome Profile
- **CookieBridge Server**: WebSocket 连接 Extension，REST API 对外服务，缓存 + 降级
- 默认端口: `8274`

## 项目路径

```
$MCR_ROOT/MediaCrawlerPro-CookieBridge
```

## 支持平台

| 平台 | Key | 域名 | 登录检测 Cookie |
|------|-----|------|----------------|
| 小红书 | xhs | xiaohongshu.com | web_session |
| 抖音 | dy | douyin.com | ttwid |
| 快手 | ks | kuaishou.com | kuaishou.server.web_st |
| B站 | bili | bilibili.com | bili_ticket |
| 微博 | wb | weibo.com | SUBP |
| 贴吧 | tieba | tieba.baidu.com | BDUSS |
| 知乎 | zhihu | zhihu.com | z_c0 |

## 启动服务

### 一键启动（推荐）

```bash
# macOS / Linux
cd "$MCR_ROOT/MediaCrawlerPro-CookieBridge"
./start.sh

# Windows
start.bat
```

自动安装 uv、项目依赖并启动服务。

### 手动启动

```bash
cd "$MCR_ROOT/MediaCrawlerPro-CookieBridge/server"
uv sync
uv run app.py
```

### Chrome Extension 安装

1. 打开 `chrome://extensions/`
2. 启用"开发者模式"
3. 点击"加载已解压的扩展程序"，选择 `extension/` 目录
4. 多账号：在不同 Chrome Profile 中分别安装 Extension

## 与 Python 爬虫集成

CookieBridge 已作为第三种 Cookie 来源集成到 `MediaCrawlerPro-Python` 的账号池管理中。

```bash
# 使用 CookieBridge 模式运行爬虫
ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run --project "$MCR_ROOT/MediaCrawlerPro-Python" \
  "$MCR_ROOT/MediaCrawlerPro-Python/main.py" \
  --platform bili --type search --keywords "deepseek"
```

相关配置（`config/base_config.py`）：
- `ACCOUNT_POOL_SAVE_TYPE` — 设为 `cookie_bridge`
- `COOKIE_BRIDGE_URL` — CookieBridge 服务地址，默认 `http://localhost:8274`

**注意**：如果 shell 环境中存在 `ACCOUNT_POOL_SAVE_TYPE` 环境变量，会覆盖配置文件中的默认值。

## REST API

### 健康检查 `GET /ping`

```bash
curl -s http://localhost:8274/ping
# {"biz_code":0,"msg":"OK!","isok":true,"data":{"message":"pong"}}
```

### 查看账号列表 `GET /api/accounts`

返回所有已注册的客户端（Chrome Profile）及其各平台 Cookie 状态。

```bash
curl -s http://localhost:8274/api/accounts
```

响应:
```json
{
  "biz_code": 0, "msg": "OK!", "isok": true,
  "data": {
    "accounts": [
      {
        "client_id": "550e8400-e29b-41d4-a716-446655440000",
        "connected": true,
        "nicknames": {"xhs": "风往北吹", "bili": "YangBabyZi"},
        "platforms": {
          "xhs": {"has_cookies": true, "cookie_count": 15},
          "dy": {"has_cookies": true, "cookie_count": 31},
          "ks": {"has_cookies": false, "cookie_count": 0}
        },
        "last_seen": 1234567890.0
      }
    ],
    "total": 1
  }
}
```

### 查看账号详情 `GET /api/accounts/{client_id}`

### 查看服务状态 `GET /api/status`

返回 Extension 连接状态和各平台 cookies 可用情况。

### 获取单平台 cookies `GET /api/cookies/{platform}`

```bash
# 不指定 client_id — 返回任意可用账号的 Cookie
curl -s http://localhost:8274/api/cookies/xhs

# 指定 client_id — 获取特定账号的 Cookie
curl -s "http://localhost:8274/api/cookies/xhs?client_id=550e8400-e29b-41d4-a716-446655440000"
```

响应:
```json
{
  "biz_code": 0, "msg": "OK!", "isok": true,
  "data": {
    "platform": "xhs",
    "cookies": "web_session=xxx; a1=yyy; ...",
    "source": "extension",
    "client_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

`source` 字段说明:
- `extension` — 实时从浏览器获取
- `cache` — 服务端新鲜缓存（默认 5 分钟 TTL）
- `stale_cache` — 过期缓存（Extension 离线时降级使用）
- `manual` — 通过 API 手动设置

### 获取所有平台 cookies `GET /api/cookies/all`

```bash
curl -s http://localhost:8274/api/cookies/all
curl -s "http://localhost:8274/api/cookies/all?client_id=xxx"
```

### 手动设置 cookies `POST /api/cookies/{platform}/set`

```bash
curl -X POST http://localhost:8274/api/cookies/xhs/set \
  -H 'Content-Type: application/json' \
  -d '{"cookies": "key1=value1; key2=value2", "client_id": "manual-01"}'
```

## 与 DownloadServer 集成

从 CookieBridge 获取 cookies 后，传入 DownloadServer API：

```bash
# 1. 从 CookieBridge 获取 cookies
COOKIES=$(curl -s http://localhost:8274/api/cookies/xhs | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['cookies'])")

# 2. 传给 DownloadServer
curl -s -X POST http://localhost:8205/api/v1/content_detail \
  -H 'Content-Type: application/json' \
  -d "{\"platform\": \"xhs\", \"content_url\": \"https://...\", \"cookies\": \"$COOKIES\"}"
```

## 回退行为

- Extension 断开时，Server 返回缓存中的 cookies（可能已过期），`source` 为 `stale_cache`
- CookieBridge 服务不可用时，爬虫应回退到 Excel (`xlsx`) 或 MySQL (`mysql`) 模式
- Python 爬虫的 `load_accounts_from_cookie_bridge()` 在连接失败时会抛出明确异常，提示检查服务状态
