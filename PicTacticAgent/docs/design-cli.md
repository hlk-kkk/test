# CLI 接口设计方案

## 1. 概述

为 PicTacticAgent 添加 CLI 入口，用户可通过命令行直接调用图片生成能力，无需启动 HTTP 服务。CLI 直接调用 service 层（`GenerationService` + LangGraph workflow），适用于批量脚本、CI/CD 流水线、以及 Claude Code Skill 集成。

## 2. 技术选型

| 组件 | 选型 | 理由 |
|------|------|------|
| CLI 框架 | typer (>=0.9.0) | 基于 click，类型注解驱动，自动生成帮助文档 |
| 进度显示 | rich (>=13.0.0) | 终端进度条、表格、Panel 渲染 |
| JSON 输出 | 标准库 json | 结构化输出供脚本/agent 解析 |

### pyproject.toml 变更

```toml
[project]
dependencies = [
    # ... 现有依赖 ...
    "typer>=0.9.0",
    "rich>=13.0.0",
]

[project.scripts]
pictactic = "pictactic.cli.main:app"
```

## 3. 模块结构

```
backend/src/pictactic/cli/
├── __init__.py
├── main.py          # typer.Typer() 应用入口 + 子命令注册
├── generate.py      # pictactic generate 命令实现
├── providers.py     # pictactic providers 命令实现
├── status.py        # pictactic status 命令实现
└── output.py        # 输出格式化（text/json/quiet 三种模式）
```

## 4. 命令设计

### 4.1 `pictactic generate`

核心命令，直接调用 LangGraph workflow 生成图片。

```
pictactic generate [OPTIONS] PROMPT
```

**参数对齐 `GenerationRequest`：**

| CLI 参数 | 类型 | 默认值 | 对应字段 | 说明 |
|----------|------|--------|----------|------|
| `PROMPT` | str (必选) | - | `prompt` | 生成提示词 |
| `--template` / `-t` | list[Path] | [] | `template_images` | 模板图片路径（可多次指定） |
| `--rounds` / `-r` | int | 3 | `max_rounds` | 最大生成轮次 |
| `--images` / `-n` | int | 5 | `images_per_round` | 每轮生成图片数 |
| `--threshold` / `-q` | float | 0.7 | `quality_threshold` | 质量阈值 |
| `--aspect-ratio` / `-a` | str | "16:9" | `aspect_ratio` | 画面比例 |
| `--size` / `-s` | str | "2K" | `size` | 图片尺寸 |
| `--provider` / `-p` | str | None (读 .env) | `provider` | 图片生成 provider |
| `--no-enhance` | bool flag | False | `enhance_prompt=False` | 禁用 prompt 增强 |
| `--no-ai-scoring` | bool flag | False | `enable_ai_scoring=False` | 禁用 AI 评分 |
| `--output-dir` / `-o` | Path | ./output | - | 图片保存目录 |
| `--format` / `-f` | str | "text" | - | 输出模式: text/json/quiet |

**使用示例：**

```bash
# 基本用法
pictactic generate "一只戴墨镜的猫咪在海边冲浪"

# 完整参数
pictactic generate "科技感封面" \
  --template cover1.png --template cover2.png \
  --rounds 5 --images 8 --threshold 0.8 \
  --aspect-ratio 16:9 --size 2K \
  --provider gemini \
  --output-dir ./my-images

# JSON 输出（供脚本解析）
pictactic generate "产品海报" --format json

# 静默模式（仅输出文件路径）
pictactic generate "Logo设计" --format quiet
```

### 4.2 `pictactic providers`

列出可用的图片生成 provider 及其能力。

```
pictactic providers [OPTIONS]
```

| CLI 参数 | 类型 | 默认值 | 说明 |
|----------|------|--------|------|
| `--format` / `-f` | str | "text" | 输出模式: text/json |

**输出示例（text 模式）：**

```
Available Providers
┌──────────────┬────────────────────────────┬───────┬─────────┐
│ Provider     │ Modes                      │ Batch │ Default │
├──────────────┼────────────────────────────┼───────┼─────────┤
│ gemini       │ text_to_image, i2i         │ Yes   │ *       │
│ jiekouai     │ text_to_image, i2i         │ Yes   │         │
│ antigravity  │ text_to_image, i2i         │ Yes   │         │
└──────────────┴────────────────────────────┴───────┴─────────┘
```

### 4.3 `pictactic status`

