# PicTacticAgent

**English** | [中文](README.md)

AI image generation tool — multi-round generation + auto evaluation + iterative refinement. No more "rolling the dice" with text-to-image. Ships with **199 curated prompt templates** and works as a Skill for Claude Code / OpenClaw and other AI agents.

<table>
  <tr>
    <td><img src="docs/screenshots/workspace-with-history.png" alt="Workspace" /></td>
    <td><img src="docs/screenshots/template-library.png" alt="Template Library" /></td>
  </tr>
  <tr>
    <td align="center"><b>Workspace</b> — Generate + Score + History</td>
    <td align="center"><b>Template Library</b> — 199 curated prompts, one-click generate</td>
  </tr>
</table>

## Highlights

- **199 Curated Templates** — Selected from 2,000+ X/Twitter prompts via Vision LLM tournament, spanning 6 categories: cinematic, fashion, portrait, product, artistic, surreal
- **Multi-Round Iteration** — Generates multiple candidates per round, AI scores on 6 dimensions, iterates until quality threshold is met
- **Skill Integration** — One command to turn it into an image generation capability for Claude Code / OpenClaw — just tell the AI "draw me a picture"
- **Multiple Interfaces** — Web UI / CLI / REST API, pick what fits your workflow
- **Multi-Provider** — Gemini Web (free, cookie auth), Gemini, Jiekouai, Antigravity — switch with one config change

## Quick Start

```bash
# 1. Clone
git clone https://github.com/NanmiCoder/PicTacticAgent.git
cd PicTacticAgent

# 2. Configure API key (edit .env, pick a provider and fill in the key)
cp .env.example .env

# 3. One-click start
./start.sh
```

Visit http://localhost:3019 to get started.

> Provider recommendation: [Jiekouai](https://jiekou.ai/referral?invited_code=3CF8T0) — one API key covers both image generation and LLM evaluation. Sign up with GitHub for $3 free credits.

## Skill Integration

PicTacticAgent works as a **Skill** for AI agents, giving them image generation and editing capabilities.

### Claude Code

Let Claude Code read the install doc and it will handle the setup:

```
@docs/skill-installation.md Follow the guide to install PicTacticAgent Skill
```

Then just say "generate a cyberpunk cover image" and it works.

### OpenClaw

A `SKILL.md` is provided at the project root for OpenClaw skill discovery:

```bash
# Install in OpenClaw
clawhub install pictactic

# Or manually copy SKILL.md to your OpenClaw workspace
```

Once installed, trigger it from any messaging channel (Telegram / Slack / Discord etc.) by saying "generate an image of ...".

### Other Agents

Any agent that supports CLI calls can integrate via `uv run pictactic`:

```bash
uv run pictactic generate "a tech-inspired cover image"
uv run pictactic edit --source ./image.png "change the background to dark blue"
```

## CLI Quick Reference

```bash
# Text-to-image
uv run pictactic generate "your prompt"

# With options
uv run pictactic generate --rounds 3 --images 5 --provider gemini "prompt"

# Edit an image
uv run pictactic edit --source ./img.png "edit instruction"

# Async mode (recommended for agents)
uv run pictactic submit generate --format quiet "prompt"
uv run pictactic status <task_id>
uv run pictactic result <task_id> --output-dir ./output

# List providers
uv run pictactic providers
```

> Note: the prompt argument must come **after** all options.

## Tech Stack

FastAPI + LangGraph (StateGraph) | React 19 + TailwindCSS 4 | SQLite | Gemini Web / Gemini / Jiekouai / Antigravity | Typer CLI

## Documentation

| Doc | Description |
|-----|-------------|
| [Configuration](docs/CONFIGURATION.md) | Provider selection, API key setup |
| [Deployment](docs/DEPLOYMENT.md) | Docker / Linux production deployment |
| [Skill Installation](docs/skill-installation.md) | Claude Code Skill setup guide |
| [Technical Design](docs/TECHNICAL_DESIGN.md) | Architecture, workflow, data models |
| [API Docs](http://localhost:8019/docs) | Interactive API docs (after startup) |

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!
