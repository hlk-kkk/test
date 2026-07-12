"""Tests for provider settings API routes."""

import pytest
from unittest.mock import MagicMock, patch

from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient

from pictactic.api.dependencies import get_current_user
from pictactic.api.routes.settings import _is_masked, _mask_key, router
from pictactic.services.auth_service import User, UserStatus


# ========== Fixtures ==========


def _create_test_app(mock_user: User | None = None) -> FastAPI:
    """Create a fresh FastAPI app with the settings router and optional auth override."""
    app = FastAPI()
    app.include_router(router, prefix="/api/v1/settings")
    if mock_user is not None:
        app.dependency_overrides[get_current_user] = lambda: mock_user
    return app


def _make_mock_user() -> User:
    return User(
        user_id="user-123",
        email="test@example.com",
        display_name="Test User",
        status=UserStatus.ACTIVE,
    )


def _make_mock_settings() -> MagicMock:
    """Create a MagicMock that mimics the Settings object with provider attributes."""
    s = MagicMock()
    s.default_provider = "gemini"
    s.gemini_api_key = "sk-gemini-abcdefgh1234"
    s.gemini_model = "gemini-2.0-flash-exp"
    s.gemini_llm_model = "gemini-2.0-flash"
    s.jiekouai_api_key = "jk-secret-key-5678"
    s.jiekouai_api_base_url = "https://api.jiekouai.com/v3"
    s.jiekouai_llm_base_url = "https://api.jiekouai.com/openai"
    s.jiekouai_llm_model = "gpt-5-mini"
    s.antigravity_api_key = "ag-key-9012"
    s.antigravity_api_base_url = "https://api.antigravity.com"
    s.antigravity_model = "flux-1"
    s.antigravity_llm_model = "gpt-4o"
    return s


# ========== GET /provider ==========


