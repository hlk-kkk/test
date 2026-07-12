# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Content Remix Agent** 是一个基于 LangGraph Agent 的内容二次创作助手。用户粘贴小红书、抖音、B站、快手的视频/图文链接，Agent 自动调用工具提取内容、转录语音、分析结构、生成二创文案。

**核心能力:**
- 对话式 Agent 交互（LangGraph + Tools）
- 多平台链接解析（小红书、抖音、B站、快手）
- 视频语音转文字（SenseVoice，备用 faster-whisper）
- LLM 内容结构分析（爆款元素、创作技巧识别）
- 多模式二创文案生成（仿写、改写、总结、拆解）
- 会话记忆持久化（PostgreSQL）
- 实时 SSE 流式输出

## Technology Stack

**Backend:** Python 3.10+ / FastAPI / LangGraph / LangChain / SenseVoice (FunASR) / ffmpeg / uv
**Frontend:** Vue 3 / TypeScript / Pinia / Tailwind CSS / DaisyUI / Vite 7

## System Architecture

```
用户浏览器 (React + SSE)
        │
        ▼
FastAPI Backend (/api/v1/remix/analyze)
        │
        ▼
┌─────────────────────────────────────┐
│         RemixAgent (LangGraph)      │
│    create_agent + 3 Tools           │
└──────────────┬──────────────────────┘
               │
    ┌──────────┼──────────┐
    ▼          ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐
│ parse  │ │ fetch  │ │process │
│ _link  │ │_content│ │ _video │
│ Tool   │ │ Tool   │ │ Tool   │
└────────┘ └────┬───┘ └────┬───┘
               │           │
    DownloadServer    Download + Extract + ASR
        Client        (内置于 process_video)

分析和灵感生成由 Agent 直接通过 System Prompt 完成

Memory: MySQL / PostgreSQL / InMemory
```

### LangChain 1.0 Agent 模式

本项目采用 LangChain 1.0 的最新 Agent 模式:

**核心 API:**
- `create_agent()`: 创建 Agent (替代旧版 `create_react_agent`)
- `init_chat_model()`: 统一的模型初始化 (自动适配 Provider)
- `AgentState`: Agent 状态基类 (自带 messages 字段)
- `ToolRuntime`: Tool 运行时上下文 (访问 context、stream_writer)

**状态管理:**
- `RemixAgentState`: 继承自 `AgentState`，定义 Agent 可变状态
- `RemixContext`: 请求级别上下文 (session_id, mode, instruction)
- `Command(update={...})`: Tool 返回状态更新

**记忆系统:**
- 短期记忆 (Checkpointer): 会话级别对话历史，通过 `thread_id` 隔离
- 长期记忆 (Store): 跨会话持久化数据 (如用户偏好)
- PostgresSaver: 生产环境持久化存储
- MemorySaver: 开发环境内存存储

### Agent Tools

当前项目使用 **3 个核心工具**，分析和灵感生成由 Agent 直接通过 System Prompt 完成：

| Tool | 功能 | 触发条件 |
|------|------|----------|
| `parse_link` | 解析链接，识别平台和内容ID | Agent 收到 URL 时 |
| `fetch_content` | 获取内容详情（标题、描述、视频URL等） | 解析成功后 |
| `process_video` | 下载视频 + 提取音频 + ASR 转录 | 内容包含视频时自动调用 |

**注意**: 分析内容和生成灵感不再作为独立工具，而是由 Agent 根据 System Prompt 直接完成，这样 Agent 可以更好地理解上下文并进行多轮对话。

### Data Flow (SSE Events)

| Event | Trigger | Data |
|-------|---------|------|
| `tool_start` | Tool 开始执行 | `{tool, message}` |
| `tool_end` | Tool 执行完成 | `{tool, state_update}` |
| `agent` | Agent 响应流 | `{content}` |
| `done` | 全部完成 | `{session_id, content_info, analysis, copywriting}` |
| `error` | 发生错误 | `{code, message}` |

## Essential Commands

