# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/agent/middleware/omni_media_injection.py
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
全模态媒体注入中间件

当全模态模式启用时，将视频或音频 URL 注入到最后一条 HumanMessage 中。
视频优先级高于音频（有视频则不注入音频）。
与 multimodal_injection_middleware（图片注入）对称设计。

DashScope 的 OpenAI-compatible Omni 音频输入要求使用
`input_audio: {"data": ..., "format": ...}`，不能只传 `url`。

基于 LangChain 1.0 的 @before_model 中间件模式实现。
"""
from typing import Optional, Dict, Any
from urllib.parse import urlparse
from pathlib import Path

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


def _infer_audio_format(audio_url: str) -> str:
    """从音频 URL 推断格式；DashScope 需要显式提供 format 字段。"""
    path = urlparse(audio_url).path or audio_url
    suffix = Path(path).suffix.lower().lstrip(".")
    if suffix in {"wav", "mp3", "m4a", "aac", "amr", "3gp", "3gpp"}:
        return suffix
    return "wav"


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
        media_content = {
            "type": "input_audio",
            "input_audio": {
                "data": audio_url,
                "format": _infer_audio_format(audio_url),
            },
        }
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