class TestGetProviderSettings:
    """Tests for GET /api/v1/settings/provider."""

    @pytest.mark.asyncio
    async def test_get_provider_settings_success(self):
        """Verify all fields returned and API keys are masked."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with patch("pictactic.api.routes.settings.settings", mock_settings):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/settings/provider")

        assert response.status_code == 200
        data = response.json()

        # Top-level field
        assert data["default_provider"] == "gemini"

        # Gemini section
        assert data["gemini"]["api_key"] == "****1234"
        assert data["gemini"]["model"] == "gemini-2.0-flash-exp"
        assert data["gemini"]["llm_model"] == "gemini-2.0-flash"

        # Jiekouai section
        assert data["jiekouai"]["api_key"] == "****5678"
        assert data["jiekouai"]["api_base_url"] == "https://api.jiekouai.com/v3"
        assert data["jiekouai"]["llm_base_url"] == "https://api.jiekouai.com/openai"
        assert data["jiekouai"]["llm_model"] == "gpt-5-mini"

        # Antigravity section
        assert data["antigravity"]["api_key"] == "****9012"
        assert data["antigravity"]["api_base_url"] == "https://api.antigravity.com"
        assert data["antigravity"]["model"] == "flux-1"
        assert data["antigravity"]["llm_model"] == "gpt-4o"

    @pytest.mark.asyncio
    async def test_get_provider_settings_unauthenticated(self):
        """Without auth override, returns 401."""
        app = _create_test_app()  # No mock user — dependency not overridden
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.get("/api/v1/settings/provider")

        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_get_provider_settings_masks_short_keys(self):
        """Keys with <= 4 chars return just '****'."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()
        mock_settings.gemini_api_key = "abcd"  # exactly 4 chars
        mock_settings.jiekouai_api_key = "xy"  # 2 chars
        mock_settings.antigravity_api_key = ""  # empty

        with patch("pictactic.api.routes.settings.settings", mock_settings):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/settings/provider")

        data = response.json()
        assert data["gemini"]["api_key"] == "****"
        assert data["jiekouai"]["api_key"] == "****"
        assert data["antigravity"]["api_key"] == "****"

    @pytest.mark.asyncio
    async def test_get_provider_settings_masks_long_keys(self):
        """Longer keys show '****' + last 4 chars."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()
        mock_settings.gemini_api_key = "abcdefghijklmnop"  # 16 chars, last 4 = "mnop"

        with patch("pictactic.api.routes.settings.settings", mock_settings):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.get("/api/v1/settings/provider")

        data = response.json()
        assert data["gemini"]["api_key"] == "****mnop"


# ========== PUT /provider ==========


class TestUpdateProviderSettings:
    """Tests for PUT /api/v1/settings/provider."""

    @pytest.mark.asyncio
    async def test_update_default_provider(self):
        """Update default_provider, verify env file write and in-memory apply."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch("pictactic.api.routes.settings.update_env_file") as mock_env,
            patch("pictactic.api.routes.settings.apply_settings_in_memory") as mock_apply,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={"default_provider": "jiekouai"},
                )

        assert response.status_code == 200
        mock_env.assert_called_once()
        env_arg = mock_env.call_args[0][0]
        assert env_arg["DEFAULT_PROVIDER"] == "jiekouai"

        mock_apply.assert_called_once()
        apply_arg = mock_apply.call_args[0][0]
        assert apply_arg["DEFAULT_PROVIDER"] == "jiekouai"

    @pytest.mark.asyncio
    async def test_update_gemini_api_key(self):
        """Update gemini API key with a real (non-masked) value."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch("pictactic.api.routes.settings.update_env_file") as mock_env,
            patch("pictactic.api.routes.settings.apply_settings_in_memory") as mock_apply,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={"gemini": {"api_key": "sk-new-real-key-12345"}},
                )

        assert response.status_code == 200
        env_arg = mock_env.call_args[0][0]
        assert env_arg["GEMINI_API_KEY"] == "sk-new-real-key-12345"

    @pytest.mark.asyncio
    async def test_skip_masked_api_key(self):
        """Sending a masked value ('****xxxx') as API key should NOT trigger update for that key."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch("pictactic.api.routes.settings.update_env_file") as mock_env,
            patch("pictactic.api.routes.settings.apply_settings_in_memory") as mock_apply,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={
                        "gemini": {"api_key": "****1234", "model": "gemini-pro"},
                    },
                )

        assert response.status_code == 200
        env_arg = mock_env.call_args[0][0]
        # Masked API key should be skipped
        assert "GEMINI_API_KEY" not in env_arg
        # But model should still be updated
        assert env_arg["GEMINI_MODEL"] == "gemini-pro"

    @pytest.mark.asyncio
    async def test_update_multiple_providers(self):
        """Update fields across multiple providers at once."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch("pictactic.api.routes.settings.update_env_file") as mock_env,
            patch("pictactic.api.routes.settings.apply_settings_in_memory") as mock_apply,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={
                        "default_provider": "antigravity",
                        "jiekouai": {"llm_model": "gpt-5"},
                        "antigravity": {
                            "api_key": "new-ag-key",
                            "model": "flux-2",
                        },
                    },
                )

        assert response.status_code == 200
        env_arg = mock_env.call_args[0][0]
        assert env_arg["DEFAULT_PROVIDER"] == "antigravity"
        assert env_arg["JIEKOUAI_LLM_MODEL"] == "gpt-5"
        assert env_arg["ANTIGRAVITY_API_KEY"] == "new-ag-key"
        assert env_arg["ANTIGRAVITY_MODEL"] == "flux-2"

    @pytest.mark.asyncio
    async def test_update_returns_fresh_settings(self):
        """Verify the response reflects updated values (re-reads from settings)."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        def fake_apply(updates):
            """Simulate apply_settings_in_memory by mutating mock_settings."""
            for key, value in updates.items():
                field_name = key.lower()
                setattr(mock_settings, field_name, value)

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch("pictactic.api.routes.settings.update_env_file"),
            patch(
                "pictactic.api.routes.settings.apply_settings_in_memory",
                side_effect=fake_apply,
            ),
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={"default_provider": "jiekouai"},
                )

        assert response.status_code == 200
        data = response.json()
        assert data["default_provider"] == "jiekouai"

    @pytest.mark.asyncio
    async def test_update_unauthenticated(self):
        """Without auth, returns 401."""
        app = _create_test_app()  # No mock user
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.put(
                "/api/v1/settings/provider",
                json={"default_provider": "jiekouai"},
            )

        assert response.status_code == 401

    @pytest.mark.asyncio
    async def test_env_write_failure_still_applies_in_memory(self):
        """When update_env_file raises OSError, apply_settings_in_memory is still called."""
        mock_user = _make_mock_user()
        app = _create_test_app(mock_user)
        mock_settings = _make_mock_settings()

        with (
            patch("pictactic.api.routes.settings.settings", mock_settings),
            patch(
                "pictactic.api.routes.settings.update_env_file",
                side_effect=OSError("disk full"),
            ) as mock_env,
            patch("pictactic.api.routes.settings.apply_settings_in_memory") as mock_apply,
        ):
            transport = ASGITransport(app=app)
            async with AsyncClient(transport=transport, base_url="http://test") as client:
                response = await client.put(
                    "/api/v1/settings/provider",
                    json={"default_provider": "jiekouai"},
                )

        assert response.status_code == 200
        mock_env.assert_called_once()
        # apply_settings_in_memory should be called in the except block
        mock_apply.assert_called_once()
        apply_arg = mock_apply.call_args[0][0]
        assert apply_arg["DEFAULT_PROVIDER"] == "jiekouai"


# ========== Helper functions ==========


class TestMaskHelpers:
    """Tests for _mask_key and _is_masked helper functions."""

    def test_mask_key_empty(self):
        """Empty string returns '****'."""
        assert _mask_key("") == "****"

    def test_mask_key_short(self):
        """3-char key returns '****'."""
        assert _mask_key("abc") == "****"

    def test_mask_key_long(self):
        """Longer key returns '****' + last 4 chars."""
        assert _mask_key("abcdefgh") == "****efgh"
        assert _mask_key("12345") == "****2345"

    def test_is_masked_true(self):
        """'****abcd' is detected as masked."""
        assert _is_masked("****abcd") is True
        assert _is_masked("****") is True

    def test_is_masked_false(self):
        """'sk-xxx' is not masked."""
        assert _is_masked("sk-xxx") is False
        assert _is_masked("real-api-key") is False

    def test_is_masked_none(self):
        """None is not masked."""
        assert _is_masked(None) is False
