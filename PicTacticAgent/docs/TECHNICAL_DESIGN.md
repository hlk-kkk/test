# PicTacticAgent - 技术设计文档

## 1. 技术栈

### 1.1 后端
| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 语言 | Python 3.11+ | 主要开发语言 |
| 包管理 | uv | 快速 Python 包管理器 |
| 工作流框架 | LangChain 1.0+ + LangGraph 1.0+ | StateGraph 工作流编排 |
| Web 框架 | FastAPI + Uvicorn | 异步 API 服务 |
| 数据库 | SQLite | 数据持久化 |
| 图片存储 | Local FS | 本地文件系统 |

### 1.2 前端
| 组件 | 技术选型 | 说明 |
|------|----------|------|
| UI 框架 | React 19 + Vite 7 | 现代前端框架 |
| 样式 | Tailwind CSS v4 | 原子化 CSS |
| 实时通信 | WebSocket | 进度推送 |

### 1.3 AI 服务
| 服务 | 用途 | 说明 |
|------|------|------|
| GPT-4o | Agent 推理 | 提示词增强、评估决策 |
| GPT-4o-mini | 轻量任务 | 策略选择 |
| Gemini 3 Pro Image | 图片生成 | 默认 Provider，支持生图+编辑 |
| Antigravity (OpenAI 兼容) | 图片生成 | 反向代理 Provider，仅支持生图 |
| CLIP | 图文匹配 | 评分辅助 |

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend (React + Vite)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │ Prompt Input │  │ Image Gallery│  │ Config Panel │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└─────────────────────────────────────────────────────────────────┘
                              │
                    REST API / WebSocket
                              │
┌─────────────────────────────────────────────────────────────────┐
│                      Backend (FastAPI)                           │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────┐         │
│  │ API Routes │  │ WebSocket    │  │ Storage Service  │         │
│  │            │  │ Handler      │  │                  │         │
│  └────────────┘  └──────────────┘  └──────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│              StateGraph Workflow (LangGraph)                     │
│                                                                  │
│  START ──► check_template ──┬──► analyze_node ──┐               │
│                             │                    │               │
│                             └──► enhance_node ◄─┘               │
│                                       │                          │
│                                       ▼                          │
│                              generate_node ◄─────────┐          │
│                                       │               │          │
│                                       ▼               │          │
│                              evaluate_node            │          │
│                                       │               │          │
│                                       ▼               │          │
│                             should_continue           │          │
│                              /         \              │          │
│                         continue      finalize        │          │
│                            │             │            │          │
│                            ▼             ▼            │          │
│                      prepare_next   finalize_node     │          │
│                            │             │            │          │
│                            └─────────────┼────────────┘          │
│                                          ▼                       │
│                                         END                      │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────────┐
│                        Storage Layer                             │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────┐         │
│  │  SQLite    │  │ Local FS     │  │ Vision LLM       │         │
│  │ (Metadata) │  │ (Images)     │  │ (GPT-4o)         │         │
│  └────────────┘  └──────────────┘  └──────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 StateGraph 工作流

```
用户输入 Prompt + 配置
        │
        ▼
┌──────────────────┐
│  check_template  │  ◄── 检查是否有参考图片
└──────────────────┘
        │
    有参考图?
    /       \
   是        否
   │          │
   ▼          │
┌──────────────────────┐
│   analyze_node       │  ◄── 分析参考图片风格
└──────────────────────┘
        │              │
        └──────┬───────┘
               ▼
┌──────────────────────┐
│   enhance_node       │  ◄── 增强提示词（考虑反馈）
└──────────────────────┘
        │
        ▼
┌──────────────────────┐
│   generate_node      │  ◄── 并行生成图片，流式输出
└──────────────────────┘
        │
        ▼
┌──────────────────────┐
│   evaluate_node      │  ◄── 比较排序，打分评审
└──────────────────────┘
        │
        ▼
┌──────────────────────┐
│  should_continue     │  ◄── 判断是否继续
└──────────────────────┘
   /            \
继续           结束
  │              │
  ▼              ▼
prepare_next   finalize_node
  │              │
  └──► enhance ──► END
```

