#!/bin/bash
# Hero3 一键启动脚本 - 同时运行 Go 后端、两个玩家前端和 GM 后台

set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

# 递归列出指定进程及其所有子进程，确保 go run 生成的临时后端不会残留。
collect_process_tree() {
  local root_pid="$1"
  local child_pid
  for child_pid in $(pgrep -P "$root_pid" 2>/dev/null || true); do
    collect_process_tree "$child_pid"
  done
  echo "$root_pid"
}

# 停止脚本启动的一整棵进程树。
stop_process_tree() {
  local root_pid="${1:-}"
  local process_ids
  [[ "$root_pid" =~ ^[0-9]+$ ]] || return 0
  process_ids="$(collect_process_tree "$root_pid")"
  [ -n "$process_ids" ] || return 0
  kill $process_ids 2>/dev/null || true
  wait "$root_pid" 2>/dev/null || true
}

# 统一停止本脚本启动的服务，避免退出后残留开发进程。
cleanup() {
  [ "${CLEANUP_DONE:-false}" = "true" ] && return 0
  CLEANUP_DONE=true
  echo ""
  echo "正在停止所有服务..."
  stop_process_tree "${GO_PID:-}"
  stop_process_tree "${WEB_PID:-}"
  stop_process_tree "${WLSG_PID:-}"
  stop_process_tree "${ADMIN_PID:-}"
  stop_process_tree "${DB_TUNNEL_PID:-}"
  echo "已停止。"
}

# 响应手动停止时按成功退出，不把主动 Ctrl+C 误报成服务崩溃。
handle_shutdown() {
  cleanup
  exit 0
}

trap cleanup EXIT
trap handle_shutdown INT TERM

# 启动前确认固定开发端口空闲，避免把旧服务的健康响应误判为本次启动成功。
ensure_port_available() {
  local port="$1"
  local service_name="$2"
  if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "❌ $service_name 启动端口 $port 已被占用，请先停止旧开发环境。"
    return 1
  fi
}

# 安全加载 KEY=VALUE 格式的 .env，避免 DSN 中的 & 被 shell 当成控制符。
load_env_file() {
  local env_file="$1"
  [ -f "$env_file" ] || return 0
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" || "$line" == \#* || "$line" != *=* ]] && continue
    local key="${line%%=*}"
    local value="${line#*=}"
    key="${key%"${key##*[![:space:]]}"}"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    export "$key=$value"
  done < "$env_file"
}

# 将 MySQL TCP DSN 的连接地址切换到本机 SSH 隧道，保留账号、密码、库名和参数。
use_database_tunnel() {
  local local_port="$1"
  if [[ "${HERO3_DATABASE_DSN:-}" =~ ^(.+@tcp\()[^\)]*(\)/.+)$ ]]; then
    export HERO3_DATABASE_DSN="${BASH_REMATCH[1]}127.0.0.1:${local_port}${BASH_REMATCH[2]}"
    return 0
  fi
  echo "❌ HERO3_DATABASE_DSN 不是可切换隧道的 MySQL TCP DSN。"
  return 1
}

# 等待 Go 后端健康检查，避免后台进程已退出但脚本仍提示启动成功。
wait_for_backend() {
  local backend_port="$1"
  local attempt
  for attempt in $(seq 1 30); do
    if ! kill -0 "$GO_PID" 2>/dev/null; then
      wait "$GO_PID" 2>/dev/null || true
      echo "❌ Go 后端启动失败，请查看上方错误日志。"
      return 1
    fi
    if curl -fsS "http://localhost:${backend_port}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "❌ Go 后端健康检查超时：http://localhost:${backend_port}/healthz"
  return 1
}

