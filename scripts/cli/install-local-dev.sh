#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
  SCRIPT_PATH="$0"
  case "$SCRIPT_PATH" in
    /*|*/*) ;;
    *) SCRIPT_PATH="./$SCRIPT_PATH" ;;
  esac
  exec /usr/bin/env bash "$SCRIPT_PATH" "$@"
fi
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
normalize_exit_codes

LISTEN_ADDR=""

usage() {
  cat <<'USAGE'
用法:
  ./scripts/dev/install.sh [参数]

说明:
  本地源码模式启动安装器（固定 dev 模式）

参数:
  --listen <addr>    Web 页面监听地址，如 127.0.0.1:18083
  --help             显示帮助
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --listen)
      shift
      if [ "$#" -eq 0 ]; then
        usage_error "--listen 缺少参数"
      fi
      LISTEN_ADDR="$1"
      ;;
    *)
      usage_error "不支持的参数: $1"
      ;;
  esac
  shift
done

if ! command -v go >/dev/null 2>&1; then
  error "未检测到 Go，请先安装 Go（https://go.dev/dl/）"
  exit 1
fi

INSTALLER_ARGS=(run ./tools/installer/cmd/lunafox-installer --root-dir "$ROOT_DIR" --dev)
if [ -n "$LISTEN_ADDR" ]; then
  INSTALLER_ARGS+=(--listen "$LISTEN_ADDR")
fi

cd "$ROOT_DIR"
exec go "${INSTALLER_ARGS[@]}"
