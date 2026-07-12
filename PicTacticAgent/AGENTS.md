# Repository Guidelines

## Project Structure & Module Organization
`backend/src/pictactic/` contains the FastAPI app and core logic:
- `api/` routes, dependencies, and websocket handlers
- `agents/` LangGraph workflow (`nodes/`, `tools/`, `workflow.py`)
- `services/`, `providers/`, `db/`, `models/`, `core/`

`frontend/src/` is the Vite + React UI:
- `pages/` route-level screens
- `components/` reusable UI blocks
- `hooks/` shared state/data hooks (`useAuth`, `useGeneration`, etc.)
- `services/` API/auth clients and `locales/` i18n files

Tests live in `tests/unit/`, `tests/integration/`, and `tests/e2e/`. Runtime data is under `data/` and `storage/`; docs are in `docs/`.

## Build, Test, and Development Commands
- `./start.sh`: one-command startup for backend (`:8019`) and frontend (`:3019`).
- `uv sync`: install/sync Python dependencies.
- `uv run uvicorn backend.src.pictactic.api.app:app --reload --port 8019`: run backend only.
- `cd frontend && npm install && npm run dev`: run frontend only.
- `uv run pytest`: run all Python tests.
- `uv run pytest tests/unit -v` or `uv run pytest tests/integration -v`: run targeted suites.
- `uv run ruff check .` and `uv run mypy backend/src`: lint and type-check backend.
- `cd frontend && npm run lint && npm run build`: lint and production-build frontend.

## Coding Style & Naming Conventions
Python style is governed by Ruff (`line-length = 100`, Python 3.11). Use 4-space indentation, `snake_case` for functions/modules, and `PascalCase` for classes/schemas.

Frontend uses ESLint (React Hooks + Vite config). Use 2-space indentation, `PascalCase` for components/pages (`LoginPage.jsx`), and `camelCase` for hooks/utilities (`useTaskHistory.js`).

## Testing Guidelines
Pytest is the test framework (`testpaths = tests`, file pattern `test_*.py`). Place tests in the matching layer directory (`unit`, `integration`, `e2e`) and keep names behavior-focused (for example, `test_auth_service.py`).

No fixed coverage gate is configured; new features should include unit tests and update integration/e2e tests when API or workflow behavior changes.

## Commit & Pull Request Guidelines
Follow the existing commit style from history: Conventional Commit prefixes such as `feat:`, `fix:`, `docs:`, `chore:`. Keep each commit scoped to one change.

For PRs, include:
- clear summary and motivation
- linked issue(s) when applicable
- test evidence (commands + results)
- screenshots/GIFs for UI changes
- notes on `.env` or migration impacts (for example, `data/pictactic.db` changes)

## Security & Configuration Tips
Copy `.env.example` to `.env` and never commit real keys. Validate OAuth/SMTP/JWT settings locally before opening PRs, and use non-production secrets in development.
