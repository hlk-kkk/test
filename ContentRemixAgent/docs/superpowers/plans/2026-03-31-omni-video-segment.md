# Omni Video Segment Analysis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable omni-modal models to directly process video input — short videos in one shot, long videos via semantic segmentation with SubAgent chain analysis.

**Architecture:** Extend `process_video` with duration-based branching (≤400s direct upload, >400s ASR→segment→SubAgent chain). New `VideoTranscoder` service handles ffmpeg 720p transcoding and splitting. Two new tools (`prepare_video_segments`, `analyze_video_segment`) orchestrate long video analysis. Existing middleware is renamed to handle both audio and video URL injection.

**Tech Stack:** Python 3.10+ / ffmpeg / qiniu SDK / LangChain 1.0 / LangGraph

---

### Task 1: Config — Video Settings

**Files:**
- Modify: `backend/config/settings.py`

- [ ] **Step 1: Add OMNI_VIDEO config items**

In `backend/config/settings.py`, add after the Qiniu config section (after `QINIU_DELETE_AFTER_DAYS`):

```python
    # ========== 全模态视频配置 ==========
    OMNI_VIDEO_MAX_DURATION: int = 400       # 直传最大时长（秒），超出则分段
    OMNI_VIDEO_SEGMENT_MAX: int = 360        # 分段最大时长（秒），留余量
```

- [ ] **Step 2: Commit**

```bash
git add backend/config/settings.py
git commit -m "feat: add OMNI_VIDEO config items"
```

---

### Task 2: State — omni_video_url + video_segments

**Files:**
- Modify: `backend/agent/state.py`

- [ ] **Step 1: Add VideoSegmentDict TypedDict**

In `backend/agent/state.py`, add after the `TranscriptDict` class (after line 94):

```python
class VideoSegmentDict(TypedDict, total=False):
    """视频分段信息"""
    index: int           # 段序号 (从1开始)
    start: float         # 开始时间（秒）
    end: float           # 结束时间（秒）
    title: str           # 段落标题（LLM 生成）
    video_url: str       # 七牛云 URL
```

- [ ] **Step 2: Add state fields to RemixAgentState**

Add after `omni_audio_url` field:

```python
    # 全模态视频 URL（短视频直传，≤400s）
    omni_video_url: Annotated[Optional[str], last_value]

    # 视频分段列表（长视频分段后的段落信息）
    video_segments: Annotated[Optional[List[VideoSegmentDict]], last_value]
```

Add `List` to the typing imports if not already there.

- [ ] **Step 3: Commit**

```bash
git add backend/agent/state.py
git commit -m "feat: add omni_video_url and video_segments state"
```

---

### Task 3: VideoTranscoder Service

**Files:**
- Create: `backend/services/video_transcoder.py`

- [ ] **Step 1: Create VideoTranscoder**

