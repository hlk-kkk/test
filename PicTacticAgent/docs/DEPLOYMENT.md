# PicTacticAgent 生产部署指南

本文档用于将 PicTacticAgent 部署到远程 Linux 服务器。面向 AI Agent 或运维人员，包含所有必要的上下文和操作步骤。

---

## 项目背景

PicTacticAgent 是一个 AI 图片/视频生成工具，技术栈：

- **后端**：Python 3.11 + FastAPI + LangGraph，SQLite 数据库，uvicorn 运行在 8019 端口
- **前端**：React 19 + Vite SPA（非 Next.js），构建产物是纯静态文件（`dist/`）
- **图片生成**：通过 jiekou.ai 代理调用 Gemini API，生成的图片保存在本地磁盘
- **视频生成**：通过 jiekou.ai 代理调用 Kling/Sora/Veo，视频保存在本地磁盘
- **认证**：JWT + bcrypt，支持邮箱注册登录 + GitHub OAuth + Google OAuth

部署目标：给家人使用，少量用户，不需要高可用。

---

## 架构

```
Internet → [宿主机 Nginx :443 HTTPS] → [Docker Network]
                                          ├── nginx 容器 :80
                                          │   ├── /*           → 前端静态文件（Vite build 产物）
                                          │   ├── /api/*       → proxy_pass → backend:8019
                                          │   ├── /ws/*        → WebSocket 代理 → backend:8019
                                          │   ├── /static/*    → 从宿主机目录直接服务图片/视频
                                          │   └── /health      → proxy_pass → backend:8019
                                          │
                                          └── backend 容器 :8019（仅容器网络内部可达）
                                              ├── FastAPI + LangGraph（uvicorn）
                                              ├── SQLite 数据库 → 宿主机 ./data/db/
                                              ├── 生成的图片 → 宿主机 ./data/storage/images/
                                              └── 生成的视频 → 宿主机 ./data/storage/videos/
```

**两层 Nginx**：
- 宿主机 Nginx：负责 HTTPS 终止 + 反代到 Docker 容器的 80 端口（由部署者自行配置）
- Docker 内 Nginx 容器：负责前端静态文件服务 + API/WebSocket 反代 + 静态媒体文件服务

---

## 前置条件

服务器上需要已安装：

- Docker Engine（20.10+）
- Docker Compose V2（`docker compose` 命令，非 `docker-compose`）
- Nginx（宿主机上，用于 HTTPS 终止和反代）
- Git

确认方式：

```bash
docker --version        # Docker version 20.10+
docker compose version  # Docker Compose version v2.x
nginx -v                # nginx version
git --version
```

---

## 部署步骤

### 1. 克隆代码

```bash
cd /opt  # 或你偏好的目录
git clone https://github.com/NanmiCoder/PicTacticAgent.git
cd PicTacticAgent
```

### 2. 创建 .env.production

项目根目录下有一个 `.env.production` 模板文件。复制并修改：

```bash
cp .env.production .env.production.bak  # 备份模板
```

编辑 `.env.production`，必须修改以下字段：

```env
# ===== 必须修改 =====
JWT_SECRET_KEY=<替换为随机值>
JIEKOUAI_API_KEY=<替换为真实 API Key>

# ===== 域名（替换 yourdomain.com 为真实域名）=====
FRONTEND_URL=https://yourdomain.com
CORS_ORIGINS=["https://yourdomain.com"]

# ===== Provider =====
DEFAULT_PROVIDER=jiekouai
JIEKOUAI_API_BASE_URL=https://api.jiekou.ai/v3
JIEKOUAI_LLM_BASE_URL=https://api.jiekou.ai/openai
JIEKOUAI_LLM_MODEL=gpt-5-mini

# ===== OAuth（可选，不配置则对应登录方式不可用）=====
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_CALLBACK_URL=https://yourdomain.com/auth/callback/github
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_CALLBACK_URL=https://yourdomain.com/auth/callback/google

# ===== 内部 API Key =====
INTERNAL_API_KEY=

# ===== 存储（Docker 容器内的路径，不要修改）=====
STORAGE_PATH=/app/storage/images
DB_PATH=/app/data/pictactic.db

# ===== 视频 =====
DEFAULT_VIDEO_PROVIDER=kling
```

