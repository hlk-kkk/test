# 用户认证系统配置指南

本文档详细说明如何配置 Content Remix Agent 的用户认证系统，包括邮箱密码登录、GitHub OAuth、Google OAuth 以及邮件服务。

## 目录

- [系统概述](#系统概述)
- [快速开始](#快速开始)
- [邮箱密码登录](#邮箱密码登录)
- [GitHub OAuth 配置](#github-oauth-配置)
- [Google OAuth 配置](#google-oauth-配置)
- [SMTP 邮件服务配置](#smtp-邮件服务配置)
- [环境变量完整列表](#环境变量完整列表)
- [数据库迁移](#数据库迁移)
- [安全建议](#安全建议)
- [常见问题](#常见问题)

---

## 系统概述

### 认证方式

Content Remix Agent 支持三种认证方式：

| 方式 | 说明 | 必需配置 |
|------|------|----------|
| 邮箱 + 密码 | 传统注册登录 | JWT_SECRET_KEY, SMTP (可选) |
| GitHub OAuth | 使用 GitHub 账号登录 | GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET |
| Google OAuth | 使用 Google 账号登录 | GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET |

### 技术架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端 (React)                             │
│  LoginPage / RegisterPage / OAuthCallback / VerifyEmailPage     │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP API
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      FastAPI 后端                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ auth.py     │  │dependencies │  │   auth_service.py       │  │
│  │ (路由)       │  │ (中间件)     │  │   (核心逻辑)             │  │
│  └──────┬──────┘  └──────┬──────┘  └───────────┬─────────────┘  │
│         │                │                      │                │
│         └────────────────┼──────────────────────┘                │
│                          ▼                                       │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    服务层                                    ││
│  │ ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ ││
│  │ │oauth_client │ │email_service│ │      MySQL 数据库        │ ││
│  │ │(GitHub/Google)│ │  (SMTP)    │ │  users / tokens / ...   │ ││
│  │ └─────────────┘ └─────────────┘ └─────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### JWT 双 Token 机制

- **Access Token**: 30 分钟有效，用于 API 认证
- **Refresh Token**: 30 天有效，用于刷新 Access Token
- Token 刷新时自动轮换 Refresh Token（旧 token 立即失效）

---

## 快速开始

### 1. 最小配置（仅邮箱登录，无邮件功能）

```bash
# content_remix_agent/backend/.env

# 必需：JWT 密钥（生成方式见下方）
JWT_SECRET_KEY=your-super-secret-key-at-least-32-characters-long
```

生成安全的 JWT 密钥：

```bash
# 方式一：使用 Python
python -c "import secrets; print(secrets.token_urlsafe(32))"

# 方式二：使用 openssl
openssl rand -base64 32
```

### 2. 运行数据库迁移

```bash
cd content_remix_agent/backend

# 迁移会在服务启动时自动执行
# 或手动查看迁移文件
cat schema/migrations/V007__user_auth.sql
```

### 3. 启动服务

```bash
# 后端
uv run python scripts/run_api_server.py

# 前端
cd ../frontend && npm run dev
```

访问 http://localhost:5373/auth/login 即可使用邮箱注册/登录。

---

## 邮箱密码登录

邮箱密码登录是最基础的认证方式，只需配置 JWT 密钥即可使用。

### 配置项

```bash
# JWT 配置
JWT_SECRET_KEY=your-secret-key          # 必需，用于签名 JWT
JWT_ALGORITHM=HS256                      # 可选，默认 HS256
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=30       # 可选，Access Token 有效期
JWT_REFRESH_TOKEN_EXPIRE_DAYS=30         # 可选，Refresh Token 有效期
```

### 功能说明

- **注册**: 用户提供邮箱、密码、可选昵称完成注册
- **登录**: 邮箱 + 密码验证
- **密码要求**: 至少 6 个字符
- **密码存储**: bcrypt 哈希，不可逆

### 邮箱验证（可选）

如果配置了 SMTP 邮件服务：
- 注册后发送验证邮件
- 用户点击链接完成验证
- 未验证用户状态为 `pending`，验证后为 `active`

---

## GitHub OAuth 配置

### 步骤 1: 创建 GitHub OAuth App

1. 登录 GitHub，进入 **Settings** > **Developer settings** > **OAuth Apps**

   直达链接: https://github.com/settings/developers

2. 点击 **New OAuth App**

3. 填写应用信息：

   | 字段 | 示例值 | 说明 |
   |------|--------|------|
   | Application name | Remix AI Studio | 应用名称，用户授权时会看到 |
   | Homepage URL | http://localhost:5373 | 应用主页（开发环境） |
   | Application description | AI content remix assistant | 可选描述 |
   | Authorization callback URL | http://localhost:5373/auth/callback/github | **重要**: 必须与后端配置一致 |

4. 点击 **Register application**

5. 创建成功后，记录：
   - **Client ID**: 类似 `Ov23lixxxxxxxxxx`
   - 点击 **Generate a new client secret** 获取 **Client Secret**

   > ⚠️ Client Secret 只显示一次，请立即保存！

### 步骤 2: 配置环境变量

```bash
# content_remix_agent/backend/.env

# GitHub OAuth
GITHUB_CLIENT_ID=Ov23lixxxxxxxxxx
GITHUB_CLIENT_SECRET=your_client_secret_here
GITHUB_CALLBACK_URL=http://localhost:5373/auth/callback/github
```

### 步骤 3: 生产环境配置

部署到生产环境时，需要更新 GitHub OAuth App 设置：

1. 回到 GitHub OAuth App 设置页面
2. 更新 **Authorization callback URL** 为生产环境地址，如：
   ```
   https://your-domain.com/auth/callback/github
   ```
3. 更新环境变量：
   ```bash
   GITHUB_CALLBACK_URL=https://your-domain.com/auth/callback/github
   FRONTEND_URL=https://your-domain.com
   ```

### 权限说明

应用请求的 GitHub 权限：
- `read:user` - 读取用户基本信息
- `user:email` - 读取用户邮箱（用于创建账号）

---

## Google OAuth 配置

### 步骤 1: 创建 Google Cloud 项目

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)

2. 创建新项目或选择现有项目

3. 在搜索栏搜索 **"OAuth consent screen"** 或导航到：

   **APIs & Services** > **OAuth consent screen**

### 步骤 2: 配置 OAuth 同意屏幕

1. 选择 **User Type**:
   - **External**: 任何 Google 账号都可登录（推荐）
   - **Internal**: 仅组织内部用户（需要 Google Workspace）

2. 点击 **Create**

3. 填写应用信息：

   | 字段 | 示例值 | 说明 |
   |------|--------|------|
   | App name | Remix AI Studio | 用户看到的应用名 |
   | User support email | your@email.com | 支持邮箱 |
   | App logo | (可选) | 应用图标 |
   | Developer contact information | your@email.com | 开发者邮箱 |

4. 点击 **Save and Continue**

5. **Scopes** 页面：
   - 点击 **Add or Remove Scopes**
   - 选择以下范围：
     - `.../auth/userinfo.email` - 查看邮箱
     - `.../auth/userinfo.profile` - 查看基本资料
     - `openid` - OpenID Connect
   - 点击 **Update** 然后 **Save and Continue**

6. **Test users** 页面（如果选择了 External）：
   - 在测试阶段，需要添加测试用户邮箱
   - 点击 **Add Users** 添加你的 Google 账号
   - 点击 **Save and Continue**

7. 完成配置，点击 **Back to Dashboard**

### 步骤 3: 创建 OAuth 2.0 凭据

1. 导航到 **APIs & Services** > **Credentials**

2. 点击 **Create Credentials** > **OAuth client ID**

3. 选择 **Application type**: **Web application**

4. 填写信息：

   | 字段 | 值 |
   |------|-----|
   | Name | Remix AI Studio Web Client |
   | Authorized JavaScript origins | `http://localhost:5373` |
   | Authorized redirect URIs | `http://localhost:5373/auth/callback/google` |

   > 开发环境添加 localhost，生产环境添加实际域名

5. 点击 **Create**

6. 记录：
   - **Client ID**: 类似 `123456789-abc.apps.googleusercontent.com`
   - **Client Secret**: 类似 `GOCSPX-xxxxx`

### 步骤 4: 配置环境变量

```bash
# content_remix_agent/backend/.env

# Google OAuth
GOOGLE_CLIENT_ID=123456789-abc.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-your-client-secret
GOOGLE_CALLBACK_URL=http://localhost:5373/auth/callback/google
```

### 步骤 5: 发布应用（生产环境）

测试阶段的应用只能由添加的测试用户使用。要让所有用户都能登录：

1. 回到 **OAuth consent screen**
2. 点击 **Publish App**
3. 确认发布

> ⚠️ 发布后应用状态变为 "In production"，任何 Google 用户都可登录。

### 步骤 6: 生产环境配置

1. 回到 **Credentials** 页面
2. 编辑 OAuth client
3. 添加生产环境的域名：
   - Authorized JavaScript origins: `https://your-domain.com`
   - Authorized redirect URIs: `https://your-domain.com/auth/callback/google`

4. 更新环境变量：
   ```bash
   GOOGLE_CALLBACK_URL=https://your-domain.com/auth/callback/google
   FRONTEND_URL=https://your-domain.com
   ```

---

## SMTP 邮件服务配置

邮件服务用于发送验证邮件和密码重置邮件。如果不配置，用户仍可正常使用，但：
- 无法发送邮箱验证邮件
- 无法使用"忘记密码"功能

### 常见邮件服务商配置

#### Gmail

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password    # 注意：需要使用应用专用密码
SMTP_FROM_EMAIL=your-email@gmail.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

**获取 Gmail 应用专用密码：**
1. 访问 [Google 账号安全设置](https://myaccount.google.com/security)
2. 开启 **两步验证**
3. 访问 [应用专用密码](https://myaccount.google.com/apppasswords)
4. 选择 **邮件** + **其他 (自定义名称)**
5. 点击 **生成**，使用生成的 16 位密码

#### Outlook / Microsoft 365

```bash
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USER=your-email@outlook.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=your-email@outlook.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

#### 阿里云企业邮箱

```bash
SMTP_HOST=smtp.qiye.aliyun.com
SMTP_PORT=465
SMTP_USER=your-email@your-domain.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=your-email@your-domain.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

#### 腾讯企业邮箱

```bash
SMTP_HOST=smtp.exmail.qq.com
SMTP_PORT=465
SMTP_USER=your-email@your-domain.com
SMTP_PASSWORD=your-password
SMTP_FROM_EMAIL=your-email@your-domain.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

#### SendGrid

```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASSWORD=your-sendgrid-api-key
SMTP_FROM_EMAIL=noreply@your-domain.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

#### Mailgun

```bash
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@your-domain.mailgun.org
SMTP_PASSWORD=your-mailgun-password
SMTP_FROM_EMAIL=noreply@your-domain.com
SMTP_FROM_NAME=Remix AI Studio
SMTP_USE_TLS=true
```

### 测试邮件配置

启动后端后，可以通过注册新用户来测试邮件发送功能。

---

## 环境变量完整列表

以下是所有认证相关的环境变量：

```bash
# ============================================================
# JWT 配置（必需）
# ============================================================
JWT_SECRET_KEY=your-secret-key-at-least-32-chars    # 必需
JWT_ALGORITHM=HS256                                   # 可选，默认 HS256
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=30                    # 可选，默认 30
JWT_REFRESH_TOKEN_EXPIRE_DAYS=30                      # 可选，默认 30

# ============================================================
# GitHub OAuth（可选）
# ============================================================
GITHUB_CLIENT_ID=                                     # GitHub OAuth App Client ID
GITHUB_CLIENT_SECRET=                                 # GitHub OAuth App Client Secret
GITHUB_CALLBACK_URL=http://localhost:5373/auth/callback/github

# ============================================================
# Google OAuth（可选）
# ============================================================
GOOGLE_CLIENT_ID=                                     # Google OAuth Client ID
GOOGLE_CLIENT_SECRET=                                 # Google OAuth Client Secret
GOOGLE_CALLBACK_URL=http://localhost:5373/auth/callback/google

# ============================================================
# SMTP 邮件服务（可选）
# ============================================================
SMTP_HOST=                                            # SMTP 服务器地址
SMTP_PORT=587                                         # SMTP 端口，默认 587
SMTP_USER=                                            # SMTP 用户名
SMTP_PASSWORD=                                        # SMTP 密码
SMTP_FROM_EMAIL=                                      # 发件人邮箱
SMTP_FROM_NAME=Remix AI Studio                        # 发件人名称
SMTP_USE_TLS=true                                     # 是否使用 TLS

# ============================================================
# 前端 URL（用于邮件链接）
# ============================================================
FRONTEND_URL=http://localhost:5373                    # 前端地址
```

---

## 数据库迁移

### 新增表结构

认证系统新增以下数据库表：

| 表名 | 说明 |
|------|------|
| `users` | 用户主表 |
| `user_oauth_accounts` | OAuth 账号关联表 |
| `email_verification_tokens` | 邮箱验证/密码重置令牌 |
| `user_refresh_tokens` | Refresh Token 白名单 |

### 现有表修改

为以下表添加 `user_id` 字段实现用户数据隔离：

- `agent_sessions`
- `agent_checkpoints`
- `agent_store`

### 迁移文件

迁移文件位于：`backend/schema/migrations/V007__user_auth.sql`

迁移会在服务启动时自动执行（由 `migration_service` 处理）。

### 手动执行迁移

如需手动执行：

```bash
cd content_remix_agent/backend

# 查看迁移 SQL
cat schema/migrations/V007__user_auth.sql

# 手动执行（使用 MySQL 客户端）
mysql -u root -p remix_agent < schema/migrations/V007__user_auth.sql
```

---

## 安全建议

### 1. JWT 密钥安全

```bash
# 生成足够长的随机密钥（至少 32 字符）
python -c "import secrets; print(secrets.token_urlsafe(32))"

# 不要硬编码在代码中
# 不要提交到 Git
# 使用环境变量或密钥管理服务
```

### 2. 生产环境配置

```bash
# 1. 使用 HTTPS
FRONTEND_URL=https://your-domain.com

# 2. 更新 OAuth 回调地址
GITHUB_CALLBACK_URL=https://your-domain.com/auth/callback/github
GOOGLE_CALLBACK_URL=https://your-domain.com/auth/callback/google

# 3. 配置 CORS
# 在 settings.py 中更新 CORS_ORIGINS
```

### 3. 密码策略

当前实现：
- 最小长度 6 字符
- bcrypt 哈希存储

建议增强（自行修改 `auth_service.py`）：
- 要求包含大小写字母、数字、特殊字符
- 检测常见弱密码

### 4. 登录限流

当前实现未包含登录限流，建议在生产环境添加：
- 使用 Redis 实现
- 失败 N 次后锁定 M 分钟
- 可添加验证码

---

## 常见问题

### Q: OAuth 登录报错 "redirect_uri_mismatch"

**原因**: 配置的回调 URL 与 OAuth 应用设置不一致

**解决**:
1. 检查 `.env` 中的 `GITHUB_CALLBACK_URL` 或 `GOOGLE_CALLBACK_URL`
2. 确保与 GitHub/Google 开发者控制台中配置的完全一致
3. 注意区分 http 和 https

### Q: 邮件发送失败

**检查清单**:
1. SMTP 配置是否正确
2. 密码是否使用了应用专用密码（Gmail 等需要）
3. 端口是否被防火墙阻止
4. 查看后端日志获取详细错误

### Q: Token 过期太快

**调整配置**:
```bash
# 延长 Access Token 有效期（分钟）
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=60

# 延长 Refresh Token 有效期（天）
JWT_REFRESH_TOKEN_EXPIRE_DAYS=90
```

### Q: 如何禁用某种登录方式？

- **禁用 GitHub OAuth**: 不配置 `GITHUB_CLIENT_ID` 和 `GITHUB_CLIENT_SECRET`
- **禁用 Google OAuth**: 不配置 `GOOGLE_CLIENT_ID` 和 `GOOGLE_CLIENT_SECRET`
- **禁用邮箱注册**: 需要修改后端代码禁用 `/register` 端点

### Q: OAuth 登录后如何获取用户信息？

OAuth 登录成功后：
1. 前端收到 `access_token` 和 `refresh_token`
2. 存储到 `localStorage`
3. 后续请求自动携带 `Authorization: Bearer <token>`
4. 调用 `GET /api/v1/auth/me` 获取用户信息

### Q: 如何实现"记住我"功能？

当前实现：
- Refresh Token 默认 30 天有效
- 存储在 `localStorage`
- 关闭浏览器后重新打开仍保持登录

如需更短的"不记住"时间，可以：
- 前端使用 `sessionStorage` 替代 `localStorage`
- 或在登出时主动清除 Token

### Q: 多设备登录如何处理？

当前实现支持多设备同时登录：
- 每个设备有独立的 Refresh Token
- Token 存储在 `user_refresh_tokens` 白名单表
- 修改密码时会撤销所有设备的 Token

---

## API 端点参考

### 认证端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 邮箱注册 |
| POST | `/api/v1/auth/login` | 邮箱登录 |
| POST | `/api/v1/auth/logout` | 登出 |
| POST | `/api/v1/auth/refresh` | 刷新 Token |
| GET | `/api/v1/auth/me` | 获取当前用户 |
| GET | `/api/v1/auth/oauth/providers` | 获取 OAuth 提供商状态 |
| GET | `/api/v1/auth/oauth/{provider}/authorize` | 获取 OAuth 授权 URL |
| POST | `/api/v1/auth/oauth/{provider}/callback` | OAuth 回调 |
| POST | `/api/v1/auth/verify-email` | 验证邮箱 |
| POST | `/api/v1/auth/resend-verification` | 重发验证邮件 |
| POST | `/api/v1/auth/forgot-password` | 请求重置密码 |
| POST | `/api/v1/auth/reset-password` | 重置密码 |

### 请求示例

**注册**:
```bash
curl -X POST http://localhost:8001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "display_name": "Test User"
  }'
```

**登录**:
```bash
curl -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

**获取当前用户**:
```bash
curl http://localhost:8001/api/v1/auth/me \
  -H "Authorization: Bearer <access_token>"
```

**刷新 Token**:
```bash
curl -X POST http://localhost:8001/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<refresh_token>"
  }'
```

---

## 更新日志

- **2025-01-30**: 初始版本，支持邮箱登录、GitHub OAuth、Google OAuth
