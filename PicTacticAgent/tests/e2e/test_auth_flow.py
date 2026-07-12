"""E2E tests for authentication flows.

These tests require a running server. Set SKIP_SERVER_TESTS=false to run them.
"""

import asyncio
import os

import httpx
import pytest

API_URL = "http://localhost:8019/api/v1/auth"

# Skip marker for tests that require running server
requires_server = pytest.mark.skipif(
    os.getenv("SKIP_SERVER_TESTS", "true").lower() == "true",
    reason="Server tests skipped (set SKIP_SERVER_TESTS=false to run)",
)


@requires_server
class TestAuthenticationE2E:
    """End-to-end tests for authentication flows."""

    @pytest.fixture(autouse=True)
    def setup(self):
        """Set up test fixtures."""
        self.test_email = f"test_{os.urandom(4).hex()}@example.com"
        self.test_password = "secure_password_123"
        self.test_display_name = "E2E Test User"

    @pytest.mark.asyncio
    async def test_full_registration_flow(self):
        """Test complete registration flow."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # 1. Register new user
            print(f"Registering user: {self.test_email}")
            response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                    "display_name": self.test_display_name,
                },
            )

            if response.status_code == 200:
                data = response.json()
                print(f"Registration successful: {data['user']['email']}")

                assert data["access_token"] is not None
                assert data["refresh_token"] is not None
                assert data["user"]["email"] == self.test_email
                assert data["user"]["display_name"] == self.test_display_name
                assert data["user"]["email_verified"] is False  # Not verified yet

                return data
            else:
                print(f"Registration failed: {response.status_code} - {response.text}")
                pytest.fail(f"Registration failed: {response.text}")

    @pytest.mark.asyncio
    async def test_full_login_flow(self):
        """Test complete login flow."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register a user
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if register_response.status_code != 200:
                print(f"Registration step failed: {register_response.text}")
                pytest.skip("Could not register user for login test")

            # Now login
            print(f"Logging in user: {self.test_email}")
            response = await client.post(
                f"{API_URL}/login",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if response.status_code == 200:
                data = response.json()
                print(f"Login successful")

                assert data["access_token"] is not None
                assert data["refresh_token"] is not None
                assert data["user"]["email"] == self.test_email

                return data
            else:
                print(f"Login failed: {response.status_code} - {response.text}")
                pytest.fail(f"Login failed: {response.text}")

    @pytest.mark.asyncio
    async def test_token_refresh_flow(self):
        """Test token refresh flow."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register and login
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if register_response.status_code != 200:
                pytest.skip("Could not register user for refresh test")

            tokens = register_response.json()
            original_access_token = tokens["access_token"]
            refresh_token = tokens["refresh_token"]

            # Wait a moment to ensure new tokens are different
            await asyncio.sleep(1)

            # Refresh tokens
            print("Refreshing tokens...")
            response = await client.post(
                f"{API_URL}/refresh",
                json={"refresh_token": refresh_token},
            )

            if response.status_code == 200:
                data = response.json()
                print("Token refresh successful")

                assert data["access_token"] is not None
                assert data["refresh_token"] is not None
                # New tokens should be different
                assert data["access_token"] != original_access_token
                assert data["refresh_token"] != refresh_token

                return data
            else:
                print(f"Token refresh failed: {response.status_code} - {response.text}")
                pytest.fail(f"Token refresh failed: {response.text}")

    @pytest.mark.asyncio
    async def test_get_current_user(self):
        """Test getting current user info."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                    "display_name": self.test_display_name,
                },
            )

            if register_response.status_code != 200:
                pytest.skip("Could not register user")

            tokens = register_response.json()
            access_token = tokens["access_token"]

            # Get current user
            print("Getting current user info...")
            response = await client.get(
                f"{API_URL}/me",
                headers={"Authorization": f"Bearer {access_token}"},
            )

            if response.status_code == 200:
                data = response.json()
                print(f"Got user info: {data['email']}")

                assert data["email"] == self.test_email
                assert data["display_name"] == self.test_display_name

                return data
            else:
                print(f"Get user failed: {response.status_code} - {response.text}")
                pytest.fail(f"Get user failed: {response.text}")

    @pytest.mark.asyncio
    async def test_logout_flow(self):
        """Test logout flow."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if register_response.status_code != 200:
                pytest.skip("Could not register user")

            tokens = register_response.json()
            refresh_token = tokens["refresh_token"]

            # Logout
            print("Logging out...")
            response = await client.post(
                f"{API_URL}/logout",
                json={"refresh_token": refresh_token},
            )

            assert response.status_code == 200
            assert response.json()["success"] is True
            print("Logout successful")

            # Try to refresh with the revoked token (should fail)
            print("Attempting to use revoked token...")
            refresh_response = await client.post(
                f"{API_URL}/refresh",
                json={"refresh_token": refresh_token},
            )

            # Should fail because token is revoked
            assert refresh_response.status_code == 401
            print("Token correctly rejected after logout")

    @pytest.mark.asyncio
    async def test_login_wrong_password(self):
        """Test login with wrong password."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if register_response.status_code != 200:
                pytest.skip("Could not register user")

            # Try to login with wrong password
            print("Attempting login with wrong password...")
            response = await client.post(
                f"{API_URL}/login",
                json={
                    "email": self.test_email,
                    "password": "wrong_password",
                },
            )

            assert response.status_code == 401
            print("Wrong password correctly rejected")

    @pytest.mark.asyncio
    async def test_login_nonexistent_user(self):
        """Test login with non-existent user."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Attempting login with non-existent user...")
            response = await client.post(
                f"{API_URL}/login",
                json={
                    "email": "nonexistent_user@example.com",
                    "password": "some_password",
                },
            )

            assert response.status_code == 401
            print("Non-existent user correctly rejected")

    @pytest.mark.asyncio
    async def test_register_duplicate_email(self):
        """Test registering with duplicate email."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First registration
            response1 = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if response1.status_code != 200:
                pytest.skip("Could not complete first registration")

            # Second registration with same email
            print("Attempting duplicate registration...")
            response2 = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": "different_password",
                },
            )

            assert response2.status_code == 400
            print("Duplicate email correctly rejected")

    @pytest.mark.asyncio
    async def test_register_weak_password(self):
        """Test registering with weak password."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Attempting registration with weak password...")
            response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": "12345",  # Less than 6 characters
                },
            )

            assert response.status_code == 400
            print("Weak password correctly rejected")

    @pytest.mark.asyncio
    async def test_protected_endpoint_without_token(self):
        """Test accessing protected endpoint without token."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Attempting to access protected endpoint without token...")
            response = await client.get(f"{API_URL}/me")

            assert response.status_code in [401, 403]
            print("Unauthorized access correctly rejected")

    @pytest.mark.asyncio
    async def test_protected_endpoint_with_invalid_token(self):
        """Test accessing protected endpoint with invalid token."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Attempting to access protected endpoint with invalid token...")
            response = await client.get(
                f"{API_URL}/me",
                headers={"Authorization": "Bearer invalid_token"},
            )

            assert response.status_code in [401, 403]
            print("Invalid token correctly rejected")


@requires_server
class TestOAuthE2E:
    """End-to-end tests for OAuth flows."""

    @pytest.mark.asyncio
    async def test_get_oauth_providers(self):
        """Test getting available OAuth providers."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Getting OAuth providers...")
            response = await client.get(f"{API_URL}/oauth/providers")

            assert response.status_code == 200
            data = response.json()
            print(f"OAuth providers: github={data['github']}, google={data['google']}")

            assert "github" in data
            assert "google" in data

    @pytest.mark.asyncio
    async def test_github_authorize_url(self):
        """Test getting GitHub OAuth authorization URL."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Getting GitHub OAuth authorize URL...")
            response = await client.get(f"{API_URL}/oauth/github/authorize")

            if response.status_code == 200:
                data = response.json()
                print(f"Got authorize URL: {data['authorize_url'][:50]}...")

                assert "authorize_url" in data
                assert "state" in data
                assert "github.com" in data["authorize_url"]
            elif response.status_code == 400:
                print("GitHub OAuth not configured (expected in test environment)")
            else:
                pytest.fail(f"Unexpected status: {response.status_code}")

    @pytest.mark.asyncio
    async def test_google_authorize_url(self):
        """Test getting Google OAuth authorization URL."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Getting Google OAuth authorize URL...")
            response = await client.get(f"{API_URL}/oauth/google/authorize")

            if response.status_code == 200:
                data = response.json()
                print(f"Got authorize URL: {data['authorize_url'][:50]}...")

                assert "authorize_url" in data
                assert "state" in data
                assert "google" in data["authorize_url"].lower()
            elif response.status_code == 400:
                print("Google OAuth not configured (expected in test environment)")
            else:
                pytest.fail(f"Unexpected status: {response.status_code}")


