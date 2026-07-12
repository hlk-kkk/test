# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/agent/tools/video_segment_tool.py
# GitHub: https://github.com/NanmiCoder
# Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
#
# 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
# 1. 不得用于任何商业用途。
# 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
# 3. 不得进行大规模爬取或对平台造成运营干扰。
# 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
# 5. 不得用于任何非法或不当的用途。
#
# 详细许可条款请参阅项目根目录下的LICENSE文件。
# 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。

"""
视频分段分析工具

提供两个工具：
1. prepare_video_segments: 对长视频进行语义分段、切割、上传
2. analyze_video_segment: 使用子 Agent 分析单个视频段落

用于全模态模型处理超过 400s 的长视频场景。
"""
import shutil
import uuid
from pathlib import Path

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


def _normalize_analyzed_segments(analyzed_segments: list[int] | None) -> list[int]:
    """规范化已分析段落索引列表。"""
    normalized = []
    for value in analyzed_segments or []:
        if isinstance(value, int) and value > 0 and value not in normalized:
            normalized.append(value)
    return sorted(normalized)


def _remaining_segment_indices(video_segments: list[dict], analyzed_segments: list[int] | None) -> list[int]:
    """返回尚未分析的段落索引。"""
    analyzed_set = set(_normalize_analyzed_segments(analyzed_segments))
    indices = []
    for segment in video_segments:
        index = segment.get("index")
        if isinstance(index, int) and index > 0 and index not in analyzed_set:
            indices.append(index)
    return indices


def _build_segment_progress_hint(video_segments: list[dict], analyzed_segments: list[int] | None) -> str:
    """构建段落分析进度提示，帮助主 Agent 决定下一步。"""
    normalized = _normalize_analyzed_segments(analyzed_segments)
    remaining = _remaining_segment_indices(video_segments, normalized)

    if remaining:
        remaining_text = ", ".join(str(i) for i in remaining)
        next_index = remaining[0]
        return (
            f"剩余未分析段落: {remaining_text}。\n"
            f"下一步请调用 analyze_video_segment(segment_index={next_index})，"
            f"并将当前返回的“段落摘要”作为 previous_summary 传给下一段。"
        )

    if normalized:
        return (
            "所有分段均已分析完成。现在直接汇总所有段落分析结果，输出最终完整报告。"
            "不要再次调用 prepare_video_segments 或 analyze_video_segment。"
        )

    return "尚未开始任何分段分析。"


# ============================================================================
# Tool: prepare_video_segments
# ============================================================================

