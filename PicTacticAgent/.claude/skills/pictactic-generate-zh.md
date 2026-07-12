---
name: pictactic
description: |
  AI 图片/视频生成与编辑工具（PicTacticAgent）。支持文生图、参考图风格迁移、图片编辑，以及视频生成。
  触发场景：
  (1) 根据提示词生成图片 - "生成一张..."、"帮我画..."、"创建图片..."
  (2) 基于参考图（文生图时的风格参考）生成风格一致的图片 - "根据这张图生成..."、"参照这张图片的风格..."
  (3) 批量生成多张候选图 - "生成几张...让我选"
  (4) 对已有图片进行编辑修改 - "修改这张图片..."、"把这张图片改成..."、"编辑图片..."
  (5) 生成视频 - "生成一段视频..."、"帮我做个视频..."、"创建视频..."
  (6) 使用 /pictactic 命令
allowed-tools: Bash(uv:*)
---

# PicTactic 图片生成 Skill

## 项目路径配置

**注意**：所有命令需在 PicTacticAgent 项目目录下通过 `uv run pictactic` 执行。

```bash
# 将 PROJECT_DIR 替换为实际项目路径
PROJECT_DIR=/path/to/PicTacticAgent
```

## 异步工作流（推荐 Agent 使用）

异步模式：提交任务 → 立即返回 task_id → 轮询状态 → 获取结果。避免长时间阻塞导致超时。

### 第一步：提交任务

```bash
# 提交生成任务
cd $PROJECT_DIR && uv run pictactic submit generate --format quiet "一只可爱的猫咪在樱花树下"
# → 输出 task_id（单行，立即退出）
# 若后端未运行会自动启动

# 完整参数
cd $PROJECT_DIR && uv run pictactic submit generate \
  [--rounds 3] [--images 5] [--top-k 3] [--threshold 0.7] \
  [--aspect-ratio 16:9] [--size 2K] [--provider jiekouai] \
  [--no-enhance] [--no-ai-scoring] \
  [--template /path/to/ref.jpg] \
  --format quiet \
  "<提示词>"

# 提交编辑任务
cd $PROJECT_DIR && uv run pictactic submit edit \
  --source /path/to/image.png \
  [--count 1] [--aspect-ratio 16:9] [--size 2K] \
  [--provider gemini] [--enhance] \
  --format quiet \
  "<编辑指令>"
```

### 第二步：轮询状态（重复直到 status=completed/failed）

```bash
cd $PROJECT_DIR && uv run pictactic status <task_id> --format json
# → {"task_id":"...","status":"generating","current_round":2,"images_generated":8,...}
```

```bash
# 紧凑格式用于脚本
cd $PROJECT_DIR && uv run pictactic status <task_id> --format quiet
# → generating 2/3 8 0.85
# 格式: status round/max images_count best_score
```

### 第三步：获取结果（status=completed 时）

```bash
cd $PROJECT_DIR && uv run pictactic result <task_id> --output-dir ./output --format quiet
# → 输出图片本地路径，每行一个
```

### 异步快速示例（完整流程）

```bash
# 1. 提交
TASK_ID=$(cd $PROJECT_DIR && uv run pictactic submit generate --format quiet "一条威武的龙")

# 2. 轮询（定期检查）
cd $PROJECT_DIR && uv run pictactic status $TASK_ID --format json

# 3. 完成后下载
cd $PROJECT_DIR && uv run pictactic result $TASK_ID --output-dir ./output --format quiet
```

## 直接模式（阻塞到完成，适合交互使用）

### 文生图（generate 命令）

> **重要**：`prompt` 必须放在所有 `--options` 之后，否则 typer 解析失败。

```bash
cd $PROJECT_DIR && uv run pictactic generate \
  [--rounds 3] \          # 最大迭代轮数（1-10，默认 3）
  [--images 5] \          # 每轮候选图数量（1-10，默认 5）
  [--top-k 3] \           # 最终输出最优图数量（默认 3）
  [--threshold 0.7] \     # 质量阈值，达到后自动停止（0-1，默认 0.7）
  [--aspect-ratio 16:9] \ # 宽高比：1:1 / 16:9 / 9:16 / 4:3 / 3:4
  [--size 2K] \           # 图片尺寸：1K / 2K / 4K
  [--provider jiekouai] \ # 指定 Provider（可选）
  [--no-enhance] \        # 禁用提示词 LLM 自动增强
  [--no-ai-scoring] \     # 禁用 AI 图片评分
  [--output-dir ./output] \ # 图片保存目录
  [--format quiet] \      # 输出格式：text（默认）/ json / quiet（仅路径）
  "<提示词>"              # ← prompt 必须放最后
```

