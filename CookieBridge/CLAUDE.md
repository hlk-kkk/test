# CookieBridge Development Guide

## Project Structure
- `server/` - Python Tornado server (port 8274)
- `extension/` - Chrome Extension (Manifest V3)
- `docs/` - Protocol documentation

## Key Conventions
- Response format: `{"biz_code": 0, "msg": "OK!", "isok": true, "data": {...}}`
- Cookies are always plain strings: `key1=value1; key2=value2; ...`
- Follow DownloadServer patterns for Tornado handlers/models
- Use uv for Python dependency management
- Python 3.12

## Platform Keys
xhs, dy, ks, bili, wb, tieba, zhihu

## Server Port
8274 (default, configurable via APP_PORT env var)
