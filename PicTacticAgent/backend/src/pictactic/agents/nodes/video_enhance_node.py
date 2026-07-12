"""Node: enhance user prompt for video generation."""

import logging

from langchain_core.messages import HumanMessage, SystemMessage
from langgraph.config import get_stream_writer

from ...core.config import create_vision_llm
from ..video_state import VideoGenerationState

logger = logging.getLogger(__name__)

VIDEO_ENHANCEMENT_SYSTEM_PROMPT = """You are an expert at crafting prompts for AI video generation.

Enhance the user's prompt to produce better video output. Focus on describing:
- Motion and movement (character actions, object trajectories)
- Camera movement (pan, tilt, zoom, dolly, tracking shot)
- Temporal progression (how the scene evolves over time)
- Cinematic qualities (lighting changes, mood transitions, pacing)
- Visual atmosphere (color palette, depth of field, film grain)

Guidelines:
- Keep the user's core intent
- Add specific motion and temporal details
- Include cinematic modifiers (smooth motion, dynamic camera, fluid transitions)
- Describe the scene as a continuous sequence, not a static image
- Keep prompts focused and not overly long (max 150 words)
- Use comma-separated phrases for clarity

DO NOT:
- Add content the user didn't request
- Change the subject matter
- Include explanations in your output
- Use quotation marks around the prompt"""


async def video_enhance_node(state: VideoGenerationState) -> dict:
    """Enhance the user prompt for better video generation.

    This node uses an LLM to transform the user's prompt into
    an optimized version with better motion, camera, and cinematic
    details. Only enhances on the first round; subsequent rounds
    reuse the existing enhanced prompt.

    Args:
        state: Current video workflow state.

    Returns:
        Dict with enhanced_prompt and status to update state.
    """
    writer = get_stream_writer()
    if writer:
        writer(
            {
                "type": "status_change",
                "status": "enhancing",
                "current_round": state.get("current_round", 0),
            }
        )

    # Check if prompt enhancement is disabled
    if not state.get("enhance_prompt_enabled", True):
        user_prompt = state.get("user_prompt", "")
        logger.info("Video prompt enhancement disabled, using original prompt")
        return {
            "enhanced_prompt": user_prompt,
            "status": "enhanced",
        }

    # Skip re-enhancement if we already have an enhanced prompt (subsequent rounds)
    existing_enhanced = state.get("enhanced_prompt")
    if existing_enhanced and state.get("current_round", 0) > 0:
        logger.info("Reusing existing enhanced prompt for round %d", state.get("current_round", 0))
        return {
            "enhanced_prompt": existing_enhanced,
            "status": "enhanced",
        }

    user_prompt = state.get("user_prompt", "")

    llm = create_vision_llm(
        temperature=0.7,
        max_retries=3,
    )

    user_message = f"""Enhance this prompt for AI video generation:

Original prompt: {user_prompt}

Target: high-quality cinematic video with smooth motion and compelling visuals.

Create an enhanced version that will produce better video results.
Output ONLY the enhanced prompt, nothing else."""

    try:
        response = await llm.ainvoke(
            [
                SystemMessage(content=VIDEO_ENHANCEMENT_SYSTEM_PROMPT),
                HumanMessage(content=user_message),
            ]
        )
        # Handle different response formats (str or list of content blocks)
        content = response.content
        if isinstance(content, list):
            content = " ".join(
                block.get("text", "") if isinstance(block, dict) else str(block)
                for block in content
            )
        enhanced = content.strip().strip("\"'")
    except Exception as e:
        # Fallback to original prompt to keep the workflow running
        logger.warning("video_enhance_node failed, using original prompt: %s", e)
        enhanced = user_prompt

    logger.info("Video prompt enhanced: %s...", enhanced[:100] if len(enhanced) > 100 else enhanced)
    return {
        "enhanced_prompt": enhanced,
        "status": "enhanced",
    }
