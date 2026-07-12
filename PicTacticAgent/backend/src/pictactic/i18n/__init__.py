"""Internationalization support for PicTacticAgent."""

from .messages import MESSAGES


def get_text(key: str, lang: str = "en", **kwargs) -> str:
    """Get translated text by key and language.

    Args:
        key: Message key (dot-separated path)
        lang: Language code ('en' or 'zh')
        **kwargs: Interpolation parameters

    Returns:
        Translated string with parameters interpolated
    """
    lang = lang if lang in ("en", "zh") else "en"
    text = MESSAGES.get(lang, {}).get(key) or MESSAGES.get("en", {}).get(key, key)
    if kwargs:
        try:
            text = text.format(**kwargs)
        except (KeyError, IndexError):
            pass
    return text


def detect_language(accept_language: str | None = None, lang_param: str | None = None) -> str:
    """Detect preferred language from request.

    Priority: lang query param > Accept-Language header > default (en)
    """
    if lang_param and lang_param in ("en", "zh"):
        return lang_param
    if accept_language:
        for part in accept_language.split(","):
            tag = part.split(";")[0].strip().lower()
            if tag.startswith("zh"):
                return "zh"
            if tag.startswith("en"):
                return "en"
    return "en"
