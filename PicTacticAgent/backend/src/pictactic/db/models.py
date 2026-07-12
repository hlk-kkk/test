"""SQLAlchemy ORM models."""

from datetime import datetime, timezone
from typing import TYPE_CHECKING

from sqlalchemy import JSON, Boolean, DateTime, Float, ForeignKey, Integer, String, Text
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship

if TYPE_CHECKING:
    pass


def _utcnow() -> datetime:
    return datetime.now(timezone.utc)


class Base(DeclarativeBase):
    pass


# ========== User Authentication Models ==========


class User(Base):
    """用户模型"""

    __tablename__ = "users"

    user_id: Mapped[str] = mapped_column(String(36), primary_key=True)
    email: Mapped[str] = mapped_column(String(255), unique=True, index=True)
    password_hash: Mapped[str | None] = mapped_column(String(255), nullable=True)
    display_name: Mapped[str | None] = mapped_column(String(100), nullable=True)
    avatar_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
    status: Mapped[int] = mapped_column(Integer, default=0)  # 0=pending, 1=active, 2=disabled
    email_verified_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=_utcnow, onupdate=_utcnow
    )

    # Relationships
    oauth_accounts: Mapped[list["UserOAuthAccount"]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )
    refresh_tokens: Mapped[list["UserRefreshToken"]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )
    tasks: Mapped[list["TaskRecord"]] = relationship(back_populates="user")
    templates: Mapped[list["PromptTemplate"]] = relationship(
        back_populates="user", cascade="all, delete-orphan"
    )


class UserOAuthAccount(Base):
    """OAuth 关联账号"""

    __tablename__ = "user_oauth_accounts"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="CASCADE"), index=True
    )
    provider: Mapped[str] = mapped_column(String(20))  # github, google
    provider_user_id: Mapped[str] = mapped_column(String(100))
    provider_email: Mapped[str | None] = mapped_column(String(255), nullable=True)
    access_token: Mapped[str | None] = mapped_column(Text, nullable=True)
    refresh_token: Mapped[str | None] = mapped_column(Text, nullable=True)
    raw_data: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=_utcnow, onupdate=_utcnow
    )

    user: Mapped["User"] = relationship(back_populates="oauth_accounts")


class EmailVerificationToken(Base):
    """邮箱验证令牌"""

    __tablename__ = "email_verification_tokens"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="CASCADE"), index=True
    )
    token: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    token_type: Mapped[str] = mapped_column(String(20))  # email_verify, password_reset
    expires_at: Mapped[datetime] = mapped_column(DateTime)
    used_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)


class UserRefreshToken(Base):
    """Refresh Token 白名单"""

    __tablename__ = "user_refresh_tokens"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    user_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="CASCADE"), index=True
    )
    token_hash: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    device_info: Mapped[str | None] = mapped_column(String(255), nullable=True)
    ip_address: Mapped[str | None] = mapped_column(String(45), nullable=True)
    expires_at: Mapped[datetime] = mapped_column(DateTime)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)

    user: Mapped["User"] = relationship(back_populates="refresh_tokens")


class OAuthState(Base):
    """OAuth CSRF 防护 state"""

    __tablename__ = "oauth_states"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    state: Mapped[str] = mapped_column(String(64), unique=True, index=True)
    provider: Mapped[str] = mapped_column(String(20))
    expires_at: Mapped[datetime] = mapped_column(DateTime)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)


class TaskRecord(Base):
    __tablename__ = "tasks"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    user_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="SET NULL"), nullable=True, index=True
    )
    status: Mapped[str] = mapped_column(String(20), default="pending")
    original_prompt: Mapped[str] = mapped_column(Text, default="")
    enhanced_prompt: Mapped[str | None] = mapped_column(Text, nullable=True)
    template_images: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    config: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    template_analysis: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    total_rounds: Mapped[int] = mapped_column(Integer, default=0)
    total_images: Mapped[int] = mapped_column(Integer, default=0)
    best_score: Mapped[float] = mapped_column(Float, default=0.0)
    error: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=_utcnow, onupdate=_utcnow
    )
    completed_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)

    rounds: Mapped[list["RoundRecord"]] = relationship(
        back_populates="task", cascade="all, delete-orphan", order_by="RoundRecord.round_number"
    )
    images: Mapped[list["ImageRecord"]] = relationship(
        back_populates="task", cascade="all, delete-orphan"
    )
    user: Mapped["User | None"] = relationship(back_populates="tasks")


class RoundRecord(Base):
    __tablename__ = "rounds"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    task_id: Mapped[str] = mapped_column(String(36), ForeignKey("tasks.id", ondelete="CASCADE"))
    round_number: Mapped[int] = mapped_column(Integer)
    best_score: Mapped[float] = mapped_column(Float, default=0.0)
    prompt_used: Mapped[str] = mapped_column(Text, default="")
    feedback: Mapped[str | None] = mapped_column(Text, nullable=True)

    task: Mapped["TaskRecord"] = relationship(back_populates="rounds")
    images: Mapped[list["ImageRecord"]] = relationship(
        back_populates="round", cascade="all, delete-orphan"
    )


