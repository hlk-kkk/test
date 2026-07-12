"""CLI HTTP client for communicating with the PicTactic backend API."""

import logging
import subprocess
import sys
import time
from pathlib import Path

import httpx

logger = logging.getLogger(__name__)

# Project root: 4 levels up from this file
_PROJECT_ROOT = Path(__file__).resolve().parents[4]

DEFAULT_API_URL = "http://localhost:8019"
_HEALTH_TIMEOUT = 15  # seconds to wait for backend to start


def _get_internal_api_key() -> str:
    """Read internal_api_key from settings."""
    from ..core.config import settings

    return settings.internal_api_key


def get_api_headers() -> dict[str, str]:
    """Build request headers with X-API-Key authentication."""
    key = _get_internal_api_key()
    headers = {}
    if key:
        headers["X-API-Key"] = key
    return headers


def api_get(url: str, timeout: int = 30) -> httpx.Response:
    """Send authenticated GET request."""
    resp = httpx.get(url, headers=get_api_headers(), timeout=timeout)
    _check_response(resp)
    return resp


def api_post(url: str, json: dict | None = None, timeout: int = 60) -> httpx.Response:
    """Send authenticated POST request."""
    resp = httpx.post(url, json=json, headers=get_api_headers(), timeout=timeout)
    _check_response(resp)
    return resp


def _check_response(resp: httpx.Response) -> None:
    """Raise descriptive errors for common HTTP status codes."""
    if resp.status_code == 401:
        raise ApiError("认证失败：API Key 无效或未配置 (HTTP 401)")
    if resp.status_code == 404:
        raise ApiError(f"资源未找到 (HTTP 404): {resp.url}")
    resp.raise_for_status()


def ensure_backend_running(api_url: str = DEFAULT_API_URL) -> None:
    """Check if backend is reachable; auto-start if not."""
    health_url = f"{api_url}/health"

    # Quick health check
    try:
        resp = httpx.get(health_url, timeout=3)
        if resp.status_code == 200:
            return
    except (httpx.ConnectError, httpx.TimeoutException):
        pass

    # Auto-start backend
    logger.info("Backend not running, starting automatically...")
    try:
        subprocess.Popen(
            [
                sys.executable, "-m", "uvicorn",
                "backend.src.pictactic.api.app:app",
                "--port", api_url.rsplit(":", 1)[-1] if ":" in api_url.rsplit("/", 1)[-1] else "8019",
            ],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            cwd=str(_PROJECT_ROOT),
        )
    except OSError as e:
        raise ApiError(f"无法启动后端进程: {e}") from e

    # Poll health endpoint
    deadline = time.monotonic() + _HEALTH_TIMEOUT
    while time.monotonic() < deadline:
        time.sleep(1)
        try:
            resp = httpx.get(health_url, timeout=3)
            if resp.status_code == 200:
                logger.info("Backend started successfully")
                return
        except (httpx.ConnectError, httpx.TimeoutException):
            continue

    raise ApiError(f"后端启动超时（{_HEALTH_TIMEOUT}s），请手动启动后端")


class ApiError(Exception):
    """CLI API client error."""