```python
# -*- coding: utf-8 -*-
"""
视频转码和切割服务

提供 720p 转码和按时间点切割视频的能力。
所有 ffmpeg 操作通过 asyncio.to_thread 包装，避免阻塞事件循环。
"""
import asyncio
import subprocess
from pathlib import Path
from typing import List, Dict

from utils.logger import logger


class VideoTranscodeError(Exception):
    """视频转码错误"""
    pass


class VideoTranscoder:
    """视频转码和切割服务"""

    async def get_duration(self, video_path: str) -> float:
        """
        获取视频时长（秒）

        Args:
            video_path: 视频文件路径

        Returns:
            时长（秒）

        Raises:
            VideoTranscodeError: 无法获取时长
        """
        def _probe():
            cmd = [
                "ffprobe", "-v", "error",
                "-show_entries", "format=duration",
                "-of", "default=noprint_wrappers=1:nokey=1",
                video_path
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
            if result.returncode != 0:
                raise VideoTranscodeError(f"ffprobe failed: {result.stderr}")
            return float(result.stdout.strip())

        return await asyncio.to_thread(_probe)

    async def transcode_720p(self, input_path: str, output_path: str) -> str:
        """
        转码视频为 720p

        Args:
            input_path: 输入视频路径
            output_path: 输出视频路径

        Returns:
            输出文件路径

        Raises:
            VideoTranscodeError: 转码失败
        """
        def _transcode():
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            cmd = [
                "ffmpeg", "-y", "-i", input_path,
                "-vf", "scale=-2:720",
                "-c:v", "libx264", "-preset", "fast", "-crf", "23",
                "-c:a", "aac", "-b:a", "128k",
                output_path
            ]
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
            if result.returncode != 0:
                raise VideoTranscodeError(f"ffmpeg transcode failed: {result.stderr[:500]}")
            logger.info(f"[VideoTranscoder] Transcoded to 720p: {output_path}")
            return output_path

        return await asyncio.to_thread(_transcode)

    async def split_and_transcode(
        self,
        input_path: str,
        segments: List[Dict],
        output_dir: str,
    ) -> List[str]:
        """
        按时间点切割视频并转码 720p

        Args:
            input_path: 输入视频路径
            segments: 分段列表 [{"start": 0.0, "end": 185.5, "title": "..."}, ...]
            output_dir: 输出目录

        Returns:
            各段文件路径列表

        Raises:
            VideoTranscodeError: 切割/转码失败
        """
        Path(output_dir).mkdir(parents=True, exist_ok=True)
        output_paths = []

        for i, seg in enumerate(segments):
            output_path = str(Path(output_dir) / f"seg_{i + 1:03d}.mp4")

            def _split(inp=input_path, out=output_path, start=seg["start"], end=seg["end"]):
                cmd = [
                    "ffmpeg", "-y",
                    "-i", inp,
                    "-ss", str(start),
                    "-to", str(end),
                    "-vf", "scale=-2:720",
                    "-c:v", "libx264", "-preset", "fast", "-crf", "23",
                    "-c:a", "aac", "-b:a", "128k",
                    out
                ]
                result = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
                if result.returncode != 0:
                    raise VideoTranscodeError(
                        f"ffmpeg split failed for segment {i + 1} ({start}-{end}): {result.stderr[:500]}"
                    )
                return out

            path = await asyncio.to_thread(_split)
            output_paths.append(path)
            logger.info(f"[VideoTranscoder] Split segment {i + 1}/{len(segments)}: {seg['start']:.1f}s-{seg['end']:.1f}s -> {path}")

        return output_paths
```

- [ ] **Step 2: Commit**

```bash
git add backend/services/video_transcoder.py
git commit -m "feat: VideoTranscoder service for 720p transcoding and splitting"
```

---

### Task 4: QiniuStorageService — upload_video

**Files:**
- Modify: `backend/services/qiniu_storage.py`

- [ ] **Step 1: Add upload_video method**

Add after the `upload_audio` method:

```python
    async def upload_video(self, local_path: str, key: str) -> str:
        """
        上传视频文件到七牛云

        Args:
            local_path: 本地视频文件路径
            key: 七牛存储 key，如 video/{session_id}/{content_id}/seg_001.mp4

        Returns:
            公网访问 URL

        Raises:
            QiniuStorageError: 上传失败或配置不完整
        """
        if not self.is_available():
            raise QiniuStorageError(
                "七牛云配置不完整，请在 .env 中配置 QINIU_ACCESS_KEY, QINIU_SECRET_KEY, QINIU_BUCKET, QINIU_DOMAIN"
            )

        try:
            url = await asyncio.to_thread(self._sync_upload_video, local_path, key)
            logger.info(f"[Qiniu] Video uploaded: {key} -> {url}")
            return url
        except QiniuStorageError:
            raise
        except Exception as e:
            raise QiniuStorageError(f"视频上传失败: {e}") from e

    def _sync_upload_video(self, local_path: str, key: str) -> str:
        """同步上传视频（在线程池中执行）"""
        import qiniu

        auth = qiniu.Auth(settings.QINIU_ACCESS_KEY, settings.QINIU_SECRET_KEY)

        policy = {
            "deleteAfterDays": settings.QINIU_DELETE_AFTER_DAYS,
        }

        token = auth.upload_token(
            settings.QINIU_BUCKET, key, 3600, policy=policy
        )

        ret, info = qiniu.put_file(token, key, local_path, mime_type="video/mp4")

        if ret is None:
            raise QiniuStorageError(f"七牛视频上传失败: {info}")

        domain = settings.QINIU_DOMAIN.rstrip("/")
        if not domain.startswith(("http://", "https://")):
            domain = f"https://{domain}"

        return f"{domain}/{key}"
```

- [ ] **Step 2: Update module docstring**

Change the module docstring from:
```python
"""
七牛云存储服务 - 上传音频文件并获取公网 URL

用于全模态模型（omni）的音频直传场景。
音频上传后自动设置过期删除（默认 1 天）。
"""
```

