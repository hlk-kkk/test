# MediaCrawlerPro-CookieBridge

浏览器 Cookie 自动提取与同步桥接服务，支持多账号同时在线。

## 架构概览

```
Chrome Extension (MV3)  ── WebSocket ──>  CookieBridge Server (Python/Tornado)  <── HTTP ──  MediaCrawlerPro-Python
```

- **Chrome Extension**：通过 `chrome.cookies` API 自动读取浏览器中各平台的 Cookie
- **CookieBridge Server**：WebSocket + REST API 桥接服务，管理多个浏览器客户端的 Cookie 数据
- **MediaCrawlerPro-Python**：通过 REST API 获取 Cookie，替代手动复制粘贴到 Excel/MySQL

## 核心特性

- **多账号支持**：多个 Chrome Profile 各自安装 Extension，同时连接同一个 Server
- **自动同步**：浏览器中 Cookie 变化时自动推送到 Server，无需手动操作
- **持久化 Client ID**：每个浏览器实例生成唯一 UUID，跨会话不变
- **昵称识别**：自动尝试提取各平台的登录用户昵称（best-effort）
- **向后兼容**：API 不带 `client_id` 参数时，自动返回任意可用账号的 Cookie
- **缓存 + 降级**：优先返回新鲜缓存，Extension 离线时返回过期缓存

## 支持平台

| 平台 | 标识 | 域名 | 登录检测 Cookie |
|------|------|------|----------------|
| 小红书 | xhs | xiaohongshu.com | web_session |
| 抖音 | dy | douyin.com | ttwid |
| 快手 | ks | kuaishou.com | kuaishou.server.web_st |
| B站 | bili | bilibili.com | bili_ticket |
| 微博 | wb | weibo.com / weibo.cn | SUBP |
| 贴吧 | tieba | tieba.baidu.com | BDUSS |
| 知乎 | zhihu | zhihu.com | z_c0 |

## 快速开始

### 1. 启动 Server

**一键启动（推荐）**：自动安装 uv、项目依赖并启动服务

```bash
# macOS / Linux
./start.sh

# Windows
start.bat
```

**手动启动**：

```bash
cd server
uv sync
uv run app.py
```

Server 默认运行在 `http://localhost:8274`，可通过环境变量 `APP_PORT` 修改端口。

### 2. 安装 Chrome Extension

1. 打开 Chrome 浏览器，地址栏输入 `chrome://extensions/` 回车，进入扩展管理页面
2. 开启页面右上角的「开发者模式」开关
3. 点击左上角「加载已解压的扩展程序」按钮
4. 在弹出的文件选择窗口中，定位到本仓库的 `extension/` 目录（即 `MediaCrawlerPro-CookieBridge/extension/`），选中该文件夹后点击「选择」

```
MediaCrawlerPro-CookieBridge/
├── server/       ← Server 代码（步骤 1）
└── extension/    ← 选择这个目录加载到 Chrome（步骤 2）
    ├── manifest.json
    ├── background.js
    ├── popup.html
    ├── popup.js
    ├── popup.css
    └── icons/
```

安装成功后，Chrome 工具栏会出现 CookieBridge 图标，点击可查看连接状态。Extension 会自动生成 Client ID 并连接 Server（需先启动 Server）。

### 3. 多账号使用

如需管理多个账号，在不同的 Chrome Profile 中分别安装 Extension：

1. Chrome 右上角头像 → 「添加个人资料」
2. 在新 Profile 中重复步骤 2 安装 Extension
3. 在不同 Profile 中登录不同平台账号
4. 每个 Profile 的 Extension 会自动获得独立的 Client ID

### 4. 验证连接

```bash
# 健康检查
curl http://localhost:8274/ping

# 查看连接状态（显示所有已连接的客户端）
curl http://localhost:8274/api/status

# 查看已注册账号列表
curl http://localhost:8274/api/accounts
```

## API 参考

### 统一响应格式

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": { ... }
}
```

### 接口列表

| 接口 | 方法 | 说明 |
|------|------|------|
| `/ping` | GET | 健康检查 |
| `/api/status` | GET | 服务状态，含多客户端连接信息 |
| `/api/accounts` | GET | 所有已注册账号列表 |
| `/api/accounts/{client_id}` | GET | 指定账号详情（含各平台 Cookie 状态） |
| `/api/cookies/{platform}` | GET | 获取指定平台 Cookie |
| `/api/cookies/all` | GET | 获取所有平台 Cookie |
| `/api/cookies/{platform}/set` | POST | 手动设置 Cookie |
| `/ws` | WS | WebSocket 端点（支持多连接） |

### 获取 Cookie

```bash
# 不指定 client_id → 返回任意可用账号的 Cookie（向后兼容）
curl http://localhost:8274/api/cookies/xhs

# 指定 client_id → 获取特定账号的 Cookie
curl "http://localhost:8274/api/cookies/xhs?client_id=550e8400-e29b-41d4-a716-446655440000"

