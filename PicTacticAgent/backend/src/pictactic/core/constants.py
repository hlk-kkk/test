"""Constants and default values for PicTacticAgent."""

# Generation defaults
DEFAULT_MAX_ROUNDS = 3
DEFAULT_IMAGES_PER_ROUND = 5
DEFAULT_QUALITY_THRESHOLD = 0.7

# Image settings
SUPPORTED_ASPECT_RATIOS = ["1:1", "16:9", "9:16", "4:3", "3:4"]
SUPPORTED_IMAGE_SIZES = ["1K", "2K", "4K"]

# Task statuses
STATUS_PENDING = "pending"
STATUS_ENHANCING = "enhancing"
STATUS_GENERATING = "generating"
STATUS_REVIEWING = "reviewing"
STATUS_COMPLETED = "completed"
STATUS_FAILED = "failed"
STATUS_CANCELLED = "cancelled"

# File extensions
ALLOWED_IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".gif", ".webp"}

# API rate limits
MAX_CONCURRENT_GENERATIONS = 5
GENERATION_TIMEOUT_SECONDS = 120
GENERATION_HTTP_TIMEOUT = 300.0
IMAGE_DOWNLOAD_TIMEOUT = 10.0

# Cache TTL
CACHE_TTL_TEMPLATE_ANALYSIS = 3600  # 1 hour
CACHE_TTL_TASK_PROGRESS = 60  # 1 minute

# LLM settings
MAX_REFERENCE_IMAGES = 3
