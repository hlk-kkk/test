# CookieBridge 架构设计文档

## 为什么需要 CookieBridge？

做自媒体平台爬虫，绕不开一个核心问题：**Cookie 怎么来？**

传统做法是手动打开浏览器 → F12 → 复制 Cookie → 粘贴到配置文件。这种方式有几个致命缺陷：

1. **繁琐低效**：每个平台都要手动操作，7 个平台就要重复 7 次
2. **时效性差**：Cookie 过期后需要重新手动获取，无法感知过期时间
3. **多账号噩梦**：管理 N 个账号 × M 个平台 = N×M 次手动操作
4. **无法自动化**：CI/CD 场景下完全无法使用

CookieBridge 的设计目标就一个：**让 Cookie 获取完全自动化，你只管在浏览器里登录，Cookie 自动就到了爬虫手里。**

## 整体架构

```
┌─────────────────┐     WebSocket      ┌──────────────────┐     REST API      ┌──────────────────┐
│  Chrome 浏览器   │ ◄──────────────► │  CookieBridge    │ ◄──────────────► │  爬虫程序         │
│  (Extension)     │    实时双向通信     │  Server          │    按需获取       │  (Python)         │
│                  │                    │  (Tornado)       │                   │                   │
│  - 监听 Cookie   │                    │  - Cookie 缓存    │                  │  - 请求 Cookie    │
│  - 主动推送变更   │                    │  - 客户端管理     │                  │  - 注入请求头      │
│  - 检测登录状态   │                    │  - REST API       │                  │  - 账号轮换       │
└─────────────────┘                    └──────────────────┘                   └──────────────────┘
```

三个组件各司其职，通过 WebSocket 和 REST API 两种协议串联。

## 核心设计思想

### 1. 推拉结合的数据同步

CookieBridge 不是单纯的"拉取"模型，而是**推拉结合**：

- **推（Proactive Push）**：浏览器 Cookie 一旦变化，Extension 立即通过 WebSocket 推送给 Server
- **拉（On-Demand Pull）**：爬虫请求 Cookie 时，Server 如果缓存不够新，会主动向 Extension 拉取最新值

```
Cookie变化 → Extension监听到 → 500ms防抖 → WebSocket推送 → Server缓存更新
                                                                    ↑
爬虫请求 → Server检查缓存 → 新鲜？直接返回 / 过期？向Extension拉取 ──┘
```

为什么要推拉结合？

- 纯推：Server 缓存总是最新的，爬虫获取零延迟。但如果 Server 重启，缓存丢失
- 纯拉：每次都要等 Extension 响应，增加延迟
- 推拉结合：日常走缓存（推保证新鲜），异常时能实时补救（拉兜底）

### 2. 三级降级策略

当爬虫请求 Cookie 时，Server 的查询策略是逐级降级的：

```
第一级：新鲜缓存（TTL 5分钟内）
  ↓ 没有
第二级：向在线 Extension 实时拉取（10秒超时）
  ↓ 超时或离线
第三级：过期缓存（有总比没有好）
  ↓ 也没有
返回错误
```

这个设计保证了**最大可用性**：即使 Extension 断线了，只要缓存里有数据，爬虫就不会完全停摆。这在实际使用中非常重要——你不需要 24 小时保持浏览器打开。

### 3. Client ID 多账号隔离

V2 版本引入了 `client_id` 机制来支持多账号：

```
Chrome Profile A  →  Extension实例A  →  client_id: "uuid-aaa"  →  账号甲的 Cookie
Chrome Profile B  →  Extension实例B  →  client_id: "uuid-bbb"  →  账号乙的 Cookie
```

关键设计：
- **client_id = UUID**，在 Extension 首次安装时生成，持久化在 `chrome.storage.local`
- **Cookie 存储双重键**：`(client_id, platform)` 组合作为唯一标识，完全隔离不同账号
- **向后兼容**：API 不传 `client_id` 时，自动返回任意可用账号的 Cookie

为什么用 Chrome Profile 而不是单浏览器多账号？因为大多数平台的 Cookie 是 domain 级别的，同一个浏览器 Profile 下同一平台只能登录一个账号。Chrome Profile 天然提供了完美的隔离环境。

## 服务端架构

### 分层设计