**生成随机密钥**：

```bash
# JWT_SECRET_KEY
openssl rand -base64 32

# INTERNAL_API_KEY（可留空，后端会自动生成，但 Docker 中建议显式设置）
openssl rand -base64 32
```

**安全校验**：后端启动时会检查 `JWT_SECRET_KEY`，如果是默认值、长度不足 32、或以 `<` 开头（模板占位符），将**拒绝启动**并报错。

### 3. 创建数据目录

Docker Compose 使用宿主机绑定挂载，数据存储在 `docker/data/` 下：

```bash
mkdir -p docker/data/storage/images
mkdir -p docker/data/storage/videos
mkdir -p docker/data/db
```

这些目录会持久化所有生成的图片、视频和 SQLite 数据库。备份时只需备份 `docker/data/` 目录。

### 4. 构建并启动容器

```bash
cd docker
docker compose up --build -d
```

首次构建约需 3-5 分钟（下载基础镜像 + npm install + pip install）。

**查看启动状态**：

```bash
docker compose ps
```

预期输出：两个容器均为 `running` 状态，backend 的 Health 列显示 `healthy`。

**查看日志**：

```bash
# 所有容器
docker compose logs -f

# 仅后端
docker compose logs -f backend

# 仅 nginx
docker compose logs -f nginx
```

**验证后端健康**：

```bash
# 从宿主机直接访问 Docker nginx 容器
curl http://localhost:80/health
# 预期返回: {"status":"ok"}
```

### 5. 配置宿主机 Nginx（HTTPS 反代）

这一步由部署者根据自己的 HTTPS 证书方案配置。以下是参考配置：

```nginx
server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    # 普通 HTTP 请求
    location / {
        proxy_pass http://127.0.0.1:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 需要特殊处理
    location /ws/ {
        proxy_pass http://127.0.0.1:80;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
    }
}

# HTTP → HTTPS 重定向
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$host$request_uri;
}
```

配置完成后：

```bash
nginx -t          # 测试配置语法
systemctl reload nginx  # 重新加载
```

### 6. 配置 OAuth（可选）

#### GitHub OAuth

1. 访问 https://github.com/settings/developers → "OAuth Apps" → "New OAuth App"
2. 填写：
   - **Application name**: PicTacticAgent
   - **Homepage URL**: `https://yourdomain.com`
   - **Authorization callback URL**: `https://yourdomain.com/auth/callback/github`
3. 创建后获取 Client ID 和 Client Secret
4. 填入 `.env.production` 的 `GITHUB_CLIENT_ID` 和 `GITHUB_CLIENT_SECRET`

#### Google OAuth

1. 访问 https://console.cloud.google.com/apis/credentials
2. 创建 "OAuth 2.0 Client ID"，类型选 "Web application"
3. 添加 **Authorized redirect URIs**: `https://yourdomain.com/auth/callback/google`
4. 获取 Client ID 和 Client Secret
5. 填入 `.env.production` 的 `GOOGLE_CLIENT_ID` 和 `GOOGLE_CLIENT_SECRET`

**OAuth 回调流程说明**：
- 前端路由 `/auth/callback/:provider` 接收 OAuth 重定向
- 前端拿到 `code` 参数后调用后端 `POST /api/v1/auth/oauth/{provider}/callback`
- 后端用 `code` 向 GitHub/Google 换取用户信息，创建或关联用户

修改 `.env.production` 后需要重启容器生效：

```bash
cd docker
docker compose up -d  # 重新读取 env_file
```

### 7. 注册第一个用户

OAuth 配置好之后，直接访问 `https://yourdomain.com/auth/register` 注册账号即可（支持邮箱注册或 OAuth 登录）。

不需要 SMTP 邮件服务，邮箱注册无需验证即可使用。

---

## 目录结构说明

部署后，项目目录结构：

```
PicTacticAgent/
├── .env.production          # 生产环境配置（敏感，已 gitignore）
├── docker/
│   ├── docker-compose.yml   # 容器编排
│   ├── Dockerfile           # 后端镜像
│   ├── Dockerfile.frontend  # 前端 + Nginx 镜像
│   ├── nginx.conf           # 容器内 Nginx 配置
│   └── data/                # 持久化数据（宿主机绑定挂载）
│       ├── storage/
│       │   ├── images/      # 生成的图片（UUID 文件名，如 gemini_a1b2c3d4_0.jpg）
│       │   └── videos/      # 生成的视频
│       └── db/
│           └── pictactic.db # SQLite 数据库
├── backend/                 # Python 源码（构建时复制进镜像）
├── frontend/                # React 源码（构建时 npm build 进镜像）
└── ...
```

