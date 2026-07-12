"""Tests for authentication service."""

import pytest
from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, patch, MagicMock

from pictactic.services.auth_service import (
    AuthService,
    User,
    UserStatus,
    RegisterRequest,
    LoginRequest,
    TokenPair,
)


class TestAuthServicePassword:
    """Tests for password hashing and verification."""

    def setup_method(self):
        """Set up test fixtures."""
        self.auth_service = AuthService()

    def test_hash_password(self):
        """Test password hashing generates valid hash."""
        password = "test_password_123"
        hashed = self.auth_service.hash_password(password)

        assert hashed is not None
        assert hashed != password
        assert len(hashed) > 50  # bcrypt hashes are long

    def test_verify_password_correct(self):
        """Test verifying correct password."""
        password = "correct_password"
        hashed = self.auth_service.hash_password(password)

        assert self.auth_service.verify_password(password, hashed) is True

    def test_verify_password_incorrect(self):
        """Test verifying incorrect password."""
        password = "correct_password"
        wrong_password = "wrong_password"
        hashed = self.auth_service.hash_password(password)

        assert self.auth_service.verify_password(wrong_password, hashed) is False

    def test_verify_password_empty(self):
        """Test verifying empty password."""
        password = "correct_password"
        hashed = self.auth_service.hash_password(password)

        assert self.auth_service.verify_password("", hashed) is False

    def test_verify_password_invalid_hash(self):
        """Test verifying with invalid hash."""
        assert self.auth_service.verify_password("password", "invalid_hash") is False

    def test_hash_token(self):
        """Test token hashing."""
        token = "test_token_123"
        hashed = self.auth_service._hash_token(token)

        assert hashed is not None
        assert len(hashed) == 64  # SHA256 produces 64 hex characters
        assert hashed != token


class TestAuthServiceJWT:
    """Tests for JWT token operations."""

    def setup_method(self):
        """Set up test fixtures."""
        self.auth_service = AuthService()

    def test_create_access_token(self):
        """Test creating access token."""
        user_id = "test-user-123"
        token = self.auth_service.create_access_token(user_id)

        assert token is not None
        assert isinstance(token, str)
        assert len(token) > 0

    def test_create_access_token_with_extra_claims(self):
        """Test creating access token with extra claims."""
        user_id = "test-user-123"
        extra_claims = {"role": "admin", "custom": "value"}
        token = self.auth_service.create_access_token(user_id, extra_claims)

        payload = self.auth_service.decode_token(token)
        assert payload is not None
        assert payload.get("role") == "admin"
        assert payload.get("custom") == "value"

    def test_create_refresh_token(self):
        """Test creating refresh token."""
        user_id = "test-user-123"
        token, expires_at = self.auth_service.create_refresh_token(user_id)

        assert token is not None
        assert isinstance(token, str)
        assert expires_at > datetime.now(timezone.utc)

    def test_create_token_pair(self):
        """Test creating token pair."""
        user_id = "test-user-123"
        token_pair = self.auth_service.create_token_pair(user_id)

        assert isinstance(token_pair, TokenPair)
        assert token_pair.access_token is not None
        assert token_pair.refresh_token is not None
        assert token_pair.token_type == "Bearer"
        assert token_pair.expires_in > 0

    def test_decode_valid_token(self):
        """Test decoding valid token."""
        user_id = "test-user-123"
        token = self.auth_service.create_access_token(user_id)

        payload = self.auth_service.decode_token(token)
        assert payload is not None
        assert payload.get("sub") == user_id
        assert payload.get("type") == "access"

    def test_decode_invalid_token(self):
        """Test decoding invalid token."""
        payload = self.auth_service.decode_token("invalid_token")
        assert payload is None

    def test_decode_empty_token(self):
        """Test decoding empty token."""
        payload = self.auth_service.decode_token("")
        assert payload is None

    def test_verify_access_token_valid(self):
        """Test verifying valid access token."""
        user_id = "test-user-123"
        token = self.auth_service.create_access_token(user_id)

        result = self.auth_service.verify_access_token(token)
        assert result == user_id

    def test_verify_access_token_invalid(self):
        """Test verifying invalid access token."""
        result = self.auth_service.verify_access_token("invalid_token")
        assert result is None

    def test_verify_access_token_refresh_token(self):
        """Test verifying refresh token as access token fails."""
        user_id = "test-user-123"
        token, _ = self.auth_service.create_refresh_token(user_id)

        result = self.auth_service.verify_access_token(token)
        assert result is None

    def test_verify_refresh_token_valid(self):
        """Test verifying valid refresh token."""
        user_id = "test-user-123"
        token, _ = self.auth_service.create_refresh_token(user_id)

        result = self.auth_service.verify_refresh_token(token)
        assert result == user_id

    def test_verify_refresh_token_access_token(self):
        """Test verifying access token as refresh token fails."""
        user_id = "test-user-123"
        token = self.auth_service.create_access_token(user_id)

        result = self.auth_service.verify_refresh_token(token)
        assert result is None


