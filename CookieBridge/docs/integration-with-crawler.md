# CookieBridge 与爬虫的协作集成

## 背景：爬虫的 Cookie 困境

MediaCrawlerPro-Python 是一个支持 7 大自媒体平台的数据采集框架。每个平台的 API 请求都需要携带合法的用户 Cookie 才能通过认证。

在没有 CookieBridge 之前，爬虫项目提供了两种 Cookie 来源：

| 方式 | 流程 | 痛点 |
|------|------|------|
| **Excel** | 手动复制 Cookie → 粘贴到 xlsx 文件 → 爬虫读取 | 频繁过期需反复操作，无状态追踪 |
| **MySQL** | 手动复制 Cookie → 写入数据库 → 爬虫读取 | 有状态追踪，但获取 Cookie 仍然是手动的 |

**核心矛盾**：不管存储介质多高级，"从浏览器复制 Cookie"这一步始终是手动的。

CookieBridge 解决的就是这最后一公里：**让 Cookie 从浏览器到爬虫全程自动流转。**

## 三种模式对比

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Cookie 来源对比                                    │
├──────────┬──────────────┬──────────────┬───────────────────────────┤
│          │   Excel      │   MySQL       │   CookieBridge            │
├──────────┼──────────────┼──────────────┼───────────────────────────┤
│ 获取方式  │ 手动复制粘贴  │ 手动写入数据库 │ 浏览器登录即自动同步        │
│ 更新方式  │ 手动替换文件  │ 手动UPDATE    │ Cookie变化实时推送          │
│ 过期感知  │ 无           │ 有(status字段) │ 有(登录检测Cookie)          │
│ 多账号    │ 多行数据     │ 多行记录       │ 多Chrome Profile自动管理   │
│ 持久化    │ 文件         │ 数据库         │ 内存(重启后自动恢复)        │
│ 适用场景  │ 快速测试     │ 生产环境       │ 自动化/频繁更新场景         │
│ 外部依赖  │ 无           │ MySQL         │ CookieBridge Server       │
└──────────┴──────────────┴──────────────┴───────────────────────────┘
```

## 集成架构

```
                        MediaCrawlerPro 生态
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ┌────────────────┐        ┌─────────────────────────────────┐   │
│  │ CookieBridge   │        │  MediaCrawlerPro-Python          │  │
│  │                │        │                                   │  │
│  │  Extension ──WebSocket──► Server ──REST API──► AccountPool │  │
│  │  (浏览器端)     │        │  (Port 8274)        (内存)       │  │
│  │                │        │                       │           │  │
│  └────────────────┘        │                  ┌────▼────┐      │  │
│                            │                  │ Platform │      │  │
│                            │                  │ Client   │      │  │
│                            │                  │ (XHS/DY/ │      │  │
│                            │                  │  KS/...) │      │  │
│                            │                  └────┬─────┘      │  │
│                            │                       │            │  │
│                            │                  HTTP请求+Cookie    │  │
│                            │                       ▼            │  │
│                            │                  目标平台API        │  │
│                            └─────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

## 爬虫端如何接入

### 配置切换

在 MediaCrawlerPro-Python 中，只需修改一个配置即可切换到 CookieBridge：

```python
# config/base_config.py
ACCOUNT_POOL_SAVE_TYPE = "cookie_bridge"   # 可选: "xlsx" | "mysql" | "cookie_bridge"
COOKIE_BRIDGE_URL = "http://localhost:8274" # CookieBridge Server 地址
```

或通过环境变量：

```bash
export ACCOUNT_POOL_SAVE_TYPE=cookie_bridge
export COOKIE_BRIDGE_URL=http://localhost:8274
```

### 加载流程

爬虫启动时的 Cookie 加载流程：

```python
# pkg/account_pool/pool.py 中的核心逻辑（简化）

async def load_accounts_from_cookie_bridge(platform):
    # 1. 获取所有已注册的客户端
    accounts = GET(f"{COOKIE_BRIDGE_URL}/api/accounts")

    for client in accounts["data"]["accounts"]:
        # 2. 检查该客户端是否有目标平台的 Cookie
        if client["platforms"][platform]["has_cookies"]:
            # 3. 获取具体的 Cookie 字符串
            cookies = GET(f"{COOKIE_BRIDGE_URL}/api/cookies/{platform}?client_id={client['client_id']}")

            # 4. 构建账号模型，加入内存池
            account = AccountInfoModel(
                account_name=client["nicknames"].get(platform, client["client_id"]),
                cookies=cookies["data"]["cookies"],   # 纯字符串格式
                platform_name=platform,
                status=NORMAL
            )
            account_pool.append(account)
```

### 运行时的账号管理

加载完成后，CookieBridge 的账号和 Excel/MySQL 的账号在使用上**完全一致**：

```python
# 对平台 Client 来说，Cookie 来源是透明的
class XhsClient:
    @property
    def headers(self):
        return {
            "cookie": self.account_info.account.cookies,  # 不管来自哪里，都是字符串
            ...
        }
```

账号轮换逻辑也是通用的：