# 等待前端服务真正开始响应，依赖安装或 Vite 启动失败时立即终止整套环境。
wait_for_frontend() {
  local service_name="$1"
  local service_pid="$2"
  local service_url="$3"
  local attempt
  for attempt in $(seq 1 30); do
    if ! kill -0 "$service_pid" 2>/dev/null; then
      wait "$service_pid" 2>/dev/null || true
      echo "❌ $service_name 启动失败，请查看上方错误日志。"
      return 1
    fi
    if curl -fsS "$service_url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "❌ $service_name 健康检查超时：$service_url"
  return 1
}

# 持续监控全部服务；任一服务意外退出都让脚本失败并触发统一清理。
monitor_services() {
  local service_entry
  local service_name
  local service_pid
  while true; do
    for service_entry in "Go 后端:$GO_PID" "Web 玩家端:$WEB_PID" "武林三国玩家端:$WLSG_PID" "GM 后台:$ADMIN_PID"; do
      service_name="${service_entry%%:*}"
      service_pid="${service_entry##*:}"
      if ! kill -0 "$service_pid" 2>/dev/null; then
        wait "$service_pid" 2>/dev/null || true
        echo "❌ $service_name 运行中异常退出，正在停止其余服务。"
        return 1
      fi
    done
    sleep 1
  done
}

# 启动 Go 后端
echo "🚀 启动 Go 后端..."
cd "$ROOT_DIR/go"
load_env_file .env
BACKEND_PORT="${HERO3_PORT:-8080}"
if [ -n "${HERO3_ADMIN_TOKEN:-}" ] && [ -z "${VITE_ADMIN_TOKEN:-}" ]; then
  export VITE_ADMIN_TOKEN="$HERO3_ADMIN_TOKEN"
fi

ensure_port_available "$BACKEND_PORT" "Go 后端"
ensure_port_available 5173 "Web 玩家端"
ensure_port_available 5174 "GM 后台"
ensure_port_available 5175 "武林三国玩家端"

if [ "${HERO3_DB_TUNNEL_ENABLED:-false}" = "true" ]; then
  DB_TUNNEL_LOCAL_PORT="${HERO3_DB_TUNNEL_LOCAL_PORT:-3307}"
  echo "🔌 启动服务器数据库 SSH 隧道..."
  if lsof -nP -iTCP:"$DB_TUNNEL_LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "   端口 $DB_TUNNEL_LOCAL_PORT 已在监听，复用已有隧道。"
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
  if ! lsof -nP -iTCP:"$DB_TUNNEL_LOCAL_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "❌ 数据库 SSH 隧道启动失败，本机端口 $DB_TUNNEL_LOCAL_PORT 未监听。"
    exit 1
  fi
  use_database_tunnel "$DB_TUNNEL_LOCAL_PORT"
fi

go run ./cmd/server &
GO_PID=$!
wait_for_backend "$BACKEND_PORT"

# 启动 Web 前端
echo "🚀 启动 Web 前端..."
cd "$ROOT_DIR/web"
pnpm run dev --port 5173 --strictPort &
WEB_PID=$!

# 启动武林三国风格 Web 前端
echo "🚀 启动武林三国 Web 前端..."
cd "$ROOT_DIR/web-wlsg"
pnpm run dev --port 5175 --strictPort &
WLSG_PID=$!

# 启动 GM 后台
echo "🚀 启动 GM 后台..."
cd "$ROOT_DIR/admin"
pnpm run dev --port 5174 --strictPort &
ADMIN_PID=$!

wait_for_frontend "Web 玩家端" "$WEB_PID" "http://localhost:5173/"
wait_for_frontend "武林三国玩家端" "$WLSG_PID" "http://localhost:5175/"
wait_for_frontend "GM 后台" "$ADMIN_PID" "http://localhost:5174/admin/"

echo ""
echo "✅ Hero3 开发环境已启动"
echo "   Web 玩家端:      http://localhost:5173"
echo "   GM 后台:         http://localhost:5174"
echo "   武林三国玩家端:  http://localhost:5175"
echo "   Go 后端:         http://localhost:$BACKEND_PORT"
echo ""
echo "按 Ctrl+C 停止所有服务"

monitor_services