To:
```python
"""
七牛云存储服务 - 上传音频/视频文件并获取公网 URL

用于全模态模型（omni）的媒体直传场景。
文件上传后自动设置过期删除（默认 1 天）。
"""
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/qiniu_storage.py
git commit -m "feat: add upload_video to QiniuStorageService"
```

---

### Task 5: process_video — Short Video Branch

**Files:**
- Modify: `backend/agent/tools/video_processor_tool.py`

- [ ] **Step 1: Replace omni audio-only branch with duration-based branching**

In `_process_video_internal`, the current omni branch (lines 132-191) only handles audio upload. Replace the entire omni branch block with a new version that checks video duration:

Find the block starting with `# === Omni 分支：全模态模型直接理解音频，跳过 ASR ===` and ending before `# 获取音频时长`.

Replace it with:

```python
        # === Omni 分支：全模态模型直接处理视频/音频 ===
        if runtime is not None and content_info is not None:
            from llm_provider import is_omni_enabled
            config_name = getattr(runtime.context, 'config_name', None)

            if is_omni_enabled(config_name):
                from services.qiniu_storage import QiniuStorageService, QiniuStorageError
                from services.video_transcoder import VideoTranscoder
                from config import settings as app_settings

                qiniu_service = QiniuStorageService()
                if qiniu_service.is_available():
                    try:
                        # 获取视频时长
                        transcoder = VideoTranscoder()
                        video_duration = await transcoder.get_duration(video_path)
                        max_duration = app_settings.OMNI_VIDEO_MAX_DURATION

                        session_id = getattr(runtime.context, 'session_id', 'unknown')
                        content_id = content_info.get("content_id", "unknown")

                        if video_duration <= max_duration:
                            # --- 短视频：转码 720p → 上传 → 直接视频输入 ---
                            if step_callback:
                                await step_callback("start", "transcode_video", "转码视频为 720p...")

                            output_720p = str(Path(video_path).parent / "video_720p.mp4")
                            await transcoder.transcode_720p(video_path, output_720p)

                            if step_callback:
                                await step_callback("end", "transcode_video", "视频转码完成")

                            if step_callback:
                                await step_callback("start", "upload_video", "上传视频到云端...")

                            qiniu_key = f"video/{session_id}/{content_id}/full.mp4"
                            video_public_url = await qiniu_service.upload_video(output_720p, qiniu_key)

                            if step_callback:
                                await step_callback("end", "upload_video", "视频已上传到云端")

                            mins, secs = divmod(int(video_duration), 60)
                            duration_str = f"{mins}:{secs:02d}"

                            logger.info(f"[process_video] Omni short video: uploaded to {video_public_url}")

                            await adispatch_custom_event(
                                "content_info",
                                {"content_info": dict(content_info)},
                            )

                            return Command(update={
                                "messages": [ToolMessage(
                                    content=f"短视频({duration_str})已转码720p并上传，全模态模型将直接分析视频内容。",
                                    tool_call_id=runtime.tool_call_id
                                )],
                                "omni_video_url": video_public_url,
                                "current_stage": "视频已上传，等待模型分析",
                            })

                        else:
                            # --- 长视频：走 ASR 获取转录，后续由 prepare_video_segments 分段 ---
                            logger.info(f"[process_video] Omni long video ({video_duration:.0f}s > {max_duration}s), "
                                        f"falling through to ASR for transcript + segment planning")
                            # 不 return，继续走下面的 ASR 流程
                            # ASR 完成后，主 Agent 会调用 prepare_video_segments

                    except QiniuStorageError as e:
                        logger.warning(f"[process_video] Qiniu upload failed: {e}, falling back to ASR")
                    except Exception as e:
                        logger.warning(f"[process_video] Omni video processing failed: {e}, falling back to ASR")
                else:
                    logger.warning("[process_video] Qiniu not configured, falling back to ASR for omni model")
```

Note: `Path` import — add `from pathlib import Path` at the top of the file if not already there.

- [ ] **Step 2: Commit**

```bash
git add backend/agent/tools/video_processor_tool.py
git commit -m "feat: short video 720p transcode + upload in process_video omni branch"
```

---

### Task 6: prepare_video_segments + analyze_video_segment Tools

**Files:**
- Create: `backend/agent/tools/video_segment_tool.py`

- [ ] **Step 1: Create the tool file with both tools**

