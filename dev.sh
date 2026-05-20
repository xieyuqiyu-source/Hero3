#!/bin/bash
# Hero3 一键启动脚本 - 同时运行 Go 后端和 Web 前端

set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

cleanup() {
  echo ""
  echo "正在停止所有服务..."
  kill $GO_PID $WEB_PID 2>/dev/null || true
  wait $GO_PID $WEB_PID 2>/dev/null || true
  echo "已停止。"
}

trap cleanup EXIT INT TERM

# 启动 Go 后端
echo "🚀 启动 Go 后端..."
cd "$ROOT_DIR/go"
go run ./cmd/server &
GO_PID=$!

# 启动 Web 前端
echo "🚀 启动 Web 前端..."
cd "$ROOT_DIR/web"
pnpm dev &
WEB_PID=$!

echo ""
echo "✅ Hero3 开发环境已启动"
echo "   前端: http://localhost:5173"
echo "   后端: http://localhost:8080"
echo ""
echo "按 Ctrl+C 停止所有服务"

wait