查询正在运行的任务状态（连接到后端服务）。此命令需要后端 HTTP 服务运行。

```
pictactic status [TASK_ID] [OPTIONS]
```

| CLI 参数 | 类型 | 默认值 | 说明 |
|----------|------|--------|------|
| `TASK_ID` | str (可选) | None | 指定 task_id；省略则列出所有任务 |
| `--api-url` | str | http://localhost:8019 | 后端 API 地址 |
| `--format` / `-f` | str | "text" | 输出模式: text/json |

## 5. 输出模式设计

### 5.1 text 模式（默认，人类可读）

使用 `rich` 库渲染：
- 启动时显示配置摘要 Panel
- 每轮显示 Rich Progress Bar（`[Round 1/3] Generating... 3/5`）
- 评分阶段显示 Table（图片路径 + 分数）
- 完成后显示 Best Images 汇总 Panel

```
╭─ PicTactic Generation ─────────────────────╮
│  Prompt:     一只戴墨镜的猫咪在海边冲浪        │
│  Provider:   gemini                         │
│  Rounds:     3 × 5 images                   │
│  Threshold:  0.70                           │
╰─────────────────────────────────────────────╯

[Round 1/3] Generating ━━━━━━━━━━━━━━━━━━ 5/5 100%
[Round 1/3] Evaluating ━━━━━━━━━━━━━━━━━━ 5/5 100%

  Best: 0.85 → ./output/img_001.png

[Round 2/3] Generating ━━━━━━━━━━━━━━━━━━ 5/5 100%
...

╭─ Results ───────────────────────────────────╮
│  Total images:  15                          │
│  Best score:    0.92                        │
│  Output:        ./output/                   │
│  Best images:                               │
│    1. ./output/img_007.png  (0.92)          │
│    2. ./output/img_012.png  (0.88)          │
│    3. ./output/img_001.png  (0.85)          │
╰─────────────────────────────────────────────╯
```

### 5.2 json 模式（机器/agent 可读）

每个阶段输出一行 JSON（JSONL 格式），字段与 `TaskProgress` / `GenerationResult` 对齐：

```jsonl
{"event":"start","config":{"prompt":"...","provider":"gemini","max_rounds":3,"images_per_round":5}}
{"event":"round_start","round":1}
{"event":"image_ready","round":1,"image":{"id":"...","url":"...","local_path":"./output/img_001.png"}}
{"event":"round_end","round":1,"best_score":0.85}
{"event":"complete","total_images":15,"best_score":0.92,"best_images":[...]}
```

### 5.3 quiet 模式（管道友好）

仅输出最终 best images 的本地文件路径，每行一个：

```
./output/img_007.png
./output/img_012.png
./output/img_001.png
```

## 6. 核心实现方案

### 6.1 直接调用 Service 层

CLI 不启动 FastAPI，直接构建 workflow 并 `astream`。关键是复用 `create_generation_workflow()` 和 `create_initial_state()`，绕过 HTTP 层。

```python
# backend/src/pictactic/cli/generate.py 核心逻辑伪代码

import asyncio
import typer
from ..agents.workflow import create_generation_workflow
from ..agents.state import create_initial_state

async def run_generation(prompt: str, **kwargs) -> dict:
    workflow = create_generation_workflow()
    initial_state = create_initial_state(
        user_prompt=prompt,
        template_images=kwargs.get("template_images", []),
        max_rounds=kwargs.get("max_rounds", 3),
        images_per_round=kwargs.get("images_per_round", 5),
        quality_threshold=kwargs.get("quality_threshold", 0.7),
        aspect_ratio=kwargs.get("aspect_ratio", "16:9"),
        image_size=kwargs.get("size", "2K"),
        provider=kwargs.get("provider", "gemini"),
        enhance_prompt_enabled=kwargs.get("enhance_prompt", True),
        enable_ai_scoring=kwargs.get("enable_ai_scoring", True),
    )

    config = {"configurable": {"thread_id": f"cli-{uuid4()}"}}

    async for event in workflow.astream(initial_state, config=config, stream_mode="custom"):
        yield event  # 交给 output formatter 处理

    final_snapshot = await workflow.aget_state(config)
    yield {"event": "final_state", "state": final_snapshot.values}
```

### 6.2 模板图片处理

CLI 接收本地文件路径，需要转换为 base64 编码传入 workflow：

