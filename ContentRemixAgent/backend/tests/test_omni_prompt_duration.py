# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_omni_prompt_duration.py
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

"""Tests for omni prompt duration rendering."""

from types import SimpleNamespace

from agent.prompts import _build_remix_dynamic_prompt


def test_remix_dynamic_prompt_uses_qwen_omni_duration(monkeypatch):
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)
    monkeypatch.setattr("llm_provider.get_omni_video_max_duration", lambda config_name: 3600)

    request = SimpleNamespace(runtime=SimpleNamespace(context=SimpleNamespace(mode="analyze", config_name="qwen-omni")))
    prompt = _build_remix_dynamic_prompt(request)

    assert "3600" in prompt


def test_remix_dynamic_prompt_uses_default_omni_duration(monkeypatch):
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)
    monkeypatch.setattr("llm_provider.get_omni_video_max_duration", lambda config_name: 400)

    request = SimpleNamespace(runtime=SimpleNamespace(context=SimpleNamespace(mode="analyze", config_name="other-omni")))
    prompt = _build_remix_dynamic_prompt(request)

    assert "400" in prompt