---

## 文件存储机制

- 所有 provider（jiekouai/gemini/antigravity）生成图片后，都下载到本地并保存为 JPEG
- API 返回相对路径 URL 如 `/static/images/gemini_a1b2c3d4_0.jpg`，不依赖外部 CDN
- 视频同理，URL 格式为 `/static/videos/xxx.mp4`
- Docker 内 Nginx 容器通过 `alias` 直接从挂载的 volume 提供静态文件服务
- 宿主机路径：`docker/data/storage/images/` 和 `docker/data/storage/videos/`
- 不会自动清理文件，需要手动管理磁盘空间

---

## 数据备份

所有数据都在宿主机 `docker/data/` 目录下，备份该目录即可：

```bash
# 简单备份（建议用 cron 定期执行）
tar czf /backup/pictactic-$(date +%Y%m%d).tar.gz -C /opt/PicTacticAgent/docker data/
```

SQLite 数据库在线备份（更安全，不会复制到不一致状态）：

```bash
# 使用 SQLite 的 .backup 命令
sqlite3 /opt/PicTacticAgent/docker/data/db/pictactic.db ".backup /backup/pictactic-db-$(date +%Y%m%d).db"
```

---

## 容器内 Nginx 配置说明

配置文件：`docker/nginx.conf`，被复制到容器内 `/etc/nginx/conf.d/default.conf`。

| 路径 | 处理方式 | 说明 |
|------|---------|------|
| `/api/v1/auth/login` | rate limit 5r/m + proxy → backend:8019 | 登录防暴力破解 |
| `/api/v1/auth/register` | rate limit 5r/m + proxy → backend:8019 | 注册防滥用 |
| `/api/v1/generation/` | rate limit 10r/m + proxy → backend:8019 | 图片生成限速，timeout 300s |
| `/api/v1/video/` | rate limit 10r/m + proxy → backend:8019 | 视频生成限速，timeout 300s |
| `/api/*` | proxy → backend:8019 | 其他 API（设置、模板等） |
| `/ws/*` | WebSocket proxy → backend:8019 | 实时进度推送，timeout 86400s |
| `/health` | proxy → backend:8019 | 健康检查 |
| `/static/images/*` | 直接从 volume 提供文件 | 7 天缓存 |
| `/static/videos/*` | 直接从 volume 提供文件 | 7 天缓存 |
| `/*` | 前端 SPA 静态文件 | `try_files` fallback 到 `index.html` |

安全 headers：`server_tokens off`、`X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy`。

---

## 常用运维命令

```bash
cd /opt/PicTacticAgent/docker

# 启动
docker compose up -d

# 停止
docker compose down

# 重启（修改 .env.production 后）
docker compose up -d

# 重新构建（代码更新后）
git pull
docker compose up --build -d

# 查看日志
docker compose logs -f backend
docker compose logs -f nginx

# 查看容器状态
docker compose ps

# 进入后端容器调试
docker compose exec backend bash

# 查看磁盘使用
du -sh docker/data/storage/images/
du -sh docker/data/storage/videos/
du -sh docker/data/db/
```

---

## 更新部署

```bash
cd /opt/PicTacticAgent
git pull origin main
cd docker
docker compose up --build -d
```

数据不会丢失，因为持久化数据在宿主机 `docker/data/` 目录，不在镜像内。

---

## 故障排查

### 后端启动失败

```bash
docker compose logs backend
```

常见错误：

| 错误信息 | 原因 | 解决 |
|---------|------|------|
| `RuntimeError: JWT_SECRET_KEY must be set` | `.env.production` 中 JWT_SECRET_KEY 未修改 | 用 `openssl rand -base64 32` 生成并填入 |
| `ModuleNotFoundError` | pip install 失败 | 重新构建：`docker compose build --no-cache backend` |
| `sqlite3.OperationalError: unable to open database file` | 数据目录权限问题 | `chmod -R 777 docker/data/db/` |

