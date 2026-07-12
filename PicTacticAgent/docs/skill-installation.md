# PicTacticAgent — Claude Code Skill 安装指南

本文档是给 AI Agent 阅读的安装指南。按照步骤执行，即可将 PicTacticAgent 安装为 Claude Code Skill，让任意 Coding Agent 或 AI Agent 都能通过 `pictactic` 命令快速生成和编辑图片。

---

## 功能概述

安装后，Skill 支持以下三种图片操作模式：

| 模式 | 说明 | 触发示例 |
|------|------|---------|
| **文生图** | 根据提示词直接生成图片 | "帮我生成一张赛博朋克风格的封面图" |
| **参考图生成** | 上传参考图，生成风格一致的图片（文生图时的风格参考） | "根据这张参考图生成一张同风格的图片" |
| **图片编辑** | 对已生成的图片进行文本引导的编辑修改 | "把这张图片的天空改成夕阳色调" |

> **注意**：图片编辑（edit 命令）通过 CLI 直接调用，不需要启动 Web UI。

---

## 前提条件检查

在安装前，请先确认以下工具已安装：

```bash
# 检查 Python 版本（需要 3.11+）
python3 --version

# 检查 uv 是否安装
uv --version

# 如果没有 uv，执行以下命令安装
curl -LsSf https://astral.sh/uv/install.sh | sh
```

---

## 第一步：克隆并安装项目

```bash
# 1. 克隆仓库（选择一个合适的目录）
git clone https://github.com/NanmiCoder/PicTacticAgent.git
cd PicTacticAgent

# 2. 安装 Python 依赖
uv sync

# 3. 验证 CLI 安装成功
uv run pictactic --help
```

成功输出示例：
```
Usage: pictactic [OPTIONS] COMMAND [ARGS]...

  PicTactic - AI-driven image generation from the command line.

Commands:
  generate   Generate images from a text prompt
  edit       Edit an existing image with a text prompt
  providers  List available image providers
  status     Check the status of a generation task
```

---

## 第二步：配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env
```

然后编辑 `.env` 文件，**至少配置以下必填项**（选择一个 Provider）：

### 选项 A：使用 Jiekouai 中转站（推荐）

Jiekouai 是一个 API 中转代理服务，**一个 Key 同时搞定两件事**：
- **图片生成**（文生图 + 图片编辑）via `/v3` 端点
- **LLM 评估**（提示词增强、图片 AI 打分）via `/openai` 端点

```env
DEFAULT_PROVIDER=jiekouai
JIEKOUAI_API_KEY=<你的 Jiekouai API Key>
JIEKOUAI_API_BASE_URL=https://api.jiekou.ai/v3
JIEKOUAI_LLM_BASE_URL=https://api.jiekou.ai/openai
JIEKOUAI_LLM_MODEL=gpt-5-mini
```

> **说明**：`JIEKOUAI_LLM_MODEL` 是用于提示词增强和图片评分的语言模型，推荐 `gpt-5-mini`（性价比高），也可用 `gpt-4o`（效果更好但更贵）。

### 选项 B：使用 Gemini 官方 API

```env
DEFAULT_PROVIDER=gemini
GEMINI_API_KEY=<你的 Google AI Studio API Key>
GEMINI_MODEL=gemini-3-pro-image-preview    # 生图模型
GEMINI_LLM_MODEL=gemini-3-flash-preview    # 评估模型
```

API Key 获取地址：https://aistudio.google.com/apikey

### 选项 C：使用 Antigravity 反代（仅支持文生图）

```env
DEFAULT_PROVIDER=antigravity
ANTIGRAVITY_API_KEY=<你的 API Key>
ANTIGRAVITY_API_BASE_URL=http://127.0.0.1:8045/v1
ANTIGRAVITY_MODEL=gemini-3-pro-image       # 生图模型
ANTIGRAVITY_LLM_MODEL=gemini-3-flash       # 评估模型
```

> **注意**：Antigravity 仅支持文生图（TEXT_TO_IMAGE），**不支持**参考图生成和图片编辑（IMAGE_TO_IMAGE）。

### 验证 Provider 配置

```bash
uv run pictactic providers
```

成功输出示例：
```
┌─────────────┬──────────────────────────┬───────┬─────────┐
│ Provider    │ Modes                    │ Batch │ Default │
├─────────────┼──────────────────────────┼───────┼─────────┤
│ jiekouai    │ text_to_image, image_to_ │ No    │ *       │
│ gemini      │ text_to_image, image_to_ │ Yes   │         │
│ antigravity │ text_to_image            │ No    │         │
└─────────────┴──────────────────────────┴───────┴─────────┘
```

---

## 第三步：安装 Claude Code Skill

### 关于项目内 Skill 模板

PicTacticAgent 项目在 `.claude/skills/pictactic-generate.md` 中维护了一份 Skill 模板文件，包含完整的命令说明和参数文档。安装时直接复制此文件并替换路径占位符即可，避免了手动维护两份文档的同步问题。

### 安装步骤

```bash
# 1. 获取当前项目的绝对路径
PROJECT_DIR=$(pwd)
echo "项目目录：$PROJECT_DIR"

