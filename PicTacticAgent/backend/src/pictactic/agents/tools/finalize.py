"""Tool: finalize the workflow by selecting the best images."""

import json
from typing import Annotated

from langchain_core.messages import ToolMessage
from langchain_core.tools import InjectedToolCallId, tool
from langgraph.prebuilt import InjectedState
from langgraph.types import Command

from ..state import GenerationState


@tool
async def finalize(
    best_image_indexes: list[int],
    tool_call_id: Annotated[str, InjectedToolCallId],
    state: Annotated[GenerationState, InjectedState()],
) -> Command:
    """Finalize the workflow by selecting the best images.
    Args:
        best_image_indexes: 1-based indexes of the best images from the current round
    Updates state with best_images and sets status to completed."""
    generated_images = state.get("generated_images", [])
    current_round = state.get("current_round", 1)

    current_round_images = [
        img for img in generated_images if img.get("round_number") == current_round
    ]

    best_images = []
    for idx in best_image_indexes:
        if 1 <= idx <= len(current_round_images):
            best_images.append(current_round_images[idx - 1])

    if not best_images and current_round_images:
        sorted_imgs = sorted(current_round_images, key=lambda x: x.get("score", 0), reverse=True)
        best_images = sorted_imgs[:3]

    final_result = {
        "success": True,
        "best_images": best_images,
        "total_rounds": current_round,
        "total_images": len(generated_images),
    }

    return Command(update={
        "best_images": best_images,
        "status": "completed",
        "final_result": final_result,
        "messages": [ToolMessage(content=json.dumps(final_result, ensure_ascii=False), tool_call_id=tool_call_id)],
    })
