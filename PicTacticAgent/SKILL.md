---
name: pictactic
version: 1.0.0
description: AI image/video generation and editing. Multi-round generation with auto evaluation for best results. 199 curated prompt templates included.
author: NanmiCoder
tags:
  - image-generation
  - ai-art
  - text-to-image
  - image-editing
  - video-generation
triggers:
  - "generate an image"
  - "draw me"
  - "create a picture"
  - "edit this image"
  - "generate a video"
  - "make a cover"
---

# PicTactic — AI Image Generation Skill

Generate, evaluate, and iterate AI images from text prompts. Supports text-to-image, image editing, reference-based style transfer, and video generation.

## Setup

```bash
# 1. Install
git clone https://github.com/NanmiCoder/PicTacticAgent.git
cd PicTacticAgent && uv sync

# 2. Configure (pick a provider, fill in API key)
cp .env.example .env

# 3. Verify
uv run pictactic providers
```

> See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for provider details.

## Commands

```bash
PROJECT_DIR=/path/to/PicTacticAgent

# Generate image
cd $PROJECT_DIR && uv run pictactic generate --format quiet "a cyberpunk city at night"

# Edit image
cd $PROJECT_DIR && uv run pictactic edit --source ./img.png --format quiet "change background to sunset"

# Generate video
cd $PROJECT_DIR && uv run pictactic video --format quiet "ocean waves at golden hour"

# Async mode (for long tasks)
TASK_ID=$(cd $PROJECT_DIR && uv run pictactic submit generate --format quiet "prompt")
cd $PROJECT_DIR && uv run pictactic status $TASK_ID --format quiet
cd $PROJECT_DIR && uv run pictactic result $TASK_ID --output-dir ./output --format quiet
```

## Key Options

| Option | Default | Description |
|--------|---------|-------------|
| `--rounds N` | 3 | Iteration rounds (1-10) |
| `--images N` | 5 | Candidates per round (1-10) |
| `--top-k N` | 3 | Final output count |
| `--aspect-ratio` | 16:9 | 1:1, 16:9, 9:16, 4:3, 3:4 |
| `--size` | 2K | 1K, 2K, 4K |
| `--format quiet` | text | Output paths only (for scripting) |

## Output

- `--format quiet`: one image/video path per line
- `--format json`: structured JSONL with scores and metadata
- Exit code 0 = success, 2 = generation failed, 3 = not completed

## Notes

- Prompt argument must come **after** all options
- Image editing requires `gemini` or `jiekouai` provider
- Video supports `kling`, `sora`, `veo` providers
