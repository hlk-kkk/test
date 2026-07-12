# PicTacticAgent 配置指南

本文档介绍如何配置 PicTacticAgent 的 API Key 和相关设置。

## 快速开始

PicTacticAgent 支持四种 Provider，**只需配置一套 API Key / Cookie 即可同时支持生图和评估功能**。

```bash
# 1. 复制配置模板
cp .env.example .env

# 2. 选择一个 Provider 并配置对应的 API Key
# 3. 启动服务
uv run uvicorn pictactic.api.app:app --reload
```

## Provider 选择

| Provider | 特点 | 适用场景 |
|----------|------|----------|
| **gemini_web** | 逆向代理，免费，使用 Google 账号 Cookie | 有 Google 账号的用户，零成本体验 |
| **gemini** | 官方 SDK，稳定可靠 | 海外用户，需要 API Key |
| **jiekouai** | 国内中转站，访问便捷 | 国内用户，无法直连 Google |
| **antigravity** | 本地反代，灵活配置 | 高级用户，自建服务 |

## 配置详解

### 场景 1: Gemini Web（推荐，免费）

通过逆向工程直接与 gemini.google.com 交互，使用 Google 账号 Cookie 认证，无需 API Key，完全免费。

```bash
# .env 配置
DEFAULT_PROVIDER=gemini_web

# Cookie 获取方法：
# 1. 登录 https://gemini.google.com
# 2. F12 打开开发者工具 → Application → Cookies → gemini.google.com
# 3. 找到 __Secure-1PSID 和 __Secure-1PSIDTS 的值

GEMINI_WEB_SECURE_1PSID=your-cookie-value
GEMINI_WEB_SECURE_1PSIDTS=your-cookie-value

# 代理 (可选，如需代理访问 Google)
GEMINI_WEB_PROXY=http://127.0.0.1:7890

# 生图模型 (可选，默认 gemini-3-pro)
GEMINI_WEB_MODEL=gemini-3-pro
```

**支持的模型：**
- `gemini-3-pro` - 高质量生图，推荐
- `gemini-3-flash` - 更快速
- `gemini-3-pro-plus` - 最高质量（需 Google One 订阅）

**注意事项：**
- Cookie 会过期，需定期更新（库会自动刷新，但长期不用需要重新获取）
- 生成的图片右下角会有 Gemini 四角星水印
- 评估功能需要配合其他 Provider 的 LLM（如 jiekouai），或单独配置 `OPENAI_API_KEY`

### 场景 2: 官方 Gemini

使用 Google 官方 Gemini API，需要能够直连 Google 服务。

```bash
# .env 配置
DEFAULT_PROVIDER=gemini

# API Key (从 https://aistudio.google.com/apikey 获取)
GEMINI_API_KEY=your-google-gemini-api-key

# 生图模型 (可选，默认 gemini-3-pro-image-preview)
GEMINI_MODEL=gemini-3-pro-image-preview

# 评估模型 (可选，默认 gemini-3-flash-preview)
GEMINI_LLM_MODEL=gemini-3-flash-preview
```

**支持的生图模型：**
- `gemini-3-pro-image-preview` - 高质量，支持 1K/2K/4K，推荐

**支持的评估模型（Gemini 3 系列）：**
- `gemini-3-flash-preview` - 100万上下文，快速响应，推荐
- `gemini-3-pro-preview` - 100万上下文，更强推理能力

### 场景 3: Jiekouai 中转站

使用 Jiekouai 提供的 API 中转服务，国内可直接访问。

```bash
# .env 配置
DEFAULT_PROVIDER=jiekouai

# API Key (从 jiekou.ai 获取)
JIEKOUAI_API_KEY=your-jiekouai-api-key

# 生图 API 地址 (可选，默认值如下)
JIEKOUAI_API_BASE_URL=https://api.jiekou.ai/v3

# 评估 LLM API 地址 (可选，默认值如下)
JIEKOUAI_LLM_BASE_URL=https://api.jiekou.ai/openai

# 评估模型 (可选，默认 gpt-4o)
JIEKOUAI_LLM_MODEL=gpt-4o
```

**支持的评估模型：**
- `gpt-4o` - OpenAI GPT-4o，推荐
- `gpt-4o-mini` - 更快更便宜
- 其他 OpenAI 兼容模型

### 场景 4: Antigravity 反代

使用本地或自建的 OpenAI 兼容反代服务。

```bash
# .env 配置
DEFAULT_PROVIDER=antigravity

# API Key
ANTIGRAVITY_API_KEY=your-antigravity-api-key

# 反代地址 (可选，默认值如下)
ANTIGRAVITY_API_BASE_URL=http://127.0.0.1:8045/v1

# 生图模型 (可选，默认 gemini-3-pro-image)
ANTIGRAVITY_MODEL=gemini-3-pro-image

# 评估模型 (可选，默认 gemini-3-flash)
ANTIGRAVITY_LLM_MODEL=gemini-3-flash
```

