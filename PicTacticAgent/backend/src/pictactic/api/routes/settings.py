"""Provider settings API routes."""

import logging

from fastapi import APIRouter, Depends
from pydantic import BaseModel

from ...core.config import apply_settings_in_memory, settings, update_env_file
from ...services.auth_service import User
from ..dependencies import get_current_user

logger = logging.getLogger(__name__)

router = APIRouter()


# ========== Schemas ==========


class GeminiSettings(BaseModel):
    api_key: str | None = None
    model: str | None = None
    llm_model: str | None = None


class JiekouaiSettings(BaseModel):
    api_key: str | None = None
    api_base_url: str | None = None
    llm_base_url: str | None = None
    llm_model: str | None = None


class AntigravitySettings(BaseModel):
    api_key: str | None = None
    api_base_url: str | None = None
    model: str | None = None
    llm_model: str | None = None


class GeminiWebSettings(BaseModel):
    secure_1psid: str | None = None
    secure_1psidts: str | None = None
    model: str | None = None
    proxy: str | None = None


class ProviderSettingsResponse(BaseModel):
    default_provider: str
    gemini: GeminiSettings
    jiekouai: JiekouaiSettings
    antigravity: AntigravitySettings
    gemini_web: GeminiWebSettings


class ProviderSettingsUpdate(BaseModel):
    default_provider: str | None = None
    gemini: GeminiSettings | None = None
    jiekouai: JiekouaiSettings | None = None
    antigravity: AntigravitySettings | None = None
    gemini_web: GeminiWebSettings | None = None


# ========== Helper ==========


def _mask_key(key: str) -> str:
    """Mask API key, showing only last 4 chars."""
    if not key or len(key) <= 4:
        return "****"
    return "****" + key[-4:]


def _is_masked(value: str | None) -> bool:
    """Check if a value is a masked API key."""
    return value is not None and value.startswith("****")


# ========== Routes ==========


@router.get("/provider", response_model=ProviderSettingsResponse)
async def get_provider_settings(user: User = Depends(get_current_user)):
    """Get current provider settings (API keys are masked)."""
    return ProviderSettingsResponse(
        default_provider=settings.default_provider,
        gemini=GeminiSettings(
            api_key=_mask_key(settings.gemini_api_key),
            model=settings.gemini_model,
            llm_model=settings.gemini_llm_model,
        ),
        jiekouai=JiekouaiSettings(
            api_key=_mask_key(settings.jiekouai_api_key),
            api_base_url=settings.jiekouai_api_base_url,
            llm_base_url=settings.jiekouai_llm_base_url,
            llm_model=settings.jiekouai_llm_model,
        ),
        antigravity=AntigravitySettings(
            api_key=_mask_key(settings.antigravity_api_key),
            api_base_url=settings.antigravity_api_base_url,
            model=settings.antigravity_model,
            llm_model=settings.antigravity_llm_model,
        ),
        gemini_web=GeminiWebSettings(
            secure_1psid=_mask_key(settings.gemini_web_secure_1psid),
            secure_1psidts=_mask_key(settings.gemini_web_secure_1psidts),
            model=settings.gemini_web_model,
            proxy=settings.gemini_web_proxy,
        ),
    )


@router.put("/provider", response_model=ProviderSettingsResponse)
async def update_provider_settings(
    data: ProviderSettingsUpdate,
    user: User = Depends(get_current_user),
):
    """Update provider settings. Masked API key values (****xxxx) are skipped."""
    env_updates: dict[str, str] = {}

    if data.default_provider is not None:
        env_updates["DEFAULT_PROVIDER"] = data.default_provider

    if data.gemini:
        if data.gemini.api_key and not _is_masked(data.gemini.api_key):
            env_updates["GEMINI_API_KEY"] = data.gemini.api_key
        if data.gemini.model is not None:
            env_updates["GEMINI_MODEL"] = data.gemini.model
        if data.gemini.llm_model is not None:
            env_updates["GEMINI_LLM_MODEL"] = data.gemini.llm_model

    if data.jiekouai:
        if data.jiekouai.api_key and not _is_masked(data.jiekouai.api_key):
            env_updates["JIEKOUAI_API_KEY"] = data.jiekouai.api_key
        if data.jiekouai.api_base_url is not None:
            env_updates["JIEKOUAI_API_BASE_URL"] = data.jiekouai.api_base_url
        if data.jiekouai.llm_base_url is not None:
            env_updates["JIEKOUAI_LLM_BASE_URL"] = data.jiekouai.llm_base_url
        if data.jiekouai.llm_model is not None:
            env_updates["JIEKOUAI_LLM_MODEL"] = data.jiekouai.llm_model

    if data.antigravity:
        if data.antigravity.api_key and not _is_masked(data.antigravity.api_key):
            env_updates["ANTIGRAVITY_API_KEY"] = data.antigravity.api_key
        if data.antigravity.api_base_url is not None:
            env_updates["ANTIGRAVITY_API_BASE_URL"] = data.antigravity.api_base_url
        if data.antigravity.model is not None:
            env_updates["ANTIGRAVITY_MODEL"] = data.antigravity.model
        if data.antigravity.llm_model is not None:
            env_updates["ANTIGRAVITY_LLM_MODEL"] = data.antigravity.llm_model

    if data.gemini_web:
        if data.gemini_web.secure_1psid and not _is_masked(data.gemini_web.secure_1psid):
            env_updates["GEMINI_WEB_SECURE_1PSID"] = data.gemini_web.secure_1psid
        if data.gemini_web.secure_1psidts and not _is_masked(data.gemini_web.secure_1psidts):
            env_updates["GEMINI_WEB_SECURE_1PSIDTS"] = data.gemini_web.secure_1psidts
        if data.gemini_web.model is not None:
            env_updates["GEMINI_WEB_MODEL"] = data.gemini_web.model
        if data.gemini_web.proxy is not None:
            env_updates["GEMINI_WEB_PROXY"] = data.gemini_web.proxy

    if env_updates:
        try:
            update_env_file(env_updates)
            apply_settings_in_memory(env_updates)
            logger.info("Provider settings updated: %s", list(env_updates.keys()))
        except OSError:
            logger.warning(
                "Failed to write provider settings to .env", exc_info=True
            )
            # Still apply in-memory even if file write fails
            apply_settings_in_memory(env_updates)

    # Return updated settings (re-read from memory)
    return await get_provider_settings(user)
