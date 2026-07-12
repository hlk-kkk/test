# 全模态视频分段分析设计

> 日期: 2026-03-31
> 状态: 已确认
> 前置: omni-audio-input（已完成）

## 背景

Qwen3.5-Omni 支持视频输入，但限制 ≤400s、720P。社交媒体视频可能长达 20 分钟以上。本设计在 omni-audio-input 基础上，新增视频直传（短视频）和分段分析（长视频）能力。

## 核心决策

| 决策项 | 结论 |
|--------|------|
| 短视频阈值 | ≤ 400s 直传，> 400s 分段 |
| 视频转码 | 必须转码 720p（ffmpeg） |
| SubAgent 模式 | Tool 模式（非 SubGraph） |
| 视频传递方式 | 上传七牛云，传 URL |
| 分段策略 | LLM 语义分段（非硬切），每段 ≤ 360s（留余量） |
| 上下文传递 | 滚动 summary（每段子 Agent 自动生成压缩摘要） |

## 架构概览

```
视频下载 → 获取时长
    │
    ├─ [非 omni 模型] → 现有 ASR 流程（完全不变）
    │
    ├─ [omni + 短视频 ≤ 400s]
    │   → ffmpeg 转码 720p
    │   → 上传七牛
    │   → 中间件注入视频 URL → 模型直接看+听
    │
    └─ [omni + 长视频 > 400s]
         1. 提取音频 → ASR → 全文
         2. LLM 语义分段 → [{start, end, title}, ...]
         3. ffmpeg 按时间点切割 + 转码 720p
         4. 各段上传七牛
         5. 主 Agent 逐段派发 analyze_video_segment
            - SubAgent 1: seg1_url → analysis_1 + summary_1
            - SubAgent 2: seg2_url + summary_1 → analysis_2 + summary_2
            - SubAgent N: segN_url + summary_{N-1} → analysis_N
         6. 主 Agent 汇总 → 完整分析报告
```

## 详细设计

### 1. 配置项 (settings.py)

```python
# 全模态视频配置
OMNI_VIDEO_MAX_DURATION: int = 400       # 直传最大时长（秒）
OMNI_VIDEO_RESOLUTION: str = "720p"      # 转码目标分辨率
OMNI_VIDEO_SEGMENT_MAX: int = 360        # 分段最大时长（秒），留 40s 余量
```

### 2. VideoTranscoder 服务

**文件:** `backend/services/video_transcoder.py`

职责：视频转码和切割，所有 ffmpeg 操作通过 `asyncio.to_thread` 包装。

```python
class VideoTranscoder:
    """视频转码和切割服务"""

    async def get_duration(self, video_path: str) -> float:
        """获取视频时长（秒），使用 ffprobe"""

    async def transcode_720p(self, input_path: str, output_path: str) -> str:
        """
        转码为 720p
        ffmpeg -i input -vf "scale=-2:720" -c:a copy output
        返回输出文件路径
        """

    async def split_and_transcode(
        self, input_path: str, segments: list[dict], output_dir: str
    ) -> list[str]:
        """
        按时间点切割并转码 720p
        对每个 segment: ffmpeg -i input -ss start -to end -vf "scale=-2:720" seg_N.mp4
        返回各段文件路径列表
        """
```

### 3. process_video Tool 扩展

在现有 omni 音频分支之前，增加视频时长判断：

```
process_video 被调用 → content_info 中有 video_url
    │
    ├─ [非 omni] → 现有 ASR 流程（不变）
    │
    └─ [omni] → 下载视频 → 获取时长
         │
         ├─ [≤ 400s] → 转码 720p → 上传七牛
         │   返回 Command:
         │     omni_video_url = "https://cdn.../video.mp4"
         │     ToolMessage: "短视频(3:20)已上传，模型将直接分析视频"
         │
         └─ [> 400s] → 提取音频 → ASR → 返回 transcript
             保留视频文件 (persist_video=True)
             返回 Command:
               transcript = {text, segments}
               ToolMessage: "长视频(12:30)已转录(8234字)，需分段分析"
```

