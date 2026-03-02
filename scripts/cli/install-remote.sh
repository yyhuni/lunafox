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
CHANNEL_SCHEMA_VERSION="${LUNAFOX_CHANNEL_SCHEMA_VERSION:-3}"

# CI smoke only: override manifest raw endpoints.
GITHUB_RAW_BASE="${LUNAFOX_RELEASE_GITHUB_RAW_BASE:-https://raw.githubusercontent.com}"
GITEE_RAW_BASE="${LUNAFOX_RELEASE_GITEE_RAW_BASE:-https://gitee.com}"

CHANNEL="stable"
SOURCE="auto"
VERSION_OVERRIDE=""
PUBLIC_URL=""
PUBLIC_HOST=""
PUBLIC_PORT=""
NON_INTERACTIVE=0
TARGET_VERSION=""
CHANNEL_SOURCE=""
CHANNEL_SOURCE_URL=""
VERSION_SOURCE=""
VERSION_SOURCE_URL=""
ASSET_SOURCE=""
ASSET_SOURCE_URL=""
FAILED_DOWNLOADS=()

usage() {
	cat <<'USAGE'
用法:
  ./install.sh [安装器参数...]

示例:
  ./install.sh
  ./install.sh --version v1.5.13
  ./install.sh --channel canary
  ./install.sh --source gitee --channel stable --public-url https://10.0.0.8:18443
  ./install.sh --public-host 10.0.0.8 --public-port 18443 --non-interactive

常用参数:
  --version <ver>            指定安装版本（例如 v1.5.13）
  --channel <name>           版本通道（默认 stable）
  --source <auto|github|gitee>
                            下载源策略（默认 auto=清单 gitee 后 github；安装器测速后在 gitee/github 间自动选择）
  --public-url <url>         公网访问地址（仅支持 localhost/IPv4），如 https://10.0.0.8:18443
  --public-host <host>       公网主机（仅支持 localhost/IPv4），如 10.0.0.8
  --public-port <port>       公网端口（1-65535，必填）
  --non-interactive          禁用交互向导，需显式提供公网地址

地址输入规则:
  1) --public-url 与 --public-host/--public-port 二选一
  2) --public-host 必须配合 --public-port
  3) 主机仅支持 localhost 或 IPv4

行为说明:
  install 默认会在安装前执行轻清理（compose down --remove-orphans + 清理残留 lunafox-agent 容器），不会删除卷/证书/.env。
  如需完全重置，请在项目根目录运行 ./uninstall.sh（或 ./uninstall.sh --keep-data 仅卸载服务）。
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
	--web)
		usage_error "--web 已移除：安装器现在仅支持终端交互"
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

CHANNEL="$(printf '%s' "$CHANNEL" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')"
if ! printf '%s' "$CHANNEL" | grep -Eq '^[a-z0-9._-]+$'; then
	usage_error "--channel 格式不合法：仅支持字母/数字/点/下划线/中划线"
fi

if [ -n "$PUBLIC_URL" ] && { [ -n "$PUBLIC_HOST" ] || [ -n "$PUBLIC_PORT" ]; }; then
	usage_error "--public-url 与 --public-host/--public-port 不能同时使用"
fi
if [ -n "$PUBLIC_PORT" ] && [ -z "$PUBLIC_HOST" ]; then
	usage_error "--public-port 需要配合 --public-host 使用"
fi
if [ -n "$PUBLIC_HOST" ] && [ -z "$PUBLIC_PORT" ]; then
	usage_error "--public-host 需要配合 --public-port 使用"
fi

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

manifest_source_candidates() {
	case "$SOURCE" in
	auto) echo "gitee github" ;;
	github) echo "github" ;;
	gitee) echo "gitee" ;;
	esac
}

release_manifest_source_candidates() {
	case "$SOURCE" in
	auto)
		if [ "$VERSION_SOURCE" = "github" ]; then
			echo "github gitee"
		else
			echo "gitee github"
		fi
		;;
	github) echo "github" ;;
	gitee) echo "gitee" ;;
	esac
}

asset_source_candidates() {
	case "$SOURCE" in
	auto) echo "gitee github" ;;
	github) echo "github" ;;
	gitee) echo "gitee" ;;
	esac
}

append_failed_download() {
	local kind="$1"
	local source="$2"
	local url="$3"
	FAILED_DOWNLOADS+=("${kind}|${source}|${url}")
}

print_failed_download_summary() {
	if [ "${#FAILED_DOWNLOADS[@]}" -eq 0 ]; then
		return
	fi
	error "下载失败明细："
	local item=""
	for item in "${FAILED_DOWNLOADS[@]}"; do
		local kind="${item%%|*}"
		local rest="${item#*|}"
		local source="${rest%%|*}"
		local url="${rest#*|}"
		error "  - ${kind} source=${source} url=${url}"
	done
}

