#!/bin/bash
# Hero3 一键启动脚本 - 同时运行 Go 后端、玩家前端和 GM 后台

set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

cleanup() {
  echo ""
  echo "正在停止所有服务..."
  kill ${GO_PID:-} ${WEB_PID:-} ${ADMIN_PID:-} ${DB_TUNNEL_PID:-} 2>/dev/null || true
  wait ${GO_PID:-} ${WEB_PID:-} ${ADMIN_PID:-} ${DB_TUNNEL_PID:-} 2>/dev/null || true
  echo "已停止。"
}

trap cleanup EXIT INT TERM

# 启动 Go 后端
echo "🚀 启动 Go 后端..."
cd "$ROOT_DIR/go"
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

if [ "${HERO3_DB_TUNNEL_ENABLED:-false}" = "true" ]; then
  echo "🔌 启动服务器数据库 SSH 隧道..."
  if lsof -nP -iTCP:${HERO3_DB_TUNNEL_LOCAL_PORT:-3307} -sTCP:LISTEN >/dev/null 2>&1; then
    echo "   端口 ${HERO3_DB_TUNNEL_LOCAL_PORT:-3307} 已在监听，复用已有隧道。"
  else
    if [ -n "${HERO3_DB_SSH_PASSWORD:-}" ] && command -v sshpass >/dev/null 2>&1; then
      SSHPASS="$HERO3_DB_SSH_PASSWORD" sshpass -e ssh \
        -o StrictHostKeyChecking=accept-new \
        -N \
        -L "${HERO3_DB_TUNNEL_LOCAL_PORT:-3307}:127.0.0.1:${HERO3_DB_REMOTE_PORT:-3306}" \
        "${HERO3_DB_SSH_USER:-root}@${HERO3_DB_SSH_HOST}" &
    else
      ssh \
        -o StrictHostKeyChecking=accept-new \
        -N \
        -L "${HERO3_DB_TUNNEL_LOCAL_PORT:-3307}:127.0.0.1:${HERO3_DB_REMOTE_PORT:-3306}" \
        "${HERO3_DB_SSH_USER:-root}@${HERO3_DB_SSH_HOST}" &
    fi
    DB_TUNNEL_PID=$!
    sleep 1
  fi
fi

go run ./cmd/server &
GO_PID=$!

# 启动 Web 前端
echo "🚀 启动 Web 前端..."
cd "$ROOT_DIR/web"
pnpm dev &
WEB_PID=$!

# 启动 GM 后台
echo "🚀 启动 GM 后台..."
cd "$ROOT_DIR/admin"
pnpm dev &
ADMIN_PID=$!

echo ""
echo "✅ Hero3 开发环境已启动"
echo "   前端: http://localhost:5173"
echo "   后台: http://localhost:5174"
echo "   后端: http://localhost:8080"
echo ""
echo "按 Ctrl+C 停止所有服务"

wait
