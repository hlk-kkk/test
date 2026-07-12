"""
FastAPI 依赖注入 - 认证相关

提供:
- get_current_user: 从 JWT Token 获取当前用户
- get_current_user_optional: 可选认证（未登录返回 None）
- require_verified_user: 要求邮箱已验证
"""

from typing import Optional

from fastapi import Depends, HTTPException, Request, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from ..services.auth_service import User, UserStatus, auth_service

# HTTP Bearer Token 提取器
bearer_scheme = HTTPBearer(auto_error=False)


async def get_token_from_header(
    credentials: HTTPAuthorizationCredentials = Depends(bearer_scheme),
) -> Optional[str]:
    """从 Authorization 头提取 Token"""
    if credentials:
        return credentials.credentials
    return None


async def get_current_user(
    token: Optional[str] = Depends(get_token_from_header),
) -> User:
    """
    获取当前用户（必须已登录）

    Usage:
        @router.get("/me")
        async def get_me(user: User = Depends(get_current_user)):
            return user
    """
    if not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="未提供认证信息",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user_id = auth_service.verify_access_token(token)
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token 无效或已过期",
            headers={"WWW-Authenticate": "Bearer"},
        )

    user = await auth_service.get_user_by_id(user_id)
    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="用户不存在",
            headers={"WWW-Authenticate": "Bearer"},
        )

    if user.status == UserStatus.DISABLED:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="账号已被禁用",
        )

    return user


async def get_current_user_optional(
    token: Optional[str] = Depends(get_token_from_header),
) -> Optional[User]:
    """
    获取当前用户（可选，未登录返回 None）

    Usage:
        @router.get("/data")
        async def get_data(user: Optional[User] = Depends(get_current_user_optional)):
            if user:
                # 已登录用户
                ...
            else:
                # 未登录用户
                ...
    """
    if not token:
        return None

    user_id = auth_service.verify_access_token(token)
    if not user_id:
        return None

    user = await auth_service.get_user_by_id(user_id)
    if not user or user.status == UserStatus.DISABLED:
        return None

    return user


async def require_verified_user(
    user: User = Depends(get_current_user),
) -> User:
    """
    要求邮箱已验证的用户

    Usage:
        @router.post("/sensitive-action")
        async def sensitive_action(user: User = Depends(require_verified_user)):
            ...
    """
    if user.status == UserStatus.PENDING:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="请先验证邮箱",
        )
    return user


async def get_current_user_or_api_key(
    request: Request,
    token: Optional[str] = Depends(get_token_from_header),
) -> User:
    """
    认证：优先 JWT Bearer，失败后检查 X-API-Key 头。

    API Key 匹配时返回 sentinel User（user_id="__api_key__"）。
    """
    # Try JWT first
    if token:
        user_id = auth_service.verify_access_token(token)
        if user_id:
            user = await auth_service.get_user_by_id(user_id)
            if user and user.status != UserStatus.DISABLED:
                return user

    # Fallback to X-API-Key
    api_key = request.headers.get("X-API-Key")
    if api_key:
        from ..core.config import settings

        if api_key == settings.internal_api_key and settings.internal_api_key:
            return User(
                user_id="__api_key__",
                email="internal@api.local",
                status=UserStatus.ACTIVE,
            )

    raise HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="未提供有效认证信息",
        headers={"WWW-Authenticate": "Bearer"},
    )


def effective_user_id(user: User) -> Optional[str]:
    """Return user_id for ownership checks; None for API key sentinel (skip ownership)."""
    if user.user_id == "__api_key__":
        return None
    return user.user_id


def get_client_ip(request: Request) -> str:
    """获取客户端 IP"""
    # 检查代理头
    forwarded = request.headers.get("X-Forwarded-For")
    if forwarded:
        return forwarded.split(",")[0].strip()

    real_ip = request.headers.get("X-Real-IP")
    if real_ip:
        return real_ip

    # 直接连接
    if request.client:
        return request.client.host

    return "unknown"


def get_user_agent(request: Request) -> str:
    """获取用户代理"""
    return request.headers.get("User-Agent", "unknown")[:255]