print_retry_hints() {
	warn "你可以尝试以下命令快速重试："
	if [ -n "$TARGET_VERSION" ]; then
		warn "  1) ./install.sh --version ${TARGET_VERSION} --source gitee"
		warn "  2) ./install.sh --version ${TARGET_VERSION} --source github"
	else
		warn "  1) ./install.sh --channel stable --source gitee"
		warn "  2) ./install.sh --channel stable --source github"
	fi
	if [ "$CHANNEL" != "stable" ]; then
		warn "  3) 若你只想先装稳定版：./install.sh --channel stable"
	fi
}

raw_path_url() {
	local source="$1"
	local relative_path="$2"
	case "$source" in
	github)
		printf '%s/%s/%s/%s' "${GITHUB_RAW_BASE%/}" "$GITHUB_REPO" "$RELEASE_CHANNEL_BRANCH" "$relative_path"
		;;
	gitee)
		printf '%s/%s/raw/%s/%s' "${GITEE_RAW_BASE%/}" "$GITEE_REPO" "$RELEASE_CHANNEL_BRANCH" "$relative_path"
		;;
	esac
}

manifest_url() {
	local source="$1"
	local manifest_name="$2"
	raw_path_url "$source" "channels/${manifest_name}"
}

download_with_retry() {
	local url="$1"
	local output="$2"
	local desc="$3"
	local source="$4"
	local delays=(1 2 4)
	local attempt=1
	local attempts_total="${#delays[@]}"
	local rc=0
	local err_file=""
	local err_msg=""
	for delay in "${delays[@]}"; do
		err_file="$(mktemp "${TMP_DIR}/curl-err.XXXXXX")"
		if curl -fsSL --connect-timeout 10 --max-time 60 -o "$output" "$url" 2>"$err_file"; then
			rm -f "$err_file"
			return 0
		else
			rc=$?
		fi
		err_msg="$(tr '\n' ' ' <"$err_file" | sed 's/[[:space:]]\+/ /g' | cut -c1-220)"
		rm -f "$err_file"
		if [ "$attempt" -lt "$attempts_total" ]; then
			info "$desc 下载失败（source=${source}，第 ${attempt}/${attempts_total} 次，url=${url}，rc=${rc}）${err_msg:+，错误: ${err_msg}}，${delay}s 后重试"
			sleep "$delay"
		else
			error "$desc 下载失败（source=${source}，第 ${attempt}/${attempts_total} 次，url=${url}，rc=${rc}）${err_msg:+，错误: ${err_msg}}"
		fi
		attempt=$((attempt + 1))
	done
	return 1
}

validate_version_value() {
	local version="$1"
	if ! printf '%s' "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.]+)?$'; then
		error "版本号格式不合法: $version"
		error "期望格式示例: v1.5.13 或 v1.6.0-rc.1"
		exit 1
	fi
}

validate_sha256_value() {
	local value="$1"
	local key_name="$2"
	if ! printf '%s' "$value" | grep -Eq '^[a-f0-9]{64}$'; then
		error "版本清单字段 ${key_name} 不合法（必须为 64 位小写 sha256）"
		exit 1
	fi
}

validate_asset_name() {
	local value="$1"
	local key_name="$2"
	if ! printf '%s' "$value" | grep -Eq '^[A-Za-z0-9._-]+$'; then
		error "版本清单字段 ${key_name} 不合法（仅支持字母/数字/点/下划线/中划线）"
		exit 1
	fi
}

validate_manifest_base_url() {
	local value="$1"
	local key_name="$2"
	if ! printf '%s' "$value" | grep -Eq '^https?://'; then
		error "版本清单字段 ${key_name} 不合法（必须以 http:// 或 https:// 开头）"
		exit 1
	fi
}

validate_release_manifest_ref() {
	local value="$1"
	local key_name="$2"
	if ! printf '%s' "$value" | grep -Eq '^manifests/[A-Za-z0-9._-]+\.yaml$'; then
		error "版本清单字段 ${key_name} 不合法（必须为 manifests/<name>.yaml）"
		exit 1
	fi
}

read_yaml_value() {
	local file="$1"
	local key="$2"
	awk -v key="$key" '
		BEGIN { FS=":" }
		$0 ~ "^[[:space:]]*"key"[[:space:]]*:" {
			value=substr($0, index($0, ":")+1)
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
			gsub(/^["'"'"']|["'"'"']$/, "", value)
			print value
			exit
		}
	' "$file"
}

normalize_version() {
	local version="$1"
	version="$(printf '%s' "$version" | tr -d '[:space:]')"
	version="${version#v}"
	version="${version#V}"
	printf '%s' "$version"
}

