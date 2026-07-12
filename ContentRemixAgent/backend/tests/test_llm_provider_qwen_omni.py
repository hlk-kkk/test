# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_llm_provider_qwen_omni.py
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

"""Tests for Qwen Omni compatibility settings in llm_provider."""

import llm_provider


def _make_db_config(model_name: str, base_url: str) -> dict:
    return {
        "provider": "openai",
        "model_name": model_name,
        "api_key": "test-key",
        "base_url": base_url,
        "enable_thinking": False,
        "thinking_budget_tokens": 4096,
        "reasoning_effort": "high",
    }


def test_get_llm_forces_streaming_for_dashscope_qwen_omni(monkeypatch):
    captured = {}

    def fake_chat_openai(*, model, **kwargs):
        captured["model"] = model
        captured["kwargs"] = kwargs
        return {"model": model, "kwargs": kwargs}

    monkeypatch.setattr(
        llm_provider,
        "_get_active_db_config",
        lambda: _make_db_config(
            "qwen3.5-omni-plus",
            "https://dashscope.aliyuncs.com/compatible-mode/v1",
        ),
    )
    monkeypatch.setattr(llm_provider, "ChatOpenAI", fake_chat_openai)

    result = llm_provider.get_llm(enable_debug=False)

    assert result["model"] == "qwen3.5-omni-plus"
    assert captured["kwargs"]["streaming"] is True
    assert captured["kwargs"]["model_kwargs"] == {
        "stream_options": {"include_usage": True},
    }


def test_get_llm_does_not_force_streaming_for_regular_openai_model(monkeypatch):
    captured = {}

    def fake_chat_openai(*, model, **kwargs):
        captured["model"] = model
        captured["kwargs"] = kwargs
        return {"model": model, "kwargs": kwargs}

    monkeypatch.setattr(
        llm_provider,
        "_get_active_db_config",
        lambda: _make_db_config(
            "gpt-4o-mini",
            "https://api.openai.com/v1",
        ),
    )
    monkeypatch.setattr(llm_provider, "ChatOpenAI", fake_chat_openai)

    result = llm_provider.get_llm(enable_debug=False)

    assert result["model"] == "gpt-4o-mini"
    assert "streaming" not in captured["kwargs"]
    assert "model_kwargs" not in captured["kwargs"]


def test_get_omni_video_max_duration_uses_qwen_limit(monkeypatch):
    monkeypatch.setattr(
        llm_provider,
        "_get_active_db_config",
        lambda: _make_db_config(
            "qwen3.5-omni-plus",
            "https://dashscope.aliyuncs.com/compatible-mode/v1",
        ),
    )

    assert llm_provider.get_omni_video_max_duration() == llm_provider.settings.QWEN_OMNI_VIDEO_MAX_DURATION


def test_get_omni_video_max_duration_keeps_default_for_other_models(monkeypatch):
    monkeypatch.setattr(
        llm_provider,
        "_get_active_db_config",
        lambda: _make_db_config(
            "gpt-4o",
            "https://api.openai.com/v1",
        ),
    )

    assert llm_provider.get_omni_video_max_duration() == llm_provider.settings.OMNI_VIDEO_MAX_DURATION
