"""Tests for CLI api_client module."""

import time
from unittest.mock import MagicMock, patch

import httpx
import pytest

from pictactic.cli.api_client import (
    ApiError,
    _check_response,
    api_get,
    api_post,
    ensure_backend_running,
    get_api_headers,
)


class TestGetApiHeaders:
    """Tests for get_api_headers."""

    @patch("pictactic.cli.api_client._get_internal_api_key", return_value="test-key")
    def test_returns_header_with_key(self, mock_key):
        headers = get_api_headers()
        assert headers == {"X-API-Key": "test-key"}

    @patch("pictactic.cli.api_client._get_internal_api_key", return_value="")
    def test_returns_empty_headers_when_no_key(self, mock_key):
        headers = get_api_headers()
        assert headers == {}


class TestCheckResponse:
    """Tests for _check_response."""

    def test_401_raises_api_error(self):
        resp = MagicMock(spec=httpx.Response)
        resp.status_code = 401
        with pytest.raises(ApiError, match="认证失败"):
            _check_response(resp)

    def test_404_raises_api_error(self):
        resp = MagicMock(spec=httpx.Response)
        resp.status_code = 404
        resp.url = "http://localhost:8019/api/v1/test"
        with pytest.raises(ApiError, match="资源未找到"):
            _check_response(resp)

    def test_200_does_not_raise(self):
        resp = MagicMock(spec=httpx.Response)
        resp.status_code = 200
        resp.raise_for_status = MagicMock()
        _check_response(resp)
        resp.raise_for_status.assert_called_once()

    def test_500_calls_raise_for_status(self):
        resp = MagicMock(spec=httpx.Response)
        resp.status_code = 500
        resp.raise_for_status.side_effect = httpx.HTTPStatusError(
            "Server Error", request=MagicMock(), response=resp
        )
        with pytest.raises(httpx.HTTPStatusError):
            _check_response(resp)


class TestApiGet:
    """Tests for api_get."""

    @patch("pictactic.cli.api_client.get_api_headers", return_value={"X-API-Key": "k"})
    @patch("pictactic.cli.api_client.httpx.get")
    def test_passes_headers_and_timeout(self, mock_get, mock_headers):
        mock_resp = MagicMock(spec=httpx.Response)
        mock_resp.status_code = 200
        mock_resp.raise_for_status = MagicMock()
        mock_get.return_value = mock_resp

        result = api_get("http://localhost:8019/health", timeout=10)
        mock_get.assert_called_once_with(
            "http://localhost:8019/health",
            headers={"X-API-Key": "k"},
            timeout=10,
        )
        assert result is mock_resp

    @patch("pictactic.cli.api_client.get_api_headers", return_value={})
    @patch("pictactic.cli.api_client.httpx.get")
    def test_401_raises_api_error(self, mock_get, mock_headers):
        mock_resp = MagicMock(spec=httpx.Response)
        mock_resp.status_code = 401
        mock_get.return_value = mock_resp

        with pytest.raises(ApiError, match="认证失败"):
            api_get("http://localhost:8019/api")


class TestApiPost:
    """Tests for api_post."""

    @patch("pictactic.cli.api_client.get_api_headers", return_value={"X-API-Key": "k"})
    @patch("pictactic.cli.api_client.httpx.post")
    def test_passes_json_and_headers(self, mock_post, mock_headers):
        mock_resp = MagicMock(spec=httpx.Response)
        mock_resp.status_code = 200
        mock_resp.raise_for_status = MagicMock()
        mock_post.return_value = mock_resp

        payload = {"prompt": "test"}
        result = api_post("http://localhost:8019/api/v1/generation/", json=payload)
        mock_post.assert_called_once_with(
            "http://localhost:8019/api/v1/generation/",
            json=payload,
            headers={"X-API-Key": "k"},
            timeout=60,
        )
        assert result is mock_resp

    @patch("pictactic.cli.api_client.get_api_headers", return_value={})
    @patch("pictactic.cli.api_client.httpx.post")
    def test_404_raises_api_error(self, mock_post, mock_headers):
        mock_resp = MagicMock(spec=httpx.Response)
        mock_resp.status_code = 404
        mock_resp.url = "http://localhost:8019/api/v1/missing"
        mock_post.return_value = mock_resp

        with pytest.raises(ApiError, match="资源未找到"):
            api_post("http://localhost:8019/api/v1/missing", json={})


class TestEnsureBackendRunning:
    """Tests for ensure_backend_running."""

    @patch("pictactic.cli.api_client.httpx.get")
    def test_already_running(self, mock_get):
        mock_resp = MagicMock()
        mock_resp.status_code = 200
        mock_get.return_value = mock_resp

        ensure_backend_running("http://localhost:8019")
        mock_get.assert_called_once_with("http://localhost:8019/health", timeout=3)

    @patch("pictactic.cli.api_client.time.sleep")
    @patch("pictactic.cli.api_client.time.monotonic")
    @patch("pictactic.cli.api_client.subprocess.Popen")
    @patch("pictactic.cli.api_client.httpx.get")
    def test_starts_backend_on_connect_error(self, mock_get, mock_popen, mock_mono, mock_sleep):
        # First call: ConnectError (health check fails)
        # Second call (poll): success
        mock_resp_ok = MagicMock()
        mock_resp_ok.status_code = 200
        mock_get.side_effect = [httpx.ConnectError("refused"), mock_resp_ok]

        # monotonic: start=0, deadline=15, loop check=1 (< 15)
        mock_mono.side_effect = [0, 1]

        ensure_backend_running("http://localhost:8019")
        mock_popen.assert_called_once()
        assert mock_get.call_count == 2

    @patch("pictactic.cli.api_client.time.sleep")
    @patch("pictactic.cli.api_client.time.monotonic")
    @patch("pictactic.cli.api_client.subprocess.Popen")
    @patch("pictactic.cli.api_client.httpx.get")
    def test_timeout_raises_api_error(self, mock_get, mock_popen, mock_mono, mock_sleep):
        # All health checks fail
        mock_get.side_effect = httpx.ConnectError("refused")

        # monotonic: start=0, deadline=15, loop iterations > 15
        mock_mono.side_effect = [0, 1, 16]

        with pytest.raises(ApiError, match="后端启动超时"):
            ensure_backend_running("http://localhost:8019")

    @patch("pictactic.cli.api_client.subprocess.Popen")
    @patch("pictactic.cli.api_client.httpx.get")
    def test_popen_oserror_raises_api_error(self, mock_get, mock_popen):
        mock_get.side_effect = httpx.ConnectError("refused")
        mock_popen.side_effect = OSError("Permission denied")

        with pytest.raises(ApiError, match="无法启动后端进程"):
            ensure_backend_running("http://localhost:8019")
