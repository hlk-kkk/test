"""Tool: evaluate and rank generated images."""

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

COMPARATIVE_PROMPT_TEMPLATE = """你是图片评审专家。以下是 {n} 张候选图片。
用户需求：{prompt}
{reference_note}
请对比所有图片，从最好到最差排序。
输出 JSON：{{"rankings": [序号], "reviews": [{{"index": 1, "brief": "简评"}}], "best_reason": "..."}}"""


@tool
async def evaluate_and_rank(
    original_prompt: str,
    tool_call_id: Annotated[str, InjectedToolCallId],
    state: Annotated[GenerationState, InjectedState()],
    reference_image_url: str = "",
) -> Command:
    """Evaluate and rank generated images from the current round using comparative assessment.
    Image URLs are read from state automatically — do NOT pass image_urls.
    Args:
        original_prompt: The user's original prompt
        reference_image_url: Optional reference image URL for comparison
    Returns a Command that updates state with scored images."""
    # Read image URLs from state for the current round
    generated_images = list(state.get("generated_images", []))
    current_round = state.get("current_round", 1)
    current_round_images = [
        img for img in generated_images if img.get("round_number") == current_round
    ]
    image_urls = [img["url"] for img in current_round_images]

    n = len(image_urls)
    if n == 0:
        return Command(update={
            "status": "reviewed",
            "messages": [ToolMessage(content="No images to evaluate.", tool_call_id=tool_call_id)],
        })

    # Convert image URLs to base64 data URIs for the vision LLM
    all_urls = list(image_urls)
    if reference_image_url:
        all_urls.append(reference_image_url)

    try:
        data_uris = await batch_url_to_data_uri(all_urls)
        image_data_uris = data_uris[:n]
        ref_data_uri = data_uris[n] if reference_image_url else ""
        logger.info("Successfully converted %d images to base64 data URIs", len(image_data_uris))
    except Exception as e:
        logger.warning("Failed to download images for evaluation: %s (type=%s)", e, type(e).__name__)
        # Fallback: use original URLs (may still fail at LLM side)
        image_data_uris = image_urls
        ref_data_uri = reference_image_url

    reference_note = ""
    if reference_image_url:
        reference_note = "参考照片已附上，重点关注人像一致性。"

    prompt_text = COMPARATIVE_PROMPT_TEMPLATE.format(
        n=n, prompt=original_prompt, reference_note=reference_note,
    )

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
        data = _parse_json(response.content, n)
        logger.info("Evaluation LLM returned rankings: %s", data.get("rankings"))
    except Exception as e:
        logger.warning("Evaluation LLM call failed: %s (type=%s)", e, type(e).__name__)
        data = {"rankings": list(range(1, n + 1)), "reviews": [], "best_reason": f"Evaluation failed: {e}"}

    # Assign scores based on ranking
    score_map: dict[int, float] = {}
    review_map: dict[int, str] = {}
    for rank_pos, img_index in enumerate(data.get("rankings", [])):
        score_map[img_index] = max(0.95 - rank_pos * 0.05, 0.10)
    for r in data.get("reviews", []):
        review_map[r.get("index", 0)] = r.get("brief", "")

    # Update current round images with scores
    for i, img in enumerate(current_round_images):
        img_index = i + 1
        img["score"] = score_map.get(img_index, 0.5)
        img["review"] = review_map.get(img_index, "")

    # Calculate quality check info to help LLM decide whether to continue
    quality_threshold = state.get("quality_threshold", 0.7)
    max_rounds = state.get("max_rounds", 3)
    best_score = max((img.get("score", 0) for img in current_round_images), default=0)

    quality_met = best_score >= quality_threshold
    rounds_exhausted = current_round >= max_rounds

    quality_check = {
        "current_round": current_round,
        "max_rounds": max_rounds,
        "best_score": round(best_score, 2),
        "quality_threshold": quality_threshold,
        "quality_met": quality_met,
        "should_continue": not quality_met and not rounds_exhausted,
        "suggestion": "可以调用 finalize 完成" if quality_met or rounds_exhausted else "继续优化提示词并生成下一轮",
    }

    # Combine evaluation data with quality check
    result_data = {**data, "quality_check": quality_check}
    summary = json.dumps(result_data, ensure_ascii=False)

    return Command(update={
        "generated_images": generated_images,
        "status": "reviewed",
        "messages": [ToolMessage(content=summary, tool_call_id=tool_call_id)],
    })


def _parse_json(text: str, n: int) -> dict:
    """Extract JSON from LLM response."""
    try:
        start = text.find("{")
        end = text.rfind("}") + 1
        if start != -1 and end > 0:
            return json.loads(text[start:end])
    except (json.JSONDecodeError, ValueError):
        pass
    return {"rankings": list(range(1, n + 1)), "reviews": [], "best_reason": ""}