```python
# -*- coding: utf-8 -*-
"""
视频分段分析工具

提供两个工具：
1. prepare_video_segments: 对长视频进行语义分段、切割、上传
2. analyze_video_segment: 使用子 Agent 分析单个视频段落

用于全模态模型处理超过 400s 的长视频场景。
"""
import json
import shutil
import uuid
from pathlib import Path
from typing import Optional

from langchain_core.callbacks.manager import adispatch_custom_event
from langchain_core.messages import ToolMessage, HumanMessage, SystemMessage
from langchain_core.tools import tool
from langgraph.prebuilt import ToolRuntime
from langgraph.types import Command

from agent.errors import RemixToolException, RemixErrorCode
from agent.state import RemixContext
from config import settings
from utils.logger import logger


# ============================================================================
# 语义分段 Prompt
# ============================================================================

SEGMENT_PROMPT = """你是一个视频内容分段专家。根据以下视频转录文本，将内容划分为语义完整的段落。

要求：
- 每段时长不超过 {max_duration} 秒
- 在语义自然断点处分段（话题切换、章节过渡、论点转换等）
- 避免在句子中间切断
- 每段给出简短标题（10字以内）
- 段落之间不要有时间间隙或重叠

转录文本（带时间戳）：
{transcript}

严格输出以下 JSON 格式，不要包含其他内容：
```json
[
  {{"start": 0.0, "end": 185.5, "title": "开场介绍与背景"}},
  {{"start": 185.5, "end": 348.0, "title": "核心观点阐述"}},
  {{"start": 348.0, "end": 520.0, "title": "案例分析"}}
]
```"""


# ============================================================================
# 子 Agent 分析 Prompt
# ============================================================================

SEGMENT_ANALYSIS_PROMPT = """你是一个专业的视频内容分析师，正在分析一个长视频的某个段落。

请结合视频画面和声音进行分析，输出以下内容：

## 本段分析

### 内容要点
- 列出本段的核心信息和关键观点

### 表达技巧
- 分析使用的表达技巧和话术

### 视觉元素
- 画面构图、字幕设计、转场效果等

### 情感与节奏
- 语气变化、节奏把控、情绪表达

---SUMMARY---
用 2-3 句话概括本段核心内容和关键信息。这段摘要将作为后续段落分析的上下文。"""


# ============================================================================
# Tool: prepare_video_segments
# ============================================================================

@tool
async def prepare_video_segments(runtime: ToolRuntime[RemixContext]) -> Command:
    """对长视频进行语义分段、切割视频、上传到云端。

    前置条件: 需要先调用 process_video 获取转录文本(transcript)。
    仅在长视频(>400s)且使用全模态模型时调用。

    处理流程:
    1. 使用 LLM 对转录文本进行语义分段
    2. ffmpeg 按分段时间点切割视频并转码 720p
    3. 上传各段到七牛云
    4. 返回分段列表（含各段视频 URL）
    """
    # 验证前置条件
    transcript = runtime.state.get("transcript")
    if not transcript or not transcript.get("text"):
        return Command(update={
            "messages": [ToolMessage(
                content="转录文本尚未就绪，请先调用 process_video 获取转录文本。",
                tool_call_id=runtime.tool_call_id,
            )],
        })

    content_info = runtime.state.get("content_info")
    if not content_info:
        return Command(update={
            "messages": [ToolMessage(
                content="内容信息尚未就绪，请先调用 fetch_content。",
                tool_call_id=runtime.tool_call_id,
            )],
        })

    try:
        from services.video_transcoder import VideoTranscoder, VideoTranscodeError
        from services.qiniu_storage import QiniuStorageService, QiniuStorageError
        from llm_provider import get_llm
        from utils.json_utils import extract_json_from_text

        session_id = runtime.context.session_id
        config_name = getattr(runtime.context, 'config_name', None)
        content_id = content_info.get("content_id", "unknown")

        # ====== Step 1: LLM 语义分段 ======
        await adispatch_custom_event("sub_step_start", {
            "step_id": "semantic_segment",
            "label": "正在进行语义分段...",
            "parent_tool": "prepare_video_segments",
        })

        # 构建带时间戳的转录文本
        segments_text = transcript.get("segments", [])
        if segments_text:
            transcript_with_ts = "\n".join(
                f"[{int(s['start'] // 60)}:{int(s['start'] % 60):02d} - "
                f"{int(s['end'] // 60)}:{int(s['end'] % 60):02d}] {s['text']}"
                for s in segments_text
            )
        else:
            transcript_with_ts = transcript["text"]

        max_seg_duration = settings.OMNI_VIDEO_SEGMENT_MAX

        # 使用 LLM 分段（用同一个模型配置，但不启用 thinking 以加速）
        segment_llm = get_llm(temperature=0.1, config_name=config_name, enable_thinking=False)
        prompt = SEGMENT_PROMPT.format(
            max_duration=max_seg_duration,
            transcript=transcript_with_ts,
        )

        segment_response = await segment_llm.ainvoke([HumanMessage(content=prompt)])
        segment_plan = extract_json_from_text(segment_response.content)

        if not segment_plan or not isinstance(segment_plan, list):
            raise RemixToolException(
                RemixErrorCode.INTERNAL_ERROR,
                "LLM 语义分段结果解析失败",
                {"response": segment_response.content[:200]},
            )

        await adispatch_custom_event("sub_step_end", {
            "step_id": "semantic_segment",
            "message": f"已分为 {len(segment_plan)} 段",
        })

        logger.info(f"[prepare_video_segments] LLM segmentation: {len(segment_plan)} segments")

        # ====== Step 2: 查找视频文件 ======
        # 视频文件在 process_video 的 persist_video 模式下保留
        # 通过 content_info 中的 local_video_url 或 temp 目录查找
        from services.video_downloader import VideoDownloader
        downloader = VideoDownloader()

        # 查找视频文件：优先 local_video_url，其次遍历 temp 目录
        video_path = None
        local_video = content_info.get("local_video_url")
        if local_video and Path(local_video).exists():
            video_path = local_video
        else:
            # 在 temp 目录中查找最近的视频文件
            temp_dir = Path(settings.VIDEO_TEMP_DIR)
            if temp_dir.exists():
                for mp4 in sorted(temp_dir.rglob("*.mp4"), key=lambda p: p.stat().st_mtime, reverse=True):
                    video_path = str(mp4)
                    break

        if not video_path or not Path(video_path).exists():
            # 需要重新下载
            video_url = content_info.get("video_url")
            audio_url = content_info.get("audio_url")
            download_url = video_url or audio_url
            if not download_url:
                return Command(update={
                    "messages": [ToolMessage(
                        content="无法找到视频文件，请重新调用 process_video。",
                        tool_call_id=runtime.tool_call_id,
                    )],
                })
            task_id = str(uuid.uuid4())
            video_path = await downloader.download(download_url, task_id)

        # ====== Step 3: ffmpeg 切割 + 转码 720p ======
        await adispatch_custom_event("sub_step_start", {
            "step_id": "split_video",
            "label": f"正在切割视频为 {len(segment_plan)} 段并转码 720p...",
            "parent_tool": "prepare_video_segments",
        })

        transcoder = VideoTranscoder()
        output_dir = str(Path(settings.VIDEO_TEMP_DIR) / f"segments_{session_id}")
        segment_paths = await transcoder.split_and_transcode(
            video_path, segment_plan, output_dir
        )

        await adispatch_custom_event("sub_step_end", {
            "step_id": "split_video",
            "message": f"视频切割完成，共 {len(segment_paths)} 段",
        })

        # ====== Step 4: 上传各段到七牛 ======
        await adispatch_custom_event("sub_step_start", {
            "step_id": "upload_segments",
            "label": "正在上传视频分段...",
            "parent_tool": "prepare_video_segments",
        })

        qiniu_service = QiniuStorageService()
        video_segments = []

        for i, (seg_info, seg_path) in enumerate(zip(segment_plan, segment_paths)):
            qiniu_key = f"video/{session_id}/{content_id}/seg_{i + 1:03d}.mp4"
            seg_url = await qiniu_service.upload_video(seg_path, qiniu_key)

            video_segments.append({
                "index": i + 1,
                "start": seg_info["start"],
                "end": seg_info["end"],
                "title": seg_info.get("title", f"段落 {i + 1}"),
                "video_url": seg_url,
            })

        await adispatch_custom_event("sub_step_end", {
            "step_id": "upload_segments",
            "message": f"已上传 {len(video_segments)} 段视频",
        })

        # 清理临时分段文件
        try:
            shutil.rmtree(output_dir, ignore_errors=True)
        except Exception as e:
            logger.warning(f"Cleanup segment temp dir failed: {e}")

        # 构建返回消息
        segment_desc = "\n".join(
            f"{s['index']}. {s['title']} "
            f"({int(s['start'] // 60)}:{int(s['start'] % 60):02d}-"
            f"{int(s['end'] // 60)}:{int(s['end'] % 60):02d})"
            for s in video_segments
        )

        return Command(update={
            "messages": [ToolMessage(
                content=f"视频已分为 {len(video_segments)} 段：\n{segment_desc}\n\n"
                        f"请依次调用 analyze_video_segment 分析每段。",
                tool_call_id=runtime.tool_call_id,
            )],
            "video_segments": video_segments,
            "current_stage": f"视频已分为 {len(video_segments)} 段",
        })

    except RemixToolException:
        raise
    except Exception as e:
        logger.error(f"[prepare_video_segments] Failed: {e}", exc_info=True)
        return Command(update={
            "messages": [ToolMessage(
                content=f"视频分段失败: {str(e)}",
                tool_call_id=runtime.tool_call_id,
            )],
            "current_stage": "视频分段失败",
        })


# ============================================================================
# Tool: analyze_video_segment
# ============================================================================

@tool
async def analyze_video_segment(
    segment_index: int,
    segment_video_url: str,
    segment_title: str,
    previous_summary: str = "",
    runtime: ToolRuntime[RemixContext] = None,
) -> Command:
    """分析单个视频段落。创建子 Agent 调用全模态模型直接分析视频内容。

    Args:
        segment_index: 段序号 (从1开始)
        segment_video_url: 该段视频的云端 URL
        segment_title: 段落标题
        previous_summary: 前序段落的压缩摘要（首段为空字符串）

    Returns:
        Command 包含分析结果和压缩摘要
    """
    try:
        from llm_provider import get_llm

        config_name = getattr(runtime.context, 'config_name', None)

        await adispatch_custom_event("sub_step_start", {
            "step_id": f"analyze_seg_{segment_index}",
            "label": f"正在分析第 {segment_index} 段: {segment_title}...",
            "parent_tool": "analyze_video_segment",
        })

        # 创建子 Agent 模型（使用同一个 omni 模型配置）
        sub_model = get_llm(temperature=0.3, config_name=config_name, enable_thinking=False)

        # 构建消息
        context_text = ""
        if previous_summary:
            context_text = f"\n\n【前序段落摘要】\n{previous_summary}\n"

        user_content = [
            {"type": "text", "text": (
                f"请分析第 {segment_index} 段视频: 「{segment_title}」"
                f"{context_text}"
            )},
            {"type": "video_url", "video_url": {"url": segment_video_url}},
        ]

        messages = [
            SystemMessage(content=SEGMENT_ANALYSIS_PROMPT),
            HumanMessage(content=user_content),
        ]

        # 调用模型
        response = await sub_model.ainvoke(messages)
        full_response = response.content

        # 解析 analysis 和 summary（按 ---SUMMARY--- 分割）
        if "---SUMMARY---" in full_response:
            parts = full_response.split("---SUMMARY---", 1)
            analysis = parts[0].strip()
            summary = parts[1].strip()
        else:
            analysis = full_response
            summary = full_response[:200]  # fallback: 取前200字作为摘要

        await adispatch_custom_event("sub_step_end", {
            "step_id": f"analyze_seg_{segment_index}",
            "message": f"第 {segment_index} 段分析完成",
        })

        logger.info(
            f"[analyze_video_segment] Segment {segment_index} analyzed: "
            f"analysis={len(analysis)} chars, summary={len(summary)} chars"
        )

        return Command(update={
            "messages": [ToolMessage(
                content=(
                    f"## 第 {segment_index} 段分析: {segment_title}\n\n"
                    f"{analysis}\n\n"
                    f"**段落摘要（供后续段落参考）:** {summary}"
                ),
                tool_call_id=runtime.tool_call_id,
            )],
            "current_stage": f"第 {segment_index} 段分析完成",
        })

    except Exception as e:
        logger.error(f"[analyze_video_segment] Segment {segment_index} failed: {e}", exc_info=True)
        return Command(update={
            "messages": [ToolMessage(
                content=f"第 {segment_index} 段分析失败: {str(e)}",
                tool_call_id=runtime.tool_call_id,
            )],
            "current_stage": f"第 {segment_index} 段分析失败",
        })
```

