---
name: pictactic
description: |
  AI image/video generation and editing tool (PicTacticAgent). Supports text-to-image, reference-based style transfer, editing existing images, and video generation.
  Triggers:
  (1) Generate images from prompts - "generate an image...", "draw me...", "create a picture..."
  (2) Generate style-consistent images from reference - "generate based on this image...", "match this style..."
  (3) Batch generate candidate images - "generate a few... let me pick"
  (4) Edit existing images - "modify this image...", "change this picture to...", "edit the image..."
  (5) Generate videos - "generate a video...", "create a video...", "make a video..."
  (6) Use /pictactic command
allowed-tools: Bash(uv:*)
---

# PicTactic Image Generation Skill

## Project Path Configuration

**Note**: All commands must be run via `uv run pictactic` from the PicTacticAgent project directory.

```bash
# Replace PROJECT_DIR with the actual project path
PROJECT_DIR=/path/to/PicTacticAgent
```

## Async Workflow (Recommended for Agent Use)

The async workflow submits tasks to the backend, returns immediately with a task_id, and allows polling for status/results. This prevents timeouts during long-running generation tasks.

### Step 1: Submit Task

```bash
# Submit a generation task
cd $PROJECT_DIR && uv run pictactic submit generate --format quiet "a cute cat under cherry blossoms"
# → outputs task_id (single line, exits immediately)
# Auto-starts backend if not running

# With full options
cd $PROJECT_DIR && uv run pictactic submit generate \
  [--rounds 3] [--images 5] [--top-k 3] [--threshold 0.7] \
  [--aspect-ratio 16:9] [--size 2K] [--provider jiekouai] \
  [--no-enhance] [--no-ai-scoring] \
  [--template /path/to/ref.jpg] \
  --format quiet \
  "<prompt>"

# Submit an edit task
cd $PROJECT_DIR && uv run pictactic submit edit \
  --source /path/to/image.png \
  [--count 1] [--aspect-ratio 16:9] [--size 2K] \
  [--provider gemini] [--enhance] \
  --format quiet \
  "<edit instruction>"
```

### Step 2: Poll Status (repeat until status=completed/failed)

```bash
cd $PROJECT_DIR && uv run pictactic status <task_id> --format json
# → {"task_id":"...","status":"generating","current_round":2,"images_generated":8,...}
```

```bash
# Compact format for scripting
cd $PROJECT_DIR && uv run pictactic status <task_id> --format quiet
# → generating 2/3 8 0.85
# Format: status round/max images_count best_score
```

### Step 3: Get Results (when status=completed)

```bash
cd $PROJECT_DIR && uv run pictactic result <task_id> --output-dir ./output --format quiet
# → outputs image local paths, one per line
```

### Quick Async Example (full flow)

```bash
# 1. Submit
TASK_ID=$(cd $PROJECT_DIR && uv run pictactic submit generate --format quiet "a majestic dragon")

# 2. Poll (check periodically)
cd $PROJECT_DIR && uv run pictactic status $TASK_ID --format json

# 3. Download when done
cd $PROJECT_DIR && uv run pictactic result $TASK_ID --output-dir ./output --format quiet
```

## Direct Mode (blocks until done, for interactive/quick use)

### Text-to-Image (generate command)

> **Important**: `prompt` must come AFTER all `--options`, otherwise typer parsing will fail.

```bash
cd $PROJECT_DIR && uv run pictactic generate \
  [--rounds 3] \          # Max iteration rounds (1-10, default 3)
  [--images 5] \          # Candidate images per round (1-10, default 5)
  [--top-k 3] \           # Number of top-scored images to output (default 3)
  [--threshold 0.7] \     # Quality threshold, auto-stop when reached (0-1, default 0.7)
  [--aspect-ratio 16:9] \ # Aspect ratio: 1:1 / 16:9 / 9:16 / 4:3 / 3:4
  [--size 2K] \           # Image size: 1K / 2K / 4K
  [--provider jiekouai] \ # Specify provider (optional)
  [--no-enhance] \        # Disable LLM prompt auto-enhancement
  [--no-ai-scoring] \     # Disable AI image scoring
  [--output-dir ./output] \ # Image save directory
  [--format quiet] \      # Output format: text (default) / json / quiet (paths only)
  "<prompt>"              # <- prompt must come last
```

