"""Tool: enhance user prompt for image generation."""

import logging
from typing import Annotated

from langchain_core.messages import HumanMessage, SystemMessage, ToolMessage
from langchain_core.tools import InjectedToolCallId, tool
from langgraph.types import Command

from ...core.config import create_vision_llm

logger = logging.getLogger(__name__)

ENHANCEMENT_SYSTEM_PROMPT = """You are an expert at crafting prompts for AI image generation.

Your goal is to transform user prompts into optimized versions that produce:
- High visual quality
- Clear composition
- Visually appealing results
- Professional aesthetics

Guidelines:
- Keep the user's core intent
- Add specific visual details (lighting, perspective, depth, textures)
- Include quality modifiers (high resolution, detailed, professional)
- Incorporate style elements when provided
- Keep prompts focused and not overly long (max 150 words)
- Use comma-separated phrases for clarity

DO NOT:
- Add content the user didn't request
- Change the subject matter
- Include explanations in your output
- Use quotation marks around the prompt"""


@tool
async def enhance_prompt(
    original_prompt: str,
    tool_call_id: Annotated[str, InjectedToolCallId],
    feedback: str = "",
    template_analysis_summary: str = "",
) -> Command:
    """Enhance the user's prompt for better image generation results.
    Args:
        original_prompt: The user's original prompt text
        feedback: Optional feedback from a previous evaluation round
        template_analysis_summary: Optional summary of template analysis results
    Returns a Command that updates state with enhanced_prompt."""
    llm = create_vision_llm(
        temperature=0.7,
        max_retries=3,
    )

    if feedback:
        user_message = f"""Improve this image generation prompt based on feedback:

Current prompt: {original_prompt}

Feedback from evaluation: {feedback}

Create an improved version that addresses the feedback while maintaining the original intent.
Output ONLY the improved prompt, nothing else."""
    else:
        style_context = ""
        if template_analysis_summary:
            style_context = f"\n\nReference style information:\n{template_analysis_summary}"

        user_message = f"""Enhance this prompt for AI image generation:

Original prompt: {original_prompt}

Target: high-quality image
{style_context}

Create an enhanced version that will produce better results.
Output ONLY the enhanced prompt, nothing else."""

    try:
        response = await llm.ainvoke([
            SystemMessage(content=ENHANCEMENT_SYSTEM_PROMPT),
            HumanMessage(content=user_message),
        ])
        enhanced = response.content.strip().strip('"\'')
    except Exception as e:
        # Fallback to original prompt to keep the workflow running.
        logger.warning("enhance_prompt failed, using original prompt: %s", e)
        enhanced = original_prompt
        return Command(update={
            "enhanced_prompt": enhanced,
            "status": "enhanced",
            "messages": [ToolMessage(content=enhanced, tool_call_id=tool_call_id)],
        })
    return Command(update={
        "enhanced_prompt": enhanced,
        "status": "enhanced",
        "messages": [ToolMessage(content=enhanced, tool_call_id=tool_call_id)],
    })