@tool
async def prepare_video_segments(runtime: ToolRuntime[RemixContext]) -> Command:
    """根据转录文本对长视频进行语义分段，切割视频并上传到云端。

    前置条件: 需要先调用 process_video 获取转录文本(transcript)。
    仅在长视频(>400s)且使用全模态模型时调用。

    处理流程:
    1. 使用 LLM 对转录文本进行语义分段
    2. ffmpeg 按分段时间点切割视频并转码 720p
    3. 上传各段到七牛云
    4. 返回分段列表（含各段视频 URL）
    """
    # 缓存检查：如果已有分段结果，直接返回
    existing_segments = runtime.state.get("video_segments")
    if existing_segments:
        analyzed_segments = runtime.state.get("analyzed_segments", [])
        progress_hint = _build_segment_progress_hint(existing_segments, analyzed_segments)
        return Command(update={
            "messages": [ToolMessage(
                content=f"视频已分过段，共 {len(existing_segments)} 段。\n{progress_hint}",
                tool_call_id=runtime.tool_call_id,
            )],
        })

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

        # 使用 LLM 分段
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

        # 获取视频时长用于边界校验
        video_duration = transcript.get("duration", 0)
        if not video_duration and segments_text:
            video_duration = max(s.get("end", 0) for s in segments_text)

        # 校验和修正分段边界
        if video_duration > 0:
            valid_segments = []
            for seg in segment_plan:
                seg_start = seg.get("start", 0)
                seg_end = seg.get("end", 0)
                # 跳过起始时间超过视频时长的无效段
                if seg_start >= video_duration:
                    logger.warning(
                        f"[prepare_video_segments] Skipping segment: "
                        f"start={seg_start} >= duration={video_duration}"
                    )
                    continue
                # 钳制结束时间到视频时长
                if seg_end > video_duration:
                    logger.warning(
                        f"[prepare_video_segments] Clamping segment end: "
                        f"{seg_end} -> {video_duration}"
                    )
                    seg["end"] = video_duration
                # 跳过时长为0的段
                if seg["end"] <= seg_start:
                    continue
                valid_segments.append(seg)
            segment_plan = valid_segments

        if not segment_plan:
            raise RemixToolException(
                RemixErrorCode.INTERNAL_ERROR,
                "分段校验后无有效段落",
                {"video_duration": video_duration},
            )

        await adispatch_custom_event("sub_step_end", {
            "step_id": "semantic_segment",
            "message": f"已分为 {len(segment_plan)} 段",
        })

        logger.info(f"[prepare_video_segments] LLM segmentation: {len(segment_plan)} segments")

        # ====== Step 2: 查找视频文件 ======
        from services.video_downloader import VideoDownloader
        downloader = VideoDownloader()

        video_path = None
        local_video = content_info.get("local_video_url")
        if local_video and Path(local_video).exists():
            video_path = local_video
        else:
            # 在 temp 目录的任务子目录中查找，排除 segments_* 目录
            # segments_* 是之前分段产生的文件，不是原始视频
            temp_dir = Path(settings.VIDEO_TEMP_DIR)
            if temp_dir.exists():
                # 搜索常见视频格式（B站下载的是 .m4s 格式）
                candidates = []
                for pattern in ("*/*.mp4", "*/*.m4s", "*/*.flv", "*/*.mkv"):
                    for f in temp_dir.glob(pattern):
                        if not f.parent.name.startswith("segments_"):
                            candidates.append(f)
                if candidates:
                    candidates.sort(key=lambda p: p.stat().st_mtime, reverse=True)
                    video_path = str(candidates[0])
                    logger.info(f"[prepare_video_segments] Found video in temp: {video_path}")

        if not video_path or not Path(video_path).exists():
            # 需要重新下载视频
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
            logger.info(f"[prepare_video_segments] Re-downloading video from: {download_url[:100]}...")
            video_path = await downloader.download(download_url, task_id)

        logger.info(f"[prepare_video_segments] Using video: {video_path}")

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

        # 构建简短返回消息（不含 URL，避免 Agent 将原始数据回显到报告中）
        # URL 由 analyze_video_segment 自动从 state 中查找
        indices = ", ".join(str(s['index']) for s in video_segments)

        return Command(update={
            "messages": [ToolMessage(
                content=f"视频已分为 {len(video_segments)} 段。"
                        f"请依次调用 analyze_video_segment(segment_index=N)，N={indices}。"
                        f"第 1 段 previous_summary 为空，后续段传入前一段返回的摘要。"
                        f"所有段落分析完成后，汇总输出完整分析报告。",
                tool_call_id=runtime.tool_call_id,
            )],
            "video_segments": video_segments,
            "analyzed_segments": [],
            "current_stage": f"视频已分为 {len(video_segments)} 段",
        })

    except RemixToolException:
        # 确保所有 sub_step 都结束，避免前端进度卡住
        for step_id in ("semantic_segment", "split_video", "upload_segments"):
            try:
                await adispatch_custom_event("sub_step_end", {
                    "step_id": step_id,
                    "message": "失败",
                })
            except Exception:
                pass
        raise
    except Exception as e:
        logger.error(f"[prepare_video_segments] Failed: {e}", exc_info=True)
        # 确保所有 sub_step 都结束
        for step_id in ("semantic_segment", "split_video", "upload_segments"):
            try:
                await adispatch_custom_event("sub_step_end", {
                    "step_id": step_id,
                    "message": "失败",
                })
            except Exception:
                pass
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
    previous_summary: str = "",
    runtime: ToolRuntime[RemixContext] = None,
) -> Command:
    """分析单个视频段落。创建子 Agent 调用全模态模型直接分析视频内容。

    视频 URL 和标题自动从 state 中按 segment_index 查找，无需手动传入。

    Args:
        segment_index: 段序号 (从1开始)
        previous_summary: 前序段落的压缩摘要（首段为空字符串）

    Returns:
        Command 包含分析结果和压缩摘要
    """
    try:
        from llm_provider import get_llm

        # 从 state 查找分段信息
        video_segments = runtime.state.get("video_segments", [])
        analyzed_segments = _normalize_analyzed_segments(runtime.state.get("analyzed_segments", []))
        seg_info = next((s for s in video_segments if s["index"] == segment_index), None)
        if not seg_info:
            return Command(update={
                "messages": [ToolMessage(
                    content=f"未找到第 {segment_index} 段信息，请先调用 prepare_video_segments。",
                    tool_call_id=runtime.tool_call_id,
                )],
            })

        if segment_index in analyzed_segments:
            progress_hint = _build_segment_progress_hint(video_segments, analyzed_segments)
            return Command(update={
                "messages": [ToolMessage(
                    content=(
                        f"第 {segment_index} 段已分析完成，无需重复调用。\n"
                        f"{progress_hint}"
                    ),
                    tool_call_id=runtime.tool_call_id,
                )],
                "current_stage": f"第 {segment_index} 段已分析过",
            })

        segment_video_url = seg_info["video_url"]
        segment_title = seg_info["title"]
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

        updated_analyzed_segments = _normalize_analyzed_segments(analyzed_segments + [segment_index])
        progress_hint = _build_segment_progress_hint(video_segments, updated_analyzed_segments)

        return Command(update={
            "messages": [ToolMessage(
                content=(
                    f"## 第 {segment_index} 段分析: {segment_title}\n\n"
                    f"{analysis}\n\n"
                    f"**段落摘要（供后续段落参考）:** {summary}\n\n"
                    f"**进度提示:** {progress_hint}"
                ),
                tool_call_id=runtime.tool_call_id,
            )],
            "analyzed_segments": updated_analyzed_segments,
            "current_stage": (
                "所有视频段落分析完成，等待汇总"
                if not _remaining_segment_indices(video_segments, updated_analyzed_segments)
                else f"第 {segment_index} 段分析完成"
            ),
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
