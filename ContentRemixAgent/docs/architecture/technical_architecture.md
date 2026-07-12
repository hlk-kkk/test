# Content Remix Agent 技术架构文档（最新版）

> 目标：从系统视角说明整体架构、关键组件、数据流与时序，并给出关键流程图/时序图，方便新人快速理解与运维排障。

## 1. 总览

Content Remix Agent 是一个面向内容创作者的 AI 二创系统，核心流程是：
1) 前端提交链接/指令 → 2) 后端创建会话与意图识别 → 3) RemixAgent 调用工具获取内容与转录 → 4) LLM 生成分析与灵感 → 5) SSE 流式返回并持久化。

核心技术栈：
- FastAPI + SSE（sse-starlette）
- LangChain 1.0 + LangGraph create_agent（ToolRuntime + Checkpointer/Store）
- MySQL（会话、Auth、LLM Config、Insight Modes、Agent Memory）
- DownloadServer（内容抓取与下载）
- ASR（FunASR 本地模型 / Bcut 云端 API）
- 资源存储（本地文件系统挂载 /assets）

## 2. 系统架构图

```mermaid
flowchart LR
    User((用户)) --> FE[React 前端]
    FE -- SSE/HTTP --> API[FastAPI API]

    API --> Agent[RemixAgent<br/>LangGraph create_agent]
    API --> SessionMgr[Session Manager<br/>会话/消息持久化]

    Agent --> Tools[Tools: parse_link / fetch_content / process_video]
    Agent --> LLM[LLM Provider<br/>OpenAI/Anthropic/DeepSeek/Ollama]
    Agent --> Memory[Agent Memory<br/>Checkpointer + Store]

    Tools --> LinkParser[LinkParser Service]
    Tools --> DownloadClient[DownloadServer Client]
    Tools --> VideoPipeline[Video Pipeline]
    Tools --> AssetStore[Asset Storage]

    DownloadClient --> DownloadServer[DownloadServer]
    DownloadServer --> MediaDB[(MediaCrawlerPro DB)]
    DownloadServer --> SignSrv[Signature Service]

    VideoPipeline --> Downloader[VideoDownloader]
    VideoPipeline --> Extractor[AudioExtractor/ffmpeg]
    VideoPipeline --> ASR[ASR Service<br/>FunASR/Bcut]

    SessionMgr --> AgentDB[(MySQL Agent DB)]
    Memory --> AgentDB
    API --> AgentDB

    AssetStore --> Assets[(assets/ 本地文件)]
```

## 3. 关键运行流程（时序）

### 3.1 /api/v1/remix/analyze 主流程（SSE）

```mermaid
sequenceDiagram
    participant U as User
    participant FE as Frontend
    participant API as FastAPI
    participant IM as IntentClassifier
    participant A as RemixAgent
    participant T as Tools
    participant DS as DownloadServer
    participant ASR as ASRService
    participant DB as MySQL

    U->>FE: 粘贴链接 + 指令
    FE->>API: POST /remix/analyze (SSE)
    API->>DB: 创建 session + 记录用户消息
    API->>IM: 可选意图识别（未指定 mode）
    IM-->>API: mode + reasoning

    API->>A: create_remix_agent()
    API->>A: astream_events(messages, context)

    A->>T: parse_link
    T-->>A: parsed_link

    A->>T: fetch_content
    T->>DS: 拉取内容信息
    DS-->>T: content_info
    T-->>A: content_info

    A->>T: process_video (若有视频)
    T->>ASR: 转录音频（FunASR/Bcut）
    ASR-->>T: transcript
    T-->>A: transcript

    A-->>API: astream_events
    API-->>FE: SSE 事件流（thinking/tool/content/done）
    API->>DB: 持久化 assistant 消息 + segments
    FE-->>U: 实时展示分析与灵感
```

### 3.2 视频处理子流程（process_video）

```mermaid
sequenceDiagram
    participant T as process_video
    participant VD as VideoDownloader
    participant AE as AudioExtractor
    participant ASR as ASRService
    participant AS as AssetStorage

    T->>VD: 下载视频/音频
    VD-->>T: 本地视频路径
    T->>AE: ffmpeg 提取音频
    AE-->>T: 音频文件
    T->>ASR: 转录音频
    ASR-->>T: text + segments
    T->>AS: 持久化视频/封面（可选）
    AS-->>T: local_video_url
```

### 3.3 应用启动流程（lifespan）

```mermaid
sequenceDiagram
    participant API as FastAPI
    participant MS as MigrationService
    participant DS as DownloadServer
    participant MM as MemoryManager
    participant IM as InsightModeService
    participant ASR as ASRService

    API->>MS: run_migrations()
    MS-->>API: 迁移完成
    API->>DS: ping()
    DS-->>API: ok
    API->>MM: initialize()
    MM-->>API: Memory 就绪
    API->>IM: initialize_default_modes()
    API->>IM: preload_mode_prompts()
    API->>ASR: preload()
    ASR-->>API: 模型就绪
```

## 4. 数据与存储结构

### 4.1 数据分类
- **Agent 会话与消息**：session、message、segments（用于前端恢复进度）
- **Agent Memory**：短期 Checkpointer + 长期 Store（跨会话记忆）
- **运行配置**：LLM Config、Insight Modes、Cookies
- **认证系统**：用户、OAuth、刷新令牌、邮件验证
- **静态资源**：封面、视频、合并后的媒体与元数据（本地 assets）

### 4.2 数据库与文件存储关系

```mermaid
erDiagram
    agent_sessions ||--o{ agent_session_messages : contains
    agent_checkpoints ||--o{ agent_checkpoint_writes : has

    agent_sessions {
        string session_id
        string title
        string first_message
    }
    agent_session_messages {
        string message_id
        string role
        text content
        json segments_json
    }
    agent_checkpoints {
        string thread_id
    }
    agent_checkpoint_writes {
        string thread_id
        json checkpoint
    }
    agent_store {
        string namespace
        string key
        json value
    }
    llm_configs {
        string provider
        string model_name
        bool is_active
    }
    insight_modes {
        string mode_key
        bool is_active
    }
    cookies {
        string platform
        text cookies
    }
```

> 说明：数据库表结构以 `content_remix_agent/backend/schema/migrations` 为准。文件资源存储在 `content_remix_agent/backend/assets` 并通过 `/assets` 静态路由暴露。

## 5. SSE 事件类型（核心约定）

常见事件：
- `thinking_start` / `thinking_chunk` / `thinking_end`
- `tool_call_start` / `tool_call_finish` / `tool_progress`
- `sub_step_start` / `sub_step_end`（视频处理子步骤）
- `content_info` / `transcript`
- `content_chunk`（最终报告）
- `done` / `error`

事件处理在 `agent/stream/` 内统一封装，前端直接消费标准化事件。

## 6. 关键组件责任边界

- **API 层（api/routes）**：请求校验、会话持久化、SSE 组织与错误处理
- **Agent（agent/）**：策略与推理、工具编排、上下文管理与记忆
- **Services（services/）**：下载、转录、资源存储、LLM 配置、OAuth 等外部依赖封装
- **DB（schema/ + services/migration_service）**：迁移与数据一致性

## 7. 依赖与边界条件

- 必须启动 **DownloadServer**（启动时强依赖 ping 成功）
- MySQL 用于：Agent 数据库 + LLM 配置 + Auth + Insight Modes + Cookies
- ASR 可选：
  - FunASR 本地模型（GPU/CPU）
  - Bcut 云端 API（无需模型下载）
- LLM Provider 通过数据库或 `.env` 选择