probe_asset_source() {
	local source="$1"
	local url="$2"
	local err_file=""
	local err_msg=""
	local out=""
	local http_code=""
	local time_total=""
	local speed_download=""
	local rc=0

	err_file="$(mktemp "${TMP_DIR}/probe-curl-err.${source}.XXXXXX")"
	if out="$(curl -fsS -L --range 0-131071 --connect-timeout 2 --max-time 6 -o /dev/null -w '%{http_code} %{time_total} %{speed_download}' "$url" 2>"$err_file")"; then
		:
	else
		rc=$?
		err_msg="$(tr '\n' ' ' <"$err_file" | sed 's/[[:space:]]\+/ /g' | cut -c1-220)"
		rm -f "$err_file"
		info "安装器源测速失败（source=${source}，url=${url}，rc=${rc}）${err_msg:+，错误: ${err_msg}}" >&2
		return 1
	fi
	rm -f "$err_file"

	read -r http_code time_total speed_download <<<"$out"
	if [ "$http_code" != "200" ] && [ "$http_code" != "206" ]; then
		info "安装器源测速失败（source=${source}，url=${url}，http=${http_code}）" >&2
		return 1
	fi
	if [ -z "$time_total" ]; then
		return 1
	fi
	info "安装器源测速结果（source=${source}，time=${time_total}s，speed=${speed_download}B/s）" >&2
	printf '%s\n' "$time_total"
	return 0
}