**关键改进**：
- 使用 StateGraph 替代 ReAct Agent，工作流更加确定可控
- 节点化设计，每个节点职责单一、易于测试
- 迭代时 `prepare_next` → `enhance_node` 形成闭环，支持基于反馈优化

---

## 3. 目录结构

```
PicTacticAgent/
├── pyproject.toml                 # 项目配置
├── README.md
├── .env.example
├── .gitignore
├── start.sh                       # 一键启动脚本
│
├── backend/src/pictactic/        # Python 后端
│   ├── __init__.py
│   │
│   ├── core/                      # 核心配置
│   │   ├── __init__.py
│   │   ├── config.py              # 配置管理
│   │   └── constants.py           # 常量定义
│   │
│   ├── agents/                    # StateGraph 工作流
│   │   ├── __init__.py
│   │   ├── workflow.py            # 工作流定义与编排
│   │   ├── conditions.py          # 条件判断函数
│   │   ├── state.py               # 共享状态定义
│   │   ├── prompts.py             # LLM 提示词模板
│   │   │
│   │   ├── nodes/                 # 工作流节点
│   │   │   ├── __init__.py
│   │   │   ├── analyze_node.py    # 模板分析节点
│   │   │   ├── enhance_node.py    # 提示词增强节点
│   │   │   ├── generate_node.py   # 图片生成节点
│   │   │   ├── evaluate_node.py   # 图片评估节点
│   │   │   ├── prepare_next_node.py # 准备下轮节点
│   │   │   └── finalize_node.py   # 结果输出节点
│   │   │
│   │   └── tools/                 # 节点工具函数
│   │       ├── __init__.py
│   │       ├── analyze_template.py
│   │       ├── enhance_prompt.py
│   │       ├── generate_images.py
│   │       ├── evaluate_and_rank.py
│   │       ├── finalize.py
│   │       └── _image_utils.py    # 图片处理工具
│   │
│   ├── providers/                 # 文生图 API 提供商
│   │   ├── __init__.py            # Provider 注册 & 能力查询
│   │   ├── base.py                # 基础接口 (ImageProvider, GenerationMode)
│   │   ├── gemini.py              # Gemini (text_to_image + image_to_image)
│   │   └── antigravity.py         # Antigravity OpenAI 兼容 (text_to_image only)
│   │
│   ├── models/                    # 数据模型
│   │   ├── __init__.py
│   │   ├── schemas.py             # Pydantic 模型
│   │   └── enums.py               # 枚举定义
│   │
│   ├── services/                  # 业务服务
│   │   ├── __init__.py
│   │   └── generation_service.py
│   │
│   ├── db/                        # 数据库
│   │   ├── __init__.py
│   │   └── database.py            # SQLite 数据库
│   │
│   └── api/                       # FastAPI 应用
│       ├── __init__.py
│       ├── app.py
│       ├── routes/
│       │   └── generation.py
│       └── websocket.py
│
├── frontend/src/                  # React + Vite 前端
│   ├── components/                # React 组件
│   ├── hooks/                     # 自定义 Hooks
│   ├── lib/                       # 工具库 & API 客户端
│   └── locales/                   # 国际化
│
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── docker/
│   ├── Dockerfile
│   ├── Dockerfile.frontend
│   └── docker-compose.yml
│
└── docs/
    ├── PRD.md
    └── TECHNICAL_DESIGN.md
```

---

## 4. 核心模块设计

### 4.1 共享状态定义