# 获取所有平台 Cookie
curl http://localhost:8274/api/cookies/all
curl "http://localhost:8274/api/cookies/all?client_id=550e8400-e29b-41d4-a716-446655440000"
```

响应示例：

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": {
    "platform": "xhs",
    "cookies": "web_session=xxx; a1=yyy; ...",
    "source": "extension",
    "client_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

`source` 字段说明：
- `extension` — 实时从浏览器获取
- `cache` — 服务端新鲜缓存（默认 5 分钟 TTL）
- `stale_cache` — 过期缓存（Extension 离线时降级使用）
- `manual` — 通过 API 手动设置

### 手动设置 Cookie

```bash
curl -X POST http://localhost:8274/api/cookies/xhs/set \
  -H 'Content-Type: application/json' \
  -d '{"cookies": "web_session=xxx; a1=yyy", "client_id": "manual-01"}'
```

`client_id` 可选，默认为 `"manual"`。

### 查看账号列表

```bash
curl http://localhost:8274/api/accounts
```

响应示例：

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": {
    "accounts": [
      {
        "client_id": "550e8400-e29b-41d4-a716-446655440000",
        "connected": true,
        "nicknames": {"xhs": "小红薯用户", "bili": "B站昵称"},
        "platforms": {"xhs": {"has_cookies": true, "cookie_count": 15}, ...},
        "last_seen": 1234567890.0
      }
    ],
    "total": 1
  }
}
```

## 与 MediaCrawlerPro-Python 集成

CookieBridge 可以替代 MediaCrawlerPro-Python 中手动管理 Excel/MySQL 账号池的方式，实现 Cookie 自动同步。

### 集成流程

```
浏览器登录平台 → Extension 自动提取 Cookie → CookieBridge Server 缓存
                                                     ↓
MediaCrawlerPro-Python 启动爬虫 ← HTTP API 获取 Cookie ←┘
```

### 对比传统方式

| | 传统方式（Excel/MySQL 账号池） | CookieBridge |
|---|---|---|
| Cookie 获取 | 手动复制粘贴到 Excel 或 MySQL | 浏览器自动同步，零手动操作 |
| Cookie 更新 | 过期后需手动重新提取 | 浏览器在线时自动保持最新 |
| 多账号管理 | 在 Excel 不同行或数据库不同记录 | 不同 Chrome Profile 自动区分 |
| 登录状态 | 无法感知，过期后爬虫报错 | 实时检测登录状态 |

### 使用方法

在 MediaCrawlerPro-Python 的爬虫代码中，通过 HTTP 请求获取 Cookie 即可：

```python
import httpx

COOKIE_BRIDGE_URL = "http://localhost:8274"

async def get_cookies_from_bridge(platform: str, client_id: str = None) -> str:
    """从 CookieBridge 获取指定平台的 Cookie"""
    url = f"{COOKIE_BRIDGE_URL}/api/cookies/{platform}"
    params = {}
    if client_id:
        params["client_id"] = client_id

    async with httpx.AsyncClient() as client:
        resp = await client.get(url, params=params)
        data = resp.json()
        if data.get("isok"):
            return data["data"]["cookies"]
        raise Exception(f"获取 Cookie 失败: {data.get('msg')}")
```

> 具体的集成方式取决于你的使用场景。CookieBridge 提供标准的 REST API，任何能发 HTTP 请求的语言/工具都可以对接。

## 文档

| 文档 | 说明 |
|------|------|
| [架构设计](docs/architecture.md) | 推拉同步机制、三级降级策略、多账号隔离、服务端分层、数据流链路 |
| [与爬虫的协作集成](docs/integration-with-crawler.md) | 三种 Cookie 来源对比、为什么独立仓库、爬虫端接入方式、快速启动指南 |
| [WebSocket 协议](docs/protocol.md) | 消息类型定义、REST API 端点、错误码、平台域名映射 |

## 项目结构

```
MediaCrawlerPro-CookieBridge/
├── server/                 # Python Tornado 服务端
│   ├── app.py              # 入口文件
│   ├── config.py           # 服务配置
│   ├── router.py           # 路由注册
│   ├── apis/               # REST API Handler
│   │   ├── cookies_handler.py    # Cookie 增删查接口
│   │   ├── accounts_handler.py   # 账号管理接口
│   │   ├── status_handler.py     # 服务状态接口
│   │   └── base_handler.py       # 基础 Handler（CORS、响应格式）
│   ├── logic/
│   │   └── ws_bridge.py    # WebSocket 核心逻辑（CookieStore、ClientRegistry）
│   ├── models/             # Pydantic 数据模型
│   └── constant/           # 常量定义
├── extension/              # Chrome Extension (Manifest V3)
│   ├── manifest.json       # 扩展清单
│   ├── background.js       # Service Worker（WebSocket、Cookie 提取、昵称识别）
│   ├── popup.html          # 弹窗页面
│   ├── popup.js            # 弹窗逻辑
│   ├── popup.css           # 弹窗样式
│   └── icons/              # 扩展图标
└── docs/
    ├── architecture.md     # 架构设计文档
    ├── integration-with-crawler.md  # 与爬虫协作集成文档
    └── protocol.md         # WebSocket 协议文档
```

## 依赖

- **Server**: Python 3.12+, Tornado 6.1, Pydantic 2.x
- **Extension**: Chrome 116+（Manifest V3, chrome.alarms, chrome.cookies）
- **包管理**: 推荐使用 [uv](https://docs.astral.sh/uv/)
