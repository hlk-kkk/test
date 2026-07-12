"""Tests for authentication API routes."""

import pytest
from datetime import datetime, timezone
from unittest.mock import AsyncMock, patch, MagicMock
from fastapi import FastAPI, HTTPException
from fastapi.testclient import TestClient
from httpx import AsyncClient, ASGITransport

from pictactic.api.routes.auth import router
from pictactic.services.auth_service import (
    User,
    UserStatus,
    TokenPair,
)


# Create a test FastAPI app with only the auth router
def create_test_app():
    """Create test FastAPI application."""
    app = FastAPI()
    app.include_router(router, prefix="/api/v1/auth")
    return app


app = create_test_app()


class TestRegisterEndpoint:
    """Tests for POST /register endpoint."""

    @pytest.mark.asyncio
    async def test_register_success(self):
        """Test successful registration."""
        mock_user = User(
            user_id="new-user-id",
            email="new@example.com",
            display_name="New User",
            status=UserStatus.PENDING,
        )
        mock_tokens = TokenPair(
            access_token="access_token_123",
            refresh_token="refresh_token_456",
            expires_in=1800,
        )

        with patch(
            "pictactic.api.routes.auth.auth_service.register", new_callable=AsyncMock
        ) as mock_register:
            mock_register.return_value = (mock_user, mock_tokens)

            with patch(
                "pictactic.api.routes.auth.auth_service.create_verification_token",
                new_callable=AsyncMock,
            ) as mock_create_token:
                mock_create_token.return_value = "verification_token"

                with patch(
                    "pictactic.api.routes.auth.email_service.send_verification_email",
                    new_callable=AsyncMock,
                ):
                    transport = ASGITransport(app=app)
                    async with AsyncClient(
                        transport=transport, base_url="http://test"
                    ) as client:
                        response = await client.post(
                            "/api/v1/auth/register",
                            json={
                                "email": "new@example.com",
                                "password": "secure123",
                                "display_name": "New User",
                            },
                        )

                        assert response.status_code == 200
                        data = response.json()
                        assert data["access_token"] == "access_token_123"
                        assert data["refresh_token"] == "refresh_token_456"
                        assert data["user"]["email"] == "new@example.com"

    @pytest.mark.asyncio
    async def test_register_email_exists(self):
        """Test registration with existing email."""
        with patch(
            "pictactic.api.routes.auth.auth_service.register", new_callable=AsyncMock
        ) as mock_register:
            mock_register.side_effect = ValueError("邮箱已被注册")

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/register",
                    json={
                        "email": "existing@example.com",
                        "password": "secure123",
                    },
                )

                assert response.status_code == 400
                assert "邮箱已被注册" in response.json()["detail"]

    @pytest.mark.asyncio
    async def test_register_invalid_email(self):
        """Test registration with invalid email format."""
        transport = ASGITransport(app=app)
        async with AsyncClient(transport=transport, base_url="http://test") as client:
            response = await client.post(
                "/api/v1/auth/register",
                json={
                    "email": "invalid-email",
                    "password": "secure123",
                },
            )

            assert response.status_code == 422  # Validation error


class TestLoginEndpoint:
    """Tests for POST /login endpoint."""

    @pytest.mark.asyncio
    async def test_login_success(self):
        """Test successful login."""
        mock_user = User(
            user_id="user-123",
            email="test@example.com",
            display_name="Test User",
            status=UserStatus.ACTIVE,
            email_verified_at=datetime.now(timezone.utc),
        )
        mock_tokens = TokenPair(
            access_token="access_token_123",
            refresh_token="refresh_token_456",
            expires_in=1800,
        )

        with patch(
            "pictactic.api.routes.auth.auth_service.login", new_callable=AsyncMock
        ) as mock_login:
            mock_login.return_value = (mock_user, mock_tokens)

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/login",
                    json={
                        "email": "test@example.com",
                        "password": "correct_password",
                    },
                )

                assert response.status_code == 200
                data = response.json()
                assert data["access_token"] == "access_token_123"
                assert data["user"]["email"] == "test@example.com"
                assert data["user"]["email_verified"] is True

    @pytest.mark.asyncio
    async def test_login_wrong_credentials(self):
        """Test login with wrong credentials."""
        with patch(
            "pictactic.api.routes.auth.auth_service.login", new_callable=AsyncMock
        ) as mock_login:
            mock_login.side_effect = ValueError("邮箱或密码错误")

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/login",
                    json={
                        "email": "test@example.com",
                        "password": "wrong_password",
                    },
                )

                assert response.status_code == 401
                assert "邮箱或密码错误" in response.json()["detail"]


class TestLogoutEndpoint:
    """Tests for POST /logout endpoint."""

    @pytest.mark.asyncio
    async def test_logout_success(self):
        """Test successful logout."""
        with patch(
            "pictactic.api.routes.auth.auth_service.logout", new_callable=AsyncMock
        ) as mock_logout:
            mock_logout.return_value = True

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/logout",
                    json={"refresh_token": "some_refresh_token"},
                )

                assert response.status_code == 200
                assert response.json()["success"] is True


