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

RELEASE_CHANNEL_BRANCH="${LUNAFOX_RELEASE_CHANNEL_BRANCH:-release-channel}"
GITHUB_REPO="${LUNAFOX_RELEASE_GITHUB_REPO:-yyhuni/lunafox}"
GITEE_REPO="${LUNAFOX_RELEASE_GITEE_REPO:-yyhuni/lunafox}"
CHANNEL_SCHEMA_VERSION="${LUNAFOX_CHANNEL_SCHEMA_VERSION:-2}"

# CI smoke only: override manifest raw endpoints.
GITHUB_RAW_BASE="${LUNAFOX_RELEASE_GITHUB_RAW_BASE:-https://raw.githubusercontent.com}"
GITEE_RAW_BASE="${LUNAFOX_RELEASE_GITEE_RAW_BASE:-https://gitee.com}"

CHANNEL="stable"
SOURCE="auto"
VERSION_OVERRIDE=""
LISTEN_ADDR=""

usage() {
	cat <<'USAGE'
用法:
  ./install.sh [安装器参数...]

示例:
  ./install.sh
  ./install.sh --version v1.5.13
  ./install.sh --channel canary
  ./install.sh --source gitee --channel stable --listen 0.0.0.0:18083

常用参数:
  --version <ver>            指定安装版本（例如 v1.5.13）
  --channel <name>           版本通道（默认 stable）
  --source <auto|github|gitee>
                            下载源策略（默认 auto=github 后 gitee）
  --listen <addr>            Web 页面监听地址，如 0.0.0.0:18083
USAGE
}

while [ "$#" -gt 0 ]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	--dev)
		error "--dev 已移除，请使用 ./scripts/dev/install.sh"
		exit 2
		;;
	--version)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--version 缺少参数"
		fi
		VERSION_OVERRIDE="$1"
		;;
	--channel)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--channel 缺少参数"
		fi
		CHANNEL="$1"
		;;
	--source)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--source 缺少参数"
		fi
		SOURCE="$1"
		;;
	--listen)
		shift
		if [ "$#" -eq 0 ]; then
			usage_error "--listen 缺少参数"
		fi
		LISTEN_ADDR="$1"
		;;
	--web)
		usage_error "--web 参数已移除，install.sh 默认即 Web 安装入口"
		;;
	--)
		usage_error "不再支持参数透传，请直接使用 install.sh 提供的参数"
		;;
	*)
		usage_error "不支持的参数: $1"
		;;
	esac
	shift
done

SOURCE="$(printf '%s' "$SOURCE" | tr '[:upper:]' '[:lower:]')"
case "$SOURCE" in
auto | github | gitee) ;;
*)
	usage_error "不支持的 --source: $SOURCE（仅支持 auto/github/gitee）"
	;;
esac

if ! command -v curl >/dev/null 2>&1; then
	error "未检测到 curl，请先安装 curl"
	exit 1
fi

OS="$(uname -s)"
case "$OS" in
Linux) OS_DIR="linux" ;;
Darwin) OS_DIR="darwin" ;;
*)
	error "Go 安装器目前仅支持 Linux/macOS（amd64/arm64）"
	exit 1
	;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
x86_64 | amd64) ARCH_DIR="amd64" ;;
aarch64 | arm64) ARCH_DIR="arm64" ;;
*)
	error "不支持的 CPU 架构: $ARCH（仅支持 x86_64/aarch64）"
	exit 1
	;;
esac

PLATFORM_KEY="$(printf '%s_%s' "$OS_DIR" "$ARCH_DIR" | tr '[:lower:]' '[:upper:]')"

sha256_cmd() {
	if command -v sha256sum >/dev/null 2>&1; then
		echo "sha256sum"
		return
	fi
	if command -v shasum >/dev/null 2>&1; then
		echo "shasum -a 256"
		return
	fi
	echo ""
}

SHA_CMD="$(sha256_cmd)"
if [ -z "$SHA_CMD" ]; then
	error "未检测到 sha256 校验工具（sha256sum 或 shasum）"
	exit 1
fi

source_candidates() {
	case "$SOURCE" in
	auto) echo "github gitee" ;;
	github) echo "github" ;;
	gitee) echo "gitee" ;;
	esac
}