- [ ] **Step 2: Commit**

```bash
git add backend/agent/tools/video_segment_tool.py
git commit -m "feat: prepare_video_segments and analyze_video_segment tools"
```

---

### Task 7: Tools __init__ + Agent Factory — Register New Tools

**Files:**
- Modify: `backend/agent/tools/__init__.py`
- Modify: `backend/agent/remix_agent.py`

- [ ] **Step 1: Export new tools in __init__.py**

Add to `backend/agent/tools/__init__.py`:

```python
from agent.tools.video_segment_tool import prepare_video_segments, analyze_video_segment
```

Add to `__all__`:
```python
    "prepare_video_segments",
    "analyze_video_segment",
```

- [ ] **Step 2: Register tools in remix_agent.py**

In `create_remix_agent()`, update the tools list:

```python
from agent.tools import (
    parse_link,
    fetch_content,
    process_video,
    prepare_video_segments,
    analyze_video_segment,
)
```

```python
tools = [
    parse_link,
    fetch_content,
    process_video,
    prepare_video_segments,
    analyze_video_segment,
]
```

- [ ] **Step 3: Commit**

```bash
git add backend/agent/tools/__init__.py backend/agent/remix_agent.py
git commit -m "feat: register video segment tools in agent"
```

---

### Task 8: Middleware — Rename + Video Support