class TestUserModel:
    """Tests for User Pydantic model."""

    def test_create_user(self):
        """Test creating user model."""
        user = User(
            user_id="test-123",
            email="test@example.com",
            display_name="Test User",
            status=UserStatus.ACTIVE,
        )

        assert user.user_id == "test-123"
        assert user.email == "test@example.com"
        assert user.display_name == "Test User"
        assert user.status == UserStatus.ACTIVE

    def test_user_defaults(self):
        """Test user model defaults."""
        user = User(
            user_id="test-123",
            email="test@example.com",
        )

        assert user.display_name is None
        assert user.avatar_url is None
        assert user.status == UserStatus.PENDING
        assert user.email_verified_at is None


class TestRegisterRequest:
    """Tests for RegisterRequest model."""

    def test_valid_register_request(self):
        """Test valid registration request."""
        request = RegisterRequest(
            email="test@example.com",
            password="secure123",
            display_name="Test User",
        )

        assert request.email == "test@example.com"
        assert request.password == "secure123"
        assert request.display_name == "Test User"

    def test_register_request_optional_display_name(self):
        """Test registration without display name."""
        request = RegisterRequest(
            email="test@example.com",
            password="secure123",
        )

        assert request.display_name is None

    def test_register_request_invalid_email(self):
        """Test registration with invalid email."""
        with pytest.raises(Exception):
            RegisterRequest(
                email="invalid-email",
                password="secure123",
            )


class TestLoginRequest:
    """Tests for LoginRequest model."""

    def test_valid_login_request(self):
        """Test valid login request."""
        request = LoginRequest(
            email="test@example.com",
            password="secure123",
        )

        assert request.email == "test@example.com"
        assert request.password == "secure123"


class TestTokenPair:
    """Tests for TokenPair model."""

    def test_create_token_pair(self):
        """Test creating token pair."""
        token_pair = TokenPair(
            access_token="access_123",
            refresh_token="refresh_456",
            expires_in=1800,
        )

        assert token_pair.access_token == "access_123"
        assert token_pair.refresh_token == "refresh_456"
        assert token_pair.token_type == "Bearer"
        assert token_pair.expires_in == 1800


class TestUserStatus:
    """Tests for UserStatus enum."""

    def test_user_status_values(self):
        """Test user status enum values."""
        assert UserStatus.PENDING == 0
        assert UserStatus.ACTIVE == 1
        assert UserStatus.DISABLED == 2