**关键：** 短视频路径设置 `omni_video_url`（视频 URL），与之前的 `omni_audio_url`（音频 URL）区分。

### 4. prepare_video_segments Tool（新）

**文件:** `backend/agent/tools/video_segment_tool.py`

```python
@tool
async def prepare_video_segments(runtime: ToolRuntime[RemixContext]) -> Command:
    """根据转录文本对长视频进行语义分段，切割并上传。

    前置条件: 需要先调用 process_video 获取转录文本。
    仅在长视频(>400s)且使用全模态模型时调用。

    处理流程:
    1. 使用 LLM 对转录文本进行语义分段
    2. ffmpeg 按分段时间点切割视频并转码 720p
    3. 上传各段到七牛云
    4. 返回分段列表
    """
```

**语义分段 Prompt:**

```
你是一个视频内容分段专家。根据以下视频转录文本，将内容划分为语义完整的段落。

要求：
- 每段时长不超过 {max_duration} 秒
- 在语义自然断点处分段（话题切换、章节过渡等）
- 避免在句子中间切断
- 每段给出简短标题

转录文本（带时间戳）：
{transcript_with_timestamps}

输出 JSON 格式：
[
  {"start": 0.0, "end": 185.5, "title": "开场介绍与背景铺垫"},
  {"start": 185.5, "end": 348.0, "title": "核心观点：三个效率法则"},
  ...
]
```

**返回:**

```python
Command(update={
    "messages": [ToolMessage(content="视频已分为 4 段：\n1. 开场介绍(0:00-3:05)\n2. ...", ...)],
    "video_segments": [
        {"index": 1, "start": 0, "end": 185.5, "title": "开场介绍", "video_url": "https://..."},
        {"index": 2, "start": 185.5, "end": 348, "title": "核心观点", "video_url": "https://..."},
        ...
    ],
})
```

### 5. analyze_video_segment Tool（新）

**文件:** `backend/agent/tools/video_segment_tool.py`（同文件）

```python
@tool
async def analyze_video_segment(
    segment_index: int,
    segment_video_url: str,
    segment_title: str,
    previous_summary: str = "",
    runtime: ToolRuntime[RemixContext] = None,
) -> Command:
    """分析单个视频段落。

    创建一个子 Agent 实例，传入视频段 URL 和前序上下文，
    返回分析结果和压缩摘要。

    Args:
        segment_index: 段序号
        segment_video_url: 该段视频的七牛云 URL
        segment_title: 段落标题
        previous_summary: 前序段落的压缩摘要（首段为空）
    """
```

**子 Agent 实现:**

```python
# 创建独立的子 Agent（不共享主 Agent 的 checkpointer）
sub_model = get_llm(config_name=runtime.context.config_name)

# 构建消息：视频内容 + 上下文
messages = [
    SystemMessage(content=SEGMENT_ANALYSIS_PROMPT),
    HumanMessage(content=[
        {"type": "text", "text": f"分析第 {segment_index} 段: {segment_title}\n\n前序上下文:\n{previous_summary}"},
        {"type": "video_url", "video_url": {"url": segment_video_url}},
    ]),
]

# 直接调用模型（不需要工具，纯分析）
response = await sub_model.ainvoke(messages)

# 从响应中提取 analysis 和 summary
# summary 由 prompt 指示模型在末尾生成
```

**子 Agent System Prompt:**

```
你是一个视频内容分析师，正在分析一个长视频的第 {index} 段。

请结合视频画面和声音进行分析，输出：

## 本段分析
- 内容要点
- 关键技巧/话术
- 视觉元素分析（画面、字幕、转场）
- 情感/节奏变化

## 本段摘要（供后续段落参考）
用 2-3 句话概括本段核心内容和关键信息，作为后续段落的上下文。
用 "---SUMMARY---" 标记分隔摘要部分。
```

**返回:**

```python
# 解析 response，按 "---SUMMARY---" 分割出 analysis 和 summary
Command(update={
    "messages": [ToolMessage(
        content=f"第 {segment_index} 段分析完成:\n{analysis}\n\n摘要: {summary}",
        tool_call_id=runtime.tool_call_id
    )],
})
```

