"""FastAPI application."""

from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
from starlette.middleware.base import BaseHTTPMiddleware

from ..core.config import settings
from ..i18n import detect_language
from .routes import auth, generation, health, templates, video_generation
from .routes import settings as settings_routes
from .websocket import router as websocket_router


class LanguageMiddleware(BaseHTTPMiddleware):
    """Middleware to detect and store preferred language on request state."""

    async def dispatch(self, request: Request, call_next):
        lang_param = request.query_params.get("lang")
        accept_lang = request.headers.get("accept-language")
        request.state.lang = detect_language(accept_lang, lang_param)
        return await call_next(request)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    import logging
    import secrets

    from ..db.engine import init_db

    # Reject startup with default/weak JWT secret
    if (
        settings.jwt_secret_key == "change-this-secret-key-in-production"
        or len(settings.jwt_secret_key) < 32
        or settings.jwt_secret_key.startswith("<")
    ):
        raise RuntimeError(
            "JWT_SECRET_KEY must be set to a strong random value in production. "
            "Generate one with: openssl rand -base64 32"
        )

    await init_db()

    # Seed system templates
    from ..db.engine import get_session
    from ..db.seed import seed_system_templates

    async with get_session() as session:
        await seed_system_templates(session)

    # Auto-generate internal API key if not set
    if not settings.internal_api_key:
        logger = logging.getLogger(__name__)
        generated_key = secrets.token_urlsafe(32)

        # Write to .env file
        env_path = Path(settings.model_config.get("env_file", ".env"))
        try:
            if env_path.exists():
                content = env_path.read_text()
                if "INTERNAL_API_KEY=" in content:
                    # Replace existing empty value
                    lines = content.splitlines(keepends=True)
                    new_lines = []
                    for line in lines:
                        if line.strip().startswith("INTERNAL_API_KEY="):
                            new_lines.append(f"INTERNAL_API_KEY={generated_key}\n")
                        else:
                            new_lines.append(line)
                    env_path.write_text("".join(new_lines))
                else:
                    # Append to file
                    with env_path.open("a") as f:
                        f.write(f"\nINTERNAL_API_KEY={generated_key}\n")
            else:
                env_path.write_text(f"INTERNAL_API_KEY={generated_key}\n")

            # Update in-memory settings
            settings.__dict__["internal_api_key"] = generated_key
            logger.info("Auto-generated INTERNAL_API_KEY and saved to %s", env_path)
        except OSError:
            logger.warning("Failed to write INTERNAL_API_KEY to %s", env_path, exc_info=True)
            settings.__dict__["internal_api_key"] = generated_key

    yield

    # Shutdown: dispose database engine to flush WAL and release connections
    from ..db.engine import dispose_engine

    await dispose_engine()


def create_app() -> FastAPI:
    """Create and configure the FastAPI application."""
    application = FastAPI(
        title="PicTacticAgent API",
        description="AI-driven cover image generation with multi-agent collaboration",
        version="0.1.0",
        lifespan=lifespan,
    )

    # Configure CORS - allow all localhost ports in development
    application.add_middleware(
        CORSMiddleware,
        allow_origin_regex=r"^https?://(localhost|127\.0\.0\.1)(:\d+)?$",
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Language detection middleware
    application.add_middleware(LanguageMiddleware)

    # Include routers
    application.include_router(health.router, tags=["Health"])
    application.include_router(
        auth.router,
        prefix="/api/v1/auth",
        tags=["Authentication"],
    )
    application.include_router(
        generation.router,
        prefix="/api/v1/generation",
        tags=["Generation"],
    )
    application.include_router(
        templates.router,
        prefix="/api/v1/templates",
        tags=["Templates"],
    )
    application.include_router(
        settings_routes.router,
        prefix="/api/v1/settings",
        tags=["Settings"],
    )
    application.include_router(
        video_generation.router,
        prefix="/api/v1",
        tags=["Video Generation"],
    )
    application.include_router(websocket_router, tags=["WebSocket"])

    # Mount static files for serving generated images
    storage_path = Path(settings.storage_path)
    storage_path.mkdir(parents=True, exist_ok=True)
    application.mount(
        "/static/images", StaticFiles(directory=str(storage_path)), name="static_images"
    )

    # Mount static files for serving generated videos
    video_storage_path = storage_path.parent / "videos"
    video_storage_path.mkdir(parents=True, exist_ok=True)
    application.mount(
        "/static/videos", StaticFiles(directory=str(video_storage_path)), name="static_videos"
    )

    # Mount static files for template preview images
    templates_storage = storage_path.parent / "templates"
    templates_storage.mkdir(parents=True, exist_ok=True)
    application.mount(
        "/static/templates",
        StaticFiles(directory=str(templates_storage)),
        name="static_templates",
    )

    return application


app = create_app()
