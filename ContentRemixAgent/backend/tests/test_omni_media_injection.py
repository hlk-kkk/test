# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_omni_media_injection.py
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

"""Tests for omni media injection middleware."""

import importlib.util
import sys
from pathlib import Path
from types import ModuleType, SimpleNamespace

from langchain_core.messages import HumanMessage


def _load_module(monkeypatch):
    module_name = "omni_media_injection_under_test"
    module_path = Path(__file__).resolve().parents[1] / "agent" / "middleware" / "omni_media_injection.py"

    utils_module = ModuleType("utils")
    logger_module = ModuleType("utils.logger")

    class _Logger:
        def debug(self, *args, **kwargs):
            pass

        def info(self, *args, **kwargs):
            pass

        def warning(self, *args, **kwargs):
            pass

    logger_module.logger = _Logger()

    llm_provider_module = ModuleType("llm_provider")
    llm_provider_module.is_omni_enabled = lambda config_name=None: False

    langchain_module = ModuleType("langchain")
    agents_module = ModuleType("langchain.agents")
    middleware_module = ModuleType("langchain.agents.middleware")
    middleware_module.before_model = lambda func: func
    agents_module.middleware = middleware_module
    langchain_module.agents = agents_module

    monkeypatch.setitem(sys.modules, "utils", utils_module)
    monkeypatch.setitem(sys.modules, "utils.logger", logger_module)
    monkeypatch.setitem(sys.modules, "llm_provider", llm_provider_module)
    monkeypatch.setitem(sys.modules, "langchain", langchain_module)
    monkeypatch.setitem(sys.modules, "langchain.agents", agents_module)
    monkeypatch.setitem(sys.modules, "langchain.agents.middleware", middleware_module)

    spec = importlib.util.spec_from_file_location(module_name, module_path)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    spec.loader.exec_module(module)
    return module


def _runtime(config_name: str = "dashscope-omni"):
    return SimpleNamespace(context=SimpleNamespace(config_name=config_name))


def test_omni_media_injection_builds_dashscope_audio_payload(monkeypatch):
    module = _load_module(monkeypatch)
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)

    state = {
        "messages": [HumanMessage(content="请分析这段音频")],
        "omni_audio_url": "https://example.com/audio.wav",
    }

    result = module.omni_media_injection_middleware(state, _runtime())

    assert result is not None
    assert result["omni_audio_url"] is None

    content = result["messages"][0].content
    assert isinstance(content, list)
    assert content[0] == {"type": "text", "text": "请分析这段音频"}
    assert content[1] == {
        "type": "input_audio",
        "input_audio": {
            "data": "https://example.com/audio.wav",
            "format": "wav",
        },
    }


def test_omni_media_injection_infers_audio_format_from_url(monkeypatch):
    module = _load_module(monkeypatch)
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)

    state = {
        "messages": [HumanMessage(content="请分析这段音频")],
        "omni_audio_url": "https://example.com/audio/sample.mp3?download=1",
    }

    result = module.omni_media_injection_middleware(state, _runtime())

    assert result is not None
    assert result["messages"][0].content[1]["input_audio"]["format"] == "mp3"


def test_omni_media_injection_prefers_video_over_audio(monkeypatch):
    module = _load_module(monkeypatch)
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)

    state = {
        "messages": [HumanMessage(content="请分析这段媒体")],
        "omni_video_url": "https://example.com/video.mp4",
        "omni_audio_url": "https://example.com/audio.wav",
    }

    result = module.omni_media_injection_middleware(state, _runtime())

    assert result is not None
    content = result["messages"][0].content
    assert content[1] == {
        "type": "video_url",
        "video_url": {"url": "https://example.com/video.mp4"},
    }