manifest_url() {
	local source="$1"
	local manifest_name="$2"
	case "$source" in
	github)
		printf '%s/%s/%s/channels/%s' "${GITHUB_RAW_BASE%/}" "$GITHUB_REPO" "$RELEASE_CHANNEL_BRANCH" "$manifest_name"
		;;
	gitee)
		printf '%s/%s/raw/%s/channels/%s' "${GITEE_RAW_BASE%/}" "$GITEE_REPO" "$RELEASE_CHANNEL_BRANCH" "$manifest_name"
		;;
	esac
}

download_with_retry() {
	local url="$1"
	local output="$2"
	local desc="$3"
	local delays=(1 2 4)
	local attempt=1
	for delay in "${delays[@]}"; do
		if curl -fsSL --connect-timeout 10 --max-time 60 -o "$output" "$url"; then
			return 0
		fi
		if [ "$attempt" -lt "${#delays[@]}" ]; then
			info "$desc 下载失败（第 ${attempt} 次），${delay}s 后重试"
			sleep "$delay"
		fi
		attempt=$((attempt + 1))
	done
	return 1
}

read_env_value() {
	local file="$1"
	local key="$2"
	awk -F= -v key="$key" '$1 == key { print substr($0, index($0, "=") + 1); exit }' "$file" | tr -d '\r'
}

require_env_value() {
	local file="$1"
	local key="$2"
	local value
	value="$(read_env_value "$file" "$key")"
	if [ -z "$value" ]; then
		error "清单缺少必填字段: $key"
		exit 1
	fi
	printf '%s' "$value"
}

is_digest_ref() {
	local ref="$1"
	printf '%s' "$ref" | grep -Eq '^.+@sha256:[a-f0-9]{64}$'
}

normalize_digest_ref_list() {
	local raw="$1"
	local key_name="$2"
	local -a parts=()
	local -a normalized=()
	local seen=","
	local part=""
	local trimmed=""
	IFS=',' read -r -a parts <<<"$raw"
	for part in "${parts[@]}"; do
		trimmed="$(printf '%s' "$part" | xargs)"
		if [ -z "$trimmed" ]; then
			continue
		fi
		if ! is_digest_ref "$trimmed"; then
			error "版本清单 ${key_name} 存在非法值（必须为 digest 引用）: $trimmed"
			exit 1
		fi
		case "$seen" in
		*,"$trimmed",*) continue ;;
		esac
		seen="${seen}${trimmed},"
		normalized+=("$trimmed")
	done
	if [ "${#normalized[@]}" -eq 0 ]; then
		error "版本清单 ${key_name} 不能为空"
		exit 1
	fi
	(
		IFS=','
		printf '%s' "${normalized[*]}"
	)
}

