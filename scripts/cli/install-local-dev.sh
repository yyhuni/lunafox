#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
	SCRIPT_PATH="$0"
	case "$SCRIPT_PATH" in
	/* | */*) ;;
	*) SCRIPT_PATH="./$SCRIPT_PATH" ;;
	esac
	exec /usr/bin/env bash "$SCRIPT_PATH" "$@"
fi
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
normalize_exit_codes

PUBLIC_URL=""
PUBLIC_HOST=""
PUBLIC_PORT=""
NON_INTERACTIVE=0

usage() {
	cat <<'USAGE'
用法:
  ./scripts/dev/install.sh [参数]

说明:
  本地源码模式启动安装器（固定 dev 模式）

参数:
  --public-url <url>  公网访问地址（仅支持 localhost/IPv4），如 https://10.0.0.8:8083
  --public-host <host> 公网主机（仅支持 localhost/IPv4）
  --public-port <port> 公网端口（默认 8083）
  --non-interactive   禁用交互向导，需显式提供公网地址
  --help             显示帮助
USAGE
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	--public-url)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--public-url 缺少参数"
		fi
		PUBLIC_URL="$1"
		;;
	--public-host)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--public-host 缺少参数"
		fi
		PUBLIC_HOST="$1"
		;;
	--public-port)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--public-port 缺少参数"
		fi
		PUBLIC_PORT="$1"
		;;
	--non-interactive)
		NON_INTERACTIVE=1
		;;
	--listen)
		usage_error "--listen 已移除，请改用 --public-url 或 --public-host/--public-port"
		;;
	*)
		usage_error "不支持的参数: $1"
		;;
	esac
	shift
done

if [ -n "$PUBLIC_URL" ] && { [ -n "$PUBLIC_HOST" ] || [ -n "$PUBLIC_PORT" ]; }; then
	usage_error "--public-url 与 --public-host/--public-port 不能同时使用"
fi
if [ -n "$PUBLIC_PORT" ] && [ -z "$PUBLIC_HOST" ]; then
	usage_error "--public-port 需要配合 --public-host 使用"
fi

if ! command -v go >/dev/null 2>&1; then
	error "未检测到 Go，请先安装 Go（https://go.dev/dl/）"
	exit 1
fi

INSTALLER_ARGS=(./cmd/lunafox-installer --root-dir "$ROOT_DIR" --dev)
if [ -n "$PUBLIC_URL" ]; then
	INSTALLER_ARGS+=(--public-url "$PUBLIC_URL")
fi
if [ -n "$PUBLIC_HOST" ]; then
	INSTALLER_ARGS+=(--public-host "$PUBLIC_HOST")
fi
if [ -n "$PUBLIC_PORT" ]; then
	INSTALLER_ARGS+=(--public-port "$PUBLIC_PORT")
fi
if [ "$NON_INTERACTIVE" -eq 1 ]; then
	INSTALLER_ARGS+=(--non-interactive)
fi

cd "$ROOT_DIR/tools/installer"
exec go run "${INSTALLER_ARGS[@]}"
