"""Application configuration using pydantic-settings."""

import logging
from functools import lru_cache
from pathlib import Path
from typing import Literal

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

logger = logging.getLogger(__name__)

# 项目根目录下的 .env 文件（向上查找直到找到 .env）
_env_file = Path(__file__).resolve().parents[4] / ".env"


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=str(_env_file),
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # API Keys
    gemini_api_key: str = Field(default="", description="Official Gemini API Key (from Google AI Studio)")
    openai_api_key: str = Field(default="", description="OpenAI API Key")
    openai_api_base_url: str = Field(
        default="https://api.jiekou.ai/openai",
        description="OpenAI API Base URL (for compatible APIs)",
    )

    # Official Gemini SDK settings
    gemini_model: str = Field(
        default="gemini-3-pro-image-preview",
        description="Official Gemini model for image generation",
    )
    gemini_llm_model: str = Field(
        default="gemini-3-flash-preview",
        description="Gemini model for LLM evaluation",
    )

    # Jiekouai third-party reverse proxy settings
    jiekouai_api_key: str = Field(default="", description="Jiekouai API Key")
    jiekouai_api_base_url: str = Field(
        default="https://api.jiekou.ai/v3",
        description="Jiekouai API base URL",
    )
    jiekouai_llm_base_url: str = Field(
        default="https://api.jiekou.ai/openai",
        description="Jiekouai OpenAI-compatible endpoint for LLM",
    )
    jiekouai_llm_model: str = Field(
        default="gpt-4o",
        description="LLM model for evaluation via jiekouai",
    )

    # Antigravity (OpenAI-compatible reverse proxy)
    antigravity_api_base_url: str = Field(
        default="http://127.0.0.1:8045/v1",
        description="Antigravity API base URL",
    )
    antigravity_api_key: str = Field(default="", description="Antigravity API Key")
    antigravity_model: str = Field(
        default="gemini-3-pro-image",
        description="Model name for Antigravity provider",
    )
    antigravity_llm_model: str = Field(
        default="gemini-3-flash",
        description="LLM model for evaluation via antigravity",
    )

    # Gemini Web (reverse-proxy via cookies)
    gemini_web_secure_1psid: str = Field(
        default="", description="__Secure-1PSID cookie for Gemini Web"
    )
    gemini_web_secure_1psidts: str = Field(
        default="", description="__Secure-1PSIDTS cookie for Gemini Web"
    )
    gemini_web_proxy: str = Field(
        default="", description="Proxy URL for Gemini Web requests"
    )
    gemini_web_model: str = Field(
        default="gemini-3-pro",
        description="Model to use for Gemini Web image generation",
    )

    # Default Provider
    default_provider: str = Field(
        default="gemini",
        description="Default image generation provider (gemini_web, gemini, jiekouai, antigravity)",
    )
    default_video_provider: str = Field(
        default="kling",
        description="Default video generation provider (kling, sora, veo)",
    )

    # Image Generation Defaults
    default_aspect_ratio: str = Field(default="16:9")
    default_image_size: str = Field(default="2K")
    default_images_per_round: int = Field(default=5, ge=1, le=10)
    default_max_rounds: int = Field(default=3, ge=1, le=10)
    default_quality_threshold: float = Field(default=0.7, ge=0.0, le=1.0)

    # Storage
    storage_type: Literal["local", "s3"] = Field(default="local")
    storage_path: str = Field(default="./storage/images")

    # Thumbnail (display-quality compressed images for web viewing)
    thumbnail_max_size: int = Field(default=1200, description="Max thumbnail dimension in pixels")
    thumbnail_quality: int = Field(default=82, ge=1, le=100, description="Thumbnail JPEG quality")

    # Application
    frontend_url: str = Field(default="http://localhost:3019")
    cors_origins: list[str] = Field(
        default=[],
        description="Additional CORS origins beyond auto-generated localhost origins",
    )

    # Access Control
    email_whitelist_enabled: bool = Field(
        default=False,
        description="Enable email whitelist - only listed emails can register/login",
    )
    email_whitelist: str = Field(
        default="",
        description="Comma-separated list of allowed email addresses",
    )

    # Database
    db_path: str = Field(default="./data/pictactic.db", description="SQLite database path")

    # Evaluation
    vision_llm_model: str = Field(default="gpt-4o")

    # JWT Authentication
    jwt_secret_key: str = Field(default="change-this-secret-key-in-production")
    jwt_algorithm: str = Field(default="HS256")
    jwt_access_token_expire_minutes: int = Field(default=1440)
    jwt_refresh_token_expire_days: int = Field(default=90)

    # GitHub OAuth
    github_client_id: str = Field(default="")
    github_client_secret: str = Field(default="")
    github_callback_url: str = Field(default="http://localhost:3019/auth/callback/github")

    # Google OAuth
    google_client_id: str = Field(default="")
    google_client_secret: str = Field(default="")
    google_callback_url: str = Field(default="http://localhost:3019/auth/callback/google")

    # Internal API Key (for CLI → API authentication, auto-generated if empty)
    internal_api_key: str = Field(default="", description="Internal API key for CLI access")

    # SMTP Email
    smtp_host: str = Field(default="")
    smtp_port: int = Field(default=587)
    smtp_user: str = Field(default="")
    smtp_password: str = Field(default="")
    smtp_from_email: str = Field(default="")
    smtp_from_name: str = Field(default="PicTactic Agent")
    smtp_use_tls: bool = Field(default=True)


