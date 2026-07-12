#!/usr/bin/env bash
# CookieBridge Server 一键启动脚本 (macOS / Linux)
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_DIR="$SCRIPT_DIR/server"

echo "=========================================="
echo "  CookieBridge Server 一键启动"
echo "=========================================="

# 检查 uv 是否已安装
if ! command -v uv &> /dev/null; then
    echo "[1/3] 未检测到 uv，正在安装..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    # 让当前 shell 能找到刚安装的 uv
    export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$PATH"
    echo "[1/3] uv 安装完成"
else
    echo "[1/3] uv 已安装: $(uv --version)"
fi

# 安装依赖
echo "[2/3] 安装项目依赖..."
cd "$SERVER_DIR"
uv sync

# 启动服务
echo "[3/3] 启动 CookieBridge Server..."
echo "  地址: http://localhost:${APP_PORT:-8274}"
echo "  按 Ctrl+C 停止服务"
echo "=========================================="
uv run app.py
