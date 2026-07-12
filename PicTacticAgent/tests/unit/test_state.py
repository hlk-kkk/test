"""Tests for agent state."""

import pytest
from pictactic.agents.state import GenerationState, create_initial_state


class TestGenerationState:
    """Tests for GenerationState."""

    def test_create_initial_state_minimal(self):
        """Test creating initial state with minimal args."""
        state = create_initial_state(user_prompt="test prompt")

        assert state["user_prompt"] == "test prompt"
        assert state["template_images"] == []
        assert state["max_rounds"] == 3
        assert state["images_per_round"] == 5
        assert state["quality_threshold"] == 0.7
        assert state["current_round"] == 0
        assert state["status"] == "pending"
        assert state["generated_images"] == []
        assert state["best_images"] == []

    def test_create_initial_state_full(self):
        """Test creating initial state with all args."""
        state = create_initial_state(
            user_prompt="detailed prompt",
            template_images=["img1.jpg", "img2.jpg"],
            max_rounds=5,
            images_per_round=8,
            quality_threshold=0.85,
            aspect_ratio="1:1",
            image_size="4K",
            provider="gemini",
        )

        assert state["user_prompt"] == "detailed prompt"
        assert len(state["template_images"]) == 2
        assert state["max_rounds"] == 5
        assert state["images_per_round"] == 8
        assert state["quality_threshold"] == 0.85
        assert state["aspect_ratio"] == "1:1"
        assert state["image_size"] == "4K"
        assert state["provider"] == "gemini"

    def test_initial_state_has_all_keys(self):
        """Test that initial state has all required keys."""
        state = create_initial_state(user_prompt="test")

        required_keys = [
            "user_prompt",
            "template_images",
            "max_rounds",
            "images_per_round",
            "quality_threshold",
            "aspect_ratio",
            "image_size",
            "provider",
            "current_round",
            "status",
            "messages",
            "enhanced_prompt",
            "template_analysis",
            "generated_images",
            "best_images",
            "evaluation_feedback",
            "final_result",
            "error",
            "enhance_prompt_enabled",
        ]

        for key in required_keys:
            assert key in state, f"Missing key: {key}"