class ImageRecord(Base):
    __tablename__ = "images"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    round_id: Mapped[int | None] = mapped_column(
        Integer, ForeignKey("rounds.id", ondelete="CASCADE"), nullable=True
    )
    task_id: Mapped[str] = mapped_column(String(36), ForeignKey("tasks.id", ondelete="CASCADE"))
    url: Mapped[str] = mapped_column(Text, default="")
    local_path: Mapped[str | None] = mapped_column(Text, nullable=True)
    prompt: Mapped[str] = mapped_column(Text, default="")
    provider: Mapped[str] = mapped_column(String(20), default="gemini")
    round_number: Mapped[int] = mapped_column(Integer, default=1)
    score: Mapped[float] = mapped_column(Float, default=0.0)
    score_details: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    metadata_: Mapped[dict | None] = mapped_column("metadata", JSON, nullable=True)
    is_best: Mapped[bool] = mapped_column(Boolean, default=False)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)

    round: Mapped["RoundRecord | None"] = relationship(back_populates="images")
    task: Mapped["TaskRecord"] = relationship(back_populates="images")


# ========== Video Generation Models ==========


class VideoTaskRecord(Base):
    """Video generation task record."""

    __tablename__ = "video_tasks"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    user_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="SET NULL"), nullable=True, index=True
    )
    status: Mapped[str] = mapped_column(String(20), default="pending")
    original_prompt: Mapped[str] = mapped_column(Text, default="")
    enhanced_prompt: Mapped[str | None] = mapped_column(Text, nullable=True)
    start_image: Mapped[str | None] = mapped_column(Text, nullable=True)
    end_image: Mapped[str | None] = mapped_column(Text, nullable=True)
    config: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    total_rounds: Mapped[int] = mapped_column(Integer, default=0)
    total_videos: Mapped[int] = mapped_column(Integer, default=0)
    error: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=_utcnow, onupdate=_utcnow
    )
    completed_at: Mapped[datetime | None] = mapped_column(DateTime, nullable=True)

    rounds: Mapped[list["VideoRoundRecord"]] = relationship(
        back_populates="task",
        cascade="all, delete-orphan",
        order_by="VideoRoundRecord.round_number",
    )
    videos: Mapped[list["VideoRecord"]] = relationship(
        back_populates="task", cascade="all, delete-orphan"
    )


class VideoRoundRecord(Base):
    """Video generation round record."""

    __tablename__ = "video_rounds"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    task_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("video_tasks.id", ondelete="CASCADE")
    )
    round_number: Mapped[int] = mapped_column(Integer)

    task: Mapped["VideoTaskRecord"] = relationship(back_populates="rounds")
    videos: Mapped[list["VideoRecord"]] = relationship(
        back_populates="round", cascade="all, delete-orphan"
    )


class VideoRecord(Base):
    """Generated video record."""

    __tablename__ = "video_records"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    round_id: Mapped[int | None] = mapped_column(
        Integer, ForeignKey("video_rounds.id", ondelete="CASCADE"), nullable=True
    )
    task_id: Mapped[str] = mapped_column(
        String(36), ForeignKey("video_tasks.id", ondelete="CASCADE")
    )
    url: Mapped[str] = mapped_column(Text, default="")
    local_path: Mapped[str | None] = mapped_column(Text, nullable=True)
    video_type: Mapped[str] = mapped_column(String(10), default="mp4")
    duration: Mapped[int | None] = mapped_column(Integer, nullable=True)
    round_number: Mapped[int] = mapped_column(Integer, default=1)
    provider: Mapped[str] = mapped_column(String(20), default="kling")
    provider_task_id: Mapped[str | None] = mapped_column(String(100), nullable=True)
    metadata_: Mapped[dict | None] = mapped_column("metadata", JSON, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)

    round: Mapped["VideoRoundRecord | None"] = relationship(back_populates="videos")
    task: Mapped["VideoTaskRecord"] = relationship(back_populates="videos")


class PromptTemplate(Base):
    """Prompt template model."""

    __tablename__ = "prompt_templates"

    id: Mapped[str] = mapped_column(String(36), primary_key=True)
    user_id: Mapped[str | None] = mapped_column(
        String(36), ForeignKey("users.user_id", ondelete="CASCADE"),
        index=True, nullable=True,
    )
    title: Mapped[str] = mapped_column(String(200))
    description: Mapped[str | None] = mapped_column(Text, nullable=True)
    prompt_text: Mapped[str] = mapped_column(Text)
    category: Mapped[str] = mapped_column(String(50), default="other", index=True)
    tags: Mapped[list | None] = mapped_column(JSON, default=list)
    preview_image_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
    config: Mapped[dict | None] = mapped_column(JSON, nullable=True)
    usage_count: Mapped[int] = mapped_column(Integer, default=0)
    is_public: Mapped[bool] = mapped_column(Boolean, default=False)
    is_system: Mapped[bool] = mapped_column(Boolean, default=False)
    source_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
    source_author: Mapped[str | None] = mapped_column(String(100), nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime, default=_utcnow)
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, default=_utcnow, onupdate=_utcnow
    )

    user: Mapped["User"] = relationship(back_populates="templates")
