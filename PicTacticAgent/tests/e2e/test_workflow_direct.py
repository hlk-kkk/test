"""Test the workflow directly without API."""

import asyncio
import os
import sys

import pytest

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../backend/src'))

from dotenv import load_dotenv
load_dotenv()

from pictactic.agents.workflow import create_generation_workflow
from pictactic.agents.state import create_initial_state

# Skip marker for tests that require API keys (GEMINI_API_KEY for generation, OPENAI_API_KEY for LLM)
requires_api = pytest.mark.skipif(
    not os.getenv("GEMINI_API_KEY") or not os.getenv("OPENAI_API_KEY"),
    reason="GEMINI_API_KEY and OPENAI_API_KEY required"
)


@requires_api
async def test_workflow():
    """Test the workflow directly."""
    print("Starting workflow test...")
    print(f"OPENAI_API_KEY: {os.getenv('OPENAI_API_KEY', '')[:20]}...")
    print(f"OPENAI_API_BASE_URL: {os.getenv('OPENAI_API_BASE_URL', '')}")
    print(f"GEMINI_API_KEY: {os.getenv('GEMINI_API_KEY', '')[:20]}...")
    print()

    # Create the workflow
    workflow = create_generation_workflow()

    # Create initial state
    initial_state = create_initial_state(
        user_prompt="A simple blue gradient background",
        template_images=[],
        max_rounds=1,
        images_per_round=1,
        quality_threshold=0.3,
        aspect_ratio="16:9",
        image_size="1K",
        provider="gemini",
    )

    config = {"configurable": {"thread_id": "test-123"}}

    try:
        # Stream events from the workflow
        async for event in workflow.astream(
            initial_state,
            config=config,
            stream_mode="custom",
        ):
            print(f"Event: {type(event)}")
            if isinstance(event, dict):
                event_type = event.get("type")
                if event_type == "image_ready":
                    image = event.get("image", {})
                    print(f"  Image ready: id={image.get('id')}, round={image.get('round_number')}")
                    print(f"  URL: {image.get('url', '')[:100]}...")
            print()

        # Get final state
        final_snapshot = await workflow.aget_state(config)
        final_state = final_snapshot.values if final_snapshot else {}

        print("=" * 50)
        print("Final State:")
        print(f"  status: {final_state.get('status')}")
        print(f"  current_round: {final_state.get('current_round')}")
        print(f"  enhanced_prompt: {(final_state.get('enhanced_prompt') or '')[:100]}...")
        print(f"  generated_images: {len(final_state.get('generated_images', []))}")
        print(f"  best_images: {len(final_state.get('best_images', []))}")

        if final_state.get('best_images'):
            print("\nBest Images:")
            for i, img in enumerate(final_state['best_images'][:3]):
                print(f"  {i+1}. score={img.get('score', 0):.2f}, url={img.get('url', '')[:80]}...")

        if final_state.get('final_result'):
            print(f"\nFinal Result: {final_state['final_result']}")

    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    asyncio.run(test_workflow())
