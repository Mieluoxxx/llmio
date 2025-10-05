#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
cd "$ROOT_DIR"

TOKEN="${TOKEN:-12345}"
export TOKEN

db_dir="$ROOT_DIR/db"
frontend_dir="$ROOT_DIR/webui"

if ! command -v go >/dev/null 2>&1; then
  echo "[start.sh] go command not found" >&2
  exit 1
fi

pkg_manager=""
pkg_install_cmd=()
pkg_dev_cmd=()

if command -v pnpm >/dev/null 2>&1; then
  pkg_manager="pnpm"
  pkg_install_cmd=(pnpm --dir "$frontend_dir" install)
  pkg_dev_cmd=(pnpm --dir "$frontend_dir" dev -- --host)
elif command -v npm >/dev/null 2>&1; then
  pkg_manager="npm"
  pkg_install_cmd=(npm --prefix "$frontend_dir" install)
  pkg_dev_cmd=(npm --prefix "$frontend_dir" run dev -- --host)
else
  echo "[start.sh] neither pnpm nor npm found" >&2
  exit 1
fi

if [[ ! -d "$db_dir" ]]; then
  echo "[start.sh] Creating database directory at $db_dir"
  mkdir -p "$db_dir"
fi

if [[ ! -d "$frontend_dir/node_modules" ]]; then
  echo "[start.sh] Installing frontend dependencies using $pkg_manager..."
  "${pkg_install_cmd[@]}"
fi

backend_pid=""
frontend_pid=""
status=0

cleanup() {
  local exit_code=$?
  trap - EXIT INT TERM
  echo "\n[start.sh] Stopping services..."
  if [[ -n "$backend_pid" ]] && kill -0 "$backend_pid" 2>/dev/null; then
    kill "$backend_pid" 2>/dev/null || true
    wait "$backend_pid" 2>/dev/null || true
  fi
  if [[ -n "$frontend_pid" ]] && kill -0 "$frontend_pid" 2>/dev/null; then
    kill "$frontend_pid" 2>/dev/null || true
    wait "$frontend_pid" 2>/dev/null || true
  fi
  exit "$exit_code"
}

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

echo "[start.sh] Using TOKEN=${TOKEN}"

go run main.go &
backend_pid=$!

echo "[start.sh] Starting web UI using $pkg_manager..."
"${pkg_dev_cmd[@]}" &
frontend_pid=$!

monitor_processes() {
  while true; do
    if [[ -n "$backend_pid" ]] && ! kill -0 "$backend_pid" 2>/dev/null; then
      wait "$backend_pid" 2>/dev/null
      status=$?
      return
    fi

    if [[ -n "$frontend_pid" ]] && ! kill -0 "$frontend_pid" 2>/dev/null; then
      wait "$frontend_pid" 2>/dev/null
      status=$?
      return
    fi

    sleep 1
  done
}

monitor_processes

exit "$status"