**Quickest invocation**:
```bash
cd $PROJECT_DIR && uv run pictactic generate --format quiet "a cute cat under cherry blossoms"
```

### Reference Image Generation

```bash
cd $PROJECT_DIR && uv run pictactic generate \
  --template /path/to/reference_image.jpg \
  --rounds 3 --images 4 --format quiet \
  "<prompt>"
```

### Image Editing (edit command)

```bash
cd $PROJECT_DIR && uv run pictactic edit \
  --source /path/to/image.png \
  [--count 1] [--aspect-ratio 16:9] [--size 2K] \
  [--provider gemini] [--enhance] \
  [--output-dir ./output] [--format quiet] \
  "<edit instruction>"
```

## Output Formats

| Format | Description |
|--------|-------------|
| `--format quiet` | Output only final image paths, one per line (recommended for scripting) |
| `--format json` | JSONL event stream with progress, scores, and structured data |
| `--format text` | Rich formatted UI with progress and score details (default) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Argument error (file not found, unsupported provider, etc.) |
| 2 | Generation/edit failed (provider error, API error, etc.) |
| 3 | No valid results / task not completed |

## Parameter Reference

### submit generate / generate command

| Parameter | Short | Default | Description |
|-----------|-------|---------|-------------|
| `prompt` | - | Required | Image description prompt |
| `--template` | `-t` | None | Reference image path (can use multiple, max 3) |
| `--rounds` | `-r` | 3 | Max iteration rounds |
| `--images` | `-n` | 5 | Candidate images per round |
| `--top-k` | `-k` | 3 | Number of top-scored images to output |
| `--threshold` | `-q` | 0.7 | Quality threshold |
| `--aspect-ratio` | `-a` | 16:9 | Aspect ratio |
| `--size` | `-s` | 2K | Image size |
| `--provider` | `-p` | .env default | Specify provider |
| `--no-enhance` | - | False | Disable prompt auto-enhancement |
| `--no-ai-scoring` | - | False | Disable AI scoring |
| `--output-dir` | `-o` | ./output | Local save directory |
| `--format` | `-f` | text/quiet | Output format (text/json/quiet) |

### submit edit / edit command

| Parameter | Short | Default | Description |
|-----------|-------|---------|-------------|
| `prompt` | - | Required | Edit instruction |
| `--source` | `-s` | Required | Source image path or URL (can use multiple) |
| `--count` | `-n` | 1 | Number of outputs |
| `--aspect-ratio` | `-a` | 16:9 | Aspect ratio |
| `--size` | - | 2K | Image size |
| `--provider` | `-p` | .env default | Provider (gemini/jiekouai, antigravity not supported) |
| `--enhance` | - | False | Enable LLM enhancement for edit instructions |
| `--output-dir` | `-o` | ./output | Local save directory |
| `--format` | `-f` | text/quiet | Output format (text/json/quiet) |

## Video Generation

### Direct Mode (blocks until done)

```bash
cd $PROJECT_DIR && uv run pictactic video \
  [--start-image /path/to/start.jpg] \  # Start frame (enables i2v mode)
  [--end-image /path/to/end.jpg] \      # End frame (Kling, Veo only)
  [--rounds 2] \                        # Generation rounds (1-5, default 2)
  [--videos 2] \                        # Videos per round (1-4, default 2)
  [--duration 5] \                      # Duration in seconds (default 5)
  [--resolution 720p] \                 # Resolution: 720p, 1080p
  [--aspect-ratio 16:9] \              # Aspect ratio
  [--audio] \                           # Generate audio (Kling, Veo)
  [--provider kling] \                  # Provider: kling, sora, veo
  [--no-enhance] \                      # Disable prompt enhancement
  [--output-dir ./output] \
  [--format quiet] \
  "<prompt>"                            # <- prompt must come last
```