### 前端无法访问

```bash
# 检查 nginx 容器日志
docker compose logs nginx

# 检查容器是否在运行
docker compose ps

# 测试 Docker 内部连通性
docker compose exec nginx curl http://backend:8019/health
```

### WebSocket 连接失败

确保宿主机 Nginx 配置了 WebSocket 代理（`Upgrade` + `Connection` headers），参见上方第 5 步的 `/ws/` location 配置。

### OAuth 登录失败

1. 检查 `.env.production` 中的 Client ID / Secret 是否正确
2. 检查 GitHub/Google 控制台中的回调 URL 是否与 `.env.production` 中 `*_CALLBACK_URL` 完全一致
3. 回调 URL 格式必须是 `https://yourdomain.com/auth/callback/github`（或 google）
4. 查看后端日志：`docker compose logs backend | grep OAuth`

### 图片/视频无法显示

```bash
# 检查文件是否存在
ls -la docker/data/storage/images/
ls -la docker/data/storage/videos/

# 检查 nginx 能否访问挂载目录
docker compose exec nginx ls /app/storage/images/

# 测试静态文件服务
curl -I http://localhost:80/static/images/<filename>.jpg
```

---

## 端口占用总结

| 端口 | 用途 | 暴露范围 |
|------|------|---------|
| 宿主机 443 | 宿主机 Nginx HTTPS | 公网 |
| 宿主机 80 | Docker nginx 容器映射 | 宿主机本地（被宿主机 Nginx 反代） |
| 容器 8019 | FastAPI 后端 | 仅 Docker 内部网络（`expose`，不映射宿主机） |

如果宿主机 80 端口被占用（例如宿主机 Nginx 已监听 80），修改 `docker-compose.yml` 的端口映射：

```yaml
ports:
  - "8080:80"  # 改为宿主机 8080 映射到容器 80
```

然后宿主机 Nginx 反代目标改为 `http://127.0.0.1:8080`。

---

## .env.production 完整字段参考

| 字段 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `JWT_SECRET_KEY` | 是 | 无（必须修改） | JWT 签名密钥，`openssl rand -base64 32` 生成 |
| `JIEKOUAI_API_KEY` | 是 | 无 | jiekou.ai API Key，用于图片/视频生成和 LLM 评估 |
| `FRONTEND_URL` | 是 | `http://localhost:3019` | 前端 URL，用于 CORS 和 OAuth 重定向 |
| `CORS_ORIGINS` | 是 | `[]` | CORS 允许的源，JSON 数组格式 |
| `DEFAULT_PROVIDER` | 否 | `gemini` | 图片生成默认 provider，建议 `jiekouai` |
| `JIEKOUAI_API_BASE_URL` | 否 | `https://api.jiekou.ai/v3` | jiekou.ai 图片/视频 API 地址 |
| `JIEKOUAI_LLM_BASE_URL` | 否 | `https://api.jiekou.ai/openai` | jiekou.ai LLM API 地址（OpenAI 兼容） |
| `JIEKOUAI_LLM_MODEL` | 否 | `gpt-4o` | LLM 评估模型 |
| `GITHUB_CLIENT_ID` | 否 | `""` | GitHub OAuth Client ID |
| `GITHUB_CLIENT_SECRET` | 否 | `""` | GitHub OAuth Client Secret |
| `GITHUB_CALLBACK_URL` | 否 | 本地地址 | GitHub OAuth 回调 URL |
| `GOOGLE_CLIENT_ID` | 否 | `""` | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | 否 | `""` | Google OAuth Client Secret |
| `GOOGLE_CALLBACK_URL` | 否 | 本地地址 | Google OAuth 回调 URL |
| `INTERNAL_API_KEY` | 否 | 自动生成 | 内部 API Key，Docker 中建议显式设置 |
| `STORAGE_PATH` | 否 | `/app/storage/images` | 图片存储路径（Docker 内路径，不要改） |
| `DB_PATH` | 否 | `/app/data/pictactic.db` | SQLite 路径（Docker 内路径，不要改） |
| `DEFAULT_VIDEO_PROVIDER` | 否 | `kling` | 视频生成默认 provider（kling/sora/veo） |
