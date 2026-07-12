"""Tests for LangChain 1.0 and LangGraph 1.0 compatibility."""

import pytest


def test_langchain_imports():
    """Test that LangChain 1.0 imports work correctly."""
    from langchain.chat_models import init_chat_model
    from langchain_core.messages import HumanMessage, SystemMessage

    assert init_chat_model is not None
    assert HumanMessage is not None
    assert SystemMessage is not None


def test_langgraph_imports():
    """Test that LangGraph 1.0 imports work correctly."""
    from langgraph.graph import StateGraph, START, END
    from langgraph.graph.message import add_messages
    from langgraph.checkpoint.memory import MemorySaver

    assert StateGraph is not None
    assert START is not None
    assert END is not None
    assert add_messages is not None
    assert MemorySaver is not None


def test_state_graph_creation():
    """Test that StateGraph can be created with new API."""
    from typing import Annotated, TypedDict

    from langgraph.graph import END, START, StateGraph
    from langgraph.graph.message import add_messages

    class TestState(TypedDict):
        messages: Annotated[list, add_messages]

    builder = StateGraph(TestState)
    builder.add_node("test", lambda s: {"messages": []})
    builder.add_edge(START, "test")
    builder.add_edge("test", END)
    graph = builder.compile()

    assert graph is not None


def test_state_graph_with_context_schema():
    """Test that StateGraph supports context_schema parameter."""
    from dataclasses import dataclass
    from typing import Annotated, TypedDict

    from langgraph.graph import END, START, StateGraph
    from langgraph.graph.message import add_messages

    @dataclass
    class TestContext:
        user_id: str | None = None

    class TestState(TypedDict):
        messages: Annotated[list, add_messages]

    builder = StateGraph(TestState, context_schema=TestContext)
    builder.add_node("test", lambda s: {"messages": []})
    builder.add_edge(START, "test")
    builder.add_edge("test", END)
    graph = builder.compile()

    assert graph is not None


def test_memory_saver():
    """Test that MemorySaver works correctly."""
    from langgraph.checkpoint.memory import MemorySaver

    checkpointer = MemorySaver()
    assert checkpointer is not None


def test_init_chat_model_function():
    """Test that init_chat_model function signature is correct."""
    from langchain.chat_models import init_chat_model
    import inspect

    sig = inspect.signature(init_chat_model)
    params = list(sig.parameters.keys())

    # Should have model as first parameter
    assert "model" in params or params[0] in ["model", "model_name"]


def test_message_types():
    """Test that message types work correctly."""
    from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

    human = HumanMessage(content="Hello")
    system = SystemMessage(content="You are a helpful assistant")
    ai = AIMessage(content="Hi there!")

    assert human.content == "Hello"
    assert system.content == "You are a helpful assistant"
    assert ai.content == "Hi there!"


def test_multimodal_message():
    """Test that multimodal messages work correctly."""
    from langchain_core.messages import HumanMessage

    message = HumanMessage(
        content=[
            {"type": "text", "text": "What's in this image?"},
            {"type": "image_url", "image_url": {"url": "https://example.com/image.png"}},
        ]
    )

    assert isinstance(message.content, list)
    assert len(message.content) == 2
    assert message.content[0]["type"] == "text"
    assert message.content[1]["type"] == "image_url"