**Files:**
- Rename: `backend/agent/middleware/omni_audio_injection.py` → `backend/agent/middleware/omni_media_injection.py`
- Modify: `backend/agent/middleware/__init__.py`
- Modify: `backend/agent/remix_agent.py`

- [ ] **Step 1: Rename file and rewrite middleware**

Rename `omni_audio_injection.py` to `omni_media_injection.py` and update to support both video and audio:

```python
# -*- coding: utf-8 -*-
"""
全模态媒体注入中间件

当全模态模式启用时，将视频或音频 URL 注入到最后一条 HumanMessage 中。
视频优先级高于音频（有视频则不注入音频）。
与 multimodal_injection_middleware（图片注入）对称设计。

基于 LangChain 1.0 的 @before_model 中间件模式实现。
"""
from typing import Optional, Dict, Any

from langchain.agents.middleware import before_model
from langchain_core.messages import HumanMessage

from utils.logger import logger


def _extract_text(content) -> str:
    """从 HumanMessage content 中提取纯文本"""
    if isinstance(content, str):
        return content
    elif isinstance(content, list):
        text_parts = [
            item.get("text", "") if isinstance(item, dict) and item.get("type") == "text"
            else str(item) if not isinstance(item, dict)
            else ""
            for item in content
        ]
        return "\n".join(filter(None, text_parts))
    return str(content)


def _has_media_type(content, media_type: str) -> bool:
    """检查 content 中是否已有指定类型的媒体"""
    if not isinstance(content, list):
        return False
    return any(
        isinstance(item, dict) and item.get("type") == media_type
        for item in content
    )


@before_model
def omni_media_injection_middleware(state: Dict[str, Any], runtime: Any) -> Optional[Dict[str, Any]]:
    """
    全模态媒体注入中间件

    触发条件:
    1. 全模态已启用
    2. state 中有 omni_video_url 或 omni_audio_url
    3. 存在 HumanMessage

    优先级: 视频 > 音频
    注入后清除对应 URL，防止后续轮次重复注入。
    """
    from llm_provider import is_omni_enabled

    config_name = getattr(runtime.context, 'config_name', None) if runtime.context else None
    if not is_omni_enabled(config_name):
        return None

    # 确定要注入的媒体（视频优先）
    video_url = state.get("omni_video_url")
    audio_url = state.get("omni_audio_url")

    if video_url:
        media_content = {"type": "video_url", "video_url": {"url": video_url}}
        media_type = "video_url"
        clear_fields = {"omni_video_url": None}
        label = "video"
    elif audio_url:
        media_content = {"type": "input_audio", "input_audio": {"url": audio_url}}
        media_type = "input_audio"
        clear_fields = {"omni_audio_url": None}
        label = "audio"
    else:
        return None

    # 获取消息列表
    messages = state.get("messages", [])
    if not messages:
        return None

    # 找到最后一条 HumanMessage
    last_human_idx = -1
    for i in range(len(messages) - 1, -1, -1):
        if isinstance(messages[i], HumanMessage):
            last_human_idx = i
            break

    if last_human_idx < 0:
        return None

    last_human_msg = messages[last_human_idx]

    # 检查是否已注入（避免重复）
    if _has_media_type(last_human_msg.content, media_type):
        logger.debug(f"[OmniMediaInjection] {label} already injected, skipping")
        return None

    # 注入媒体
    text = _extract_text(last_human_msg.content)
    new_content = [
        {"type": "text", "text": text},
        media_content,
    ]

    new_messages = list(messages)
    new_messages[last_human_idx] = HumanMessage(content=new_content)

    logger.info(f"[OmniMediaInjection] Injected {label} URL into message at index {last_human_idx}")

    return {"messages": new_messages, **clear_fields}


__all__ = [
    "omni_media_injection_middleware",
]
```

