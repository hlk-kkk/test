#!/usr/bin/env python
# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/scripts/test_omni_tools_direct.py
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
全模态工具直接测试（绕过 Agent 循环）

按顺序直接调用每个工具，验证完整的全模态视频分段分析链路。
避免弱 Agent 模型的工具调用循环问题。
"""
import asyncio
import uuid
import sys
from unittest.mock import MagicMock


class FakeRuntime:
    """模拟 ToolRuntime"""
    def __init__(self, context, state, tool_call_id=None):
        self.context = context
        self._state = state
        self.tool_call_id = tool_call_id or str(uuid.uuid4())

    @property
    def state(self):
        return self._state

    def stream_writer(self, data):
        stage = data.get("stage", "")
        message = data.get("message", "")
        print(f"  📡 [{stage}] {message}")


async def main():
    from agent.state import RemixContext
    from config import settings

    url = "https://www.bilibili.com/video/BV1ZpzhBLE82/"
    config_name = "Qwen3.5-Omni"
    session_id = f"omni-direct-{uuid.uuid4().hex[:8]}"

    print("=" * 60)
    print("全模态工具直接测试")
    print(f"URL: {url}")
    print(f"Config: {config_name}")
    print(f"Session: {session_id}")
    print("=" * 60)

    context = RemixContext(
        session_id=session_id,
        mode="analyze",
        config_name=config_name,
        use_mock=False,
    )

    state = {}

    # ====== Step 1: parse_link ======
    print("\n[1/5] parse_link")
    from agent.tools.link_parser_tool import parse_link
    rt = FakeRuntime(context, state, tool_call_id="call_parse")
    result = parse_link.invoke({"url": url, "runtime": rt})
    update = result.update
    state.update({k: v for k, v in update.items() if k != "messages"})
    for m in update.get("messages", []):
        print(f"  ✅ {m.content[:200]}")

    # ====== Step 2: fetch_content ======
    print("\n[2/5] fetch_content")
    from agent.tools.fetch_content_tool import fetch_content
    rt = FakeRuntime(context, state, tool_call_id="call_fetch")
    result = await fetch_content.ainvoke({"url": url, "runtime": rt})
    update = result.update
    state.update({k: v for k, v in update.items() if k != "messages"})
    for m in update.get("messages", []):
        print(f"  ✅ {m.content[:200]}")
    print(f"  has_video: {bool(state.get('content_info', {}).get('video_url'))}")

    # ====== Step 3: process_video ======
    print("\n[3/5] process_video")
    from agent.tools.video_processor_tool import process_video
    video_url = state["content_info"]["video_url"]
    rt = FakeRuntime(context, state, tool_call_id="call_process")
    result = await process_video.ainvoke({"video_url": video_url, "runtime": rt})
    update = result.update
    state.update({k: v for k, v in update.items() if k != "messages"})
    for m in update.get("messages", []):
        print(f"  ✅ {m.content[:300]}")

    transcript = state.get("transcript")
    if transcript:
        print(f"  transcript: {len(transcript.get('text', ''))} chars, {len(transcript.get('segments', []))} segments")

    omni_video_url = state.get("omni_video_url")
    if omni_video_url:
        print(f"  omni_video_url: {omni_video_url}")
        print("  短视频直传模式，无需分段分析。测试完成！")
        return

    # ====== Step 4: prepare_video_segments ======
    print("\n[4/5] prepare_video_segments")
    from agent.tools.video_segment_tool import prepare_video_segments
    rt = FakeRuntime(context, state, tool_call_id="call_segments")
    result = await prepare_video_segments.ainvoke({"runtime": rt})
    update = result.update
    state.update({k: v for k, v in update.items() if k != "messages"})
    for m in update.get("messages", []):
        print(f"  ✅ {m.content[:500]}")

    segments = state.get("video_segments", [])
    print(f"  video_segments: {len(segments)} segments")
    for s in segments:
        print(f"    [{s['index']}] {s['title']} ({s['start']:.0f}s-{s['end']:.0f}s) -> {s['video_url']}")

    # ====== Step 5: analyze_video_segment (逐段) ======
    print("\n[5/5] analyze_video_segment (逐段)")
    from agent.tools.video_segment_tool import analyze_video_segment

    previous_summary = ""
    for seg in segments:
        print(f"\n  --- 分析第 {seg['index']} 段: {seg['title']} ---")
        rt = FakeRuntime(context, state, tool_call_id=f"call_analyze_{seg['index']}")
        try:
            result = await analyze_video_segment.ainvoke({
                "segment_index": seg["index"],
                "previous_summary": previous_summary,
                "runtime": rt,
            })
            update = result.update
            for m in update.get("messages", []):
                content = m.content
                print(f"  ✅ 分析完成 ({len(content)} chars)")
                # 提取 summary
                if "段落摘要" in content:
                    summary_start = content.index("段落摘要")
                    previous_summary = content[summary_start:][:300]
                    print(f"  📝 摘要: {previous_summary[:100]}...")
                print(f"  📄 内容预览: {content[:300]}...")
        except Exception as e:
            print(f"  ❌ 第 {seg['index']} 段失败: {e}")
            # 继续分析下一段

    print("\n" + "=" * 60)
    print("✅ 全模态工具直接测试完成!")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