```python
# backend/src/pictactic/agents/state.py
from dataclasses import dataclass, field
from typing import Annotated, Optional
from langgraph.graph.message import add_messages
from typing_extensions import NotRequired, TypedDict


class GeneratedImageInfo(TypedDict):
    """Information about a generated image."""
    id: str
    url: str
    local_path: str | None
    score: float
    review: str
    round_number: int
    provider: str


class GenerationState(TypedDict):
    """StateGraph 工作流的共享状态"""

    # Messages (用于 LLM 调用)
    messages: Annotated[list, add_messages]

    # 用户输入
    user_prompt: NotRequired[str]           # 用户原始提示词
    template_images: list[str]              # 模板图片 (base64 或 URL)

    # 分析 / 提示词
    template_analysis: Optional[dict]       # 模板分析结果
    enhanced_prompt: Optional[str]          # 增强后的提示词

    # 生成结果
    generated_images: list[GeneratedImageInfo]  # 所有生成的图片
    best_images: list[GeneratedImageInfo]       # 最佳图片列表
    current_round: int                          # 当前轮数

    # 评估反馈 (用于迭代增强)
    evaluation_feedback: NotRequired[str]

    # 状态
    status: Optional[str]                   # pending/enhanced/generated/reviewed/completed
    final_result: Optional[dict]
    error: Optional[str]

    # 配置 (从请求复制)
    max_rounds: NotRequired[int]
    images_per_round: NotRequired[int]
    aspect_ratio: NotRequired[str]
    image_size: NotRequired[str]
    provider: NotRequired[str]
    enhance_prompt_enabled: NotRequired[bool]
    quality_threshold: NotRequired[float]
```

**新增字段说明**：
- `user_prompt`: 保存用户原始提示词，用于评估时参考
- `evaluation_feedback`: 评估节点生成的改进建议，供下一轮 `enhance_node` 使用

### 4.2 StateGraph 工作流定义

```python
# backend/src/pictactic/agents/workflow.py
from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, START, StateGraph

from .conditions import check_template, should_continue
from .nodes import (
    analyze_node,
    enhance_node,
    evaluate_node,
    finalize_node,
    generate_node,
    prepare_next_node,
)
from .state import GenerationState


def create_generation_workflow(checkpointer=None):
    """创建并返回图片生成工作流。

    Args:
        checkpointer: LangGraph checkpointer 用于状态持久化。
                     如果为 None，使用内存存储。

    Returns:
        编译后的 StateGraph，可直接执行。
    """
    builder = StateGraph(GenerationState)

    # 添加所有节点
    builder.add_node("analyze", analyze_node)
    builder.add_node("enhance", enhance_node)
    builder.add_node("generate", generate_node)
    builder.add_node("evaluate", evaluate_node)
    builder.add_node("prepare_next", prepare_next_node)
    builder.add_node("finalize", finalize_node)

    # 入口：检查是否有模板图片
    builder.add_conditional_edges(
        START,
        check_template,
        {
            "analyze": "analyze",
            "enhance": "enhance",
        },
    )

    # analyze -> enhance
    builder.add_edge("analyze", "enhance")

    # enhance -> generate
    builder.add_edge("enhance", "generate")

    # generate -> evaluate
    builder.add_edge("generate", "evaluate")

    # evaluate -> should_continue (条件边)
    builder.add_conditional_edges(
        "evaluate",
        should_continue,
        {
            "continue": "prepare_next",
            "finalize": "finalize",
        },
    )

    # prepare_next -> enhance (迭代)
    builder.add_edge("prepare_next", "enhance")

    # finalize -> END
    builder.add_edge("finalize", END)

    # 编译并返回
    return builder.compile(checkpointer=checkpointer or MemorySaver())
```

### 4.2.1 条件函数

```python
# backend/src/pictactic/agents/conditions.py
from typing import Literal
from .state import GenerationState


def check_template(state: GenerationState) -> Literal["analyze", "enhance"]:
    """检查是否有模板图片。"""
    template_images = state.get("template_images", [])
    return "analyze" if template_images else "enhance"


def should_continue(state: GenerationState) -> Literal["continue", "finalize"]:
    """判断是否继续生成。

    继续条件：
    - 当前轮数 < 最大轮数 AND
    - 最高分 < 质量阈值
    """
    current_round = state.get("current_round", 0)
    max_rounds = state.get("max_rounds", 3)
    quality_threshold = state.get("quality_threshold", 0.7)
    generated_images = state.get("generated_images", [])

    # 获取当前轮最高分
    current_round_images = [
        img for img in generated_images
        if img.get("round_number") == current_round
    ]
    best_score = max(
        (img.get("score", 0) for img in current_round_images),
        default=0
    )

    # 检查终止条件
    quality_met = best_score >= quality_threshold
    rounds_exhausted = current_round >= max_rounds

    if quality_met or rounds_exhausted:
        return "finalize"
    return "continue"
```