- [ ] **Step 2: Update middleware __init__.py**

Replace the old import:
```python
from agent.middleware.omni_audio_injection import omni_audio_injection_middleware
```
With:
```python
from agent.middleware.omni_media_injection import omni_media_injection_middleware
```

Update `__all__` to replace `"omni_audio_injection_middleware"` with `"omni_media_injection_middleware"`.

- [ ] **Step 3: Update remix_agent.py**

Replace import:
```python
    omni_audio_injection_middleware,
```
With:
```python
    omni_media_injection_middleware,
```

Replace in middleware list:
```python
            omni_audio_injection_middleware,  # @before_model: 全模态音频注入
```
With:
```python
            omni_media_injection_middleware,  # @before_model: 全模态媒体（视频/音频）注入
```

- [ ] **Step 4: Delete old file**

```bash
rm backend/agent/middleware/omni_audio_injection.py
```

- [ ] **Step 5: Commit**

```bash
git add backend/agent/middleware/ backend/agent/remix_agent.py
git commit -m "feat: rename omni_audio_injection to omni_media_injection, add video support"
```

---

### Task 9: System Prompt — Video Guide

**Files:**
- Modify: `backend/agent/prompts.py`

- [ ] **Step 1: Replace OMNI_AUDIO_GUIDE with OMNI_VIDEO_GUIDE**