### Environment Setup
```bash
# Backend (from content_remix_agent/backend/)
uv sync                      # Install core dependencies (includes SenseVoice/FunASR)
uv sync --extra postgres     # Include PostgreSQL memory support (optional)
uv sync --extra dev          # Include dev dependencies (pytest)

# Optional: Install faster-whisper as fallback ASR
uv add faster-whisper

# Frontend (from content_remix_agent/frontend/)
npm install

# System requirement
brew install ffmpeg          # macOS
apt install ffmpeg           # Ubuntu/Debian
```

**注意**:
- 默认使用 SenseVoice (FunASR)，首次运行会自动下载模型（约 1GB）
- 可选安装 faster-whisper 作为备用后端: `uv add faster-whisper`

### Running Services
```bash
# Backend API (default: http://localhost:8001)
cd content_remix_agent/backend
uv run python scripts/run_api_server.py
uv run python scripts/run_api_server.py --reload  # Dev mode

# Frontend (default: http://localhost:5173)
cd content_remix_agent/frontend
npm run dev
npm run build                # Production build
```

### Testing
```bash
# Run all tests
cd content_remix_agent/backend
uv run pytest

# Run specific test file
uv run pytest tests/test_link_parser.py -v

# Run with coverage
uv run pytest --cov=. --cov-report=html

# End-to-end Agent test (requires LLM)
uv run python scripts/test_agent_e2e.py
```

### Mock Mode
设置 `USE_MOCK=true` 可跳过真实的爬虫/ASR/LLM 调用进行开发调试。

## Module Index

| Module | Path | Purpose |
|--------|------|---------|
| **Agent Core** |
| Agent Factory | `backend/agent/remix_agent.py` | 创建和配置 Agent |
| Agent State | `backend/agent/state.py` | Agent 状态和上下文定义 (TypedDict) |
| Agent Errors | `backend/agent/errors.py` | 统一错误处理 (RemixToolException) |
| System Prompt | `backend/agent/prompts.py` | Agent 系统提示词 |
| Middleware | `backend/agent/middleware.py` | Agent 中间件 |
| **Tools (3个核心工具)** |
| parse_link | `backend/agent/tools/link_parser_tool.py` | 链接解析 Tool |
| fetch_content | `backend/agent/tools/fetch_content_tool.py` | 内容获取 Tool |
| process_video | `backend/agent/tools/video_processor_tool.py` | 视频处理 Tool |
| **Stream (流式处理)** |
| StreamEventProcessor | `backend/agent/stream/processor.py` | 流式事件处理器 |
| ThinkingParser | `backend/agent/stream/parser.py` | 多 Provider Thinking 解析器 |
| **Services** |
| Link Parser | `backend/services/link_parser.py` | 多平台链接解析 |
| Download Server | `backend/services/download_server_client.py` | DownloadServer 视频下载服务 |
| Video Downloader | `backend/services/video_downloader.py` | 视频下载 |
| Audio Extractor | `backend/services/audio_extractor.py` | ffmpeg 音频提取 |
| ASR Service | `backend/services/asr_service.py` | 双后端 ASR (SenseVoice + Whisper) |
| **Utils** |
| JSON Utils | `backend/utils/json_utils.py` | LLM JSON 提取和修复 |
| Logger | `backend/utils/logger.py` | 日志工具 |
| **API** |
| API Routes | `backend/api/routes/remix.py` | SSE 端点 |
| **Config** |
| Config | `backend/config.py` | Pydantic Settings 配置管理 |
| Schemas | `backend/schemas.py` | 数据模型定义 |
| LLM Provider | `backend/llm_provider.py` | LLM 提供商抽象 |

## Key Schemas

```python
# 二创模式
class RemixMode(str, Enum):
    IMITATE = "imitate"      # 仿写 - 保留结构框架
    REWRITE = "rewrite"      # 改写 - 重新组织表达
    SUMMARIZE = "summarize"  # 总结 - 提炼要点
    ANALYZE = "analyze"      # 拆解 - 教学视角

# 支持的平台
class Platform(str, Enum):
    XHS = "xhs"              # 小红书
    DOUYIN = "dy"            # 抖音
    BILIBILI = "bilibili"    # B站
    KUAISHOU = "ks"          # 快手
```