### 4.3 enhance_node（提示词增强节点）

```python
# backend/src/pictactic/agents/nodes/enhance_node.py
import json
import logging
from langchain.chat_models import init_chat_model
from langchain_core.messages import HumanMessage, SystemMessage
from ..state import GenerationState

logger = logging.getLogger(__name__)

ENHANCEMENT_SYSTEM_PROMPT = """You are an expert at crafting prompts for AI image generation,
specifically for social media cover images and thumbnails.

Your goal is to transform user prompts into optimized versions that produce:
- High visual quality
- Clear composition
- Engaging visuals for social media
- Professional aesthetics

Guidelines:
- Keep the user's core intent
- Add specific visual details (lighting, perspective, depth)
- Include quality modifiers (high resolution, detailed, professional)
- Incorporate style elements when provided
- Keep prompts focused and not overly long (max 150 words)"""


async def enhance_node(state: GenerationState) -> dict:
    """增强用户提示词。

    该节点使用 LLM 将用户提示词转换为优化版本，考虑：
    - 模板分析结果（如果有）
    - 上一轮评估反馈（迭代模式）

    Args:
        state: 当前工作流状态。

    Returns:
        包含 enhanced_prompt 和 status 的字典。
    """
    # 检查是否禁用增强
    if not state.get("enhance_prompt_enabled", True):
        return {
            "enhanced_prompt": state.get("user_prompt", ""),
            "status": "enhanced",
        }

    user_prompt = state.get("user_prompt", "")
    feedback = state.get("evaluation_feedback", "")
    template_analysis = state.get("template_analysis")

    llm = init_chat_model(
        "gpt-4o",
        model_provider="openai",
        temperature=0.7,
    )

    if feedback:
        # 基于反馈的迭代增强
        current_prompt = state.get("enhanced_prompt") or user_prompt
        user_message = f"""Improve this image generation prompt based on feedback:

Current prompt: {current_prompt}

Feedback from evaluation: {feedback}

Create an improved version that addresses the feedback.
Output ONLY the improved prompt, nothing else."""
    else:
        # 初次增强
        style_context = ""
        if template_analysis:
            style_context = f"\n\nReference style:\n{json.dumps(template_analysis, ensure_ascii=False)}"

        user_message = f"""Enhance this prompt for AI image generation:

Original prompt: {user_prompt}
{style_context}

Create an enhanced version. Output ONLY the enhanced prompt."""

    response = await llm.ainvoke([
        SystemMessage(content=ENHANCEMENT_SYSTEM_PROMPT),
        HumanMessage(content=user_message),
    ])

    return {
        "enhanced_prompt": response.content.strip().strip('"\''),
        "status": "enhanced",
    }
```

### 4.4 generate_node（图片生成节点）

