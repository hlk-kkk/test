# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/tests/test_video_segment_progress.py
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

"""Tests for long-video segment progress helpers."""

from agent.tools.video_segment_tool import (
    _build_segment_progress_hint,
    _normalize_analyzed_segments,
    _remaining_segment_indices,
)


def test_normalize_analyzed_segments_deduplicates_and_sorts():
    assert _normalize_analyzed_segments([3, 1, 3, 2, -1, 0]) == [1, 2, 3]


def test_remaining_segment_indices_returns_unfinished_segments():
    video_segments = [{"index": 1}, {"index": 2}, {"index": 3}]
    assert _remaining_segment_indices(video_segments, [2]) == [1, 3]


def test_build_segment_progress_hint_guides_next_segment():
    video_segments = [{"index": 1}, {"index": 2}, {"index": 3}]
    hint = _build_segment_progress_hint(video_segments, [1])

    assert "剩余未分析段落: 2, 3" in hint
    assert "analyze_video_segment(segment_index=2)" in hint


def test_build_segment_progress_hint_stops_after_all_segments_done():
    video_segments = [{"index": 1}, {"index": 2}]
    hint = _build_segment_progress_hint(video_segments, [1, 2])

    assert "所有分段均已分析完成" in hint
    assert "不要再次调用 prepare_video_segments 或 analyze_video_segment" in hint