class TestRefreshEndpoint:
    """Tests for POST /refresh endpoint."""

    @pytest.mark.asyncio
    async def test_refresh_success(self):
        """Test successful token refresh."""
        mock_user = User(
            user_id="user-123",
            email="test@example.com",
            status=UserStatus.ACTIVE,
        )
        mock_tokens = TokenPair(
            access_token="new_access_token",
            refresh_token="new_refresh_token",
            expires_in=1800,
        )

        with patch(
            "pictactic.api.routes.auth.auth_service.refresh_tokens",
            new_callable=AsyncMock,
        ) as mock_refresh:
            mock_refresh.return_value = mock_tokens

            with patch(
                "pictactic.api.routes.auth.auth_service.verify_access_token"
            ) as mock_verify:
                mock_verify.return_value = "user-123"

                with patch(
                    "pictactic.api.routes.auth.auth_service.get_user_by_id",
                    new_callable=AsyncMock,
                ) as mock_get_user:
                    mock_get_user.return_value = mock_user

                    transport = ASGITransport(app=app)
                    async with AsyncClient(
                        transport=transport, base_url="http://test"
                    ) as client:
                        response = await client.post(
                            "/api/v1/auth/refresh",
                            json={"refresh_token": "old_refresh_token"},
                        )

                        assert response.status_code == 200
                        data = response.json()
                        assert data["access_token"] == "new_access_token"
                        assert data["refresh_token"] == "new_refresh_token"

    @pytest.mark.asyncio
    async def test_refresh_invalid_token(self):
        """Test token refresh with invalid token."""
        with patch(
            "pictactic.api.routes.auth.auth_service.refresh_tokens",
            new_callable=AsyncMock,
        ) as mock_refresh:
            mock_refresh.return_value = None

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/refresh",
                    json={"refresh_token": "invalid_token"},
                )

                assert response.status_code == 401


class TestMeEndpoint:
    """Tests for GET /me endpoint."""

    @pytest.mark.asyncio
    async def test_get_me_authenticated(self):
        """Test getting current user when authenticated."""
        mock_user = User(
            user_id="user-123",
            email="test@example.com",
            display_name="Test User",
            status=UserStatus.ACTIVE,
            email_verified_at=datetime.now(timezone.utc),
        )

        with patch(
            "pictactic.api.routes.auth.get_current_user", return_value=mock_user
        ):
            # Override dependency
            app.dependency_overrides[
                __import__(
                    "pictactic.api.routes.auth", fromlist=["get_current_user"]
                ).get_current_user
            ] = lambda: mock_user

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.get(
                    "/api/v1/auth/me",
                    headers={"Authorization": "Bearer valid_token"},
                )

            # Clear override
            app.dependency_overrides.clear()


class TestOAuthEndpoints:
    """Tests for OAuth endpoints."""

    @pytest.mark.asyncio
    async def test_get_oauth_providers(self):
        """Test getting available OAuth providers."""
        with patch(
            "pictactic.services.oauth_client.github_oauth"
        ) as mock_github:
            mock_github.is_configured.return_value = True

            with patch(
                "pictactic.services.oauth_client.google_oauth"
            ) as mock_google:
                mock_google.is_configured.return_value = False

                transport = ASGITransport(app=app)
                async with AsyncClient(
                    transport=transport, base_url="http://test"
                ) as client:
                    response = await client.get("/api/v1/auth/oauth/providers")

                    assert response.status_code == 200
                    data = response.json()
                    assert "github" in data
                    assert "google" in data

    @pytest.mark.asyncio
    async def test_oauth_authorize(self):
        """Test getting OAuth authorization URL."""
        mock_client = MagicMock()
        mock_client.get_authorize_url.return_value = (
            "https://github.com/login/oauth/authorize?...",
            "state_123",
        )

        with patch(
            "pictactic.api.routes.auth.get_oauth_client", return_value=mock_client
        ):
            with patch(
                "pictactic.api.routes.auth.auth_service.save_oauth_state",
                new_callable=AsyncMock,
            ):
                transport = ASGITransport(app=app)
                async with AsyncClient(
                    transport=transport, base_url="http://test"
                ) as client:
                    response = await client.get("/api/v1/auth/oauth/github/authorize")

                    assert response.status_code == 200
                    data = response.json()
                    assert "authorize_url" in data
                    assert "state" in data