```python
# backend/src/pictactic/agents/nodes/generate_node.py
import uuid
from langgraph.config import get_stream_writer
from ...providers import get_provider
from ...providers.base import GenerationParams
from ..state import GeneratedImageInfo, GenerationState


async def generate_node(state: GenerationState) -> dict:
    """并发生成图片，流式输出进度。

    该节点：
    1. 从状态读取增强后的提示词
    2. 使用 Provider 的批量流式生成
    3. 通过 get_stream_writer 流式输出每张完成的图片
    4. 更新状态中的图片列表

    Args:
        state: 当前工作流状态。

    Returns:
        包含 generated_images, current_round, status 的字典。
    """
    writer = get_stream_writer()

    # 获取提示词
    prompt = state.get("enhanced_prompt") or state.get("user_prompt", "")

    # 获取配置
    num_images = state.get("images_per_round", 5)
    provider_name = state.get("provider", "gemini")
    aspect_ratio = state.get("aspect_ratio", "16:9")
    image_size = state.get("image_size", "2K")
    current_round = state.get("current_round", 0) + 1

    # 获取参考图片
    template_images = state.get("template_images", [])

    # 创建 Provider 和参数
    provider = get_provider(provider_name)
    params = GenerationParams(
        prompt=prompt,
        aspect_ratio=aspect_ratio,
        size=image_size,
        reference_images=template_images,
    )

    # 生成图片，带流式回调
    images: list[GeneratedImageInfo] = []

    async def on_image_ready(idx: int, result):
        """每张图片完成时的回调。"""
        image_info: GeneratedImageInfo = {
            "id": str(uuid.uuid4()),
            "url": result.url,
            "local_path": result.local_path,
            "score": 0.0,
            "review": "",
            "round_number": current_round,
            "provider": provider_name,
        }
        images.append(image_info)

        # 流式输出到客户端
        if writer:
            writer({"type": "image_ready", "image": image_info})

    await provider.generate_batch_stream(
        params, num_images, on_image_ready=on_image_ready
    )

    # 合并到已有图片列表
    all_images = state.get("generated_images", []) + images

    return {
        "generated_images": all_images,
        "current_round": current_round,
        "status": "generated",
    }
```

### 4.5 evaluate_node（图片评估节点）

```python
# backend/src/pictactic/agents/nodes/evaluate_node.py
import json
from langchain.chat_models import init_chat_model
from langchain_core.messages import HumanMessage
from ..state import GenerationState

COMPARATIVE_PROMPT_TEMPLATE = """你是封面图评审专家。以下是 {n} 张候选封面图。
用户需求：{prompt}
{reference_note}
请对比所有图片，从最好到最差排序。
评估标准：
1. 与用户需求的匹配度
2. 视觉吸引力和专业感
3. 构图和色彩协调性
4. 适合社交媒体使用

输出 JSON：
{{
    "rankings": [按从好到差排序的图片序号],
    "reviews": [{{"index": 序号, "brief": "简短评价"}}],
    "best_reason": "最佳图片胜出原因",
    "improvement_suggestions": "对下一轮生成的改进建议"
}}"""


async def evaluate_node(state: GenerationState) -> dict:
    """评估并排序当前轮的图片。

    该节点使用 Vision LLM：
    1. 对比当前轮所有图片
    2. 从最佳到最差排序
    3. 基于排名分配分数
    4. 生成改进建议供下一轮使用

    Args:
        state: 当前工作流状态。

    Returns:
        包含更新后的 generated_images (带分数) 和 evaluation_feedback 的字典。
    """
    generated_images = list(state.get("generated_images", []))
    current_round = state.get("current_round", 1)
    user_prompt = state.get("user_prompt", "")
    template_images = state.get("template_images", [])

    # 获取当前轮图片
    current_round_images = [
        img for img in generated_images
        if img.get("round_number") == current_round
    ]
    image_urls = [img["url"] for img in current_round_images]
    n = len(image_urls)

    if n == 0:
        return {"status": "reviewed"}

    # 构建带图片的消息
    reference_note = "参考照片已附上，请考虑风格一致性。" if template_images else ""
    prompt_text = COMPARATIVE_PROMPT_TEMPLATE.format(
        n=n, prompt=user_prompt, reference_note=reference_note
    )

    content = [{"type": "text", "text": prompt_text}]
    # 添加参考图片和候选图片...

    llm = init_chat_model("gpt-4o", model_provider="openai", temperature=0.2)
    response = await llm.ainvoke([HumanMessage(content=content)])
    data = _parse_json(response.content, n)

    # 基于排名分配分数
    for rank_pos, img_index in enumerate(data.get("rankings", [])):
        # 第一名 0.95，每名降 0.05
        score = max(0.95 - rank_pos * 0.05, 0.10)
        if 0 <= img_index - 1 < len(current_round_images):
            current_round_images[img_index - 1]["score"] = score

    # 提取改进建议作为下一轮反馈
    improvement = data.get("improvement_suggestions", "")
    best_reason = data.get("best_reason", "")
    evaluation_feedback = f"{best_reason}\n改进建议: {improvement}" if improvement else best_reason

    return {
        "generated_images": generated_images,
        "evaluation_feedback": evaluation_feedback,
        "status": "reviewed",
    }
```

