# Repository Guidelines

## Project Structure & Module Organization
- `backend/`: FastAPI backend. Key areas: `api/` (routers), `services/` (orchestration/external calls), `models/` + `schemas.py` (data + Pydantic schemas), `scripts/` (CLI runners), `utils/`, and `tests/` (pytest). Configuration lives in `backend/config.py`.
- `frontend/`: Vue dashboard. Source under `src/` with `src/api/` (HTTP), `src/components/`, `src/stores/`, and `src/views/`. Static assets in `public/`.
- `docs/`: project documentation. Avoid editing generated artifacts like `frontend/dist/` or `frontend/node_modules/` by hand.

## Build, Test, and Development Commands
Run from repo root unless noted.
- `uv sync --extra dev`: install backend dependencies.
- `uv run python content_remix_agent/backend/scripts/run_api_server.py --reload`: start the API server with reload.
- `uv run pytest content_remix_agent/backend/tests`: run backend test suite.
- `uv run pytest content_remix_agent/backend/tests/test_api.py::test_health_check`: run a single test.
- From `frontend/`: `npm install`, `npm run dev` (local UI), `npm run build` (type-check + build).

## Coding Style & Naming Conventions
- Python: 4-space indent, `snake_case` for modules/functions, `PascalCase` for classes, `UPPER_SNAKE_CASE` for constants. Prefer type hints on public APIs and `Path` for filesystem work. Use `loguru` `logger` instead of `print`.
- Keep side effects in `backend/scripts/` or `backend/services/`; keep data models in `schemas.py`.
- Frontend: TypeScript required, avoid `any`, PascalCase components, `use*` composables, and Tailwind utilities for styling.

## Testing Guidelines
- Backend tests use pytest in `backend/tests/` with `test_*.py` naming.
- Use `uv run pytest -k "keyword"` to filter tests; keep external API calls mocked where possible.
- Frontend regression gate is `npm run build` (includes `vue-tsc`).

## Commit & Pull Request Guidelines
- Commit messages follow Conventional Commits: `feat:`, `fix:`, `refactor:`, `chore:`, `docs:` with optional scopes (e.g., `feat(frontend): ...`).
- PRs should include a short description, testing notes/commands, and screenshots or gifs for UI changes. Link related issues when available.

## Security & Configuration Tips
- Store secrets in `.env` only; never commit credentials.
- Update `backend/config.py` when adding new environment variables.
