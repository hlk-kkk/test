"""Node: evaluate and rank generated images."""

import json
import logging
from copy import deepcopy

from langchain_core.messages import HumanMessage
from langgraph.config import get_stream_writer

from ...core.config import create_vision_llm
from ..state import GenerationState
from ..tools._image_utils import batch_url_to_data_uri

logger = logging.getLogger(__name__)

COMPARATIVE_PROMPT_TEMPLATE = """你是图片评审专家。以下是 {n} 张候选图片。
用户需求：{prompt}
{reference_note}
请对比所有图片，从最好到最差排序。
评估标准：
1. 与用户需求的匹配度
2. 视觉吸引力和美感
3. 构图和色彩协调性
4. 整体质量和完成度

输出 JSON：
{{
    "rankings": [按从好到差排序的图片序号],
    "reviews": [{{"index": 序号, "brief": "简短评价"}}],
    "best_reason": "最佳图片胜出原因",
    "improvement_suggestions": "对下一轮生成的改进建议（如需继续生成）"
}}"""


async def evaluate_node(state: GenerationState) -> dict:
    """Evaluate and rank images from the current round.

    This node uses a vision LLM to:
    1. Compare all images from the current round
    2. Rank them from best to worst
    3. Assign scores based on ranking
    4. Generate improvement suggestions for potential next round

    Args:
        state: Current workflow state.

    Returns:
        Dict with updated generated_images (with scores) and evaluation_feedback.
    """
    # Push reviewing status to frontend
    writer = get_stream_writer()
    if writer:
        writer({
            "type": "status_change",
            "status": "reviewing",
            "current_round": state.get("current_round", 1),
        })

    generated_images = deepcopy(state.get("generated_images", []))
    current_round = state.get("current_round", 1)
    user_prompt = state.get("user_prompt", "")

    # Get images from current round
    current_round_images = [
        img for img in generated_images
        if img.get("round_number") == current_round
    ]

    n = len(current_round_images)
    if n == 0:
        logger.warning("No images to evaluate in round %d", current_round)
        return {
            "status": "reviewed",
            "evaluation_feedback": "本轮没有生成图片，无法评估",
        }

    # Check if AI scoring is enabled
    enable_ai_scoring = state.get("enable_ai_scoring", True)
    if not enable_ai_scoring:
        logger.info("AI scoring disabled, using default scores")
        # Use default scores (decreasing by index)
        for i, img in enumerate(current_round_images):
            img["score"] = max(0.95 - i * 0.05, 0.10)
            img["review"] = "Auto scored"

        return {
            "generated_images": generated_images,
            "evaluation_feedback": "",
            "status": "reviewed",
        }

    image_urls = [img["url"] for img in current_round_images]

    # Convert to data URIs for vision LLM
    template_images = state.get("template_images", [])
    all_urls = list(image_urls)
    if template_images:
        all_urls.extend(template_images[:1])  # Add first reference image if available

    try:
        data_uris = await batch_url_to_data_uri(all_urls)
        image_data_uris = data_uris[:n]
        ref_data_uri = data_uris[n] if template_images else ""
        logger.info("Converted %d images to base64 for evaluation", len(image_data_uris))
    except Exception as e:
        logger.warning("Failed to download images for evaluation: %s", e)
        image_data_uris = image_urls
        ref_data_uri = template_images[0] if template_images else ""

    # Build reference note
    reference_note = ""
    if template_images:
        reference_note = "参考照片已附上，请考虑风格一致性。"

    prompt_text = COMPARATIVE_PROMPT_TEMPLATE.format(
        n=n,
        prompt=user_prompt,
        reference_note=reference_note,
    )

    # Build message content with images
    content: list[dict] = [{"type": "text", "text": prompt_text}]
    if ref_data_uri:
        content.append({"type": "text", "text": "参考照片："})
        content.append({"type": "image_url", "image_url": {"url": ref_data_uri}})
    for i, uri in enumerate(image_data_uris, 1):
        content.append({"type": "text", "text": f"候选图 {i}："})
        content.append({"type": "image_url", "image_url": {"url": uri}})

    llm = create_vision_llm(
        temperature=0.2,
        max_tokens=2048,
        max_retries=3,
    )

    try:
        response = await llm.ainvoke([HumanMessage(content=content)])
        # Handle different response formats (str or list of content blocks)
        response_text = response.content
        if isinstance(response_text, list):
            response_text = " ".join(
                block.get("text", "") if isinstance(block, dict) else str(block)
                for block in response_text
            )
        data = _parse_json(response_text, n)
        logger.info("Evaluation complete: rankings=%s", data.get("rankings"))
    except Exception as e:
        logger.warning("Evaluation LLM call failed: %s", e)
        data = {
            "rankings": list(range(1, n + 1)),
            "reviews": [],
            "best_reason": f"Evaluation failed: {e}",
            "improvement_suggestions": "",
        }

    # Assign scores based on ranking
    score_map: dict[int, float] = {}
    review_map: dict[int, str] = {}
    for rank_pos, img_index in enumerate(data.get("rankings", [])):
        # Top ranked gets 0.95, decreasing by 0.05 per rank
        score_map[img_index] = max(0.95 - rank_pos * 0.05, 0.10)
    for r in data.get("reviews", []):
        review_map[r.get("index", 0)] = r.get("brief", "")

    # Update current round images with scores
    for i, img in enumerate(current_round_images):
        img_index = i + 1
        img["score"] = score_map.get(img_index, 0.5)
        img["review"] = review_map.get(img_index, "")

    # Extract improvement suggestions as feedback for next round
    improvement_suggestions = data.get("improvement_suggestions", "")
    best_reason = data.get("best_reason", "")
    evaluation_feedback = f"{best_reason}\n改进建议: {improvement_suggestions}" if improvement_suggestions else best_reason

    logger.info(
        "Evaluation scored %d images, best score: %.2f",
        len(current_round_images),
        max((img.get("score", 0) for img in current_round_images), default=0)
    )

    return {
        "generated_images": generated_images,
        "evaluation_feedback": evaluation_feedback,
        "status": "reviewed",
    }


def _parse_json(text: str, n: int) -> dict:
    """Extract JSON from LLM response."""
    try:
        start = text.find("{")
        end = text.rfind("}") + 1
        if start != -1 and end > 0:
            return json.loads(text[start:end])
    except (json.JSONDecodeError, ValueError):
        pass
    return {
        "rankings": list(range(1, n + 1)),
        "reviews": [],
        "best_reason": "",
        "improvement_suggestions": "",
    }