**评估流程改进**：
- 使用比较式排序替代独立打分，结果更稳定
- 自动生成改进建议，供迭代时增强提示词
- 分数基于排名计算，避免 Vision LLM 打分不稳定问题

---

## 5. 数据模型

### 5.1 Pydantic Schemas

```python
# src/pictactic/models/schemas.py
from pydantic import BaseModel, Field
from typing import Optional, List
from datetime import datetime
from enum import Enum


class ImageProvider(str, Enum):
    GEMINI = "gemini"
    ANTIGRAVITY = "antigravity"


class GenerationStatus(str, Enum):
    PENDING = "pending"
    ENHANCING = "enhancing"
    GENERATING = "generating"
    REVIEWING = "reviewing"
    COMPLETED = "completed"
    FAILED = "failed"


class GenerationRequest(BaseModel):
    """图片生成请求"""
    prompt: str = Field(..., description="用户输入的提示词")
    template_images: List[str] = Field(default=[], description="参考模板图片")
    max_rounds: int = Field(default=3, ge=1, le=10)
    images_per_round: int = Field(default=5, ge=1, le=10)
    quality_threshold: float = Field(default=0.7, ge=0.0, le=1.0)
    providers: List[ImageProvider] = Field(default=[ImageProvider.DALLE])


class TemplateAnalysis(BaseModel):
    """模板分析结果"""
    style: str
    colors: List[str]
    layout: str
    mood: str
    elements: List[str]
    suggested_keywords: List[str]


class GeneratedImage(BaseModel):
    """生成的图片"""
    id: str
    url: str
    local_path: Optional[str] = None
    prompt: str
    revised_prompt: Optional[str] = None
    provider: ImageProvider
    score: float = Field(default=0.0, ge=0.0, le=1.0)
    score_details: Optional[dict] = None
    created_at: datetime = Field(default_factory=datetime.utcnow)


class GenerationResult(BaseModel):
    """完整的生成结果"""
    task_id: str
    status: GenerationStatus
    original_prompt: str
    enhanced_prompt: Optional[str] = None
    template_analysis: Optional[TemplateAnalysis] = None
    rounds: List[dict] = []
    total_rounds: int = 0
    total_images: int = 0
    best_images: List[GeneratedImage] = []
    started_at: datetime
    completed_at: Optional[datetime] = None
    error: Optional[str] = None


class TaskProgress(BaseModel):
    """任务进度"""
    task_id: str
    status: GenerationStatus
    current_round: int
    max_rounds: int
    images_generated: int
    current_best_score: float
    message: str
```

---

## 6. API 设计

### 6.1 接口列表

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/v1/generation/provider/capabilities | 获取当前 Provider 能力 |
| POST | /api/v1/generation/ | 创建生成任务 |
| GET | /api/v1/generation/{task_id} | 获取任务结果 |
| POST | /api/v1/generation/{task_id}/stop | 停止任务 |
| POST | /api/v1/generation/{task_id}/continue | 继续生成 |
| POST | /api/v1/templates/analyze | 分析模板图片 |
| POST | /api/v1/templates/upload | 上传模板 |
| GET | /api/v1/templates/presets | 获取预设风格 |
| WS | /ws/progress/{task_id} | 实时进度推送 |

### 6.2 API 示例