@lru_cache
def get_settings() -> Settings:
    """Get cached settings instance."""
    s = Settings()
    logger.info("Loaded settings: default_provider=%s, env_file=%s", s.default_provider, _env_file)
    return s


settings = get_settings()


def update_env_file(updates: dict[str, str]) -> None:
    """Update .env file with given key-value pairs."""
    env_path = Path(settings.model_config.get("env_file", ".env"))

    if env_path.exists():
        content = env_path.read_text()
        lines = content.splitlines(keepends=True)
    else:
        lines = []

    remaining_updates = dict(updates)
    new_lines = []

    for line in lines:
        stripped = line.strip()
        if "=" in stripped and not stripped.startswith("#"):
            key = stripped.split("=", 1)[0].strip()
            if key in remaining_updates:
                new_lines.append(f"{key}={remaining_updates.pop(key)}\n")
                continue
        new_lines.append(line)

    # Append any remaining new keys
    for key, value in remaining_updates.items():
        new_lines.append(f"{key}={value}\n")

    env_path.write_text("".join(new_lines))


def apply_settings_in_memory(updates: dict[str, str]) -> None:
    """Apply setting updates to the in-memory settings singleton."""
    field_map = dict(Settings.model_fields.items())

    for env_key, value in updates.items():
        # Convert ENV_KEY to field_name (lowercase)
        field_name = env_key.lower()
        if field_name in field_map:
            settings.__dict__[field_name] = value


def get_llm_config() -> dict:
    """根据 default_provider 返回 LLM 配置"""
    provider = settings.default_provider

    match provider:
        case "gemini":
            return {
                "model": settings.gemini_llm_model,
                "model_provider": "google_genai",
                "api_key": settings.gemini_api_key,
                "base_url": None,
            }
        case "jiekouai":
            return {
                "model": settings.jiekouai_llm_model,
                "model_provider": "openai",
                "api_key": settings.jiekouai_api_key,
                "base_url": settings.jiekouai_llm_base_url,
            }
        case "antigravity":
            return {
                "model": settings.antigravity_llm_model,
                "model_provider": "openai",
                "api_key": settings.antigravity_api_key,
                "base_url": settings.antigravity_api_base_url,
            }
        case _:
            # Fallback to deprecated config
            return {
                "model": settings.vision_llm_model,
                "model_provider": "openai",
                "api_key": settings.openai_api_key,
                "base_url": settings.openai_api_base_url or None,
            }


def create_vision_llm(**kwargs):
    """创建 Vision LLM 实例"""
    from langchain.chat_models import init_chat_model

    config = get_llm_config()
    return init_chat_model(
        config["model"],
        model_provider=config["model_provider"],
        api_key=config["api_key"],
        base_url=config["base_url"],
        **kwargs,
    )
