"""
用户认证服务 - 核心认证逻辑

提供:
- 用户注册/登录
- JWT Token 生成与验证
- 密码哈希
- Refresh Token 管理
"""

import hashlib
import logging
import secrets
import uuid
from datetime import datetime, timedelta, timezone
from typing import Optional, Tuple

import bcrypt
import jwt
from pydantic import BaseModel, EmailStr
from sqlalchemy import select, delete, update
from sqlalchemy.ext.asyncio import AsyncSession

from ..core.config import settings
from ..db.engine import get_session, get_write_lock
from ..db.models import (
    User as UserModel,
    UserOAuthAccount,
    EmailVerificationToken,
    UserRefreshToken,
    OAuthState,
)

logger = logging.getLogger(__name__)


# ========== 数据模型 ==========


class UserStatus:
    """用户状态枚举"""

    PENDING = 0  # 待验证
    ACTIVE = 1  # 正常
    DISABLED = 2  # 禁用


class User(BaseModel):
    """用户模型（Pydantic）"""

    user_id: str
    email: str
    display_name: Optional[str] = None
    avatar_url: Optional[str] = None
    status: int = UserStatus.PENDING
    email_verified_at: Optional[datetime] = None
    created_at: Optional[datetime] = None
    updated_at: Optional[datetime] = None

    class Config:
        from_attributes = True

    @classmethod
    def from_orm_model(cls, model: UserModel) -> "User":
        """从 ORM 模型创建 Pydantic 模型"""
        return cls(
            user_id=model.user_id,
            email=model.email,
            display_name=model.display_name,
            avatar_url=model.avatar_url,
            status=model.status,
            email_verified_at=model.email_verified_at,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )


class TokenPair(BaseModel):
    """Token 对"""

    access_token: str
    refresh_token: str
    token_type: str = "Bearer"
    expires_in: int  # Access token 过期时间（秒）


class RegisterRequest(BaseModel):
    """注册请求"""

    email: EmailStr
    password: str
    display_name: Optional[str] = None


class LoginRequest(BaseModel):
    """登录请求"""

    email: EmailStr
    password: str


# ========== 认证服务 ==========