```
server/
├── app.py              # 入口，Tornado Application 初始化
├── router.py           # 路由注册，9 个路由规则
├── config.py           # 环境变量配置
├── context_vars.py     # 请求级别上下文（request_id）
├── apis/               # API 层：处理 HTTP 请求/响应
│   ├── base_handler.py     # 统一响应格式 + CORS
│   ├── cookies_handler.py  # Cookie 增删查
│   ├── accounts_handler.py # 账号管理
│   └── status_handler.py   # 状态查询
├── logic/              # 业务逻辑层
│   └── ws_bridge.py        # WebSocket处理 + CookieStore + ClientRegistry
├── models/             # 数据模型层
│   ├── base_model.py       # 通用响应模型
│   └── cookies_model.py    # Cookie 响应模型
└── constant/           # 常量定义
    ├── base_constant.py    # 平台列表、域名映射
    └── error_code.py       # 错误码枚举
```

遵循 Tornado 项目的分层惯例：Handler 只负责参数校验和响应组装，业务逻辑在 logic 层，数据结构在 models 层。

### 核心数据结构

**CookieStore — 内存级二级缓存**

```python
# 存储结构：(client_id, platform) → cookie 数据
{
    ("uuid-aaa", "xhs"): {
        "cookies": "web_session=xxx; a1=yyy",
        "timestamp": 1710000000.0,
        "source": "extension",    # 存储层只有 extension | manual 两种
        "valid": True
    }
}
```

为什么不用 Redis/数据库？

- Cookie 本身是临时性数据，浏览器关闭就可能失效
- 内存存储足够，7 个平台 × N 个账号的数据量极小
- 无状态设计让部署更简单，不依赖外部服务
- Server 重启后 Extension 会自动重连并推送最新数据

**ClientRegistry — 客户端注册表**

```python
# 存储结构：client_id → 客户端元信息
{
    "uuid-aaa": {
        "platforms": {"xhs": {"has_cookies": True, "cookie_count": 15}, ...},
        "nicknames": {"xhs": "用户昵称", "bili": "UP主名"},
        "connected": True,
        "connected_at": 1710000000.0,
        "last_seen": 1710000024.0
    }
}
```

Registry 和 Store 分离的好处：即使 Extension 断线，Registry 仍保留账号信息，Status API 能展示历史注册过的所有客户端。

### WebSocket 双向通信

```python
# ExtensionWebSocketHandler 类级别的四个核心字典：

_connections: dict[str, ExtensionWebSocketHandler]  # client_id → 当前连接
_pending_requests: dict[str, asyncio.Future]         # request_id → 等待响应的 Future
_bulk_collectors: dict[str, dict]                    # request_id → 批量请求的收集器（get_all_cookies 用）
_request_client_map: dict[str, str]                  # request_id → client_id（断线清理用）
```

Server 向 Extension 发请求时，生成 `request_id`，创建 `Future`，发送 WebSocket 消息，然后 `await` 这个 Future。Extension 响应时带上相同的 `request_id`，Server 收到后 resolve 对应的 Future。单平台请求超时 10 秒，全平台批量请求超时 15 秒。

当 Extension 断线时（`on_close`），Server 会通过 `_request_client_map` 找到该客户端所有挂起的 Future 并取消，避免资源泄漏。

这个模式让异步的 WebSocket 通信拥有了类似 HTTP 请求-响应的语义，代码写起来非常直观。

## Extension 架构

### Manifest V3 Service Worker

Extension 使用 Chrome Manifest V3 标准，核心逻辑在 `background.js`（Service Worker）：

```
background.js 职责：
├── WebSocket 生命周期管理（连接、重连、心跳）
├── Cookie 变更监听（chrome.cookies.onChanged）
├── 平台 Cookie 提取（chrome.cookies.getAll）
├── 登录状态检测（检查关键 Cookie 是否存在）
├── 昵称获取（调用平台 API，尽力而为）
└── 与 Popup UI 通信（chrome.runtime.onMessage）
```

关键设计点：

- **防抖推送**：Cookie 变化后等 500ms 再推送，避免登录过程中频繁触发
- **指数退避重连**：断线后从 1s 开始重试，最大 30s，避免 Server 未启动时的无意义请求
- **心跳保活**：每 24 秒一次，通过 `chrome.alarms` 实现（Service Worker 可能被挂起，alarm 能唤醒）

### 七大平台支持

| 平台 | 域名 | 登录检测 Cookie |
|------|------|----------------|
| 小红书 | .xiaohongshu.com | `web_session` |
| 抖音 | .douyin.com | `ttwid` |
| 快手 | .kuaishou.com | `kuaishou.server.web_st` |
| B站 | .bilibili.com | `bili_ticket` |
| 微博 | .weibo.com / .weibo.cn | `SUBP` |
| 贴吧 | .tieba.baidu.com | `BDUSS` |
| 知乎 | .zhihu.com | `z_c0` |