## Configuration (.env)

在项目根目录 `.env` 创建配置文件:

```ini
# ========== LLM Provider (推荐使用在线 API) ==========

# 方式一：OpenAI 兼容 API（推荐，支持 DeepSeek、MiniMax、Moonshot 等）
LLM_PROVIDER=openai
OPENAI_BASE_URL=https://api.minimaxi.com/v1
OPENAI_API_KEY=your_api_key
OPENAI_MODEL_NAME=MiniMax-M2.1-lightning

# 方式二：Anthropic Claude（推荐用于高质量输出，支持 Extended Thinking）
# LLM_PROVIDER=anthropic
# ANTHROPIC_API_KEY=your_api_key
# ANTHROPIC_BASE_URL=https://api.anthropic.com
# ANTHROPIC_MODEL_NAME=claude-sonnet-4-5-20250929

# 方式三：Ollama 本地模型（仅开发调试，不推荐生产使用）
# LLM_PROVIDER=ollama
# OLLAMA_BASE_URL=http://localhost:11434
# OLLAMA_MODEL_NAME=qwen3:4b

# ========== MediaCrawlerPro Database ==========
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=media_crawler_pro

# ========== Agent Memory ==========
MEMORY_BACKEND=mysql                   # memory / mysql / postgres
AGENT_DB_HOST=localhost
AGENT_DB_PORT=3306
AGENT_DB_USER=root
AGENT_DB_PASSWORD=
AGENT_DB_NAME=remix_agent

# ========== DownloadServer ==========
DOWNLOAD_SERVER_BASE=http://localhost:8205
DOWNLOAD_SERVER_TIMEOUT=60

# ========== ASR (SenseVoice - 默认) ==========
ASR_BACKEND=sensevoice                 # sensevoice (默认) / whisper (备用)
SENSEVOICE_MODEL_NAME=iic/SenseVoiceSmall
SENSEVOICE_MODEL_DIR=./models/sensevoice
SENSEVOICE_DEVICE=cuda:0               # cuda:0 (GPU) / cpu
SENSEVOICE_LANGUAGE=auto               # auto / zh / en
ASR_ENABLE_TIMESTAMPS=true             # 启用时间戳

# ========== Video Processing ==========
VIDEO_TEMP_DIR=./temp/videos
VIDEO_DOWNLOAD_TIMEOUT=120

# ========== API Server ==========
API_HOST=0.0.0.0
API_PORT=8001

# ========== Development ==========
USE_MOCK=false                         # true 跳过真实调用
LOG_LEVEL=INFO
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | 健康检查 |
| POST | `/api/v1/remix/analyze` | Agent 分析 (SSE)，支持 instruction 参数 |
| POST | `/api/v1/remix/chat` | 对话追问 (SSE)，需要 session_id |
| GET | `/api/v1/remix/modes` | 获取二创模式列表 |
| GET | `/api/v1/remix/session/{session_id}` | 获取会话状态 |

### Request Examples

**分析请求:**
```json
{
  "url": "https://v.douyin.com/xxx",
  "mode": "rewrite",
  "instruction": "只分析不生成文案",
  "session_id": "optional-session-id"
}
```

**对话请求:**
```json
{
  "message": "换一个更短的标题",
  "session_id": "required-session-id"
}
```

## Supported Link Formats

| Platform | Long URL | Short URL |
|----------|----------|-----------|
| 小红书 | `xiaohongshu.com/explore/xxx` | `xhslink.com/xxx` |
| 抖音 | `douyin.com/video/xxx` | `v.douyin.com/xxx` |
| B站 | `bilibili.com/video/BVxxx` | `b23.tv/xxx` |
| 快手 | `kuaishou.com/short-video/xxx` | `v.kuaishou.com/xxx` |

## Frontend Structure

```
frontend/src/
├── views/RemixView.vue          # 主页面
├── components/
│   ├── LinkInput.vue            # 链接输入
│   ├── ContentPreview.vue       # 内容预览（封面/图片）
│   ├── ProcessProgress.vue      # 处理进度
│   ├── AnalysisReport.vue       # 分析报告
│   ├── CopywritingPanel.vue     # 文案面板
│   └── ChatDialog.vue           # 对话追问
├── composables/
│   ├── useRemixSSE.ts           # Agent SSE 流处理
│   └── useChatSSE.ts            # Chat SSE（带 session_id）
├── stores/remix.ts              # Pinia 状态管理（含 sessionId）
└── api/client.ts                # API 客户端
```

## Agent Conversation Flow

```
User: "分析 https://v.douyin.com/xxx"
  │
  ▼