# Integration-style tests with mocked database
class TestAuthServiceWithMockedDB:
    """Tests for auth service methods that require database mocking."""

    def setup_method(self):
        """Set up test fixtures."""
        self.auth_service = AuthService()
        # Disable email whitelist for all tests
        self._whitelist_patcher = patch(
            "pictactic.services.auth_service.settings.email_whitelist_enabled", False
        )
        self._whitelist_patcher.start()

    def teardown_method(self):
        """Tear down test fixtures."""
        self._whitelist_patcher.stop()

    @pytest.mark.asyncio
    async def test_register_success(self):
        """Test successful user registration."""
        mock_user = User(
            user_id="new-user-id",
            email="new@example.com",
            display_name="New User",
            status=UserStatus.PENDING,
        )

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = None

            with patch.object(
                self.auth_service, "create_user", new_callable=AsyncMock
            ) as mock_create:
                mock_create.return_value = mock_user

                with patch.object(
                    self.auth_service, "save_refresh_token", new_callable=AsyncMock
                ):
                    request = RegisterRequest(
                        email="new@example.com",
                        password="secure123",
                        display_name="New User",
                    )

                    user, token_pair = await self.auth_service.register(request)

                    assert user.email == "new@example.com"
                    assert token_pair.access_token is not None
                    assert token_pair.refresh_token is not None

    @pytest.mark.asyncio
    async def test_register_email_exists(self):
        """Test registration with existing email."""
        existing_user = {"user_id": "existing-id", "email": "existing@example.com"}

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = existing_user

            request = RegisterRequest(
                email="existing@example.com",
                password="secure123",
            )

            with pytest.raises(ValueError, match="邮箱已被注册"):
                await self.auth_service.register(request)

    @pytest.mark.asyncio
    async def test_register_weak_password(self):
        """Test registration with weak password."""
        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = None

            request = RegisterRequest(
                email="new@example.com",
                password="12345",  # Less than 6 characters
            )

            with pytest.raises(ValueError, match="密码长度至少为 6 位"):
                await self.auth_service.register(request)

    @pytest.mark.asyncio
    async def test_login_success(self):
        """Test successful login."""
        password = "correct_password"
        hashed = self.auth_service.hash_password(password)

        user_data = {
            "user_id": "user-123",
            "email": "test@example.com",
            "password_hash": hashed,
            "display_name": "Test User",
            "avatar_url": None,
            "status": UserStatus.ACTIVE,
            "email_verified_at": datetime.now(timezone.utc),
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
        }

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = user_data

            with patch.object(
                self.auth_service, "save_refresh_token", new_callable=AsyncMock
            ):
                request = LoginRequest(
                    email="test@example.com",
                    password=password,
                )

                user, token_pair = await self.auth_service.login(request)

                assert user.email == "test@example.com"
                assert token_pair.access_token is not None

    @pytest.mark.asyncio
    async def test_login_wrong_password(self):
        """Test login with wrong password."""
        password = "correct_password"
        hashed = self.auth_service.hash_password(password)

        user_data = {
            "user_id": "user-123",
            "email": "test@example.com",
            "password_hash": hashed,
            "status": UserStatus.ACTIVE,
        }

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = user_data

            request = LoginRequest(
                email="test@example.com",
                password="wrong_password",
            )

            with pytest.raises(ValueError, match="邮箱或密码错误"):
                await self.auth_service.login(request)

    @pytest.mark.asyncio
    async def test_login_user_not_found(self):
        """Test login with non-existent user."""
        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = None

            request = LoginRequest(
                email="nonexistent@example.com",
                password="password",
            )

            with pytest.raises(ValueError, match="邮箱或密码错误"):
                await self.auth_service.login(request)

    @pytest.mark.asyncio
    async def test_login_disabled_user(self):
        """Test login with disabled user account."""
        password = "correct_password"
        hashed = self.auth_service.hash_password(password)

        user_data = {
            "user_id": "user-123",
            "email": "disabled@example.com",
            "password_hash": hashed,
            "status": UserStatus.DISABLED,
        }

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = user_data

            request = LoginRequest(
                email="disabled@example.com",
                password=password,
            )

            with pytest.raises(ValueError, match="账号已被禁用"):
                await self.auth_service.login(request)

    @pytest.mark.asyncio
    async def test_login_oauth_only_user(self):
        """Test login for OAuth-only user (no password)."""
        user_data = {
            "user_id": "user-123",
            "email": "oauth@example.com",
            "password_hash": None,  # OAuth user has no password
            "status": UserStatus.ACTIVE,
        }

        with patch.object(
            self.auth_service, "get_user_by_email", new_callable=AsyncMock
        ) as mock_get:
            mock_get.return_value = user_data

            request = LoginRequest(
                email="oauth@example.com",
                password="some_password",
            )

            with pytest.raises(ValueError, match="该账号使用第三方登录"):
                await self.auth_service.login(request)

    @pytest.mark.asyncio
    async def test_oauth_login_syncs_avatar_for_linked_user(self):
        """Test OAuth login updates avatar URL for already-linked account."""
        existing_user = User(
            user_id="user-123",
            email="oauth@example.com",
            avatar_url=None,
            status=UserStatus.ACTIVE,
        )
        updated_user = User(
            user_id="user-123",
            email="oauth@example.com",
            avatar_url="https://avatars.example.com/u/123.png",
            status=UserStatus.ACTIVE,
        )

        with patch.object(
            self.auth_service, "get_oauth_account", new_callable=AsyncMock
        ) as mock_get_oauth:
            mock_get_oauth.return_value = {"user_id": "user-123"}

            with patch.object(
                self.auth_service, "get_user_by_id", new_callable=AsyncMock
            ) as mock_get_user:
                mock_get_user.return_value = existing_user

                with patch.object(
                    self.auth_service, "update_user", new_callable=AsyncMock
                ) as mock_update_user:
                    mock_update_user.return_value = updated_user

                    with patch.object(
                        self.auth_service, "update_oauth_tokens", new_callable=AsyncMock
                    ):
                        with patch.object(
                            self.auth_service, "save_refresh_token", new_callable=AsyncMock
                        ):
                            user, token_pair, is_new = await self.auth_service.oauth_login_or_register(
                                provider="github",
                                provider_user_id="123",
                                email="oauth@example.com",
                                avatar_url="https://avatars.example.com/u/123.png",
                                access_token="oauth_access_token",
                            )

                            assert is_new is False
                            assert user.avatar_url == "https://avatars.example.com/u/123.png"
                            assert token_pair.access_token is not None
                            mock_update_user.assert_awaited_once_with(
                                "user-123",
                                avatar_url="https://avatars.example.com/u/123.png",
                            )

    @pytest.mark.asyncio
    async def test_oauth_link_existing_user_syncs_avatar(self):
        """Test linking OAuth to an existing email user updates avatar URL."""
        existing_user_data = {
            "user_id": "user-456",
            "email": "existing@example.com",
            "password_hash": "hashed_password",
            "display_name": "Existing",
            "avatar_url": None,
            "status": UserStatus.ACTIVE,
            "email_verified_at": datetime.now(timezone.utc),
            "created_at": datetime.now(timezone.utc),
            "updated_at": datetime.now(timezone.utc),
        }
        updated_user = User(
            user_id="user-456",
            email="existing@example.com",
            avatar_url="https://lh3.googleusercontent.com/a/example",
            status=UserStatus.ACTIVE,
        )

        with patch.object(
            self.auth_service, "get_oauth_account", new_callable=AsyncMock
        ) as mock_get_oauth:
            mock_get_oauth.return_value = None

            with patch.object(
                self.auth_service, "get_user_by_email", new_callable=AsyncMock
            ) as mock_get_by_email:
                mock_get_by_email.return_value = existing_user_data

                with patch.object(
                    self.auth_service, "update_user", new_callable=AsyncMock
                ) as mock_update_user:
                    mock_update_user.return_value = updated_user

                    with patch.object(
                        self.auth_service, "create_oauth_account", new_callable=AsyncMock
                    ):
                        with patch.object(
                            self.auth_service, "save_refresh_token", new_callable=AsyncMock
                        ):
                            user, _, is_new = await self.auth_service.oauth_login_or_register(
                                provider="google",
                                provider_user_id="google-456",
                                email="existing@example.com",
                                avatar_url="https://lh3.googleusercontent.com/a/example",
                                access_token="oauth_access_token",
                                refresh_token="oauth_refresh_token",
                            )

                            assert is_new is False
                            assert user.avatar_url == "https://lh3.googleusercontent.com/a/example"
                            mock_update_user.assert_awaited_once_with(
                                "user-456",
                                avatar_url="https://lh3.googleusercontent.com/a/example",
                            )

    @pytest.mark.asyncio
    async def test_refresh_tokens_success(self):
        """Test successful token refresh."""
        user_id = "user-123"
        old_refresh_token, _ = self.auth_service.create_refresh_token(user_id)

        mock_user = User(
            user_id=user_id,
            email="test@example.com",
            status=UserStatus.ACTIVE,
        )

        with patch.object(
            self.auth_service, "validate_refresh_token", new_callable=AsyncMock
        ) as mock_validate:
            mock_validate.return_value = True

            with patch.object(
                self.auth_service, "get_user_by_id", new_callable=AsyncMock
            ) as mock_get:
                mock_get.return_value = mock_user

                with patch.object(
                    self.auth_service, "revoke_refresh_token", new_callable=AsyncMock
                ):
                    with patch.object(
                        self.auth_service, "save_refresh_token", new_callable=AsyncMock
                    ):
                        new_token_pair = await self.auth_service.refresh_tokens(
                            old_refresh_token
                        )

                        assert new_token_pair is not None
                        assert new_token_pair.access_token is not None
                        assert new_token_pair.refresh_token is not None

    @pytest.mark.asyncio
    async def test_refresh_tokens_invalid_token(self):
        """Test token refresh with invalid token."""
        result = await self.auth_service.refresh_tokens("invalid_token")
        assert result is None

    @pytest.mark.asyncio
    async def test_refresh_tokens_not_in_whitelist(self):
        """Test token refresh when token not in whitelist."""
        user_id = "user-123"
        refresh_token, _ = self.auth_service.create_refresh_token(user_id)

        with patch.object(
            self.auth_service, "validate_refresh_token", new_callable=AsyncMock
        ) as mock_validate:
            mock_validate.return_value = False

            result = await self.auth_service.refresh_tokens(refresh_token)
            assert result is None

    @pytest.mark.asyncio
    async def test_logout_success(self):
        """Test successful logout."""
        with patch.object(
            self.auth_service, "revoke_refresh_token", new_callable=AsyncMock
        ) as mock_revoke:
            mock_revoke.return_value = True

            result = await self.auth_service.logout("some_refresh_token")
            assert result is True
            mock_revoke.assert_called_once_with("some_refresh_token")
