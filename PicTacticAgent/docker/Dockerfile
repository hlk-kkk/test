# Dockerfile for PicTacticAgent backend

FROM python:3.11-slim-bullseye

# Build args for mirror configuration (optional, for China servers)
ARG APT_MIRROR=""
ARG PIP_INDEX_URL=""
ARG PIP_TRUSTED_HOST=""

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

# Optionally replace apt mirror
RUN if [ -n "$APT_MIRROR" ]; then \
      sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list 2>/dev/null || \
      sed -i "s|deb.debian.org|${APT_MIRROR}|g" /etc/apt/sources.list.d/debian.sources 2>/dev/null || true; \
    fi

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy project files needed for installation
COPY pyproject.toml README.md ./
COPY backend/ ./backend/

# Install Python dependencies (use mirror if provided)
RUN if [ -n "$PIP_INDEX_URL" ]; then \
      pip install --no-cache-dir -i "$PIP_INDEX_URL" --trusted-host "$PIP_TRUSTED_HOST" .; \
    else \
      pip install --no-cache-dir .; \
    fi

# Create storage directories
RUN mkdir -p /app/storage/images /app/storage/videos /app/data

# Expose port
EXPOSE 8019

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8019/health || exit 1

# Run the application
CMD ["uvicorn", "pictactic.api.app:app", "--host", "0.0.0.0", "--port", "8019"]
