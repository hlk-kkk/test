#!/usr/bin/env python
# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/scripts/test_omni_e2e.py
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
全模态 E2E 测试脚本

测试完整链路: B站链接 → parse_link → fetch_content → process_video (omni) → Agent 分析

使用指定的全模态模型配置 (Qwen3.5-Omni)
"""
import asyncio
import json
import uuid
import sys

async def main():
    from agent.remix_agent import create_remix_agent, get_session_config
    from agent.state import RemixContext
    from langchain_core.messages import HumanMessage

    url = "https://www.bilibili.com/video/BV1ZpzhBLE82/"
    config_name = "GLM-5V-Turbo"
    session_id = f"omni-e2e-test-{uuid.uuid4().hex[:8]}"

    print(f"=" * 60)
    print(f"全模态 E2E 测试")
    print(f"URL: {url}")
    print(f"Config: {config_name}")
    print(f"Session: {session_id}")
    print(f"=" * 60)

    # 创建 Agent
    print("\n[1/2] 创建 Agent...")
    agent = create_remix_agent(config_name=config_name)
    print("Agent 创建成功")

    # 创建上下文
    context = RemixContext(
        session_id=session_id,
        mode="analyze",
        config_name=config_name,
        use_mock=False,
    )

    config = get_session_config(session_id)
    config["recursion_limit"] = 100

    # 构建消息
    user_message = f"深度拆解这个内容的创作技巧：{url}"
    messages = [HumanMessage(content=user_message)]

    print(f"\n[2/2] 开始 Agent 流式执行...")
    print("-" * 60)

    try:
        async for event in agent.astream_events(
            {"messages": messages},
            config=config,
            context=context,
            version="v2",
        ):
            kind = event.get("event", "")
            name = event.get("name", "")

            # 工具调用开始
            if kind == "on_tool_start":
                tool_name = name
                print(f"\n🔧 工具调用: {tool_name}")
                tool_input = event.get("data", {}).get("input", {})
                if tool_input and isinstance(tool_input, dict):
                    # 只打印非空的关键参数
                    for k, v in tool_input.items():
                        if v and k not in ("runtime",):
                            val_str = str(v)[:100]
                            print(f"   {k}: {val_str}")

            # 工具调用结束
            elif kind == "on_tool_end":
                tool_name = name
                output = event.get("data", {}).get("output", "")
                if hasattr(output, 'update') and isinstance(output.update, dict):
                    update = output.update
                    # 打印状态更新
                    if update.get("omni_video_url"):
                        print(f"   ✅ omni_video_url: {update['omni_video_url']}")
                    if update.get("omni_audio_url"):
                        print(f"   ✅ omni_audio_url: {update['omni_audio_url']}")
                    if update.get("transcript"):
                        t = update["transcript"]
                        text = t.get("text", "") if isinstance(t, dict) else str(t)
                        print(f"   ✅ transcript: {len(text)} chars")
                    if update.get("video_segments"):
                        segs = update["video_segments"]
                        print(f"   ✅ video_segments: {len(segs)} segments")
                    if update.get("current_stage"):
                        print(f"   📍 stage: {update['current_stage']}")

                    # 打印 ToolMessage 内容
                    msgs = update.get("messages", [])
                    for m in msgs:
                        if hasattr(m, 'content'):
                            print(f"   💬 {m.content[:200]}")

            # 自定义事件（子步骤进度）
            elif kind == "on_custom_event":
                event_name = name
                event_data = event.get("data", {})
                if event_name == "sub_step_start":
                    print(f"   ⏳ {event_data.get('label', '')}")
                elif event_name == "sub_step_end":
                    print(f"   ✅ {event_data.get('message', '')}")

            # LLM 输出
            elif kind == "on_chat_model_stream":
                chunk = event.get("data", {}).get("chunk", None)
                if chunk and hasattr(chunk, 'content') and chunk.content:
                    content = chunk.content
                    if isinstance(content, str):
                        print(content, end="", flush=True)

        print("\n" + "-" * 60)
        print("✅ E2E 测试完成!")

    except Exception as e:
        print(f"\n❌ E2E 测试失败: {type(e).__name__}: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