Agent: [调用 parse_link] → 解析成功，抖音视频
  │
  ▼
Agent: [调用 fetch_content] → 获取内容详情
  │
  ▼
Agent: [调用 process_video] → 下载视频，转录语音 (30-120s)
  │
  ▼
Agent: [直接输出] → 分析结构和爆款元素 (通过 System Prompt 引导)
  │
  ▼
Agent: [直接输出] → 生成创意灵感 (不调用工具，保持上下文连贯)
  │
  ▼
Agent: "分析完成！这是一个职场效率类视频..."

User: "换一个更短的标题"
  │
  ▼
Agent: (记住之前的分析结果，直接回复)
Agent: "好的，这里有3个更短的标题建议..."
```

**注意**: 分析和灵感生成不再作为独立工具，而是由 Agent 直接通过 System Prompt 完成，这样可以更好地保持多轮对话的上下文连贯性。

## Adding New Tool

1. 在 `backend/agent/tools/` 创建新的 Tool 文件
2. 使用 `@tool` 装饰器定义 Tool
3. 通过 `ToolRuntime` 访问上下文和流式写入器
4. 返回 `Command` 更新 Agent 状态
5. 在 `backend/agent/tools/__init__.py` 导出 Tool
6. 在 `backend/agent/remix_agent.py` 的 tools 列表中添加

```python
# backend/agent/tools/my_new_tool.py
from langchain_core.messages import ToolMessage
from langchain_core.tools import tool
from langgraph.prebuilt import ToolRuntime
from langgraph.types import Command

from agent.state import RemixContext

@tool
def my_new_tool(arg: str, runtime: ToolRuntime[RemixContext]) -> Command:
    """Tool description for Agent to understand when to use it."""
    # 发送进度更新 (通过 SSE 推送到前端)
    runtime.stream_writer({"stage": "processing", "message": "正在处理..."})

    # 访问上下文
    context = runtime.context  # RemixContext
    session_id = context.session_id

    # 处理逻辑
    result = do_something(arg)

    # 发送完成状态
    runtime.stream_writer({"stage": "done", "result": result})

    # 返回 Command 更新状态
    return Command(update={
        "messages": [ToolMessage(content=f"处理完成: {result}", tool_call_id=runtime.tool_call_id)],
        "my_field": result,
        "current_stage": "处理完成",
    })
```

## Error Codes

| Code | Description |
|------|-------------|
| `INVALID_URL` | 无法识别的链接格式 |
| `CONTENT_NOT_FOUND` | 内容不存在或已删除 |
| `COOKIES_NOT_FOUND` | 平台 cookies 未配置 |
| `FETCH_FAILED` | 内容获取失败 |
| `DOWNLOAD_FAILED` | 视频下载失败 |
| `AUDIO_EXTRACT_FAILED` | 音频提取失败 |
| `ASR_FAILED` | 语音转录失败 |
| `MISSING_STATE` | 缺少必需的状态字段 |
| `INTERNAL_ERROR` | 内部错误 |
| `AGENT_ERROR` | Agent 执行错误 |

## Relationship to comment_analysis_agent

本项目与同仓库的 `comment_analysis_agent` 是两个独立项目，但共享以下设计模式:
- FastAPI + SSE 流式响应
- LangChain/LangGraph LLM 抽象
- Vue 3 + Pinia 前端架构
- Pydantic Settings 配置管理