Find and replace the `OMNI_AUDIO_GUIDE` constant with:

```python
OMNI_VIDEO_GUIDE = """## 全模态视频/音频理解模式

你当前使用的是全模态模型，可以直接"看到"和"听到"视频/音频内容。

**短视频（≤{max_duration}秒）：**
- process_video 会直接提供完整视频
- 请结合画面、声音、字幕等多维度分析

**长视频分段分析：**
- 长视频会被语义分段后逐段提供
- 你需要按以下流程操作：
  1. 调用 prepare_video_segments 对视频分段
  2. 依次调用 analyze_video_segment 分析每段
  3. 每段分析时传入前序段落的 summary 作为上下文（第1段 previous_summary 为空）
  4. 所有段落分析完成后，汇总输出完整分析报告

**分析维度：**
- 视觉元素：画面构图、字幕设计、转场效果
- 音频元素：语气、节奏、背景音乐
- 内容结构：叙事方式、信息密度、节奏把控
"""
```

- [ ] **Step 2: Update references**

In `remix_dynamic_prompt`, replace `OMNI_AUDIO_GUIDE` with `OMNI_VIDEO_GUIDE`:

```python
    if is_omni_enabled(config_name):
        max_duration = getattr(settings, 'OMNI_VIDEO_MAX_DURATION', 400)
        full_prompt += "\n\n" + OMNI_VIDEO_GUIDE.format(max_duration=max_duration)
```

Update `__all__` to replace `"OMNI_AUDIO_GUIDE"` with `"OMNI_VIDEO_GUIDE"`.

- [ ] **Step 3: Commit**

```bash
git add backend/agent/prompts.py
git commit -m "feat: replace OMNI_AUDIO_GUIDE with OMNI_VIDEO_GUIDE"
```

---

### Task 10: Integration Verification

- [ ] **Step 1: Verify all imports**

```bash
cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "
from agent.tools import prepare_video_segments, analyze_video_segment
from agent.middleware import omni_media_injection_middleware
from services.video_transcoder import VideoTranscoder
from services.qiniu_storage import QiniuStorageService
from agent.prompts import OMNI_VIDEO_GUIDE
from agent.state import VideoSegmentDict
print('All imports OK')
"
```

Expected: `All imports OK`

- [ ] **Step 2: Verify VideoTranscoder ffmpeg dependency**

```bash
cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "
import subprocess
result = subprocess.run(['ffmpeg', '-version'], capture_output=True, text=True)
print('ffmpeg available:', result.returncode == 0)
result2 = subprocess.run(['ffprobe', '-version'], capture_output=True, text=True)
print('ffprobe available:', result2.returncode == 0)
"
```

Expected: Both `True`.

- [ ] **Step 3: Verify agent creation with new tools**

```bash
cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "
from agent.remix_agent import create_remix_agent
print('Agent creation OK')
"
```

Expected: `Agent creation OK` (no import errors).

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: omni video segment analysis — complete integration"
```
