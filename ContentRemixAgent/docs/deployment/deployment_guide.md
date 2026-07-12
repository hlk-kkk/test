# 部署指南

## 前置条件

**必需**
- Python 3.10–3.12、[uv](https://docs.astral.sh/uv/)、Node.js + npm、MySQL 8.0+、ffmpeg

**外部服务**
- **sign-srv** 和 **DownloadServer** 需提前部署并启动
  - [MediaCrawlerPro-Downloader](https://github.com/MediaCrawlerPro/MediaCrawlerPro-Downloader)（默认端口 8205）
  - [MediaCrawlerPro-SignSrv](https://github.com/MediaCrawlerPro/MediaCrawlerPro-SignSrv)（默认端口 8989）
- 三个仓库需克隆到**同一父目录**下（Docker Compose 通过相对路径引用）：

```
~/workspace/
├── MediaCrawlerPro-ContentRemixAgent/   # 本项目
├── MediaCrawlerPro-Downloader/          # 内容下载服务
└── MediaCrawlerPro-SignSrv/             # 平台签名服务
```

<details>
<summary>🤖 不想手动装？让 AI Coding Agent 帮你搞定</summary>

如果你本地有 [Claude Code](https://claude.ai/code)、[Cursor](https://cursor.com) 等 AI 编程助手，可以直接把项目目录打开，粘贴以下提示词，让 Agent 自动分析项目并安装所有依赖：

```
请帮我安装这个项目的所有依赖并启动开发环境。具体步骤：

1. 阅读项目根目录的 .env.example 和 docs/deployment/deployment_guide.md 了解项目结构
2. 检查本机是否已安装 Python 3.10+、uv、Node.js、MySQL、ffmpeg，缺少的请给出安装命令
3. 复制 .env.example 为 .env，根据本机环境填写数据库连接信息
4. 安装后端依赖（backend 目录，使用 uv sync）
5. 安装前端依赖（frontend 目录，使用 npm install）
6. 尝试运行 ./start.sh 启动项目，如果有报错请帮我修复
```

Agent 会自动读取项目配置文件，识别缺少的系统依赖并引导你完成安装。

</details>

## 方式一：本地开发（推荐）

```bash
cp .env.example .env   # 编辑数据库密码、DOWNLOAD_SERVER_BASE
./start.sh             # 一键启动前后端
```

<details>
<summary>Windows 用户</summary>

```cmd
copy .env.example .env
start.bat
```

</details>

- 前端：http://localhost:5373
- 后端：http://localhost:8001
- 启动后在前端「设置 → 大模型配置」配置 LLM

> 数据库表会在首次启动时自动创建，无需手动建表。

## 方式二：Docker Compose

```bash
cp .env.example .env                   # 编辑配置
docker-compose up -d --build           # 构建并启动所有服务
```

- 前端：http://localhost
- 后端：http://localhost:8001

首次启动会自动构建镜像、初始化数据库、执行迁移。

## 最少配置项（.env）

| 配置项 | 说明 |
|--------|------|
| `AGENT_DB_*` | MySQL 连接信息（host/port/user/password/name） |
| `DOWNLOAD_SERVER_BASE` | DownloadServer 地址，默认 `http://localhost:8205` |
| `JWT_SECRET_KEY` | 生产环境必填，至少 32 字符 |
| `ASR_BACKEND` | `bcut`（默认，B站云端 API）或 `funasr`（本地模型） |

**LLM 配置**：推荐在前端设置页配置（数据库持久化），`.env` 中的 `LLM_PROVIDER` + API Key 仅作兜底。

## 验证

```bash
curl http://localhost:8989/signsrv/pong   # sign-srv
curl http://localhost:8205/health          # DownloadServer
curl http://localhost:8001/health          # Backend
```

## 生产部署清单

- [ ] `JWT_SECRET_KEY` 设为安全随机值
- [ ] `MEMORY_BACKEND=mysql`
- [ ] LLM 配置写入数据库并激活
- [ ] 平台 Cookies 已配置（`platform_cookies` 表）
- [ ] `ASSETS_DIR` 挂载到持久化磁盘
- [ ] 前端通过 Nginx 反代 `/api` 与 `/assets`

## 常见问题

**启动时报 DownloadServer 不可用**
- 检查 `DOWNLOAD_SERVER_BASE` 是否正确，确认 DownloadServer 与 sign-srv 已启动

**抓取内容失败（COOKIES_NOT_FOUND）**
- 检查 `media_crawler_pro.platform_cookies` 表是否有对应平台的有效 Cookies

**转录失败**
- FunASR：检查模型是否下载完成；Bcut：检查网络连通性

**LLM 无法调用**
- 确认前端设置里已有激活配置，或 `.env` 中配置了 provider 与 key