@requires_server
class TestPasswordResetE2E:
    """End-to-end tests for password reset flows."""

    @pytest.fixture(autouse=True)
    def setup(self):
        """Set up test fixtures."""
        self.test_email = f"reset_{os.urandom(4).hex()}@example.com"
        self.test_password = "original_password"
        self.new_password = "new_secure_password"

    @pytest.mark.asyncio
    async def test_forgot_password_existing_user(self):
        """Test forgot password for existing user."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            # First register
            register_response = await client.post(
                f"{API_URL}/register",
                json={
                    "email": self.test_email,
                    "password": self.test_password,
                },
            )

            if register_response.status_code != 200:
                pytest.skip("Could not register user")

            # Request password reset
            print(f"Requesting password reset for: {self.test_email}")
            response = await client.post(
                f"{API_URL}/forgot-password",
                json={"email": self.test_email},
            )

            assert response.status_code == 200
            assert response.json()["success"] is True
            print("Password reset request successful")

    @pytest.mark.asyncio
    async def test_forgot_password_nonexistent_user(self):
        """Test forgot password for non-existent user (should still return success)."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Requesting password reset for non-existent user...")
            response = await client.post(
                f"{API_URL}/forgot-password",
                json={"email": "nonexistent@example.com"},
            )

            # Should still return success to prevent email enumeration
            assert response.status_code == 200
            assert response.json()["success"] is True
            print("Request completed (email enumeration prevented)")

    @pytest.mark.asyncio
    async def test_reset_password_invalid_token(self):
        """Test password reset with invalid token."""
        async with httpx.AsyncClient(timeout=30.0) as client:
            print("Attempting password reset with invalid token...")
            response = await client.post(
                f"{API_URL}/reset-password",
                json={
                    "token": "invalid_token",
                    "new_password": self.new_password,
                },
            )

            assert response.status_code == 400
            print("Invalid token correctly rejected")


if __name__ == "__main__":
    # Run tests directly for debugging
    import sys

    pytest.main([__file__, "-v", "-s"] + sys.argv[1:])