# 2. 创建 Skill 目录
mkdir -p ~/.claude/skills/pictactic

# 3. 复制项目中的 Skill 模板
cp "$PROJECT_DIR/.claude/skills/pictactic-generate.md" \
   ~/.claude/skills/pictactic/SKILL.md

# 4. 将模板中的路径占位符替换为实际项目路径
sed -i '' "s|PROJECT_DIR=/path/to/PicTacticAgent|PROJECT_DIR=$PROJECT_DIR|" \
  ~/.claude/skills/pictactic/SKILL.md

echo "✅ Skill 文件已创建：~/.claude/skills/pictactic/SKILL.md"
```

### 验证安装

```bash
# 查看 Skill 文件中的 PROJECT_DIR 是否已替换为实际路径
grep "PROJECT_DIR=" ~/.claude/skills/pictactic/SKILL.md
# 应输出实际路径，而非占位符
# 例如：PROJECT_DIR=/Users/yourname/PicTacticAgent
```

---

## 第四步：验证整体安装

```bash
# 1. 检查 Skill 文件
cat ~/.claude/skills/pictactic/SKILL.md | head -15

# 2. 测试文生图（注意：prompt 必须放在所有 options 之后）
cd <PicTacticAgent 项目路径>
uv run pictactic generate --rounds 1 --images 1 --format quiet "测试图片"
# 成功后应输出类似：./output/pictactic_abc123def.png

# 3. 测试图片编辑（用上一步输出的路径）
uv run pictactic edit \
  --source ./output/pictactic_abc123def.png --format quiet \
  "make the colors more vibrant"
# 成功后应输出类似：./output/pictactic_edit_xyz_0.png
```

---

## 可选：启动 Web UI

如需使用浏览器界面进行图片生成和管理，需要同时启动前后端：

```bash
cd <PicTacticAgent 项目路径>
./start.sh
# 前端访问：http://localhost:3019
# 后端 API：http://localhost:8019/docs
```

---

## Skill 触发示例

安装完成后，在 Claude Code 中可以这样调用：

```
/pictactic
```

或直接描述需求，AI 会自动识别并调用 Skill：

- "帮我生成一张黑白线条风格的龙纹身图案"
- "根据这张图片的风格，生成 4 张类似风格的图片"
- "生成一张 16:9 的赛博朋克城市封面图，高质量"
- "把这张图片的背景改成樱花树林"
- "对 ./output/pictactic_abc.png 进行编辑，增加雾气效果"

---

## 参数速查表

### generate 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `prompt` | - | 必填 | 图片描述提示词 |
| `--template` | `-t` | 无 | 参考图路径（可多次使用，最多 3 张） |
| `--rounds` | `-r` | 3 | 最大迭代轮数 |
| `--images` | `-n` | 5 | 每轮候选图数量（候选池大小） |
| `--top-k` | `-k` | 3 | 最终输出最优图数量 |
| `--threshold` | `-q` | 0.7 | 质量阈值（达到后自动停止） |
| `--aspect-ratio` | `-a` | 16:9 | 宽高比 |
| `--size` | `-s` | 2K | 图片尺寸 |
| `--provider` | `-p` | .env 默认 | 指定 Provider |
| `--no-enhance` | - | False | 禁用提示词自动增强 |
| `--no-ai-scoring` | - | False | 禁用 AI 图片评分 |
| `--output-dir` | `-o` | ./output | 本地保存目录 |
| `--format` | `-f` | text | 输出格式（text/json/quiet） |

### edit 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `prompt` | - | 必填 | 编辑指令 |
| `--source` | `-s` | 必填 | 源图片路径或 URL（可多次使用） |
| `--count` | `-n` | 1 | 生成数量 |
| `--aspect-ratio` | `-a` | 16:9 | 宽高比 |
| `--size` | - | 2K | 图片尺寸 |
| `--provider` | `-p` | .env 默认 | Provider（gemini/jiekouai，不支持 antigravity） |
| `--enhance` | - | False | 启用编辑指令 LLM 增强 |
| `--output-dir` | `-o` | ./output | 本地保存目录 |
| `--format` | `-f` | text | 输出格式（text/json/quiet） |