validate_manifest_schema() {
	local file="$1"
	local kind="$2"
	local schema
	schema="$(read_env_value "$file" "SCHEMA_VERSION")"
	if [ -z "$schema" ]; then
		error "${kind}缺少 SCHEMA_VERSION"
		exit 1
	fi
	if [ "$schema" != "$CHANNEL_SCHEMA_VERSION" ]; then
		error "${kind} SCHEMA_VERSION 不兼容"
		error "当前安装器支持: $CHANNEL_SCHEMA_VERSION"
		error "清单声明版本: $schema"
		exit 1
	fi
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

CHANNEL_MANIFEST=""
TARGET_VERSION="$VERSION_OVERRIDE"

if [ -z "$TARGET_VERSION" ]; then
	for source in $(source_candidates); do
		candidate="$TMP_DIR/channel-${source}.env"
		url="$(manifest_url "$source" "${CHANNEL}.env")"
		if download_with_retry "$url" "$candidate" "通道清单(${source})"; then
			CHANNEL_MANIFEST="$candidate"
			break
		fi
	done
	if [ -z "$CHANNEL_MANIFEST" ] || [ ! -s "$CHANNEL_MANIFEST" ]; then
		error "无法下载通道清单: ${CHANNEL}.env"
		exit 1
	fi
	validate_manifest_schema "$CHANNEL_MANIFEST" "通道清单(${CHANNEL})"
	TARGET_VERSION="$(require_env_value "$CHANNEL_MANIFEST" "VERSION")"
fi

TARGET_VERSION="$(printf '%s' "$TARGET_VERSION" | tr -d '[:space:]')"
if [ -z "$TARGET_VERSION" ]; then
	error "无法解析目标版本，请检查 --version 或通道清单中的 VERSION"
	exit 1
fi

VERSION_MANIFEST=""
for source in $(source_candidates); do
	candidate="$TMP_DIR/version-${source}.env"
	url="$(manifest_url "$source" "${TARGET_VERSION}.env")"
	if download_with_retry "$url" "$candidate" "版本清单(${source})"; then
		VERSION_MANIFEST="$candidate"
		break
	fi
done
if [ -z "$VERSION_MANIFEST" ] || [ ! -s "$VERSION_MANIFEST" ]; then
	error "无法下载版本清单: ${TARGET_VERSION}.env"
	exit 1
fi

validate_manifest_schema "$VERSION_MANIFEST" "版本清单(${TARGET_VERSION})"
MANIFEST_VERSION="$(require_env_value "$VERSION_MANIFEST" "VERSION")"
if [ "$MANIFEST_VERSION" != "$TARGET_VERSION" ]; then
	error "版本清单 VERSION 与目标版本不一致"
	error "目标版本: $TARGET_VERSION"
	error "清单版本: $MANIFEST_VERSION"
	exit 1
fi

ASSET_KEY="${PLATFORM_KEY}_ASSET"
SHA_KEY="${PLATFORM_KEY}_SHA256"
ASSET_NAME="$(require_env_value "$VERSION_MANIFEST" "$ASSET_KEY")"
EXPECTED_SHA="$(require_env_value "$VERSION_MANIFEST" "$SHA_KEY")"
GITHUB_BASE_URL="$(require_env_value "$VERSION_MANIFEST" "GITHUB_BASE_URL")"
GITEE_BASE_URL="$(require_env_value "$VERSION_MANIFEST" "GITEE_BASE_URL")"
# Schema v2 contract: only digest candidate lists are accepted.
AGENT_IMAGE_REFS_RAW="$(require_env_value "$VERSION_MANIFEST" "AGENT_IMAGE_REFS")"
WORKER_IMAGE_REFS_RAW="$(require_env_value "$VERSION_MANIFEST" "WORKER_IMAGE_REFS")"
AGENT_IMAGE_REFS="$(normalize_digest_ref_list "$AGENT_IMAGE_REFS_RAW" "AGENT_IMAGE_REFS")"
WORKER_IMAGE_REFS="$(normalize_digest_ref_list "$WORKER_IMAGE_REFS_RAW" "WORKER_IMAGE_REFS")"

if [ -z "$ASSET_NAME" ] || [ -z "$EXPECTED_SHA" ]; then
	error "版本清单缺少 ${ASSET_KEY} 或 ${SHA_KEY}"
	exit 1
fi
INSTALLER_BIN="$TMP_DIR/lunafox-installer"
DOWNLOADED=0
for source in $(source_candidates); do
	case "$source" in
	github) base_url="$GITHUB_BASE_URL" ;;
	gitee) base_url="$GITEE_BASE_URL" ;;
	esac
	if [ -z "${base_url:-}" ]; then
		continue
	fi
	asset_url="${base_url%/}/$ASSET_NAME"
	if download_with_retry "$asset_url" "$INSTALLER_BIN" "安装器(${source})"; then
		DOWNLOADED=1
		break
	fi
done

if [ "$DOWNLOADED" -ne 1 ]; then
	error "安装器下载失败：已尝试 source=$SOURCE 的所有可用源"
	exit 1
fi

ACTUAL_SHA="$($SHA_CMD "$INSTALLER_BIN" | awk '{print $1}')"
if [ "$ACTUAL_SHA" != "$EXPECTED_SHA" ]; then
	error "安装器 sha256 校验失败"
	error "期望: $EXPECTED_SHA"
	error "实际: $ACTUAL_SHA"
	exit 1
fi

chmod +x "$INSTALLER_BIN"

INSTALLER_ARGS=(
	--root-dir "$ROOT_DIR"
	--version "$TARGET_VERSION"
	# Installer selects first successful ref from ordered candidates.
	--agent-image-refs "$AGENT_IMAGE_REFS"
	--worker-image-refs "$WORKER_IMAGE_REFS"
)
if [ -n "$LISTEN_ADDR" ]; then
	INSTALLER_ARGS+=(--listen "$LISTEN_ADDR")
fi

cd "$ROOT_DIR"
exec "$INSTALLER_BIN" "${INSTALLER_ARGS[@]}"
