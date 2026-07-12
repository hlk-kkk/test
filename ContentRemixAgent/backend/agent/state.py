# -*- coding: utf-8 -*-
# Copyright (c) 2026 relakkes@gmail.com
#
# This file is part of MediaCrawlerPro-ContentRemixAgent project.
# Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/backend/agent/state.py
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
Agent 状态和上下文定义

基于 LangChain 1.0 最佳实践:
- RemixAgentState: Agent 状态 (可变，会话级别)
- RemixContext: Tool 运行时上下文 (不可变，请求级别)

类型说明:
- State 字段使用 TypedDict 提供类型提示，同时保持 JSON 可序列化
- 实际数据结构与 schemas.py 中的 Pydantic 模型对应
"""

from dataclasses import dataclass
from typing import Optional, Dict, Any, Annotated, List, TypedDict
from langchain.agents import AgentState
from langgraph.graph.message import add_messages


def last_value(existing: Optional[Any], new: Optional[Any]) -> Optional[Any]:
    """Last value reducer - 只保留最后一个值"""
    return new if new is not None else existing


# ========== TypedDict 类型定义 ==========
# 提供类型提示，与 schemas.py 中的 Pydantic 模型对应

class ParsedLinkDict(TypedDict, total=False):
    """链接解析结果 (对应 schemas.ParsedLinkInfo)"""
    platform: str        # xhs/dy/bilibili/ks
    content_id: str
    original_url: str
    is_short_link: bool


class ContentInfoDict(TypedDict, total=False):
    """内容信息 (对应 schemas.ContentParseResponse)"""
    platform: str
    content_id: str
    content_type: str
    title: str
    desc: str
    author_id: str
    author_name: str
    author_avatar: str
    cover_url: str
    video_url: Optional[str]
    audio_url: Optional[str]
    image_urls: List[str]
    like_count: int
    comment_count: int
    share_count: int
    collect_count: int
    view_count: int
    publish_time: int
    duration: int
    tags: List[str]
    # 本地资源 URL (由 video_processor_tool 填充)
    local_cover_url: Optional[str]
    local_video_url: Optional[str]

    # 多模态支持 (由 fetch_content_tool 填充，用于图文内容)
    local_image_paths: List[str]


class TranscriptSegmentDict(TypedDict):
    """转录分段 (对应 schemas.Segment)"""
    start: float  # 开始时间（秒）
    end: float    # 结束时间（秒）
    text: str     # 文本内容


class TranscriptDict(TypedDict, total=False):
    """转录结果 (对应 schemas.TranscriptResult)"""
    text: str                          # 完整文本
    segments: List[TranscriptSegmentDict]  # 分段信息（带时间戳）
    content_id: str                    # 关联的内容 ID（用于缓存验证）


class VideoSegmentDict(TypedDict, total=False):
    """视频分段信息"""
    index: int           # 段序号 (从1开始)
    start: float         # 开始时间（秒）
    end: float           # 结束时间（秒）
    title: str           # 段落标题（LLM 生成）
    video_url: str       # 七牛云 URL


# ========== Agent 状态定义 ==========

class RemixAgentState(AgentState):
    """
    二创 Agent 状态

    继承自 AgentState，自动包含 messages 字段（带 add_messages reducer）。
    传递给 create_agent() 的 state_schema 参数。

    简化架构：只保留数据获取相关的状态字段
    分析和灵感内容由 Agent 直接输出，不再保存到状态中

    Attributes:
        (继承) messages: 对话消息列表 (带 add_messages reducer)
        parsed_link: 解析后的链接信息 (对应 ParsedLinkInfo)
        content_info: 爬取到的内容详情 (对应 ContentParseResponse)
        transcript: 视频转录文本
        current_stage: 当前处理阶段 (用于进度展示)
        error: 最近的错误信息
        total_input_tokens: 累计输入 token 数 (用于 Context 压缩判断)
        total_output_tokens: 累计输出 token 数 (用于 Context 压缩判断)
    """

    # 链接解析结果 (对应 schemas.ParsedLinkInfo)
    parsed_link: Annotated[Optional[ParsedLinkDict], last_value]

    # 内容信息 (对应 schemas.ContentParseResponse)
    content_info: Annotated[Optional[ContentInfoDict], last_value]

    # 视频转录结果（包含完整文本和带时间戳的分段）
    transcript: Annotated[Optional[TranscriptDict], last_value]

    # 全模态音频 URL（七牛云公网 URL，供 omni 模型直接理解音频）
    omni_audio_url: Annotated[Optional[str], last_value]

    # 全模态视频 URL（短视频直传，≤400s）
    omni_video_url: Annotated[Optional[str], last_value]

    # 视频分段列表（长视频分段后的段落信息）
    video_segments: Annotated[Optional[List[VideoSegmentDict]], last_value]

    # 已完成分析的视频段序号（长视频分段分析进度）
    analyzed_segments: Annotated[Optional[List[int]], last_value]

    # 当前处理阶段 - 使用 Annotated 支持并发更新
    current_stage: Annotated[Optional[str], last_value]

    # 错误信息 - 使用 Annotated 支持并发更新
    error: Annotated[Optional[str], last_value]

    # Token 使用统计 (用于 Context 压缩)
    # 累计的输入 token 数量（从 AIMessage.usage_metadata 获取的真实数据）
    total_input_tokens: int
    # 累计的输出 token 数量
    total_output_tokens: int


@dataclass
class RemixContext:
    """
    Tool 运行时上下文 (不可变)

    通过 runtime.context 访问，包含请求级别的配置信息。
    在 agent.invoke() 或 agent.stream() 时通过 context= 参数传入。

    Attributes:
        session_id: 会话 ID (用于 checkpointer)
        user_id: 用户 ID (可选)
        mode: 二创模式 (imitate/rewrite/summarize/analyze)
        instruction: 用户自然语言指令
        use_mock: 是否使用 Mock 服务
    """

    session_id: str
    user_id: Optional[str] = None
    mode: Optional[str] = None  # imitate, rewrite, summarize, analyze
    instruction: Optional[str] = None  # 用户自然语言指令
    config_name: Optional[str] = None  # LLM 配置名称（用于判断 omni 等能力）
    use_mock: bool = False