```python
# POST /api/v1/generation/
{
    "prompt": "科技感的AI主题封面",
    "template_images": ["https://example.com/ref1.jpg"],
    "max_rounds": 3,
    "images_per_round": 5,
    "quality_threshold": 0.7,
    "providers": ["dalle"]
}

# Response
{
    "task_id": "abc123",
    "status": "pending",
    "current_round": 0,
    "max_rounds": 3,
    "images_generated": 0,
    "current_best_score": 0.0,
    "message": "任务已创建，正在启动生成流程..."
}
```

---

## 7. 文生图 Provider 设计

### 7.1 基类接口

```python
# src/pictactic/providers/base.py
from abc import ABC, abstractmethod
from enum import Enum
from pydantic import BaseModel


class GenerationMode(str, Enum):
    TEXT_TO_IMAGE = "text_to_image"
    IMAGE_TO_IMAGE = "image_to_image"


class GenerationParams(BaseModel):
    prompt: str
    negative_prompt: str | None = None
    aspect_ratio: str = "16:9"
    size: str = "2K"
    reference_images: list[str] = []

    def get_mode(self) -> GenerationMode:
        return GenerationMode.IMAGE_TO_IMAGE if self.reference_images else GenerationMode.TEXT_TO_IMAGE

    def validate_mode(self, supported_modes: list[GenerationMode]) -> None:
        """校验请求模式是否被 provider 支持"""
        mode = self.get_mode()
        if mode not in supported_modes:
            raise ProviderError(...)


class ImageProvider(ABC):
    @property
    @abstractmethod
    def name(self) -> str: ...

    @property
    @abstractmethod
    def supports_batch(self) -> bool: ...

    @property
    def supported_modes(self) -> list[GenerationMode]:
        """Provider 支持的模式，子类可覆盖以限制"""
        return [GenerationMode.TEXT_TO_IMAGE, GenerationMode.IMAGE_TO_IMAGE]

    @abstractmethod
    async def generate(self, params: GenerationParams, index: int = 0) -> GenerationResult: ...
```

### 7.2 Provider 注册与能力查询

```python
# src/pictactic/providers/__init__.py

_PROVIDERS = {
    "gemini": GeminiImageProvider,      # 支持 text_to_image + image_to_image
    "antigravity": AntigravityImageProvider,  # 仅支持 text_to_image
}

def get_provider(name: str) -> ImageProvider: ...
def get_provider_capabilities(name: str) -> dict:
    """返回 {name, supported_modes, supports_batch}"""
```

前端通过 `GET /api/v1/generation/provider/capabilities` 获取当前 provider 能力，自动禁用不支持的模式（如 antigravity 下隐藏"图片编辑"）。

### 7.3 Gemini Provider

支持 text_to_image 和 image_to_image，通过不同 API endpoint 调用。

### 7.4 Antigravity Provider

使用 OpenAI 兼容 API（反向代理），仅支持 text_to_image。

```python
class AntigravityImageProvider(ImageProvider):
    supported_modes = [GenerationMode.TEXT_TO_IMAGE]

    async def generate(self, params, index=0):
        # 使用 OpenAI SDK，response 为 markdown 格式的 base64 图片
        # ![image](data:image/jpeg;base64,...)
        # 解析 base64 并保存为本地文件
```

通过环境变量配置：
- `DEFAULT_PROVIDER` — 选择默认 provider
- `ANTIGRAVITY_API_BASE_URL` / `ANTIGRAVITY_API_KEY` / `ANTIGRAVITY_MODEL`

---

## 8. 前端设计（Gradio）

