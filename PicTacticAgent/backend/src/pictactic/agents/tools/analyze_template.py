"""Tool: analyze template/reference images."""

import json
import logging
from typing import Annotated

from langchain_core.messages import HumanMessage, ToolMessage
from langchain_core.tools import InjectedToolCallId, tool
from langgraph.prebuilt import InjectedState
from langgraph.types import Command

from ...core.config import create_vision_llm
from ..state import GenerationState
from ._image_utils import batch_url_to_data_uri

logger = logging.getLogger(__name__)

ANALYSIS_PROMPT = """Analyze this image for creating similar images.

Extract the following information:
1. **style**: The overall visual style (e.g., minimalist, vibrant, dark, artistic, realistic, abstract)
2. **colors**: List 3-5 dominant colors (use descriptive names like "deep blue", "bright orange")
3. **layout**: Describe the composition (e.g., centered, asymmetric, rule-of-thirds, full-bleed)
4. **mood**: The emotional atmosphere (e.g., energetic, calm, mysterious, playful, dramatic)
5. **elements**: Key visual elements present (e.g., gradients, textures, patterns, shapes, subjects)
6. **suggested_keywords**: Keywords useful for recreating this style in AI image generation

Respond ONLY with valid JSON in this format:
{
    "style": "...",
    "colors": ["...", "...", "..."],
    "layout": "...",
    "mood": "...",
    "elements": ["...", "..."],
    "suggested_keywords": ["...", "...", "..."]
}"""


@tool
async def analyze_template(
    tool_call_id: Annotated[str, InjectedToolCallId],
    state: Annotated[GenerationState, InjectedState()],
    image_urls: list[str] | None = None,
) -> Command:
    """Analyze template/reference images to extract style information (colors, layout, mood).
    Call this when template images are provided and you need to understand their style.
    Args:
        image_urls: List of image URLs to analyze
    Returns a Command that updates state with template_analysis."""
    resolved_images = _resolve_image_urls(image_urls, state)

    if not resolved_images:
        analysis = {
            "style": "default",
            "colors": [],
            "layout": "centered",
            "mood": "neutral",
            "elements": [],
            "suggested_keywords": [],
        }
        return Command(update={
            "template_analysis": analysis,
            "messages": [ToolMessage(content=json.dumps(analysis), tool_call_id=tool_call_id)],
        })

    llm = create_vision_llm(
        temperature=0.3,
        max_retries=3,
    )

    # Convert URLs to base64 data URIs for the vision LLM
    urls_to_analyze = resolved_images[:3]
    try:
        data_uris = await batch_url_to_data_uri(urls_to_analyze)
    except Exception as e:
        logger.warning("Failed to download template images: %s", e)
        data_uris = urls_to_analyze  # fallback to original URLs

    message_content: list[dict] = [{"type": "text", "text": ANALYSIS_PROMPT}]
    for uri in data_uris:
        message_content.append({"type": "image_url", "image_url": {"url": uri}})

    try:
        response = await llm.ainvoke([HumanMessage(content=message_content)])
        text = response.content
        json_start = text.find("{")
        json_end = text.rfind("}") + 1
        if json_start != -1 and json_end > 0:
            analysis = json.loads(text[json_start:json_end])
        else:
            analysis = {"style": "default", "colors": [], "layout": "centered",
                        "mood": "neutral", "elements": [], "suggested_keywords": []}
    except Exception as e:
        logger.warning("Template analysis failed: %s", e)
        analysis = {"style": "default", "colors": [], "layout": "centered",
                    "mood": "neutral", "elements": [], "suggested_keywords": [str(e)]}

    return Command(update={
        "template_analysis": analysis,
        "messages": [ToolMessage(content=json.dumps(analysis), tool_call_id=tool_call_id)],
    })


def _resolve_image_urls(image_urls: list[str] | None, state: GenerationState) -> list[str]:
    if isinstance(image_urls, str):
        image_urls = [image_urls]

    images = [u for u in (image_urls or []) if isinstance(u, str) and u.strip()]

    if images and any(_is_state_reference(u) for u in images):
        images = []

    if not images:
        images = [u for u in state.get("template_images", []) if isinstance(u, str) and u.strip()]

    return images


def _is_state_reference(value: str) -> bool:
    v = value.strip()
    return (
        v.startswith("state://")
        or v.startswith("state[")
        or v.startswith("state.")
        or v.startswith("state['")
        or v.startswith('state["')
    )
