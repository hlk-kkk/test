#!/bin/bash

# PicTacticAgent 一键启动脚本

set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$PROJECT_DIR"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 启动 PicTacticAgent${NC}"
echo ""

# 检查 .env 文件
if [ ! -f ".env" ]; then
    echo -e "${YELLOW}⚠️  未找到 .env 文件，从模板创建...${NC}"
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo -e "${YELLOW}   请编辑 .env 文件填写 API 密钥${NC}"
    fi
fi

# 检查 uv
if ! command -v uv &> /dev/null; then
    echo "❌ 未安装 uv，请先安装: curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi

# 检查 node
if ! command -v node &> /dev/null; then
    echo "❌ 未安装 Node.js，请先安装 Node.js 18+"
    exit 1
fi

# 同步后端依赖
echo -e "${GREEN}📦 同步后端依赖...${NC}"
uv sync --quiet

# 安装前端依赖（如果需要）
if [ ! -d "frontend/node_modules" ]; then
    echo -e "${GREEN}📦 安装前端依赖...${NC}"
    cd frontend && npm install --silent && cd ..
fi

echo ""
echo -e "${GREEN}✅ 依赖准备完成${NC}"
echo ""
echo -e "   后端: ${YELLOW}http://localhost:8019${NC}"
echo -e "   前端: ${YELLOW}http://localhost:3019${NC}"
echo -e "   API 文档: ${YELLOW}http://localhost:8019/docs${NC}"
echo ""
echo -e "${GREEN}按 Ctrl+C 停止所有服务${NC}"
echo ""

# 启动后端（后台）
uv run uvicorn backend.src.pictactic.api.app:app --reload --port 8019 &
BACKEND_PID=$!

# 启动前端（后台）
cd frontend && npm run dev &
FRONTEND_PID=$!

# 捕获退出信号，清理进程
cleanup() {
    echo ""
    echo -e "${YELLOW}🛑 停止服务...${NC}"
    kill $BACKEND_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    # 确保子进程也被终止
    pkill -P $BACKEND_PID 2>/dev/null || true
    pkill -P $FRONTEND_PID 2>/dev/null || true
    echo -e "${GREEN}✅ 已停止${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

# 等待进程
wait