```python
# src/pictactic/ui/app.py
import gradio as gr


def create_ui():
    with gr.Blocks(
        title="PicTacticAgent",
        theme=gr.themes.Soft()
    ) as demo:

        gr.Markdown("# PicTacticAgent - AI 封面图生成器")

        with gr.Row():
            with gr.Column(scale=1):
                prompt_input = gr.Textbox(
                    label="描述你想要的封面",
                    lines=3
                )
                template_upload = gr.File(
                    label="上传参考图片（可选）",
                    file_count="multiple",
                    file_types=["image"]
                )

                with gr.Accordion("高级配置", open=False):
                    max_rounds = gr.Slider(1, 10, value=3, step=1, label="最大轮次")
                    images_per_round = gr.Slider(1, 10, value=5, step=1, label="每轮数量")
                    quality_threshold = gr.Slider(0.5, 1.0, value=0.7, step=0.05, label="质量阈值")

                generate_btn = gr.Button("开始生成", variant="primary")

            with gr.Column(scale=2):
                progress = gr.Progress()
                status_text = gr.Textbox(label="状态", interactive=False)
                enhanced_prompt = gr.Textbox(label="增强后的提示词", interactive=False)
                result_gallery = gr.Gallery(label="生成结果", columns=4)

        # 事件绑定...

    return demo
```

---

## 9. 关键技术难点与解决方案

### 9.1 DALL-E 3 批量调用

**问题**：DALL-E 3 每次只能生成 1 张图片

**解决**：使用 asyncio 并行调用 + 信号量限流

```python
async def parallel_generate(provider, prompt, count):
    semaphore = asyncio.Semaphore(5)  # 限制并发

    async def limited_generate(index):
        async with semaphore:
            return await provider.generate(prompt, index)

    tasks = [limited_generate(i) for i in range(count)]
    return await asyncio.gather(*tasks, return_exceptions=True)
```

### 9.2 图片质量评估

**问题**：如何准确评估图片质量

**解决**：多维度评分

```python
async def score_image(image_path, prompt, template):
    # 1. CLIP 图文匹配度
    clip_score = await calculate_clip_score(image_path, prompt)

    # 2. Vision LLM 主观评估
    llm_score = await evaluate_with_vision_llm(image_path, prompt, template)

    # 3. 技术质量检测
    tech_score = await check_technical_quality(image_path)

    # 综合评分
    return clip_score * 0.3 + llm_score * 0.5 + tech_score * 0.2
```

### 9.3 长任务状态管理

**问题**：生成过程可能持续数分钟

**解决**：使用 LangGraph Checkpointer 实现断点续传

```python
from langgraph.checkpoint.postgres import PostgresSaver

checkpointer = PostgresSaver.from_conn_string(DATABASE_URL)
graph = workflow_builder.compile(checkpointer=checkpointer)

# 恢复任务
async def resume_task(task_id):
    config = {"configurable": {"thread_id": task_id}}
    state = await graph.aget_state(config)
    async for event in graph.astream(None, config):
        yield event
```

---

## 10. 部署方案

### 10.1 Docker Compose

```yaml
# docker/docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@db:5432/pictactic
      - REDIS_URL=redis://redis:6379
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - db
      - redis

  db:
    image: postgres:15
    environment:
      POSTGRES_DB: pictactic
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### 10.2 环境变量

```bash
# .env.example
# API Keys
OPENAI_API_KEY=sk-xxx
STABILITY_API_KEY=sk-xxx

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/pictactic

# Redis
REDIS_URL=redis://localhost:6379

# Storage
STORAGE_TYPE=local  # local / s3
S3_BUCKET=pictactic-images
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
```

---

## 11. 验证方案

### 11.1 安装依赖

```bash
# 使用 uv (推荐)
uv sync --all-extras

# 或使用 pip
pip install -e ".[dev]"
```

### 11.2 单元测试

```bash
# 使用 uv
uv run pytest tests/unit/ -v

# 或直接运行
pytest tests/unit/ -v
```

### 11.3 集成测试

```bash
# 使用 uv
uv run pytest tests/integration/ -v

# 或直接运行
pytest tests/integration/ -v
```

### 11.4 端到端测试

1. 启动服务：`docker-compose up`
2. 访问 UI：`http://localhost:8000`
3. 输入测试提示词
4. 验证生成流程和结果

### 11.5 本地开发启动

```bash
# 使用 uv 启动后端
uv run uvicorn backend.src.pictactic.api.app:app --reload --port 8000
```