每个平台通过"关键 Cookie"来判断是否已登录。这些 Cookie 是各平台认证体系的核心标识，存在即表示有效会话。

## API 设计

### 统一响应格式

```json
{
    "biz_code": 0,
    "msg": "OK!",
    "isok": true,
    "data": { ... }
}
```

所有 API 都遵循这个格式，`biz_code` 为 0 表示成功，非零表示业务错误。API 响应中的 `source` 字段有四种值：`extension`（实时拉取）、`cache`（新鲜缓存）、`stale_cache`（过期缓存）、`manual`（手动设置）。

错误响应格式略有不同，`data` 字段替换为 `extra`：

```json
{
    "biz_code": 100,
    "msg": "Extension not connected",
    "isok": false,
    "extra": null
}
```

### 错误码

| 错误码 | 含义 |
|--------|------|
| 0 | 成功 |
| 3 | 无效参数 |
| 4 | 缺少参数 |
| 100 | Extension 未连接 |
| 101 | Cookie 未找到 |
| 102 | 请求超时 |
| 103 | 账号未找到 |

### 核心端点

| 端点 | 用途 | 典型调用方 |
|------|------|-----------|
| `GET /ping` | 健康检查 | 监控 |
| `GET /api/cookies/{platform}` | 获取指定平台 Cookie | 爬虫程序 |
| `GET /api/cookies/all` | 获取所有平台 Cookie | 爬虫程序 |
| `POST /api/cookies/{platform}/set` | 手动设置 Cookie | 测试/兜底 |
| `GET /api/accounts` | 列出所有账号 | 管理/调试 |
| `GET /api/accounts/{client_id}` | 账号详情 | 管理/调试 |
| `GET /api/status` | 服务状态 | 监控 |
| `WS /ws` | WebSocket 连接 | Chrome Extension |

### 向后兼容设计

`GET /api/cookies/{platform}` 的 `client_id` 参数是可选的：

- **传 client_id**：返回指定账号的 Cookie（多账号场景）
- **不传 client_id**：自动选择任意有效账号的 Cookie（单账号场景，向后兼容）

这让从 V1 升级到 V2 时，爬虫端可以零修改接入。

## 数据流完整链路

```
1. 用户在浏览器登录小红书
        ↓
2. Chrome 设置 Cookie（web_session=xxx）
        ↓
3. Extension 监听到 chrome.cookies.onChanged
        ↓
4. 500ms 防抖后，提取 .xiaohongshu.com 全部 Cookie
        ↓
5. 格式化为字符串："web_session=xxx; a1=yyy; ..."
        ↓
6. WebSocket 发送 cookies_update 消息
        ↓
7. Server 存入 CookieStore[(client_id, "xhs")]
        ↓
8. 爬虫调用 GET /api/cookies/xhs
        ↓
9. Server 返回缓存的 Cookie 字符串
        ↓
10. 爬虫注入到 HTTP 请求头发起爬取
```

整条链路从用户登录到爬虫拿到 Cookie，全程自动，无需人工干预。

### 链路二：按需拉取（缓存过期时）

```
1. 爬虫调用 GET /api/cookies/xhs
        ↓
2. Server 检查 CookieStore → 缓存过期（超过 TTL）
        ↓
3. Server 通过 WebSocket 发送 get_cookies 请求给 Extension
        ↓
4. Extension 调用 chrome.cookies.getAll({domain: ".xiaohongshu.com"})
        ↓
5. Extension 返回 cookies_response（带 request_id）
        ↓
6. Server resolve 对应 Future，更新 CookieStore
        ↓
7. 返回最新 Cookie 给爬虫（source="extension"）
```

如果 Extension 离线或超时，Server 还会尝试返回过期缓存（source="stale_cache"），保证最大可用性。

## 环境变量配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `IS_DEBUG` | `True` | 调试模式 |
| `LOGGER_LEVEL` | `INFO` | 日志级别 |
| `APP_HOST` | `127.0.0.1` | 监听地址 |
| `APP_PORT` | `8274` | 监听端口 |
| `CACHE_TTL` | `300` | Cookie 缓存新鲜度（秒） |
| `WS_REQUEST_TIMEOUT` | `10` | WebSocket 单次请求超时（秒） |
