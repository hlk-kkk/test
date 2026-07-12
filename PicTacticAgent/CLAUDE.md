# CLAUDE.md

This file provides guidance to Claude Code for working with the PicTacticAgent project.

## Development Commands

### One-click Start
```bash
./start.sh  # Starts both backend (:8019) and frontend (:3019)
```

### Backend (Python, uv)
```bash
uv sync                          # Install/sync dependencies
uv run uvicorn backend.src.pictactic.api.app:app --reload --port 8019
uv run pytest                    # Run tests
uv run ruff check .              # Lint
uv run ruff format .             # Format
uv run mypy backend/             # Type check
```

### Frontend (React, npm)
```bash
cd frontend
npm install
npm run dev      # Dev server on :3019
npm run build    # Production build
npm run lint     # ESLint
```

### CLI (typer + rich)
```bash
# NOTE: prompt must come AFTER all options (typer callback limitation)
uv run pictactic generate "your prompt here"             # Generate images (prompt last)
uv run pictactic generate --format json "your prompt"    # JSON output for scripting
uv run pictactic edit --source ./img.png "instruction"  # Edit existing image
uv run pictactic providers                              # List available providers
uv run pictactic status <task_id>                       # Check task status (requires backend)
```

### Test Configuration
- pytest: `asyncio_mode = "auto"`, test files in `tests/`, pattern `test_*.py`
- ruff: targets Python 3.11, line-length 100, rules: E/F/W/I/N/UP/B/C4/SIM (E501 ignored)

## Architecture Overview

Full-stack app: **FastAPI + LangGraph** backend, **React 19 + TailwindCSS 4** frontend.

```
backend/src/pictactic/
├── agents/          # LangGraph workflow (core generation logic)
│   ├── workflow.py  # StateGraph definition
│   ├── nodes.py     # Node implementations (analyze, enhance, generate, evaluate, finalize)
│   ├── conditions.py# Conditional edges (check_template, should_continue)
│   └── state.py     # GenerationState TypedDict
├── api/
│   ├── app.py       # FastAPI app factory, CORS, middleware
│   ├── websocket.py # WebSocket for real-time progress
│   └── routes/      # health, auth, generation
├── cli/             # CLI entry point (typer + rich), no HTTP needed
│   ├── main.py      # typer app + sub-command registration
│   ├── generate.py  # generate command (direct workflow invocation)
│   ├── providers.py # providers command
│   ├── status.py    # status command (HTTP query)
│   └── output.py    # Output formatting (text/json/quiet)
├── providers/       # Image generation providers (gemini, jiekouai, antigravity)
│   └── base.py      # ImageProvider ABC + GenerationParams/Result models
├── core/
│   └── config.py    # pydantic-settings (reads .env from project root)
├── services/        # Business logic services
├── db/              # SQLAlchemy async + aiosqlite
└── i18n/            # Language detection middleware

frontend/src/        # React 19 SPA (Vite 7, TailwindCSS 4, react-router-dom 7)
```

### Ports & Proxy
- Backend: `localhost:8019` (API docs at `/docs`)
- Frontend: `localhost:3019` (Vite dev proxy forwards `/api`, `/ws`, `/static` to backend)

## LangGraph Workflow

The core workflow in `agents/workflow.py` uses a `StateGraph(GenerationState)`:

```
START → check_template → analyze (if template) → enhance → generate → evaluate
                      → enhance (no template) ↗            ↓
                                                    should_continue
                                                    ↓           ↓
                                               prepare_next   finalize → END
                                                    ↓
                                                 enhance (loop)
```

**State** (`GenerationState`): TypedDict with user_prompt, template_images, enhanced_prompt, generated_images, best_images, current_round, config fields (max_rounds, images_per_round, provider, etc.)

## Key Patterns

### Provider Abstraction
`ImageProvider` ABC in `providers/base.py` — each provider implements `generate()` and optionally `generate_batch()`. Providers: `gemini` (official SDK), `jiekouai` (reverse proxy), `antigravity` (OpenAI-compatible proxy). Selected via `default_provider` setting or per-request.

### Auth
JWT-based auth with bcrypt passwords. OAuth support: GitHub, Google. Routes at `/api/v1/auth/`.

### WebSocket Progress
`/ws/{task_id}` — `ConnectionManager` tracks per-task connections. Generation service pushes progress events as images complete.

### Configuration
`pydantic-settings` (`Settings` class) loads from `.env` at project root. All API keys, model names, generation defaults, JWT config, and OAuth credentials are env-configured.

### Frontend State
React 19 with `react-router-dom` v7. Path alias `@/` maps to `frontend/src/`. UI built with TailwindCSS 4 + lucide-react icons + class-variance-authority.