## 配置架构

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              DEFAULT_PROVIDER                                │
│               (gemini_web / gemini / jiekouai / antigravity)                 │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
         ┌────────────────┬───────────┼───────────┬────────────────┐
         ▼                ▼           ▼           ▼                │
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   gemini_web    │ │     gemini      │ │    jiekouai     │ │   antigravity   │
├─────────────────┤ ├─────────────────┤ ├─────────────────┤ ├─────────────────┤
│ 生图:           │ │ 生图:           │ │ 生图:           │ │ 生图:           │
│  逆向 Web API   │ │  官方 SDK       │ │  Jiekouai API   │ │  OpenAI 兼容    │
│  Cookie 认证    │ │  GEMINI_MODEL   │ │  /v3 endpoint   │ │  /v1 endpoint   │
│  免费           │ │                 │ │                 │ │                 │
├─────────────────┤ ├─────────────────┤ ├─────────────────┤ ├─────────────────┤
│ 评估:           │ │ 评估:           │ │ 评估:           │ │ 评估:           │
│  需配合其他     │ │  google_genai   │ │  OpenAI 兼容    │ │  OpenAI 兼容    │
│  Provider LLM  │ │  gemini-3-      │ │  /openai        │ │  /v1 endpoint   │
│                 │ │  flash-preview  │ │  gpt-4o         │ │  gemini-3-flash │
└─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘
```

## 其他配置项

### 图片生成默认设置

```bash
# 默认宽高比
DEFAULT_ASPECT_RATIO=16:9

# 默认图片尺寸 (1K/2K/4K)
DEFAULT_IMAGE_SIZE=2K

# 每轮生成图片数量 (1-10)
DEFAULT_IMAGES_PER_ROUND=5

# 最大迭代轮数 (1-10)
DEFAULT_MAX_ROUNDS=3

# 质量阈值 (0.0-1.0)
DEFAULT_QUALITY_THRESHOLD=0.7
```

### 存储设置

```bash
# 存储类型 (local/s3)
STORAGE_TYPE=local

# 本地存储路径
STORAGE_PATH=./storage/images
```

### 应用设置

```bash
# 前端 URL (用于 CORS)
FRONTEND_URL=http://localhost:3019

# 数据库路径
DB_PATH=./data/pictactic.db
```

## 向后兼容

如果你之前使用的是旧配置方式（独立配置 `OPENAI_API_KEY`），系统仍然支持：

```bash
# 旧配置方式 (已废弃，但仍兼容)
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_API_BASE_URL=https://api.jiekou.ai/openai
VISION_LLM_MODEL=gpt-4o
```

当 `DEFAULT_PROVIDER` 不是 `gemini`、`jiekouai` 或 `antigravity` 时，系统会自动回退到这套旧配置。

## 验证配置

运行以下命令验证配置是否正确：

```bash
# 运行 LLM 配置测试
uv run python tests/e2e/test_llm_config.py

# 测试特定 provider
DEFAULT_PROVIDER=gemini uv run python tests/e2e/test_llm_config.py
DEFAULT_PROVIDER=jiekouai uv run python tests/e2e/test_llm_config.py
DEFAULT_PROVIDER=antigravity uv run python tests/e2e/test_llm_config.py
```

## 常见问题

### Q: 为什么评估功能不工作？

A: 检查以下几点：
1. 确认 `DEFAULT_PROVIDER` 设置正确
2. 确认对应的 API Key 已配置
3. 运行测试脚本验证配置

### Q: 如何切换 Provider？

A: 只需修改 `DEFAULT_PROVIDER` 环境变量，并确保对应的 API Key 已配置：

```bash
# 切换到 Gemini Web（免费，Cookie 认证）
DEFAULT_PROVIDER=gemini_web

# 切换到 Gemini（官方 API）
DEFAULT_PROVIDER=gemini

# 切换到 Jiekouai（中转站）
DEFAULT_PROVIDER=jiekouai

# 切换到 Antigravity（反代）
DEFAULT_PROVIDER=antigravity
```

### Q: 生图和评估可以使用不同的 Provider 吗？

A: 当前版本不支持。生图和评估使用同一个 Provider 的配置，这样简化了配置复杂度。如果有特殊需求，可以通过修改代码实现。

### Q: 如何获取各 Provider 的 API Key？

| Provider | 获取方式 |
|----------|----------|
| Gemini Web | 登录 [gemini.google.com](https://gemini.google.com) → F12 → Application → Cookies 提取 |
| Gemini | [Google AI Studio](https://aistudio.google.com/apikey) |
| Jiekouai | [jiekou.ai](https://jiekou.ai) 注册获取 |
| Antigravity | 自建服务或联系服务提供商 |