**最快速调用**：
```bash
cd $PROJECT_DIR && uv run pictactic generate --format quiet "一只可爱的猫咪在樱花树下"
```

### 参考图生成

```bash
cd $PROJECT_DIR && uv run pictactic generate \
  --template /path/to/reference_image.jpg \
  --rounds 3 --images 4 --format quiet \
  "<提示词>"
```

### 图片编辑（edit 命令）

```bash
cd $PROJECT_DIR && uv run pictactic edit \
  --source /path/to/image.png \
  [--count 1] [--aspect-ratio 16:9] [--size 2K] \
  [--provider gemini] [--enhance] \
  [--output-dir ./output] [--format quiet] \
  "<编辑指令>"
```

## 输出格式

| 格式 | 说明 |
|------|------|
| `--format quiet` | 仅输出最终图片路径，每行一个（推荐，适合脚本集成） |
| `--format json` | JSONL 事件流，包含进度、评分等结构化数据 |
| `--format text` | Rich 格式化 UI，含进度、评分详情（默认） |

## 退出码

| 代码 | 含义 |
|------|------|
| 0 | 成功 |
| 1 | 参数错误（文件不存在、Provider 不支持等） |
| 2 | 生成/编辑失败（Provider 报错、API 错误等） |
| 3 | 未产生有效结果 / 任务未完成 |

## 参数速查

### submit generate / generate 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `prompt` | - | 必填 | 图片描述提示词 |
| `--template` | `-t` | 无 | 参考图路径（可多次使用，最多 3 张） |
| `--rounds` | `-r` | 3 | 最大迭代轮数 |
| `--images` | `-n` | 5 | 每轮候选图数量 |
| `--top-k` | `-k` | 3 | 最终输出最优图数量 |
| `--threshold` | `-q` | 0.7 | 质量阈值 |
| `--aspect-ratio` | `-a` | 16:9 | 宽高比 |
| `--size` | `-s` | 2K | 图片尺寸 |
| `--provider` | `-p` | .env 默认 | 指定 Provider |
| `--no-enhance` | - | False | 禁用提示词自动增强 |
| `--no-ai-scoring` | - | False | 禁用 AI 图片评分 |
| `--output-dir` | `-o` | ./output | 本地保存目录 |
| `--format` | `-f` | text/quiet | 输出格式 |

### submit edit / edit 命令

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `prompt` | - | 必填 | 编辑指令 |
| `--source` | `-s` | 必填 | 源图片路径或 URL |
| `--count` | `-n` | 1 | 生成数量 |
| `--aspect-ratio` | `-a` | 16:9 | 宽高比 |
| `--size` | - | 2K | 图片尺寸 |
| `--provider` | `-p` | .env 默认 | Provider（gemini/jiekouai） |
| `--enhance` | - | False | 启用编辑指令 LLM 增强 |
| `--output-dir` | `-o` | ./output | 本地保存目录 |
| `--format` | `-f` | text/quiet | 输出格式 |

## 视频生成

### 直接模式（阻塞到完成）

```bash
cd $PROJECT_DIR && uv run pictactic video \
  [--start-image /path/to/start.jpg] \  # 首帧图片（启用图生视频模式）
  [--end-image /path/to/end.jpg] \      # 尾帧图片（Kling、Veo 支持）
  [--rounds 2] \                        # 生成轮数（1-5，默认 2）
  [--videos 2] \                        # 每轮视频数（1-4，默认 2）
  [--duration 5] \                      # 时长（秒，默认 5）
  [--resolution 720p] \                 # 分辨率：720p、1080p
  [--aspect-ratio 16:9] \              # 宽高比
  [--audio] \                           # 生成音频（Kling、Veo 支持）
  [--provider kling] \                  # Provider：kling、sora、veo
  [--no-enhance] \                      # 禁用提示词增强
  [--output-dir ./output] \
  [--format quiet] \
  "<提示词>"                            # ← prompt 必须放最后
```

