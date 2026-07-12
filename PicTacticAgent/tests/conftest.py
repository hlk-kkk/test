"""Pytest configuration and shared fixtures."""

import os
import sys
import pytest
from pathlib import Path

# Add project root to Python path
project_root = Path(__file__).parent.parent
sys.path.insert(0, str(project_root / "backend" / "src"))

# Set test environment variables
os.environ.setdefault("PICTACTIC_ENV", "test")
os.environ.setdefault("JWT_SECRET_KEY", "test-secret-key-for-unit-tests-only")


@pytest.fixture(scope="session")
def event_loop_policy():
    """Set event loop policy for async tests."""
    import asyncio

    return asyncio.DefaultEventLoopPolicy()


@pytest.fixture
def test_user_data():
    """Provide test user data."""
    return {
        "email": "test@example.com",
        "password": "secure_password_123",
        "display_name": "Test User",
    }


@pytest.fixture
def mock_settings():
    """Provide mock settings for tests."""
    from unittest.mock import MagicMock

    settings = MagicMock()
    settings.jwt_secret_key = "test-secret-key"
    settings.jwt_algorithm = "HS256"
    settings.jwt_access_token_expire_minutes = 30
    settings.jwt_refresh_token_expire_days = 30
    settings.github_client_id = ""
    settings.github_client_secret = ""
    settings.google_client_id = ""
    settings.google_client_secret = ""
    return settings


def pytest_configure(config):
    """Configure pytest."""
    config.addinivalue_line("markers", "asyncio: mark test as async")
    config.addinivalue_line(
        "markers", "requires_server: mark test as requiring running server"
    )


def pytest_collection_modifyitems(config, items):
    """Modify test collection."""
    # Add asyncio marker to all async tests
    for item in items:
        if hasattr(item, "function") and hasattr(item.function, "__code__"):
            if item.function.__code__.co_flags & 0x80:  # CO_COROUTINE flag
                item.add_marker(pytest.mark.asyncio)
