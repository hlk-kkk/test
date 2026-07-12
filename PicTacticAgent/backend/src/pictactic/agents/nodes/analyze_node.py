"""Node: analyze template/reference images."""

import json
import logging

from langchain_core.messages import HumanMessage

from ...core.config import create_vision_llm
from ..state import GenerationState
from ..tools._image_utils import batch_url_to_data_uri

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


async def analyze_node(state: GenerationState) -> dict:
    """Analyze template images to extract style information.

    This node uses a vision LLM to analyze uploaded reference images
    and extract style characteristics (colors, layout, mood, etc.)
    that will be used to enhance the prompt.

    Args:
        state: Current workflow state containing template_images.

    Returns:
        Dict with template_analysis to update state.
    """
    template_images = state.get("template_images", [])

    if not template_images:
        # No template images, return default analysis
        analysis = {
            "style": "default",
            "colors": [],
            "layout": "centered",
            "mood": "neutral",
            "elements": [],
            "suggested_keywords": [],
        }
        return {"template_analysis": analysis}

    llm = create_vision_llm(
        temperature=0.3,
        max_retries=3,
    )

    # Convert URLs to base64 data URIs for the vision LLM
    urls_to_analyze = template_images[:3]  # Limit to 3 images
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
            analysis = {
                "style": "default",
                "colors": [],
                "layout": "centered",
                "mood": "neutral",
                "elements": [],
                "suggested_keywords": [],
            }
    except Exception as e:
        logger.warning("Template analysis failed: %s", e)
        analysis = {
            "style": "default",
            "colors": [],
            "layout": "centered",
            "mood": "neutral",
            "elements": [],
            "suggested_keywords": [str(e)],
        }

    logger.info("Template analysis completed: style=%s", analysis.get("style"))
    return {"template_analysis": analysis}