**最快速调用**：
```bash
cd $PROJECT_DIR && uv run pictactic video --format quiet "一只猫在弹钢琴"
```

### 异步模式（提交到后端）

```bash
# 提交视频任务
TASK_ID=$(cd $PROJECT_DIR && uv run pictactic submit video --format quiet "海边日落的波浪")

# 查看状态（注意 --video 标志）
cd $PROJECT_DIR && uv run pictactic status --video $TASK_ID --format json
# 或紧凑格式: status round/max videos_count [progress%]
cd $PROJECT_DIR && uv run pictactic status --video $TASK_ID --format quiet
# → generating 1/2 2 45%

# 完成后下载结果
cd $PROJECT_DIR && uv run pictactic result --video $TASK_ID --output-dir ./output --format quiet
```

### 视频参数速查

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `prompt` | - | 必填 | 视频描述提示词 |
| `--start-image` | `-i` | 无 | 首帧图片（启用图生视频） |
| `--end-image` | `-e` | 无 | 尾帧图片（Kling、Veo 支持） |
| `--rounds` | `-r` | 2 | 生成轮数 |
| `--videos` | `-n` | 2 | 每轮视频数 |
| `--duration` | `-d` | 5 | 时长（秒） |
| `--resolution` | - | 无 | 720p 或 1080p |
| `--aspect-ratio` | `-a` | 16:9 | 宽高比 |
| `--audio` | - | False | 生成音频 |
| `--provider` | `-p` | .env 默认 | kling、sora、veo |
| `--no-enhance` | - | False | 禁用提示词增强 |
| `--output-dir` | `-o` | ./output | 输出目录 |
| `--format` | `-f` | text | text/json/quiet |

### 视频 Provider 能力对比

| Provider | 模式 | 尾帧 | 音频 | 时长 |
|----------|------|------|------|------|
| Kling | 文生视频 + 图生视频 | 支持 | 支持 | 3-15 秒（范围） |
| Sora | 文生视频 + 图生视频 | 不支持 | 不支持 | 4/8/12 秒 |
| Veo | 文生视频 + 图生视频 | 支持 | 支持 | 4/6/8 秒 |

## 其他命令

```bash
# 查看可用 Provider 及支持的模式
cd $PROJECT_DIR && uv run pictactic providers

# 查看图片任务状态
cd $PROJECT_DIR && uv run pictactic status <task_id> --format json

# 查看视频任务状态
cd $PROJECT_DIR && uv run pictactic status --video <task_id> --format json
```

## 参数选择建议

| 场景 | 推荐参数 |
|------|---------|
| 快速预览 | `--rounds 1 --images 3 --no-ai-scoring` |
| 标准生成 | `--rounds 3 --images 5`（默认） |
| 高质量 | `--rounds 5 --images 8 --threshold 0.85` |
| 参考图风格迁移 | `--template ref.jpg --rounds 3 --images 4` |
| 手机壁纸 | `--aspect-ratio 9:16 --size 2K` |
| 横幅封面 | `--aspect-ratio 16:9 --size 2K` |
| 图片编辑（快速）| `edit --source img.png --format quiet "指令"` |
| 快速视频 | `video --format quiet "提示词"` |
| 图生视频（首帧）| `video --start-image img.jpg --provider kling "提示词"` |
| 带音频视频 | `video --audio --provider veo "提示词"` |

## 错误处理

| 错误 | 原因 | 解决方案 |
|------|------|---------|
| API key 无效 | .env 配置错误 | 检查 `$PROJECT_DIR/.env` 中的 `JIEKOUAI_API_KEY` |
| Provider 不支持 image_to_image | edit 命令使用了 antigravity | 改用 jiekouai 或 gemini |
| Source image not found | 源图路径错误 | 检查 `--source` 路径是否存在 |
| 输出目录无写权限 | 目录权限问题 | 指定 `--output-dir` 到有权限的目录 |
| uv 命令找不到 | uv 未安装 | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| 认证失败 (HTTP 401) | API Key 不匹配 | 重启后端以重新生成 INTERNAL_API_KEY |
