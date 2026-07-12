"""Node: enhance user prompt for image generation."""

import json
import logging

from langchain_core.messages import HumanMessage, SystemMessage
from langgraph.config import get_stream_writer

from ...core.config import create_vision_llm
from ..state import GenerationState

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


async def enhance_node(state: GenerationState) -> dict:
    """Enhance the user prompt for better image generation.

    This node uses an LLM to transform the user's prompt into
    an optimized version with better visual details and style
    guidance. It considers:
    - Template analysis (if available)
    - Previous evaluation feedback (if in iteration mode)

    Args:
        state: Current workflow state.

    Returns:
        Dict with enhanced_prompt and status to update state.
    """
    writer = get_stream_writer()
    if writer:
        writer({
            "type": "status_change",
            "status": "enhancing",
            "current_round": state.get("current_round", 0),
        })

    # Check if prompt enhancement is disabled
    if not state.get("enhance_prompt_enabled", True):
        user_prompt = state.get("user_prompt", "")
        logger.info("Prompt enhancement disabled, using original prompt")
        return {
            "enhanced_prompt": user_prompt,
            "status": "enhanced",
        }

    user_prompt = state.get("user_prompt", "")
    feedback = state.get("evaluation_feedback", "")
    template_analysis = state.get("template_analysis")

    llm = create_vision_llm(
        temperature=0.7,
        max_retries=3,
    )

    if feedback:
        # Iterative enhancement based on feedback
        current_prompt = state.get("enhanced_prompt") or user_prompt
        user_message = f"""Improve this image generation prompt based on feedback:

Current prompt: {current_prompt}

Feedback from evaluation: {feedback}

Create an improved version that addresses the feedback while maintaining the original intent.
Output ONLY the improved prompt, nothing else."""
    else:
        # Initial enhancement
        style_context = ""
        if template_analysis:
            style_context = f"\n\nReference style information:\n{json.dumps(template_analysis, ensure_ascii=False)}"

        user_message = f"""Enhance this prompt for AI image generation:

Original prompt: {user_prompt}

Target: high-quality image
{style_context}

Create an enhanced version that will produce better results.
Output ONLY the enhanced prompt, nothing else."""

    try:
        response = await llm.ainvoke([
            SystemMessage(content=ENHANCEMENT_SYSTEM_PROMPT),
            HumanMessage(content=user_message),
        ])
        # Handle different response formats (str or list of content blocks)
        content = response.content
        if isinstance(content, list):
            # Extract text from content blocks
            content = " ".join(
                block.get("text", "") if isinstance(block, dict) else str(block)
                for block in content
            )
        enhanced = content.strip().strip('"\'')
    except Exception as e:
        # Fallback to current prompt to keep the workflow running
        logger.warning("enhance_node failed, using original prompt: %s", e)
        enhanced = state.get("enhanced_prompt") or user_prompt

    logger.info("Prompt enhanced: %s...", enhanced[:100] if len(enhanced) > 100 else enhanced)
    return {
        "enhanced_prompt": enhanced,
        "status": "enhanced",
    }
