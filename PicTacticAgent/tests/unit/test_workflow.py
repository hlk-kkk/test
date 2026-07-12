"""Tests for the StateGraph workflow."""

import pytest
from pictactic.agents.conditions import check_template, should_continue
from pictactic.agents.state import create_initial_state
from pictactic.agents.workflow import create_generation_workflow


class TestConditions:
    """Tests for condition functions."""

    def test_check_template_with_images(self):
        """Test check_template returns 'analyze' when template images exist."""
        state = create_initial_state(
            user_prompt="test",
            template_images=["img1.jpg", "img2.jpg"]
        )
        result = check_template(state)
        assert result == "analyze"

    def test_check_template_without_images(self):
        """Test check_template returns 'enhance' when no template images."""
        state = create_initial_state(user_prompt="test")
        result = check_template(state)
        assert result == "enhance"

    def test_check_template_empty_list(self):
        """Test check_template returns 'enhance' for empty list."""
        state = create_initial_state(user_prompt="test", template_images=[])
        result = check_template(state)
        assert result == "enhance"

    def test_should_continue_first_round_low_score(self):
        """Test should_continue returns 'continue' on first round with low score."""
        state = create_initial_state(user_prompt="test", max_rounds=3)
        state["current_round"] = 1
        state["generated_images"] = [
            {"round_number": 1, "score": 0.5, "id": "1", "url": "url1", "local_path": None, "review": "", "provider": "gemini"}
        ]
        result = should_continue(state)
        assert result == "continue"

    def test_should_continue_quality_met(self):
        """Test should_continue returns 'finalize' when quality threshold met."""
        state = create_initial_state(user_prompt="test", quality_threshold=0.7)
        state["current_round"] = 1
        state["generated_images"] = [
            {"round_number": 1, "score": 0.85, "id": "1", "url": "url1", "local_path": None, "review": "", "provider": "gemini"}
        ]
        result = should_continue(state)
        assert result == "finalize"

    def test_should_continue_max_rounds_reached(self):
        """Test should_continue returns 'finalize' when max rounds reached."""
        state = create_initial_state(user_prompt="test", max_rounds=3)
        state["current_round"] = 3
        state["generated_images"] = [
            {"round_number": 3, "score": 0.5, "id": "1", "url": "url1", "local_path": None, "review": "", "provider": "gemini"}
        ]
        result = should_continue(state)
        assert result == "finalize"

    def test_should_continue_exactly_at_threshold(self):
        """Test should_continue returns 'finalize' when score equals threshold."""
        state = create_initial_state(user_prompt="test", quality_threshold=0.7)
        state["current_round"] = 1
        state["generated_images"] = [
            {"round_number": 1, "score": 0.7, "id": "1", "url": "url1", "local_path": None, "review": "", "provider": "gemini"}
        ]
        result = should_continue(state)
        assert result == "finalize"


class TestWorkflowCreation:
    """Tests for workflow creation."""

    def test_create_workflow(self):
        """Test that workflow can be created."""
        workflow = create_generation_workflow()
        assert workflow is not None

    def test_workflow_has_expected_nodes(self):
        """Test that workflow has all expected nodes."""
        workflow = create_generation_workflow()
        # The compiled graph should have nodes
        assert hasattr(workflow, "nodes")
        # Check node names exist in the graph
        expected_nodes = ["analyze", "enhance", "generate", "evaluate", "prepare_next", "finalize"]
        for node_name in expected_nodes:
            assert node_name in workflow.nodes, f"Missing node: {node_name}"
