# -*- coding: utf-8 -*-
import hashlib
from typing import Any, Dict, Iterable

from pydantic import BaseModel, ConfigDict, model_validator


USER_ID_SOURCE_KEYS = (
    "user_id",
    "sec_uid",
    "short_user_id",
    "user_unique_id",
    "user_link",
    "profile_url",
    "user_url_token",
    "url_token",
    "mid",
    "uid",
    "author_id",
)

NICKNAME_KEYS = ("nickname", "user_nickname")
NICKNAME_SOURCE_KEYS = ("nickname", "user_nickname", "user_name", "screen_name", "name")
RUNTIME_ONLY_FIELDS = {"url_token"}

FORBIDDEN_USER_FIELDS = {
    "user_id",
    "sec_uid",
    "short_user_id",
    "user_unique_id",
    "user_signature",
    "user_name",
    "user_link",
    "user_url_token",
    "url_token",
    "profile_url",
    "avatar",
    "user_avatar",
    "ip_location",
    "ip_address",
    "gender",
    "sex",
    "description",
}

FORBIDDEN_RAW_KEYS = FORBIDDEN_USER_FIELDS | {
    "author",
    "owner",
    "user",
    "member",
    "creator",
    "user_info",
    "author_info",
    "up_info",
}


def anonymize_user_id(user_id: Any) -> str:
    if user_id is None:
        return ""
    raw = str(user_id).strip()
    if not raw:
        return ""
    return hashlib.sha256(raw.encode("utf-8")).hexdigest()[:16]


def mask_nickname(name: Any) -> str:
    if name is None:
        return ""
    raw = str(name)
    if len(raw) <= 1:
        return "*"
    if len(raw) == 2:
        return raw[0] + "*"
    return raw[0] + "***" + raw[-1]


def _first_present(data: Dict[str, Any], keys: Iterable[str]) -> Any:
    for key in keys:
        value = data.get(key)
        if value not in (None, ""):
            return value
    return None


def sanitize_user_item(item: Dict[str, Any]) -> Dict[str, Any]:
    cleaned = dict(item)

    creator_source = _first_present(cleaned, USER_ID_SOURCE_KEYS)
    if not cleaned.get("creator_hash") and creator_source is not None:
        cleaned["creator_hash"] = anonymize_user_id(creator_source)

    for key in NICKNAME_KEYS:
        if key in cleaned:
            cleaned[key] = mask_nickname(cleaned.get(key))

    if "nickname" not in cleaned and "user_nickname" not in cleaned:
        nickname = _first_present(cleaned, NICKNAME_SOURCE_KEYS)
        if nickname:
            cleaned["nickname"] = mask_nickname(nickname)

    for key in FORBIDDEN_USER_FIELDS:
        cleaned.pop(key, None)

    return cleaned


def sanitize_raw_payload(value: Any) -> Any:
    if isinstance(value, list):
        return [sanitize_raw_payload(item) for item in value]
    if not isinstance(value, dict):
        return value

    cleaned: Dict[str, Any] = {}
    for key, raw_value in value.items():
        if key in FORBIDDEN_RAW_KEYS:
            continue
        if key in NICKNAME_SOURCE_KEYS:
            cleaned[key] = mask_nickname(raw_value)
            continue
        cleaned[key] = sanitize_raw_payload(raw_value)
    return cleaned


class PrivacyBaseModel(BaseModel):
    model_config = ConfigDict(extra="ignore")

    @model_validator(mode="before")
    @classmethod
    def sanitize_user_fields(cls, data: Any) -> Any:
        if isinstance(data, dict):
            cleaned = sanitize_user_item(data)
            for field_name in RUNTIME_ONLY_FIELDS:
                if field_name in getattr(cls, "model_fields", {}) and field_name in data:
                    cleaned[field_name] = data[field_name]
            return cleaned
        return data

    def model_dump(self, *args: Any, **kwargs: Any) -> Dict[str, Any]:
        return sanitize_user_item(super().model_dump(*args, **kwargs))
