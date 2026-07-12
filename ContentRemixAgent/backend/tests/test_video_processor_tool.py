# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_video_processor_tool.py
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

"""Tests for video processor omni path helpers."""

from agent.tools.video_processor_tool import _should_merge_separate_av_for_omni


def test_should_merge_separate_av_for_omni_when_video_and_audio_are_split(monkeypatch):
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)

    content_info = {
        "video_url": "https://example.com/video.m4s",
        "audio_url": "https://example.com/audio.m4s",
    }

    assert _should_merge_separate_av_for_omni(content_info, "qwen-omni") is True


def test_should_not_merge_when_video_and_audio_share_same_url(monkeypatch):
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: True)

    content_info = {
        "video_url": "https://example.com/full.mp4",
        "audio_url": "https://example.com/full.mp4",
    }

    assert _should_merge_separate_av_for_omni(content_info, "qwen-omni") is False


def test_should_not_merge_for_non_omni_model(monkeypatch):
    monkeypatch.setattr("llm_provider.is_omni_enabled", lambda config_name: False)

    content_info = {
        "video_url": "https://example.com/video.m4s",
        "audio_url": "https://example.com/audio.m4s",
    }

    assert _should_merge_separate_av_for_omni(content_info, "regular-model") is False
