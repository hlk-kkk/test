"""Tests for OAuth client utilities."""

from pictactic.services.oauth_client import normalize_google_avatar_url


def test_normalize_google_avatar_url_path_size_suffix():
    url = "https://lh3.googleusercontent.com/a/ACg8ocI769v3VDrOhbScDt386T-_ETVeGWdafw4kWrgJ_AiuTwsauQ=s96-c"
    normalized = normalize_google_avatar_url(url)
    assert normalized.endswith("=s256-c")


def test_normalize_google_avatar_url_query_size():
    url = "https://lh3.googleusercontent.com/a/ACg8ocI769v3VDrOhbScDt386T-_ETVeGWdafw4kWrgJ_AiuTwsauQ?sz=50"
    normalized = normalize_google_avatar_url(url)
    assert normalized.endswith("sz=256")


def test_normalize_google_avatar_url_without_size():
    url = "https://lh3.googleusercontent.com/a/ACg8ocI769v3VDrOhbScDt386T-_ETVeGWdafw4kWrgJ_AiuTwsauQ"
    normalized = normalize_google_avatar_url(url)
    assert normalized.endswith("=s256-c")


def test_normalize_google_avatar_url_non_google_unchanged():
    url = "https://avatars.githubusercontent.com/u/123?v=4"
    normalized = normalize_google_avatar_url(url)
    assert normalized == url