### 6. State 变更

`RemixAgentState` 新增：

```python
omni_video_url: Annotated[Optional[str], last_value]    # 短视频直传 URL
video_segments: Annotated[Optional[list], last_value]    # 长视频分段列表
```

`VideoSegmentDict` (TypedDict):

```python
class VideoSegmentDict(TypedDict, total=False):
    index: int
    start: float
    end: float
    title: str
    video_url: str
```

### 7. 中间件扩展

`omni_audio_injection_middleware` 扩展为 `omni_media_injection_middleware`：

检查顺序：
1. `state.omni_video_url` 非空 → 注入 `{"type": "video_url", "video_url": {"url": "..."}}`
2. `state.omni_audio_url` 非空 → 注入 `{"type": "input_audio", ...}`（已有逻辑）
3. 视频优先（有视频 URL 时不注入音频）

注入后清除对应 URL（与音频相同，防止后续轮次重复注入）。

### 8. System Prompt

在 `remix_dynamic_prompt` 中，omni 模型追加视频理解指南：

```python
OMNI_VIDEO_GUIDE = """## 视频直接理解模式

你当前使用的是全模态模型，可以直接"看到"和"听到"视频内容。

**短视频：**
- process_video 会直接提供完整视频
- 请结合画面、声音、字幕等多维度分析

**长视频分段分析：**
- 长视频会被语义分段后逐段提供
- 你需要按以下流程操作：
  1. 调用 prepare_video_segments 对视频分段
  2. 依次调用 analyze_video_segment 分析每段
  3. 每段分析时传入前序段落的 summary 作为上下文
  4. 所有段落分析完成后，汇总输出完整分析报告

**分析维度：**
- 视觉元素：画面构图、字幕设计、转场效果
- 音频元素：语气、节奏、背景音乐
- 内容结构：叙事方式、信息密度、节奏把控
"""
```

替换之前的 `OMNI_AUDIO_GUIDE`。

### 9. Agent 工厂变更

`create_remix_agent` 的工具列表新增：

```python
tools = [
    parse_link,
    fetch_content,
    process_video,
    prepare_video_segments,      # 新增
    analyze_video_segment,       # 新增
]
```

### 10. QiniuStorageService 扩展

新增视频上传方法（复用已有逻辑，调整 MIME type）：

```python
async def upload_video(self, local_path: str, key: str) -> str:
    """上传视频到七牛，返回公网 URL"""
    # mime_type="video/mp4"
```

key 命名：`video/{session_id}/{content_id}/full.mp4`（短视频）或 `video/{session_id}/{content_id}/seg_{N}.mp4`（分段）

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `config/settings.py` | 修改 | 新增 OMNI_VIDEO_* 配置 |
| `agent/state.py` | 修改 | 新增 omni_video_url, video_segments |
| `services/video_transcoder.py` | **新建** | 转码 720p + 切割 |
| `services/qiniu_storage.py` | 修改 | 新增 upload_video 方法 |
| `agent/tools/video_processor_tool.py` | 修改 | omni 短/长视频分支 |
| `agent/tools/video_segment_tool.py` | **新建** | prepare_video_segments + analyze_video_segment |
| `agent/tools/__init__.py` | 修改 | 导出新工具 |
| `agent/middleware/omni_audio_injection.py` | 修改 | 扩展为 omni_media_injection，支持视频 |
| `agent/middleware/__init__.py` | 修改 | 更新导出名 |
| `agent/prompts.py` | 修改 | OMNI_VIDEO_GUIDE 替换 OMNI_AUDIO_GUIDE |
| `agent/remix_agent.py` | 修改 | 注册新工具 |

## 不变的部分

- 非 omni 模型的完整 ASR 流程
- 现有的 parse_link、fetch_content 工具
- 图片多模态注入中间件
- 前端（无 UI 变更，分段分析对前端透明）
- 数据库（不需要新表/新字段，support_omni 已有）