class AuthService:
    """用户认证服务"""

    def __init__(self):
        self._secret_key = settings.jwt_secret_key
        self._algorithm = settings.jwt_algorithm
        self._access_expire_minutes = settings.jwt_access_token_expire_minutes
        self._refresh_expire_days = settings.jwt_refresh_token_expire_days
        self._write_lock = get_write_lock()

    # ========== 邮箱白名单 ==========

    def _check_email_whitelist(self, email: str) -> None:
        """检查邮箱是否在白名单中（仅在启用时生效）"""
        if not settings.email_whitelist_enabled:
            return
        allowed = {
            e.strip().lower()
            for e in settings.email_whitelist.split(",")
            if e.strip()
        }
        if email.lower() not in allowed:
            raise ValueError("该邮箱未被授权使用此服务，请联系管理员")

    # ========== 密码处理 ==========

    def hash_password(self, password: str) -> str:
        """哈希密码"""
        salt = bcrypt.gensalt()
        return bcrypt.hashpw(password.encode("utf-8"), salt).decode("utf-8")

    def verify_password(self, password: str, hashed: str) -> bool:
        """验证密码"""
        try:
            return bcrypt.checkpw(password.encode("utf-8"), hashed.encode("utf-8"))
        except Exception:
            return False

    def _hash_token(self, token: str) -> str:
        """哈希 token（用于存储 refresh token）"""
        return hashlib.sha256(token.encode("utf-8")).hexdigest()

    # ========== JWT Token ==========

    def create_access_token(self, user_id: str, extra_claims: dict = None) -> str:
        """创建 Access Token"""
        now = datetime.now(timezone.utc)
        expire = now + timedelta(minutes=self._access_expire_minutes)

        payload = {
            "sub": user_id,
            "type": "access",
            "iat": now,
            "exp": expire,
        }
        if extra_claims:
            payload.update(extra_claims)

        return jwt.encode(payload, self._secret_key, algorithm=self._algorithm)

    def create_refresh_token(self, user_id: str) -> Tuple[str, datetime]:
        """创建 Refresh Token，返回 (token, expires_at)"""
        now = datetime.now(timezone.utc)
        expire = now + timedelta(days=self._refresh_expire_days)

        payload = {
            "sub": user_id,
            "type": "refresh",
            "jti": secrets.token_urlsafe(32),  # 唯一标识
            "iat": now,
            "exp": expire,
        }

        token = jwt.encode(payload, self._secret_key, algorithm=self._algorithm)
        return token, expire

    def create_token_pair(self, user_id: str) -> TokenPair:
        """创建 Token 对"""
        access_token = self.create_access_token(user_id)
        refresh_token, _ = self.create_refresh_token(user_id)

        return TokenPair(
            access_token=access_token,
            refresh_token=refresh_token,
            expires_in=self._access_expire_minutes * 60,
        )

    def decode_token(self, token: str) -> Optional[dict]:
        """解码 Token"""
        try:
            payload = jwt.decode(
                token,
                self._secret_key,
                algorithms=[self._algorithm],
            )
            return payload
        except jwt.ExpiredSignatureError:
            logger.debug("Token expired")
            return None
        except jwt.InvalidTokenError as e:
            logger.debug(f"Invalid token: {e}")
            return None

    def verify_access_token(self, token: str) -> Optional[str]:
        """验证 Access Token，返回 user_id"""
        payload = self.decode_token(token)
        if not payload:
            return None
        if payload.get("type") != "access":
            return None
        return payload.get("sub")

    def verify_refresh_token(self, token: str) -> Optional[str]:
        """验证 Refresh Token，返回 user_id"""
        payload = self.decode_token(token)
        if not payload:
            return None
        if payload.get("type") != "refresh":
            return None
        return payload.get("sub")

    # ========== 用户管理 ==========

    async def get_user_by_id(self, user_id: str) -> Optional[User]:
        """根据 ID 获取用户"""
        async with get_session() as session:
            result = await session.execute(
                select(UserModel).where(UserModel.user_id == user_id)
            )
            model = result.scalar_one_or_none()
            if model:
                return User.from_orm_model(model)
            return None

    async def get_user_by_email(self, email: str) -> Optional[dict]:
        """根据邮箱获取用户（包含密码哈希）"""
        async with get_session() as session:
            result = await session.execute(
                select(UserModel).where(UserModel.email == email.lower())
            )
            model = result.scalar_one_or_none()
            if model:
                return {
                    "user_id": model.user_id,
                    "email": model.email,
                    "password_hash": model.password_hash,
                    "display_name": model.display_name,
                    "avatar_url": model.avatar_url,
                    "status": model.status,
                    "email_verified_at": model.email_verified_at,
                    "created_at": model.created_at,
                    "updated_at": model.updated_at,
                }
            return None

    async def create_user(
        self,
        email: str,
        password: Optional[str] = None,
        display_name: Optional[str] = None,
        avatar_url: Optional[str] = None,
        status: int = UserStatus.PENDING,
        email_verified: bool = False,
    ) -> User:
        """创建用户"""
        user_id = str(uuid.uuid4())
        password_hash = self.hash_password(password) if password else None
        email_verified_at = datetime.now(timezone.utc) if email_verified else None

        async with self._write_lock, get_session() as session:
            user_model = UserModel(
                user_id=user_id,
                email=email.lower(),
                password_hash=password_hash,
                display_name=display_name,
                avatar_url=avatar_url,
                status=status,
                email_verified_at=email_verified_at,
            )
            session.add(user_model)
            await session.commit()

        return User(
            user_id=user_id,
            email=email.lower(),
            display_name=display_name,
            avatar_url=avatar_url,
            status=status,
            email_verified_at=email_verified_at,
        )

    async def update_user(self, user_id: str, **kwargs) -> Optional[User]:
        """更新用户信息"""
        allowed_fields = {"display_name", "avatar_url", "status", "email_verified_at"}
        updates = {k: v for k, v in kwargs.items() if k in allowed_fields}

        if not updates:
            return await self.get_user_by_id(user_id)

        async with self._write_lock, get_session() as session:
            await session.execute(
                update(UserModel).where(UserModel.user_id == user_id).values(**updates)
            )
            await session.commit()

        return await self.get_user_by_id(user_id)

    async def update_password(self, user_id: str, new_password: str) -> bool:
        """更新用户密码"""
        password_hash = self.hash_password(new_password)

        async with self._write_lock, get_session() as session:
            result = await session.execute(
                update(UserModel)
                .where(UserModel.user_id == user_id)
                .values(password_hash=password_hash)
            )
            await session.commit()
            return result.rowcount > 0

    # ========== Refresh Token 白名单 ==========

    async def save_refresh_token(
        self,
        user_id: str,
        token: str,
        expires_at: datetime,
        device_info: str = None,
        ip_address: str = None,
    ) -> None:
        """保存 Refresh Token 到白名单"""
        token_hash = self._hash_token(token)

        async with self._write_lock, get_session() as session:
            token_model = UserRefreshToken(
                user_id=user_id,
                token_hash=token_hash,
                device_info=device_info,
                ip_address=ip_address,
                expires_at=expires_at,
            )
            session.add(token_model)
            await session.commit()

    async def validate_refresh_token(self, token: str) -> bool:
        """验证 Refresh Token 是否在白名单中"""
        token_hash = self._hash_token(token)
        now = datetime.now(timezone.utc)

        async with get_session() as session:
            result = await session.execute(
                select(UserRefreshToken.id).where(
                    UserRefreshToken.token_hash == token_hash,
                    UserRefreshToken.expires_at > now,
                )
            )
            return result.scalar_one_or_none() is not None

    async def revoke_refresh_token(self, token: str) -> bool:
        """撤销 Refresh Token"""
        token_hash = self._hash_token(token)

        async with self._write_lock, get_session() as session:
            result = await session.execute(
                delete(UserRefreshToken).where(UserRefreshToken.token_hash == token_hash)
            )
            await session.commit()
            return result.rowcount > 0

    async def revoke_all_user_tokens(self, user_id: str) -> int:
        """撤销用户的所有 Refresh Token"""
        async with self._write_lock, get_session() as session:
            result = await session.execute(
                delete(UserRefreshToken).where(UserRefreshToken.user_id == user_id)
            )
            await session.commit()
            return result.rowcount

    async def cleanup_expired_tokens(self) -> int:
        """清理过期的 Refresh Token"""
        now = datetime.now(timezone.utc)
        async with self._write_lock, get_session() as session:
            result = await session.execute(
                delete(UserRefreshToken).where(UserRefreshToken.expires_at < now)
            )
            await session.commit()
            return result.rowcount

    # ========== 邮箱验证令牌 ==========

    async def create_verification_token(
        self,
        user_id: str,
        token_type: str,
        expires_hours: int = 24,
    ) -> str:
        """创建验证令牌"""
        token = secrets.token_urlsafe(32)
        expires_at = datetime.now(timezone.utc) + timedelta(hours=expires_hours)

        async with self._write_lock, get_session() as session:
            token_model = EmailVerificationToken(
                user_id=user_id,
                token=token,
                token_type=token_type,
                expires_at=expires_at,
            )
            session.add(token_model)
            await session.commit()

        return token

    async def verify_token(self, token: str, token_type: str) -> Optional[str]:
        """验证令牌，返回 user_id"""
        now = datetime.now(timezone.utc)

        async with self._write_lock, get_session() as session:
            result = await session.execute(
                select(EmailVerificationToken).where(
                    EmailVerificationToken.token == token,
                    EmailVerificationToken.token_type == token_type,
                    EmailVerificationToken.expires_at > now,
                    EmailVerificationToken.used_at.is_(None),
                )
            )
            token_model = result.scalar_one_or_none()
            if token_model:
                # 标记为已使用
                token_model.used_at = now
                await session.commit()
                return token_model.user_id
            return None

    # ========== OAuth State ==========

    async def save_oauth_state(
        self, state: str, provider: str, expires_minutes: int = 10
    ) -> None:
        """将 OAuth state 持久化到数据库"""
        now = datetime.now(timezone.utc)
        expires_at = now + timedelta(minutes=expires_minutes)

        async with self._write_lock, get_session() as session:
            # 清理过期 state
            await session.execute(
                delete(OAuthState).where(OAuthState.expires_at < now)
            )
            # 保存新 state
            state_model = OAuthState(
                state=state,
                provider=provider,
                expires_at=expires_at,
            )
            session.add(state_model)
            await session.commit()

    async def consume_oauth_state(self, state: str) -> Optional[str]:
        """
        消费 OAuth state，返回 provider。
        一次性使用：取出后立即删除。
        """
        now = datetime.now(timezone.utc)

        async with self._write_lock, get_session() as session:
            result = await session.execute(
                select(OAuthState).where(
                    OAuthState.state == state,
                    OAuthState.expires_at > now,
                )
            )
            state_model = result.scalar_one_or_none()
            if state_model:
                provider = state_model.provider
                await session.execute(
                    delete(OAuthState).where(OAuthState.state == state)
                )
                await session.commit()
                return provider
            return None

    # ========== OAuth 关联 ==========

    async def get_oauth_account(
        self, provider: str, provider_user_id: str
    ) -> Optional[dict]:
        """获取 OAuth 关联账号"""
        async with get_session() as session:
            result = await session.execute(
                select(UserOAuthAccount).where(
                    UserOAuthAccount.provider == provider,
                    UserOAuthAccount.provider_user_id == provider_user_id,
                )
            )
            model = result.scalar_one_or_none()
            if model:
                return {
                    "id": model.id,
                    "user_id": model.user_id,
                    "provider": model.provider,
                    "provider_user_id": model.provider_user_id,
                    "provider_email": model.provider_email,
                    "access_token": model.access_token,
                    "refresh_token": model.refresh_token,
                    "raw_data": model.raw_data,
                }
            return None

    async def create_oauth_account(
        self,
        user_id: str,
        provider: str,
        provider_user_id: str,
        provider_email: str = None,
        access_token: str = None,
        refresh_token: str = None,
        raw_data: dict = None,
    ) -> None:
        """创建 OAuth 关联"""
        async with self._write_lock, get_session() as session:
            oauth_model = UserOAuthAccount(
                user_id=user_id,
                provider=provider,
                provider_user_id=str(provider_user_id),
                provider_email=provider_email,
                access_token=access_token,
                refresh_token=refresh_token,
                raw_data=raw_data,
            )
            session.add(oauth_model)
            await session.commit()

    async def update_oauth_tokens(
        self,
        provider: str,
        provider_user_id: str,
        access_token: str,
        refresh_token: str = None,
    ) -> None:
        """更新 OAuth Token"""
        async with self._write_lock, get_session() as session:
            await session.execute(
                update(UserOAuthAccount)
                .where(
                    UserOAuthAccount.provider == provider,
                    UserOAuthAccount.provider_user_id == str(provider_user_id),
                )
                .values(access_token=access_token, refresh_token=refresh_token)
            )
            await session.commit()

    # ========== 业务逻辑 ==========

    async def register(
        self,
        request: RegisterRequest,
        device_info: str | None = None,
        ip_address: str | None = None,
    ) -> Tuple[User, TokenPair]:
        """
        用户注册

        Returns:
            (user, token_pair)

        Raises:
            ValueError: 邮箱已存在
        """
        # 检查邮箱白名单
        self._check_email_whitelist(request.email)

        # 检查邮箱是否已存在
        existing = await self.get_user_by_email(request.email)
        if existing:
            raise ValueError("邮箱已被注册")

        # 密码强度验证
        if len(request.password) < 6:
            raise ValueError("密码长度至少为 6 位")

        # 创建用户（状态为待验证）
        user = await self.create_user(
            email=request.email,
            password=request.password,
            display_name=request.display_name,
            status=UserStatus.PENDING,
        )

        # 生成 Token
        token_pair = self.create_token_pair(user.user_id)

        # 保存 Refresh Token
        _, expires_at = self.create_refresh_token(user.user_id)
        await self.save_refresh_token(
            user.user_id,
            token_pair.refresh_token,
            expires_at,
            device_info=device_info,
            ip_address=ip_address,
        )

        logger.info(f"User registered: {user.email}")
        return user, token_pair

    async def login(
        self,
        request: LoginRequest,
        device_info: str | None = None,
        ip_address: str | None = None,
    ) -> Tuple[User, TokenPair]:
        """
        用户登录

        Returns:
            (user, token_pair)

        Raises:
            ValueError: 用户不存在或密码错误
        """
        # 检查邮箱白名单
        self._check_email_whitelist(request.email)

        # 获取用户
        user_data = await self.get_user_by_email(request.email)
        if not user_data:
            raise ValueError("邮箱或密码错误")

        # 验证密码
        if not user_data.get("password_hash"):
            raise ValueError("该账号使用第三方登录，请使用对应方式登录")

        if not self.verify_password(request.password, user_data["password_hash"]):
            raise ValueError("邮箱或密码错误")

        # 检查用户状态
        if user_data["status"] == UserStatus.DISABLED:
            raise ValueError("账号已被禁用")

        user = User(**{k: v for k, v in user_data.items() if k != "password_hash"})

        # 生成 Token
        token_pair = self.create_token_pair(user.user_id)

        # 保存 Refresh Token
        _, expires_at = self.create_refresh_token(user.user_id)
        await self.save_refresh_token(
            user.user_id,
            token_pair.refresh_token,
            expires_at,
            device_info=device_info,
            ip_address=ip_address,
        )

        logger.info(f"User logged in: {user.email}")
        return user, token_pair

    async def refresh_tokens(
        self,
        refresh_token: str,
        device_info: str | None = None,
        ip_address: str | None = None,
    ) -> Optional[TokenPair]:
        """
        刷新 Token

        实现 Token 轮换：旧 refresh token 失效，返回新的 token pair

        Returns:
            新的 token_pair 或 None（token 无效）
        """
        # 验证 refresh token
        user_id = self.verify_refresh_token(refresh_token)
        if not user_id:
            return None

        # 检查白名单
        if not await self.validate_refresh_token(refresh_token):
            logger.warning(f"Refresh token not in whitelist for user {user_id}")
            return None

        # 检查用户状态
        user = await self.get_user_by_id(user_id)
        if not user or user.status == UserStatus.DISABLED:
            return None

        # 撤销旧 token
        await self.revoke_refresh_token(refresh_token)

        # 生成新 token pair
        token_pair = self.create_token_pair(user_id)

        # 保存新 refresh token
        _, expires_at = self.create_refresh_token(user_id)
        await self.save_refresh_token(
            user_id,
            token_pair.refresh_token,
            expires_at,
            device_info=device_info,
            ip_address=ip_address,
        )

        logger.debug(f"Tokens refreshed for user {user_id}")
        return token_pair

    async def logout(self, refresh_token: str) -> bool:
        """
        登出（撤销 refresh token）
        """
        return await self.revoke_refresh_token(refresh_token)

    async def oauth_login_or_register(
        self,
        provider: str,
        provider_user_id: str,
        email: str,
        display_name: str = None,
        avatar_url: str = None,
        access_token: str = None,
        refresh_token: str = None,
        raw_data: dict = None,
        device_info: str | None = None,
        ip_address: str | None = None,
    ) -> Tuple[User, TokenPair, bool]:
        """
        OAuth 登录或注册

        Returns:
            (user, token_pair, is_new_user)
        """
        # 检查邮箱白名单
        effective_email = email or f"{provider}_{provider_user_id}@oauth.local"
        self._check_email_whitelist(effective_email)

        async def sync_avatar_url(user: User) -> User:
            """
            同步 OAuth 头像到用户资料。
            仅在提供了新的 avatar_url 时更新。
            """
            if not avatar_url or avatar_url == user.avatar_url:
                return user
            updated = await self.update_user(user.user_id, avatar_url=avatar_url)
            return updated or user

        # 检查是否已有 OAuth 关联
        oauth_account = await self.get_oauth_account(provider, provider_user_id)

        if oauth_account:
            # 已关联，直接登录
            user = await self.get_user_by_id(oauth_account["user_id"])
            if not user:
                raise ValueError("关联用户不存在")

            if user.status == UserStatus.DISABLED:
                raise ValueError("账号已被禁用")

            # 同步头像 URL（例如用户在 OAuth 提供商更换头像）
            user = await sync_avatar_url(user)

            # 更新 OAuth tokens
            await self.update_oauth_tokens(
                provider, provider_user_id, access_token, refresh_token
            )

            # 生成 Token
            token_pair = self.create_token_pair(user.user_id)
            _, expires_at = self.create_refresh_token(user.user_id)
            await self.save_refresh_token(
                user.user_id,
                token_pair.refresh_token,
                expires_at,
                device_info=device_info,
                ip_address=ip_address,
            )

            logger.info(f"OAuth login: {user.email} via {provider}")
            return user, token_pair, False

        # 检查邮箱是否已存在（可能用密码注册过）
        existing_user = await self.get_user_by_email(email) if email else None

        if existing_user:
            # 关联到现有用户
            user = User(**{k: v for k, v in existing_user.items() if k != "password_hash"})

            # 同步头像 URL 到已有账号
            user = await sync_avatar_url(user)

            await self.create_oauth_account(
                user_id=user.user_id,
                provider=provider,
                provider_user_id=provider_user_id,
                provider_email=email,
                access_token=access_token,
                refresh_token=refresh_token,
                raw_data=raw_data,
            )

            # 生成 Token
            token_pair = self.create_token_pair(user.user_id)
            _, expires_at = self.create_refresh_token(user.user_id)
            await self.save_refresh_token(
                user.user_id,
                token_pair.refresh_token,
                expires_at,
                device_info=device_info,
                ip_address=ip_address,
            )

            logger.info(f"OAuth linked to existing user: {user.email} via {provider}")
            return user, token_pair, False

        # 创建新用户
        user = await self.create_user(
            email=email or f"{provider}_{provider_user_id}@oauth.local",
            display_name=display_name,
            avatar_url=avatar_url,
            status=UserStatus.ACTIVE,  # OAuth 用户直接激活
            email_verified=bool(email),
        )

        # 创建 OAuth 关联
        await self.create_oauth_account(
            user_id=user.user_id,
            provider=provider,
            provider_user_id=provider_user_id,
            provider_email=email,
            access_token=access_token,
            refresh_token=refresh_token,
            raw_data=raw_data,
        )

        # 生成 Token
        token_pair = self.create_token_pair(user.user_id)
        _, expires_at = self.create_refresh_token(user.user_id)
        await self.save_refresh_token(
            user.user_id,
            token_pair.refresh_token,
            expires_at,
            device_info=device_info,
            ip_address=ip_address,
        )

        logger.info(f"OAuth user registered: {user.email} via {provider}")
        return user, token_pair, True


# 单例
auth_service = AuthService()
