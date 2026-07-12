# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_stream_event_processor.py
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

"""Tests for StreamEventProcessor final-report streaming boundaries."""

from langchain_core.messages import AIMessageChunk

from agent.stream.processor import StreamEventProcessor
from api.routes.sse_helpers import SegmentCollector


def _tool_start(name: str = "fetch_content", run_id: str = "tool-run") -> dict:
    return {
        "event": "on_tool_start",
        "name": name,
        "run_id": run_id,
        "data": {"input": {"url": "https://example.com"}},
    }


def _tool_end(name: str = "fetch_content", run_id: str = "tool-run") -> dict:
    return {
        "event": "on_tool_end",
        "name": name,
        "run_id": run_id,
        "data": {"output": "工具完成"},
    }


def _model_start() -> dict:
    return {"event": "on_chat_model_start", "name": "model", "data": {}}


def _model_stream(chunk: AIMessageChunk) -> dict:
    return {"event": "on_chat_model_stream", "name": "model", "data": {"chunk": chunk}}


def _model_end() -> dict:
    return {"event": "on_chat_model_end", "name": "model", "data": {}}


def test_final_report_start_is_emitted_before_first_final_text_delta():
    processor = StreamEventProcessor()

    list(processor.process_event(_tool_start()))
    list(processor.process_event(_tool_end()))
    list(processor.process_event(_model_start()))

    events = list(processor.process_event(_model_stream(AIMessageChunk(content="最终报告第一段"))))

    assert [event["type"] for event in events] == ["final_report_start", "text_delta"]
    assert events[1]["data"] == {
        "content": "最终报告第一段",
        "is_process_text": False,
    }


def test_direct_answer_without_tools_streams_as_final_report():
    processor = StreamEventProcessor()

    list(processor.process_event(_model_start()))
    events = list(processor.process_event(_model_stream(AIMessageChunk(content="直接回答"))))

    assert [event["type"] for event in events] == ["final_report_start", "text_delta"]
    assert events[1]["data"]["is_process_text"] is False


def test_preamble_before_later_tool_call_is_process_text_not_final_report():
    processor = StreamEventProcessor()

    list(processor.process_event(_model_start()))
    preamble_events = list(processor.process_event(_model_stream(AIMessageChunk(content="先解析链接。"))))
    tool_events = list(processor.process_event(_model_stream(AIMessageChunk(
        content="",
        tool_call_chunks=[{
            "id": "call-parse",
            "name": "parse_link",
            "args": "{}",
            "index": 0,
        }],
    ))))

    assert [event["type"] for event in preamble_events] == ["final_report_start", "text_delta"]
    assert [event["type"] for event in tool_events] == ["final_report_cancel"]


def test_analyze_preamble_is_deferred_to_process_text_before_tool_call():
    processor = StreamEventProcessor(defer_final_until_tool_started=True)

    list(processor.process_event(_model_start()))
    preamble_events = list(processor.process_event(_model_stream(AIMessageChunk(content="先解析链接。"))))
    tool_events = list(processor.process_event(_model_stream(AIMessageChunk(
        content="",
        tool_call_chunks=[{
            "id": "call-parse",
            "name": "parse_link",
            "args": "{}",
            "index": 0,
        }],
    ))))

    assert preamble_events == []
    assert [event["type"] for event in tool_events] == ["text_delta"]
    assert tool_events[0]["data"] == {
        "content": "先解析链接。",
        "is_process_text": True,
    }


def test_deferred_text_without_tool_call_still_becomes_final_report_on_turn_end():
    processor = StreamEventProcessor(defer_final_until_tool_started=True)

    list(processor.process_event(_model_start()))
    stream_events = list(processor.process_event(_model_stream(AIMessageChunk(content="直接分析结果"))))
    end_events = list(processor.process_event(_model_end()))

    assert stream_events == []
    assert [event["type"] for event in end_events] == ["final_report_start", "text_delta"]
    assert end_events[1]["data"] == {
        "content": "直接分析结果",
        "is_process_text": False,
    }


def test_analyze_inter_tool_text_after_prior_tool_is_deferred_to_process_text():
    processor = StreamEventProcessor(defer_final_until_tool_started=True)

    list(processor.process_event(_tool_start(name="parse_link", run_id="parse-run")))
    list(processor.process_event(_tool_end(name="parse_link", run_id="parse-run")))
    list(processor.process_event(_model_start()))
    inter_tool_events = list(processor.process_event(_model_stream(AIMessageChunk(content="链接已解析，接下来获取内容详情。"))))
    tool_events = list(processor.process_event(_model_stream(AIMessageChunk(
        content="",
        tool_call_chunks=[{
            "id": "call-fetch",
            "name": "fetch_content",
            "args": "{}",
            "index": 0,
        }],
    ))))

    assert inter_tool_events == []
    assert [event["type"] for event in tool_events] == ["text_delta"]
    assert tool_events[0]["data"] == {
        "content": "链接已解析，接下来获取内容详情。",
        "is_process_text": True,
    }


def test_analyze_final_report_starts_streaming_after_report_is_clear():
    processor = StreamEventProcessor(defer_final_until_tool_started=True)
    report_start = "所有数据已获取完毕，下面为你呈现完整的深度拆解分析。\n\n🔬 深度拆解"

    list(processor.process_event(_tool_start(name="process_video", run_id="video-run")))
    list(processor.process_event(_tool_end(name="process_video", run_id="video-run")))
    list(processor.process_event(_model_start()))
    events = list(processor.process_event(_model_stream(AIMessageChunk(content=report_start))))

    assert [event["type"] for event in events] == ["final_report_start", "text_delta"]
    assert events[1]["data"] == {
        "content": report_start,
        "is_process_text": False,
    }


def test_tool_call_turn_text_remains_process_text_after_prior_tool():
    processor = StreamEventProcessor()

    list(processor.process_event(_tool_start()))
    list(processor.process_event(_tool_end()))
    list(processor.process_event(_model_start()))

    chunk = AIMessageChunk(
        content="我还需要继续调用工具",
        tool_call_chunks=[{
            "id": "call-next",
            "name": "process_video",
            "args": "{}",
            "index": 0,
        }],
    )
    events = list(processor.process_event(_model_stream(chunk)))

    event_types = [event["type"] for event in events]
    assert "final_report_start" not in event_types
    assert events[-1]["type"] == "text_delta"
    assert events[-1]["data"]["is_process_text"] is True


def test_segment_collector_rolls_back_premature_final_report_to_process_text():
    collector = SegmentCollector()

    collector.process_event("final_report_start", {}, 1.0)
    collector.process_event("text_delta", {"content": "先解析链接。", "is_process_text": False}, 1.1)
    collector.process_event("final_report_cancel", {}, 1.2)
    collector.process_event("tool_call_start", {"tool": "parse_link", "input": {}}, 1.3)
    collector.process_event("tool_call_finish", {"tool": "parse_link", "output": "完成"}, 1.4)
    collector.process_event("final_report_start", {}, 2.0)
    collector.process_event("text_delta", {"content": "最终报告", "is_process_text": False}, 2.1)

    segments = collector.finalize_segments()

    assert segments[0] == {
        "type": "process_text",
        "content": "先解析链接。",
        "status": "completed",
    }
    assert segments[-1] == {
        "type": "markdown",
        "content": "最终报告",
    }
    assert collector.get_assistant_message() == "最终报告"
