"""StateGraph workflow for image generation.

This module defines a deterministic workflow that replaces the ReAct agent,
ensuring strict adherence to user configuration (rounds, images per round).

Workflow:
    START
      │
      ▼
    [check_template] ─── has template ──→ [analyze_node]
      │                                        │
      │ no template                            │
      ▼                                        ▼
    [enhance_node] ◄───────────────────────────┘
      │
      ▼
    [generate_node] ←─────────────────────────┐
      │                                       │
      ▼                                       │
    [evaluate_node]                           │
      │                                       │
      ▼                                       │
    [should_continue]                         │
      │         │                             │
      │ continue│ finalize                    │
      ▼         ▼                             │
    [prepare_next] → [enhance_node] ──────────┘
                │
                ▼
          [finalize_node]
                │
                ▼
               END
"""

import logging

from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, START, StateGraph

from .conditions import check_template, should_continue
from .nodes import (
    analyze_node,
    enhance_node,
    evaluate_node,
    finalize_node,
    generate_node,
    prepare_next_node,
)
from .state import GenerationState

logger = logging.getLogger(__name__)


def create_generation_workflow(checkpointer=None):
    """Create and return the generation workflow graph.

    Args:
        checkpointer: LangGraph checkpointer for state persistence.
                     If None, uses in-memory storage.

    Returns:
        Compiled StateGraph ready for execution.
    """
    builder = StateGraph(GenerationState)

    # Add all nodes
    builder.add_node("analyze", analyze_node)
    builder.add_node("enhance", enhance_node)
    builder.add_node("generate", generate_node)
    builder.add_node("evaluate", evaluate_node)
    builder.add_node("prepare_next", prepare_next_node)
    builder.add_node("finalize", finalize_node)

    # Entry: check if template images exist
    builder.add_conditional_edges(
        START,
        check_template,
        {
            "analyze": "analyze",
            "enhance": "enhance",
        },
    )

    # analyze -> enhance
    builder.add_edge("analyze", "enhance")

    # enhance -> generate
    builder.add_edge("enhance", "generate")

    # generate -> evaluate
    builder.add_edge("generate", "evaluate")

    # evaluate -> should_continue (conditional)
    builder.add_conditional_edges(
        "evaluate",
        should_continue,
        {
            "continue": "prepare_next",
            "finalize": "finalize",
        },
    )

    # prepare_next -> enhance (for iteration)
    builder.add_edge("prepare_next", "enhance")

    # finalize -> END
    builder.add_edge("finalize", END)

    # Compile with checkpointer
    workflow = builder.compile(checkpointer=checkpointer or MemorySaver())

    logger.info("Generation workflow compiled successfully")
    return workflow


# Convenience function to create workflow with default checkpointer
def create_generation_agent(checkpointer=None):
    """Alias for create_generation_workflow for backward compatibility.

    This function exists to maintain API compatibility with code that
    imports create_generation_agent from the agents module.
    """
    return create_generation_workflow(checkpointer)