resolve_auto_asset_order() {
	local gitee_url="${GITEE_BASE_URL%/}/$ASSET_NAME"
	local github_url="${GITHUB_BASE_URL%/}/$ASSET_NAME"
	local gitee_time=""
	local github_time=""
	local have_gitee=0
	local have_github=0

	if gitee_time="$(probe_asset_source "gitee" "$gitee_url")"; then
		have_gitee=1
	fi
	if github_time="$(probe_asset_source "github" "$github_url")"; then
		have_github=1
	fi

	if [ "$have_gitee" -eq 1 ] && [ "$have_github" -eq 1 ]; then
		if awk "BEGIN {exit !(${gitee_time} <= ${github_time})}"; then
			info "安装器源排序：gitee 优先（${gitee_time}s <= ${github_time}s）" >&2
			printf '%s\n' "gitee github"
		else
			info "安装器源排序：github 优先（${github_time}s < ${gitee_time}s）" >&2
			printf '%s\n' "github gitee"
		fi
		return 0
	fi
	if [ "$have_gitee" -eq 1 ]; then
		info "安装器源排序：仅 gitee 探测速通过，优先 gitee" >&2
		printf '%s\n' "gitee github"
		return 0
	fi
	if [ "$have_github" -eq 1 ]; then
		info "安装器源排序：仅 github 探测速通过，优先 github" >&2
		printf '%s\n' "github gitee"
		return 0
	fi

	warn "安装器源测速全部失败，回退默认顺序 gitee -> github" >&2
	printf '%s\n' "gitee github"
	return 0
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
	for source in $(manifest_source_candidates); do
		candidate="$TMP_DIR/channel-${source}.env"
		url="$(manifest_url "$source" "${CHANNEL}.env")"
		if download_with_retry "$url" "$candidate" "通道清单(${source})" "$source"; then
			CHANNEL_MANIFEST="$candidate"
			CHANNEL_SOURCE="$source"
			CHANNEL_SOURCE_URL="$url"
			break
		else
			append_failed_download "channel-manifest" "$source" "$url"
		fi
	done
	if [ -z "$CHANNEL_MANIFEST" ] || [ ! -s "$CHANNEL_MANIFEST" ]; then
		error "无法下载通道清单: ${CHANNEL}.env"
		print_failed_download_summary
		print_retry_hints
		exit 1
	fi
	validate_manifest_schema "$CHANNEL_MANIFEST" "通道清单(${CHANNEL})"
	TARGET_VERSION="$(require_env_value "$CHANNEL_MANIFEST" "VERSION")"
	info "通道清单下载成功: source=${CHANNEL_SOURCE} url=${CHANNEL_SOURCE_URL}"
fi

TARGET_VERSION="$(printf '%s' "$TARGET_VERSION" | tr -d '[:space:]')"
if [ -z "$TARGET_VERSION" ]; then
	error "无法解析目标版本，请检查 --version 或通道清单中的 VERSION"
	exit 1
fi
validate_version_value "$TARGET_VERSION"
info "目标安装版本: ${TARGET_VERSION}"

VERSION_MANIFEST=""
for source in $(manifest_source_candidates); do
	candidate="$TMP_DIR/version-${source}.env"
	url="$(manifest_url "$source" "${TARGET_VERSION}.env")"
	if download_with_retry "$url" "$candidate" "版本清单(${source})" "$source"; then
		VERSION_MANIFEST="$candidate"
		VERSION_SOURCE="$source"
		VERSION_SOURCE_URL="$url"
		break
	else
		append_failed_download "version-manifest" "$source" "$url"
	fi
done
if [ -z "$VERSION_MANIFEST" ] || [ ! -s "$VERSION_MANIFEST" ]; then
	error "无法下载版本清单: ${TARGET_VERSION}.env"
	print_failed_download_summary
	print_retry_hints
	exit 1
fi
info "版本清单下载成功: source=${VERSION_SOURCE} url=${VERSION_SOURCE_URL}"

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
validate_asset_name "$ASSET_NAME" "$ASSET_KEY"
validate_sha256_value "$EXPECTED_SHA" "$SHA_KEY"
validate_manifest_base_url "$GITHUB_BASE_URL" "GITHUB_BASE_URL"
validate_manifest_base_url "$GITEE_BASE_URL" "GITEE_BASE_URL"
RELEASE_MANIFEST_REL="$(require_env_value "$VERSION_MANIFEST" "RELEASE_MANIFEST")"
validate_release_manifest_ref "$RELEASE_MANIFEST_REL" "RELEASE_MANIFEST"

RELEASE_MANIFEST_FILE=""
for source in $(release_manifest_source_candidates); do
	candidate="$TMP_DIR/release-manifest-${source}.yaml"
	url="$(raw_path_url "$source" "$RELEASE_MANIFEST_REL")"
	if download_with_retry "$url" "$candidate" "发布清单(${source})" "$source"; then
		RELEASE_MANIFEST_FILE="$candidate"
		break
	else
		append_failed_download "release-manifest" "$source" "$url"
	fi
done
if [ -z "$RELEASE_MANIFEST_FILE" ] || [ ! -s "$RELEASE_MANIFEST_FILE" ]; then
	error "无法下载发布清单: ${RELEASE_MANIFEST_REL}"
	print_failed_download_summary
	print_retry_hints
	exit 1
fi
RELEASE_MANIFEST_VERSION="$(read_yaml_value "$RELEASE_MANIFEST_FILE" "releaseVersion")"
if [ -z "$RELEASE_MANIFEST_VERSION" ]; then
	error "发布清单缺少 releaseVersion: ${RELEASE_MANIFEST_REL}"
	exit 1
fi
if [ "$(normalize_version "$RELEASE_MANIFEST_VERSION")" != "$(normalize_version "$TARGET_VERSION")" ]; then
	error "发布清单 releaseVersion 与目标版本不一致"
	error "目标版本: $TARGET_VERSION"
	error "发布清单: $RELEASE_MANIFEST_VERSION"
	exit 1
fi

if [ -z "$ASSET_NAME" ] || [ -z "$EXPECTED_SHA" ]; then
	error "版本清单缺少 ${ASSET_KEY} 或 ${SHA_KEY}"
	exit 1
fi
INSTALLER_BIN="$TMP_DIR/lunafox-installer"
DOWNLOADED=0
ASSET_SOURCE_ORDER="$(asset_source_candidates)"
if [ "$SOURCE" = "auto" ]; then
	ASSET_SOURCE_ORDER="$(resolve_auto_asset_order)"
fi
for source in $ASSET_SOURCE_ORDER; do
	case "$source" in
	github) base_url="$GITHUB_BASE_URL" ;;
	gitee) base_url="$GITEE_BASE_URL" ;;
	esac
	if [ -z "${base_url:-}" ]; then
		continue
	fi
	asset_url="${base_url%/}/$ASSET_NAME"
	if download_with_retry "$asset_url" "$INSTALLER_BIN" "安装器(${source})" "$source"; then
		DOWNLOADED=1
		ASSET_SOURCE="$source"
		ASSET_SOURCE_URL="$asset_url"
		break
	else
		append_failed_download "installer-asset" "$source" "$asset_url"
	fi
done

if [ "$DOWNLOADED" -ne 1 ]; then
	error "安装器下载失败：已尝试 source=$SOURCE 的所有可用源"
	print_failed_download_summary
	print_retry_hints
	exit 1
fi
info "安装器下载成功: source=${ASSET_SOURCE} url=${ASSET_SOURCE_URL}"

ACTUAL_SHA="$($SHA_CMD "$INSTALLER_BIN" | awk '{print $1}')"
if [ "$ACTUAL_SHA" != "$EXPECTED_SHA" ]; then
	error "安装器 sha256 校验失败"
	error "期望: $EXPECTED_SHA"
	error "实际: $ACTUAL_SHA"
	print_retry_hints
	exit 1
fi
success "安装器校验通过: sha256=${ACTUAL_SHA}"

chmod +x "$INSTALLER_BIN"

INSTALLER_ARGS=(
	--root-dir "$ROOT_DIR"
	--version "$TARGET_VERSION"
	--release-manifest "$RELEASE_MANIFEST_FILE"
)
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

cd "$ROOT_DIR"
exec "$INSTALLER_BIN" "${INSTALLER_ARGS[@]}"