```
请求失败(5次重试) → 标记账号失效 → 从池中取下一个账号 → 继续请求
```

区别在于状态持久化：
- **MySQL 模式**：失效状态写回数据库，下次启动不会再用
- **CookieBridge 模式**：失效状态只在内存中，下次启动重新从浏览器拉取新鲜 Cookie

这正是 CookieBridge 的优势——用户只要在浏览器保持登录，重启爬虫就能拿到最新的 Cookie，不需要手动处理失效问题。

## 为什么要拆成独立仓库？

CookieBridge 没有作为 MediaCrawlerPro-Python 的子目录，而是独立仓库，这是刻意的设计：

### 1. 技术栈差异

| | CookieBridge | MediaCrawlerPro-Python |
|---|---|---|
| 语言 | Python + JavaScript | Python |
| 框架 | Tornado + Chrome Extension | httpx + Playwright |
| 运行环境 | 需要 Chrome 浏览器 | 纯后端，可 Docker 部署 |
| 依赖 | tornado, pydantic | httpx, playwright, sqlalchemy, ... |

强行合并会让依赖管理变得混乱，且给不需要 CookieBridge 的用户增加不必要的安装负担。

### 2. 生命周期独立

- CookieBridge 是一个**常驻服务**，启动后持续运行
- 爬虫是**按需运行**的，跑完任务就退出

```
时间线：
CookieBridge:  ████████████████████████████████████████  (持续运行)
爬虫任务1:         ████                                    (5分钟)
爬虫任务2:                     ██████                       (10分钟)
爬虫任务3:                                    ███           (3分钟)
```

一个 CookieBridge 实例可以服务多次爬虫运行，甚至可以同时服务多个不同的爬虫项目。

### 3. 部署灵活性

独立仓库意味着：
- CookieBridge 可以部署在有浏览器的工作机上
- 爬虫可以部署在远端服务器上
- 只需要网络互通（修改 `COOKIE_BRIDGE_URL` 指向正确地址）

```
工作电脑 (有Chrome)                  云服务器 (无GUI)
┌──────────────┐                    ┌──────────────┐
│ Chrome       │                    │ 爬虫程序      │
│ Extension    │                    │              │
│    ↕         │                    │  GET /api/   │
│ CookieBridge │◄───── HTTP ───────│  cookies/xhs │
│ Server       │                    │              │
└──────────────┘                    └──────────────┘
```

### 4. 可替换性

CookieBridge 只是 AccountPool 的一种数据源。爬虫通过配置 `ACCOUNT_POOL_SAVE_TYPE` 切换，代码无需改动。这意味着：

- 不想用 CookieBridge？切回 Excel 或 MySQL 即可
- 想自己实现 Cookie 管理？只要提供相同的 REST API 接口就能无缝替换
- CookieBridge 挂了？爬虫降级到其他 Cookie 来源继续工作

## 协作的好处总结

### 对于日常使用

1. **零操作获取 Cookie**：浏览器登录一次，Cookie 自动到位
2. **多账号开箱即用**：每个 Chrome Profile 自动识别为独立账号
3. **实时状态感知**：Extension Popup 直观显示哪些平台已登录、Cookie 数量

### 对于运维

1. **无需 Playwright**：传统方案需要 Playwright 启动浏览器模拟登录，CookieBridge 利用真实浏览器会话
2. **Cookie 自动续期**：用户正常使用浏览器时，平台会自动续期 Cookie，CookieBridge 实时同步
3. **监控友好**：`/api/status` 端点可接入监控系统，感知 Cookie 可用性

### 对于代码架构

1. **关注点分离**：Cookie 获取和爬虫逻辑彻底解耦
2. **统一接口**：不管 Cookie 从哪来，爬虫端的 `AccountInfoModel` 接口一致
3. **独立演进**：CookieBridge 可以独立迭代（加新平台、优化缓存策略），不影响爬虫代码

## 快速启动

### 第一步：启动 CookieBridge Server

```bash
cd MediaCrawlerPro-CookieBridge
./start.sh          # macOS/Linux
# 或
start.bat           # Windows
```

Server 将在 `http://localhost:8274` 启动。

### 第二步：安装 Chrome Extension

1. Chrome 打开 `chrome://extensions/`
2. 开启「开发者模式」
3. 点击「加载已解压的扩展程序」
4. 选择 `extension/` 目录

### 第三步：登录目标平台

在浏览器中正常访问并登录你要爬取的平台（小红书、抖音等）。Extension Popup 会显示登录状态。

### 第四步：配置爬虫

```bash
cd MediaCrawlerPro-Python
export ACCOUNT_POOL_SAVE_TYPE=cookie_bridge
uv run main.py --platform xhs --type search --keywords "关键词"
```

爬虫自动从 CookieBridge 获取 Cookie，开始采集。

### 多账号扩展

需要多个账号？

1. Chrome 菜单 → 「添加个人资料」创建新 Profile
2. 在新 Profile 中安装 Extension
3. 用另一个账号登录同一平台
4. 爬虫启动时自动加载所有账号到池中轮换使用