class TestEmailVerificationEndpoints:
    """Tests for email verification endpoints."""

    @pytest.mark.asyncio
    async def test_verify_email_success(self):
        """Test successful email verification."""
        mock_user = User(
            user_id="user-123",
            email="test@example.com",
            status=UserStatus.ACTIVE,
        )

        with patch(
            "pictactic.api.routes.auth.auth_service.verify_token",
            new_callable=AsyncMock,
        ) as mock_verify:
            mock_verify.return_value = "user-123"

            with patch(
                "pictactic.api.routes.auth.auth_service.update_user",
                new_callable=AsyncMock,
            ) as mock_update:
                mock_update.return_value = mock_user

                with patch(
                    "pictactic.api.routes.auth.email_service.send_welcome_email",
                    new_callable=AsyncMock,
                ):
                    transport = ASGITransport(app=app)
                    async with AsyncClient(
                        transport=transport, base_url="http://test"
                    ) as client:
                        response = await client.post(
                            "/api/v1/auth/verify-email",
                            json={"token": "valid_token"},
                        )

                        assert response.status_code == 200
                        assert response.json()["success"] is True

    @pytest.mark.asyncio
    async def test_verify_email_invalid_token(self):
        """Test email verification with invalid token."""
        with patch(
            "pictactic.api.routes.auth.auth_service.verify_token",
            new_callable=AsyncMock,
        ) as mock_verify:
            mock_verify.return_value = None

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/verify-email",
                    json={"token": "invalid_token"},
                )

                assert response.status_code == 400


class TestPasswordResetEndpoints:
    """Tests for password reset endpoints."""

    @pytest.mark.asyncio
    async def test_forgot_password_user_exists(self):
        """Test forgot password with existing user."""
        user_data = {
            "user_id": "user-123",
            "email": "test@example.com",
            "password_hash": "some_hash",
            "display_name": "Test User",
        }

        with patch(
            "pictactic.api.routes.auth.auth_service.get_user_by_email",
            new_callable=AsyncMock,
        ) as mock_get:
            mock_get.return_value = user_data

            with patch(
                "pictactic.api.routes.auth.auth_service.create_verification_token",
                new_callable=AsyncMock,
            ) as mock_create_token:
                mock_create_token.return_value = "reset_token"

                with patch(
                    "pictactic.api.routes.auth.email_service.send_password_reset_email",
                    new_callable=AsyncMock,
                ):
                    transport = ASGITransport(app=app)
                    async with AsyncClient(
                        transport=transport, base_url="http://test"
                    ) as client:
                        response = await client.post(
                            "/api/v1/auth/forgot-password",
                            json={"email": "test@example.com"},
                        )

                        assert response.status_code == 200
                        assert response.json()["success"] is True

    @pytest.mark.asyncio
    async def test_forgot_password_user_not_exists(self):
        """Test forgot password with non-existent user (should still return success)."""
        with patch(
            "pictactic.api.routes.auth.auth_service.get_user_by_email",
            new_callable=AsyncMock,
        ) as mock_get:
            mock_get.return_value = None

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/forgot-password",
                    json={"email": "nonexistent@example.com"},
                )

                # Should still return success to prevent email enumeration
                assert response.status_code == 200
                assert response.json()["success"] is True

    @pytest.mark.asyncio
    async def test_reset_password_success(self):
        """Test successful password reset."""
        with patch(
            "pictactic.api.routes.auth.auth_service.verify_token",
            new_callable=AsyncMock,
        ) as mock_verify:
            mock_verify.return_value = "user-123"

            with patch(
                "pictactic.api.routes.auth.auth_service.update_password",
                new_callable=AsyncMock,
            ) as mock_update:
                mock_update.return_value = True

                with patch(
                    "pictactic.api.routes.auth.auth_service.revoke_all_user_tokens",
                    new_callable=AsyncMock,
                ):
                    transport = ASGITransport(app=app)
                    async with AsyncClient(
                        transport=transport, base_url="http://test"
                    ) as client:
                        response = await client.post(
                            "/api/v1/auth/reset-password",
                            json={
                                "token": "valid_reset_token",
                                "new_password": "new_secure_password",
                            },
                        )

                        assert response.status_code == 200
                        assert response.json()["success"] is True

    @pytest.mark.asyncio
    async def test_reset_password_invalid_token(self):
        """Test password reset with invalid token."""
        with patch(
            "pictactic.api.routes.auth.auth_service.verify_token",
            new_callable=AsyncMock,
        ) as mock_verify:
            mock_verify.return_value = None

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/reset-password",
                    json={
                        "token": "invalid_token",
                        "new_password": "new_password",
                    },
                )

                assert response.status_code == 400

    @pytest.mark.asyncio
    async def test_reset_password_weak_password(self):
        """Test password reset with weak password."""
        with patch(
            "pictactic.api.routes.auth.auth_service.verify_token",
            new_callable=AsyncMock,
        ) as mock_verify:
            mock_verify.return_value = "user-123"

            transport = ASGITransport(app=app)
            async with AsyncClient(
                transport=transport, base_url="http://test"
            ) as client:
                response = await client.post(
                    "/api/v1/auth/reset-password",
                    json={
                        "token": "valid_token",
                        "new_password": "12345",  # Less than 6 characters
                    },
                )

                assert response.status_code == 400
                assert "密码长度至少为 6 位" in response.json()["detail"]