```python
import base64
from pathlib import Path

def load_template_images(paths: list[Path]) -> list[str]:
    """将本地图片文件转换为 base64 data URI."""
    result = []
    for path in paths:
        if not path.exists():
            raise typer.BadParameter(f"Template image not found: {path}")
        data = path.read_bytes()
        suffix = path.suffix.lower().lstrip(".")
        mime = {"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg",
                "webp": "image/webp"}.get(suffix, "image/png")
        encoded = base64.b64encode(data).decode()
        result.append(f"data:{mime};base64,{encoded}")
    return result
```

### 6.3 图片保存

生成的图片需要从 URL 下载到 `--output-dir` 指定的本地目录：

```python
import httpx
from pathlib import Path

async def save_images(images: list[dict], output_dir: Path) -> list[Path]:
    """下载生成的图片到本地目录."""
    output_dir.mkdir(parents=True, exist_ok=True)
    saved = []
    async with httpx.AsyncClient() as client:
        for img in images:
            url = img.get("url", "")
            if url.startswith("data:"):
                # base64 图片直接解码保存
                ...
            else:
                resp = await client.get(url)
                filename = f"pictactic_{img['id']}.png"
                path = output_dir / filename
                path.write_bytes(resp.content)
                saved.append(path)
    return saved
```

## 7. 退出码设计

| 退出码 | 含义 |
|--------|------|
| 0 | 成功完成 |
| 1 | 一般错误（参数错误、配置缺失） |
| 2 | 生成失败（workflow 执行异常） |
| 3 | 部分成功（有图片生成但未达到质量阈值） |

## 8. Skill 包装方案

为 Claude Code 提供 Skill 配置，使 agent 可以通过 `/generate` 命令调用 CLI。

### 8.1 Skill 定义文件

在项目根目录添加 `.claude/skills/generate.md`：

```markdown
---
name: generate
description: Generate images using PicTacticAgent CLI
command: pictactic generate
---

# /generate - Image Generation

Generate images from text prompts using the PicTacticAgent CLI.

## Usage

/generate <prompt> [options]

## Arguments

- `prompt` (required): The text description for image generation

## Options

- `--template <path>`: Template image path (can specify multiple)
- `--rounds <n>`: Max generation rounds (default: 3)
- `--images <n>`: Images per round (default: 5)
- `--threshold <f>`: Quality threshold 0.0-1.0 (default: 0.7)
- `--aspect-ratio <ratio>`: e.g. 16:9, 1:1, 9:16 (default: 16:9)
- `--size <size>`: 1K, 2K, or 4K (default: 2K)
- `--provider <name>`: gemini, jiekouai, or antigravity
- `--no-enhance`: Skip prompt enhancement
- `--output-dir <path>`: Output directory (default: ./output)

## Examples

```
/generate "A cat surfing at the beach wearing sunglasses"
/generate "Tech cover image" --template cover1.png --rounds 5
/generate "Product poster" --format json
```

The command outputs JSON (when `--format json`) that can be parsed to get
image paths and scores. Use `--format quiet` to get just file paths.
```

### 8.2 CLAUDE.md 追加

在 CLAUDE.md 中追加 CLI 使用说明：

```markdown
### CLI Usage
```bash
# Install CLI entry point
uv pip install -e .

# Generate images
pictactic generate "your prompt here"

# List providers
pictactic providers

# Check task status (requires backend running)
pictactic status [task_id] --api-url http://localhost:8019
```
```

## 9. 错误处理策略

| 场景 | 处理方式 |
|------|----------|
| API Key 未配置 | 启动时检查，提示用户设置 .env |
| 模板图片不存在 | `typer.BadParameter` 报错退出 |
| 网络请求失败 | 重试 1 次，失败后报错（text 模式显示 rich Panel） |
| Workflow 中某轮全部失败 | 继续下一轮，最终汇总时标记 partial success |
| 用户 Ctrl+C | 捕获 KeyboardInterrupt，清理临时文件，优雅退出 |

## 10. 依赖关系

```
CLI Layer (typer + rich)
    │
    ├── 直接调用 → agents/workflow.py (create_generation_workflow)
    ├── 直接调用 → agents/state.py (create_initial_state)
    ├── 读取配置 → core/config.py (settings)
    └── 查询能力 → providers/ (get_provider_capabilities)
```

CLI 不依赖 FastAPI、WebSocket、数据库等 HTTP 层组件。唯一例外是 `pictactic status` 命令需要通过 HTTP 连接后端。