**Quickest invocation**:
```bash
cd $PROJECT_DIR && uv run pictactic video --format quiet "a cat playing piano"
```

### Async Mode (submit to backend)

```bash
# Submit video task
TASK_ID=$(cd $PROJECT_DIR && uv run pictactic submit video --format quiet "ocean waves at sunset")

# Check status (note --video flag)
cd $PROJECT_DIR && uv run pictactic status --video $TASK_ID --format json
# Or compact: status round/max videos_count [progress%]
cd $PROJECT_DIR && uv run pictactic status --video $TASK_ID --format quiet
# → generating 1/2 2 45%

# Download results when done
cd $PROJECT_DIR && uv run pictactic result --video $TASK_ID --output-dir ./output --format quiet
```

### Video Parameter Reference

| Parameter | Short | Default | Description |
|-----------|-------|---------|-------------|
| `prompt` | - | Required | Video description prompt |
| `--start-image` | `-i` | None | Start frame image (enables i2v) |
| `--end-image` | `-e` | None | End frame image (Kling, Veo only) |
| `--rounds` | `-r` | 2 | Generation rounds |
| `--videos` | `-n` | 2 | Videos per round |
| `--duration` | `-d` | 5 | Duration in seconds |
| `--resolution` | - | None | 720p or 1080p |
| `--aspect-ratio` | `-a` | 16:9 | Aspect ratio |
| `--audio` | - | False | Generate audio |
| `--provider` | `-p` | .env default | kling, sora, veo |
| `--no-enhance` | - | False | Disable prompt enhancement |
| `--output-dir` | `-o` | ./output | Output directory |
| `--format` | `-f` | text | text/json/quiet |

### Video Provider Capabilities

| Provider | Modes | End Frame | Audio | Duration |
|----------|-------|-----------|-------|----------|
| Kling | t2v + i2v | Yes | Yes | 3-15s (range) |
| Sora | t2v + i2v | No | No | 4/8/12s |
| Veo | t2v + i2v | Yes | Yes | 4/6/8s |

## Other Commands

```bash
# List available providers and supported modes
cd $PROJECT_DIR && uv run pictactic providers

# Check image task status
cd $PROJECT_DIR && uv run pictactic status <task_id> --format json

# Check video task status
cd $PROJECT_DIR && uv run pictactic status --video <task_id> --format json
```

## Recommended Parameters by Scenario

| Scenario | Recommended Parameters |
|----------|----------------------|
| Quick preview | `--rounds 1 --images 3 --no-ai-scoring` |
| Standard generation | `--rounds 3 --images 5` (default) |
| High quality | `--rounds 5 --images 8 --threshold 0.85` |
| Reference style transfer | `--template ref.jpg --rounds 3 --images 4` |
| Phone wallpaper | `--aspect-ratio 9:16 --size 2K` |
| Banner cover | `--aspect-ratio 16:9 --size 2K` |
| Quick image edit | `edit --source img.png --format quiet "instruction"` |
| Quick video | `video --format quiet "prompt"` |
| Video with start frame | `video --start-image img.jpg --provider kling "prompt"` |
| Video with audio | `video --audio --provider veo "prompt"` |

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| Invalid API key | .env misconfigured | Check `JIEKOUAI_API_KEY` in `$PROJECT_DIR/.env` |
| Provider doesn't support image_to_image | edit command used with antigravity | Switch to jiekouai or gemini |
| Source image not found | Wrong source path | Verify `--source` path exists |
| Output directory not writable | Permission issue | Specify `--output-dir` to a writable directory |
| uv command not found | uv not installed | `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| 认证失败 (HTTP 401) | API Key mismatch | Restart backend to regenerate INTERNAL_API_KEY |
