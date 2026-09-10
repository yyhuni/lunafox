#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
#
# The public lifecycle owns only host configuration and Compose orchestration.
# Product initialization remains inside the closed bootstrap image so the
# public projection never needs a standalone installer or product source.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/compose.yaml"
ENV_FILE="$ROOT_DIR/.env"
CHANNEL_DIR="$ROOT_DIR/channels"
MANIFEST_DIR="$ROOT_DIR/manifests"
STATE_DIR="$ROOT_DIR/.lunafox"
ENGINE_INVENTORY_PATH="$STATE_DIR/engine-inventory.yaml"

PROJECT_NAME="lunafox"
PROJECT_NETWORK="lunafox_network"
AGENT_CONTAINER="lunafox-agent"
SSL_VOLUME="lunafox_ssl"
RECEIPT_VOLUME="lunafox_receipt"
EXTERNAL_VOLUME_OWNER_LABEL="io.lunafox.deployment"
EXTERNAL_VOLUME_OWNER_VALUE="public"
MIN_DOCKER_API="1.45"
MIN_COMPOSE_VERSION="2.24.0"
DEFAULT_CHANNEL="stable"
DEFAULT_REGISTRY="dockerhub"
CANONICAL_RELEASE_CHANNEL_URL="https://raw.githubusercontent.com/yyhuni/lunafox/release-channel"
RELEASE_METADATA_CONNECT_TIMEOUT="10"
RELEASE_METADATA_MAX_TIME="30"

OWNED_VOLUMES=(lunafox_postgres lunafox_data lunafox_engine_execution lunafox_agent_state lunafox_loki "$SSL_VOLUME" "$RECEIPT_VOLUME")
ENGINE_PACKAGE_SELECTED_REFS=()

COMMAND=""
CHANNEL="$DEFAULT_CHANNEL"
REGISTRY="$DEFAULT_REGISTRY"
PUBLIC_HOST=""
PUBLIC_PORT="443"
RESET=0
PURGE=0
CONFIRM=0
CHANNEL_SET=0
REGISTRY_SET=0
PUBLIC_PORT_SET=0

# All temporary paths are registered centrally so an early failure cannot
# leave certificate material or interpolation secrets on the host.  RETURN
# traps are intentionally avoided because nested functions overwrite them.
TEMP_FILES=()
TEMP_DIRS=()

cleanup_temporary_paths() {
	local path
	for path in "${TEMP_FILES[@]-}"; do
		[ -n "$path" ] && rm -f -- "$path"
	done
	for path in "${TEMP_DIRS[@]-}"; do
		[ -n "$path" ] && rm -rf -- "$path"
	done
	return 0
}
trap cleanup_temporary_paths EXIT

register_temp_file() {
	TEMP_FILES+=("$1")
}

register_temp_dir() {
	TEMP_DIRS+=("$1")
}

atomic_install_metadata_file() {
	local source="$1" target="$2" label="$3"
	if [ -e "$target" ] || [ -L "$target" ]; then
		die "refusing to overwrite existing release metadata: $label"
	fi
	# The target directories are created only after validation. `mv -n` keeps a
	# concurrent writer from turning the final rename into an overwrite; the
	# source check below converts a no-op into an explicit failure.
	mv -n -- "$source" "$target" || die "could not atomically install $label"
	[ ! -e "$source" ] || die "release metadata appeared while installing $label"
	if [ ! -f "$target" ] || [ -L "$target" ]; then
		die "atomically installed $label is missing or unsafe"
	fi
}

die() {
	printf 'LunaFox deployment error: %s\n' "$*" >&2
	exit 1
}

usage_error() {
	printf 'LunaFox deployment error: %s\n' "$*" >&2
	exit 2
}

warn() {
	printf 'LunaFox deployment warning: %s\n' "$*" >&2
}

info() {
	printf 'LunaFox deployment: %s\n' "$*"
}

usage() {
	cat <<'USAGE'
Usage:
  ./install.sh [--channel stable|canary] [--registry dockerhub|ghcr] --public-host <host> [--public-port <1-65535>]
  ./install.sh --reset --confirm [--channel stable|canary] [--registry dockerhub|ghcr] --public-host <host> [--public-port <1-65535>]
  ./start.sh | ./restart.sh | ./stop.sh | ./uninstall.sh
  ./uninstall.sh --purge --confirm
  ./start.sh --status

Only Ubuntu Server 22.04/24.04 with rootful native Linux Docker Engine is
officially supported. Docker Hub is the default; --registry ghcr switches the
entire immutable release closure. No automatic registry fallback is performed.
When channels/ and manifests/ are absent during install, the selected release
metadata is fetched from the canonical public release-channel automatically.
USAGE
}

version_at_least() {
	local actual="$1" required="$2" actual_part required_part index
	local -a actual_parts required_parts
	local old_ifs="$IFS"
	IFS=.
	# Keep this comparison independent of GNU sort so the public script also
	# works with the Bash/coreutils set shipped by macOS.
	read -r -a actual_parts <<<"$actual"
	read -r -a required_parts <<<"$required"
	IFS="$old_ifs"
	for index in 0 1 2 3; do
		actual_part="${actual_parts[$index]:-0}"
		required_part="${required_parts[$index]:-0}"
		case "$actual_part" in '' | *[!0-9]*) return 1 ;; esac
		case "$required_part" in '' | *[!0-9]*) return 1 ;; esac
		if [ "$actual_part" -gt "$required_part" ]; then return 0; fi
		if [ "$actual_part" -lt "$required_part" ]; then return 1; fi
	done
	return 0
}

normalize_api_version() {
	printf '%s' "$1" | sed -E 's/[^0-9.].*$//'
}

parse_command() {
	[ "$#" -ge 1 ] || {
		usage >&2
		exit 2
	}
	COMMAND="$1"
	shift
	case "$COMMAND" in install | start | restart | stop | uninstall | status) ;; *)
		usage >&2
		exit 2
		;;
	esac

	while [ "$#" -gt 0 ]; do
		case "$1" in
		--channel)
			[ "$#" -ge 2 ] || die "--channel requires a value"
			CHANNEL="$2"
			CHANNEL_SET=1
			shift 2
			;;
		--registry)
			[ "$#" -ge 2 ] || die "--registry requires a value"
			REGISTRY="$2"
			REGISTRY_SET=1
			shift 2
			;;
		--public-host)
			[ "$#" -ge 2 ] || die "--public-host requires a value"
			PUBLIC_HOST="$2"
			shift 2
			;;
		--public-port)
			[ "$#" -ge 2 ] || die "--public-port requires a value"
			PUBLIC_PORT="$2"
			PUBLIC_PORT_SET=1
			shift 2
			;;
		--reset)
			RESET=1
			shift
			;;
		--purge)
			PURGE=1
			shift
			;;
		--confirm)
			CONFIRM=1
			shift
			;;
		# Keep the retired root dev mode fail-fast and point users at its explicit replacement.
		--dev)
			case "$COMMAND" in
			install) usage_error "--dev has been removed; use ./scripts/dev/install.sh" ;;
			start) usage_error "请使用 ./scripts/dev/start.sh" ;;
			restart) usage_error "请使用 ./scripts/dev/restart.sh" ;;
			*) usage_error "--dev is only available through the development entrypoints" ;;
			esac
			;;
		--status)
			[ "$COMMAND" = start ] || die "--status is only valid with start.sh"
			COMMAND=status
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*) die "unsupported argument: $1" ;;
		esac
	done

	case "$CHANNEL" in stable | canary) ;; *) die "--channel must be stable or canary" ;; esac
	case "$REGISTRY" in dockerhub | ghcr) ;; *) die "--registry must be dockerhub or ghcr" ;; esac
	case "$PUBLIC_PORT" in '' | *[!0-9]*) die "--public-port must be a number" ;; esac
	if [ "$PUBLIC_PORT" -lt 1 ] || [ "$PUBLIC_PORT" -gt 65535 ]; then
		die "--public-port must be between 1 and 65535"
	fi

	case "$COMMAND" in
	install)
		[ "$PURGE" -eq 0 ] || die "--purge is only valid with uninstall.sh"
		if [ "$RESET" -eq 1 ] && [ "$CONFIRM" -ne 1 ]; then die "--reset requires --confirm"; fi
		[ -n "$PUBLIC_HOST" ] || die "install requires --public-host"
		;;
	uninstall)
		[ "$RESET" -eq 0 ] || die "--reset is only valid with install.sh"
		if [ "$PURGE" -eq 1 ] && [ "$CONFIRM" -ne 1 ]; then die "--purge requires --confirm"; fi
		;;
	start | restart | stop | status)
		[ "$RESET" -eq 0 ] || die "--reset is only valid with install.sh"
		[ "$PURGE" -eq 0 ] || die "--purge is only valid with uninstall.sh"
		[ -z "$PUBLIC_HOST" ] || die "--public-host is only valid with install.sh"
		[ "$CHANNEL_SET" -eq 0 ] || die "--channel is only valid with install.sh"
		[ "$REGISTRY_SET" -eq 0 ] || die "--registry is only valid with install.sh"
		[ "$PUBLIC_PORT_SET" -eq 0 ] || die "--public-port is only valid with install.sh"
		;;
	esac
}

require_docker_access() {
	command -v docker >/dev/null 2>&1 || die "docker is required"
	if ! docker info >/dev/null 2>&1; then
		die "current user cannot access Docker; grant Docker access before retrying (no implicit sudo is used)"
	fi
	if [ "$(docker context show 2>/dev/null)" != "default" ]; then
		die "only the local default Docker context is supported; do not use a remote context"
	fi
	local endpoint operating_system
	endpoint="$(docker context inspect default --format '{{(index .Endpoints "docker").Host}}' 2>/dev/null || true)"
	case "$endpoint" in
	unix:///var/run/docker.sock | unix:///run/docker.sock) ;;
	*) die "Docker must use the local rootful Linux socket; Docker Desktop and remote contexts are not supported" ;;
	esac
	operating_system="$(docker info --format '{{.OperatingSystem}}' 2>/dev/null || true)"
	case "$operating_system" in *Docker\ Desktop*) die "Docker Desktop is not supported by this deployment" ;; esac
	if [ "$(docker info --format '{{.OSType}}' 2>/dev/null || true)" != "linux" ]; then
		die "Docker daemon must use Linux containers"
	fi
	# `.Rootless` is not exposed by every Docker API version. SecurityOptions is
	# stable across versions and explicitly identifies rootless daemons.
	local security_options
	security_options="$(docker info --format '{{json .SecurityOptions}}' 2>/dev/null || true)"
	if [[ "$security_options" =~ [Rr]ootless ]]; then
		die "rootless Docker is not supported by this deployment"
	fi

	local client_api daemon_api
	client_api="$(normalize_api_version "$(docker version --format '{{.Client.APIVersion}}' 2>/dev/null || true)")"
	daemon_api="$(normalize_api_version "$(docker version --format '{{.Server.APIVersion}}' 2>/dev/null || true)")"
	if [ -z "$client_api" ] || ! version_at_least "$client_api" "$MIN_DOCKER_API"; then
		die "Docker client API must be >= $MIN_DOCKER_API"
	fi
	if [ -z "$daemon_api" ] || ! version_at_least "$daemon_api" "$MIN_DOCKER_API"; then
		die "Docker daemon API must be >= $MIN_DOCKER_API"
	fi

	docker compose version >/dev/null 2>&1 || die "Docker Compose v2 plugin >= $MIN_COMPOSE_VERSION is required; legacy docker-compose is not supported"
	local compose_version
	compose_version="$(docker compose version --short 2>/dev/null | sed 's/^v//')"
	if [ -z "$compose_version" ] || ! version_at_least "$compose_version" "$MIN_COMPOSE_VERSION"; then
		die "Docker Compose plugin must be >= $MIN_COMPOSE_VERSION"
	fi
}

warn_support_matrix() {
	local id="" version=""
	if [ -r /etc/os-release ]; then
		# shellcheck disable=SC1091
		. /etc/os-release
		id="${ID:-}"
		version="${VERSION_ID:-}"
	fi
	if [ "$id" != ubuntu ] || { [ "$version" != 22.04 ] && [ "$version" != 24.04 ]; }; then
		warn "this host is outside the official Ubuntu Server 22.04/24.04 matrix; capability checks continue but support is experimental"
	fi
	case "$(docker info --format '{{.Architecture}}' 2>/dev/null || true)" in amd64 | x86_64 | arm64 | aarch64) ;; *) die "Docker architecture must be amd64 or arm64" ;; esac
}

compose() {
	docker compose --project-name "$PROJECT_NAME" --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

compose_environment_is_renderable() {
	[ -f "$ENV_FILE" ] && docker compose --project-name "$PROJECT_NAME" --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config -q >/dev/null 2>&1
}

compose_receipt_helper() {
	local service="$1"
	shift
	compose_environment_is_renderable || die "completion receipt helper requires a valid .env"
	compose --profile helpers run --rm --no-deps "$@" "$service"
}

invalidate_receipt_without_env() {
	# A damaged configuration must not block revocation of a stale completion
	# claim. This is the same read-write invalidator used by Compose, isolated
	# from all product volumes and networks before destructive cleanup begins.
	docker run --rm --network none \
		-v "$RECEIPT_VOLUME:/receipt" \
		-v "$ROOT_DIR/scripts/deploy/receipt/invalidate.sh:/usr/local/lib/lunafox/receipt-invalidate.sh:ro" \
		busybox:1.36.1 /bin/sh /usr/local/lib/lunafox/receipt-invalidate.sh >/dev/null ||
		die "could not invalidate the completion receipt"
}

validate_compose_config() {
	if ! compose config -q >/dev/null; then
		die "root Compose configuration is invalid; no resource mutation was performed"
	fi
}

ensure_loki_plugin() {
	local enabled
	enabled="$(docker plugin ls --format '{{.Name}} {{.Enabled}}' | awk '$2 == "true" && $1 ~ /(^|\/)loki(-docker-driver)?(:|$)/ { print $1; exit }')"
	[ -n "$enabled" ] || die "enabled Loki Docker logging plugin is required; install/enable it explicitly before retrying"
}

ensure_volume_capabilities() {
	# Probe with disposable volumes so a failed clean install cannot accidentally
	# become non-empty state before the clean-state decision is made.
	local suffix data execution probe
	suffix="$$-$(date +%s)"
	data="lunafox-preflight-data-$suffix"
	execution="lunafox-preflight-execution-$suffix"
	probe="lunafox-preflight-$suffix"
	docker volume create "$data" >/dev/null
	docker volume create "$execution" >/dev/null
	[ "$(docker volume inspect --format '{{.Name}}' "$data")" = "$data" ] || die "Docker volume identity check failed"
	if ! docker run --rm --pull=missing --network none --mount "type=volume,src=$data,dst=/data" busybox:1.36.1 sh -ec 'mkdir -p /data/subpath && test -d /data/subpath' >/dev/null; then
		docker volume rm "$data" "$execution" >/dev/null 2>&1 || true
		die "named volume mount capability check failed"
	fi
	docker create --name "$probe" --network none \
		--mount "type=volume,src=$data,dst=/data,volume-subpath=subpath" \
		--mount "type=volume,src=$execution,dst=/execution" busybox:1.36.1 true >/dev/null || {
		docker rm -f "$probe" >/dev/null 2>&1 || true
		docker volume rm "$data" "$execution" >/dev/null 2>&1 || true
		die "volume-subpath or sibling Engine Container capability check failed"
	}
	docker rm -f "$probe" >/dev/null || die "failed to remove volume capability probe"
	docker volume rm "$data" "$execution" >/dev/null || die "failed to remove volume capability probes"
}

preflight_mutating() {
	require_docker_access
	warn_support_matrix
	ensure_loki_plugin
	ensure_volume_capabilities
}

validate_host() {
	local host="$1"
	[ -n "$host" ] || die "public host is required"
	if is_ipv4 "$host" || is_ipv6_literal "$host" || is_dns_name "$host"; then return; fi
	die "public host must be a DNS name, IPv4 address, or IPv6 literal without URL syntax"
}

is_ipv4() {
	local host="$1" part
	[[ "$host" =~ ^[0-9]{1,3}(\.[0-9]{1,3}){3}$ ]] || return 1
	local old_ifs="$IFS"
	IFS=.
	read -r -a parts <<<"$host"
	IFS="$old_ifs"
	for part in "${parts[@]}"; do
		[ "$part" -ge 0 ] && [ "$part" -le 255 ] || return 1
	done
}

is_ipv6_literal() {
	local host="$1"
	[[ "$host" == *:* && "$host" =~ ^[0-9A-Fa-f:]+$ ]]
}

is_dns_name() {
	local host="$1"
	[[ "$host" =~ ^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*\.?$ ]] && [ "${#host}" -le 253 ]
}

public_url_host() {
	if is_ipv6_literal "$PUBLIC_HOST"; then printf '[%s]' "$PUBLIC_HOST"; else printf '%s' "$PUBLIC_HOST"; fi
}

certificate_san() {
	if is_ipv4 "$PUBLIC_HOST" || is_ipv6_literal "$PUBLIC_HOST"; then printf 'IP:%s' "$PUBLIC_HOST"; else printf 'DNS:%s' "$PUBLIC_HOST"; fi
}

certificate_text_san() {
	if is_ipv4 "$PUBLIC_HOST" || is_ipv6_literal "$PUBLIC_HOST"; then printf 'IP Address:%s' "$PUBLIC_HOST"; else printf 'DNS:%s' "$PUBLIC_HOST"; fi
}

channel_file() { printf '%s/%s.env' "$CHANNEL_DIR" "$1"; }

release_metadata_state() {
	local channels_present=0 manifests_present=0
	[ -e "$CHANNEL_DIR" ] && channels_present=1
	[ -e "$MANIFEST_DIR" ] && manifests_present=1
	if [ "$channels_present" -eq 0 ] && [ "$manifests_present" -eq 0 ]; then
		printf 'absent\n'
		return
	fi
	if [ "$channels_present" -eq 1 ] && [ "$manifests_present" -eq 1 ] &&
		[ -d "$CHANNEL_DIR" ] && [ ! -L "$CHANNEL_DIR" ] &&
		[ -d "$MANIFEST_DIR" ] && [ ! -L "$MANIFEST_DIR" ]; then
		printf 'present\n'
		return
	fi
	die "release metadata is incomplete; channels/ and manifests/ must both be complete directories (remove both and retry on a clean checkout)"
}

download_canonical_release_file() {
	local relative="$1" destination="$2" url http_code curl_error
	case "$relative" in
	channels/stable.env | channels/canary.env | manifests/v[0-9]*.yaml) ;;
	*) die "refusing to fetch an unsupported release metadata path: $relative" ;;
	esac
	command -v curl >/dev/null 2>&1 || die "curl is required to fetch release metadata"
	url="${CANONICAL_RELEASE_CHANNEL_URL}/${relative}"
	curl_error="$(mktemp)"
	register_temp_file "$curl_error"
	# Redirects are disabled beyond the canonical response: a changed host must
	# never become an implicit release metadata source.
	if ! http_code="$(curl --disable --fail --silent --show-error --location --max-redirs 0 \
		--proto '=https' --proto-redir '=https' --tlsv1.2 \
		--connect-timeout "$RELEASE_METADATA_CONNECT_TIMEOUT" \
		--max-time "$RELEASE_METADATA_MAX_TIME" --write-out '%{http_code}' \
		--output "$destination" "$url" 2>"$curl_error")"; then
		if [ "$http_code" = 404 ] && [ "$relative" = channels/stable.env ]; then
			die "stable channel is not published; install the first canary explicitly with --channel canary"
		fi
		die "could not fetch canonical release metadata: $relative (HTTP ${http_code:-000})"
	fi
	[ "$http_code" = 200 ] || die "canonical release metadata returned unexpected HTTP status for $relative: ${http_code:-000}"
	[ -s "$destination" ] || die "canonical release metadata is empty: $relative"
	[ ! -L "$destination" ] || die "canonical release metadata is a symlink: $relative"
}

file_mode() {
	local file="$1" mode
	mode="$(stat -c '%a' -- "$file" 2>/dev/null || true)"
	if [ -z "$mode" ]; then
		mode="$(stat -f '%Lp' "$file" 2>/dev/null || true)"
	fi
	[ -n "$mode" ] || return 1
	printf '%03d\n' "$((10#$mode))"
}

validate_env_file_structure() {
	local file="$1"
	[ -f "$file" ] || die "environment file is missing: $file"
	if ! awk -F= '
    /^[[:space:]]*$/ || /^[[:space:]]*#/ { next }
    $0 ~ /\r$/ || NF < 2 || $1 !~ /^[A-Za-z_][A-Za-z0-9_]*$/ || seen[$1]++ {
      invalid = 1
    }
    END { exit invalid ? 1 : 0 }
  ' "$file"; then
		die "environment file contains duplicate, malformed, or CRLF entries: $file"
	fi
}

env_value() {
	local file="$1" key="$2"
	awk -F= -v target="$key" '$1 == target {print substr($0, index($0, "=") + 1); exit}' "$file"
}

require_env_keys() {
	local file="$1" key value
	shift
	for key in "$@"; do
		value="$(env_value "$file" "$key")"
		[ -n "$value" ] || die "environment file is missing required value: $key"
	done
}

require_env_entries() {
	local file="$1" key
	shift
	for key in "$@"; do
		if ! awk -F= -v target="$key" '$1 == target { found = 1 } END { exit found ? 0 : 1 }' "$file"; then
			die "environment file is missing required key: $key"
		fi
	done
}

resolve_manifest_path() {
	local relative="$1" candidate manifest_dir manifest_real root_real
	[[ "$relative" =~ ^manifests/v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?\.yaml$ ]] || die "release manifest path must be a repository-relative manifests/vX.Y.Z*.yaml path"
	[[ "$relative" != *$'\n'* && "$relative" != *$'\r'* && "$relative" != *'..'* && "$relative" != /* ]] || die "release manifest path is unsafe"
	candidate="$ROOT_DIR/$relative"
	[ -f "$candidate" ] || die "release manifest is missing: $relative"
	[ ! -L "$candidate" ] || die "release manifest must not be a symlink: $relative"
	manifest_dir="$(cd -- "$(dirname -- "$candidate")" 2>/dev/null && pwd -P)" || die "release manifest directory is unavailable"
	manifest_real="$manifest_dir/$(basename -- "$candidate")"
	root_real="$(cd -- "$ROOT_DIR" && pwd -P)"
	case "$manifest_real" in
	"$root_real"/*) ;;
	*) die "release manifest escapes the repository root" ;;
	esac
	MANIFEST_PATH="$manifest_real"
}

validate_channel_record() {
	local file key expected_tag
	file="$1"
	[ -f "$file" ] || die "release channel record is missing: $file"
	[ ! -L "$file" ] || die "release channel record must not be a symlink: $file"
	validate_env_file_structure "$file"
	require_env_keys "$file" SCHEMA_VERSION VERSION RELEASE_MANIFEST RELEASE_MANIFEST_SHA256
	[ "$(env_value "$file" SCHEMA_VERSION)" = 3 ] || die "release channel must use schema v3"
	RELEASE_TAG="$(env_value "$file" VERSION)"
	MANIFEST_REL="$(env_value "$file" RELEASE_MANIFEST)"
	MANIFEST_SHA256="$(env_value "$file" RELEASE_MANIFEST_SHA256)"
	[[ "$RELEASE_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] || die "release channel VERSION is invalid"
	expected_tag="$(basename "$MANIFEST_REL" .yaml)"
	[ "$expected_tag" = "$RELEASE_TAG" ] || die "release channel manifest and VERSION do not match"
	[ "$MANIFEST_REL" = "manifests/${RELEASE_TAG}.yaml" ] || die "release channel manifest path is invalid"
	[[ "$MANIFEST_SHA256" =~ ^[a-f0-9]{64}$ ]] || die "release channel digest is invalid"
	while IFS= read -r key; do
		case "$key" in SCHEMA_VERSION | VERSION | RELEASE_MANIFEST | RELEASE_MANIFEST_SHA256) ;; *) die "release channel contains unsupported key: $key" ;; esac
	done < <(awk -F= '/^[A-Za-z_][A-Za-z0-9_]*=/{print $1}' "$file")
}

verify_manifest_digest() {
	local file="$1" expected="$2" actual
	[ -f "$file" ] || die "release manifest is missing: $MANIFEST_REL"
	[ ! -L "$file" ] || die "release manifest must not be a symlink: $MANIFEST_REL"
	actual="$(sha256sum "$file" | awk '{print $1}')"
	[ "$actual" = "$expected" ] || die "release manifest digest does not match the channel"
}

exact_checkout_release_tag() {
	local tags tag found=""
	[ -e "$ROOT_DIR/.git" ] || return 0
	command -v git >/dev/null 2>&1 || die "git is required to validate an exact release tag checkout"
	tags="$(git -C "$ROOT_DIR" tag --points-at HEAD 2>/dev/null || true)"
	while IFS= read -r tag; do
		[ -n "$tag" ] || continue
		[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] || continue
		if [ -n "$found" ] && [ "$found" != "$tag" ]; then
			die "exact checkout has multiple release tags; refusing ambiguous release identity"
		fi
		found="$tag"
	done <<<"$tags"
	[ -z "$found" ] || [ "$found" = "$RELEASE_TAG" ] || die "release channel VERSION $RELEASE_TAG does not match exact checkout tag $found"
}

hydrate_release_channel_metadata() {
	local state channel_tmp staging_dir staged_manifest channel_target manifest_target channel_install_tmp manifest_install_tmp
	state="$(release_metadata_state)"
	[ "$state" = absent ] || return 0

	channel_tmp="$(mktemp)"
	register_temp_file "$channel_tmp"
	download_canonical_release_file "channels/${CHANNEL}.env" "$channel_tmp"
	validate_channel_record "$channel_tmp"
	exact_checkout_release_tag

	staging_dir="$(mktemp -d "$ROOT_DIR/.release-metadata.XXXXXX")"
	register_temp_dir "$staging_dir"
	staged_manifest="$staging_dir/$(basename -- "$MANIFEST_REL")"
	download_canonical_release_file "$MANIFEST_REL" "$staged_manifest"
	verify_manifest_digest "$staged_manifest" "$MANIFEST_SHA256"
	MANIFEST_PATH="$staged_manifest"
	validate_manifest_for_release

	# Recheck the empty-state boundary immediately before creating either target
	# directory; a concurrent writer must not cause an existing deployment to be
	# silently merged with freshly fetched metadata.
	if [ -e "$CHANNEL_DIR" ] || [ -e "$MANIFEST_DIR" ]; then
		die "release metadata appeared while fetching; refusing to merge or overwrite it"
	fi
	mkdir "$CHANNEL_DIR" "$MANIFEST_DIR" || die "could not create release metadata directories"
	channel_target="$CHANNEL_DIR/${CHANNEL}.env"
	manifest_target="$MANIFEST_DIR/$(basename -- "$MANIFEST_REL")"
	channel_install_tmp="$(mktemp "$CHANNEL_DIR/.${CHANNEL}.env.XXXXXX")"
	manifest_install_tmp="$(mktemp "$MANIFEST_DIR/.$(basename -- "$MANIFEST_REL").XXXXXX")"
	register_temp_file "$channel_install_tmp"
	register_temp_file "$manifest_install_tmp"
	chmod 0644 "$channel_install_tmp" "$manifest_install_tmp"
	cp "$channel_tmp" "$channel_install_tmp" || die "could not stage release channel record"
	cp "$staged_manifest" "$manifest_install_tmp" || die "could not stage release manifest"
	atomic_install_metadata_file "$channel_install_tmp" "$channel_target" "release channel record"
	atomic_install_metadata_file "$manifest_install_tmp" "$manifest_target" "release manifest"
	MANIFEST_PATH="$manifest_target"
	info "release metadata fetched from canonical release-channel"
}

require_channel() {
	local file
	file="$(channel_file "$CHANNEL")"
	if [ ! -f "$file" ]; then
		if [ "$CHANNEL" = stable ]; then
			die "stable channel is not published; install the first canary explicitly with --channel canary"
		fi
		die "release channel is missing: $CHANNEL"
	fi
	validate_channel_record "$file"
	exact_checkout_release_tag
	resolve_manifest_path "$MANIFEST_REL"
	verify_manifest_digest "$MANIFEST_PATH" "$MANIFEST_SHA256"
}

yaml_runtime_refs() {
	local component="$1" manifest="$2"
	awk -v wanted="$component" '
    function emit(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      gsub(/^[\047\"]|[\047\"]$/, "", value)
      if (value != "") print value
    }
    /^runtimeImages:[[:space:]]*$/ { active = 1; next }
    /^enginePackages:[[:space:]]*$/ { exit }
    active && /^  - name: / {
      name = $0
      sub(/^  - name: /, "", name)
      in_component = (name == wanted)
      next
    }
    active && in_component && /^    refs:[[:space:]]*\[/ {
      values = $0
      sub(/^.*\[/, "", values)
      sub(/\][[:space:]]*$/, "", values)
      count = split(values, refs, ",")
      for (idx = 1; idx <= count; idx++) emit(refs[idx])
      next
    }
    active && in_component && /^      - / {
      value = $0
      sub(/^      - /, "", value)
      emit(value)
    }
  ' "$manifest"
}

yaml_engine_package_refs() {
	local manifest="$1"
	awk '
    function emit(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      gsub(/^[\047\"]|[\047\"]$/, "", value)
      if (value != "") print block "\t" value
    }
    /^enginePackages:[[:space:]]*$/ { active = 1; next }
    !active { next }
    /^  - refs:[[:space:]]*\[/ {
      block++
      values = $0
      sub(/^.*\[/, "", values)
      sub(/\][[:space:]]*$/, "", values)
      count = split(values, refs, ",")
      for (idx = 1; idx <= count; idx++) emit(refs[idx])
      next
    }
    /^  - refs:[[:space:]]*$/ { block++; next }
    /^  - / { invalid = 1; next }
    /^      - / {
      value = $0
      sub(/^      - /, "", value)
      emit(value)
      next
    }
    /^[[:space:]]*$/ { next }
    { invalid = 1 }
    END { if (!active || block == 0 || invalid) exit 1 }
  ' "$manifest"
}

validate_engine_package_closure() {
	local block ref repository digest refs_file index
	local -a blocks=() counts=() docker_refs=() ghcr_refs=()
	ENGINE_PACKAGE_SELECTED_REFS=()
	refs_file="$(mktemp)"
	register_temp_file "$refs_file"
	if ! yaml_engine_package_refs "$MANIFEST_PATH" >"$refs_file"; then
		die "release manifest Engine Package structure is invalid"
	fi
	while IFS=$'\t' read -r block ref; do
		if [ -z "$block" ] || [ -z "$ref" ]; then
			die "release manifest contains an empty Engine Package ref"
		fi
		case "$block" in '' | *[!0-9]*) die "release manifest Engine Package block is invalid" ;; esac
		index=$((block - 1))
		if [ -z "${blocks[$index]+x}" ]; then
			[ "$block" -eq $((${#blocks[@]} + 1)) ] || die "release manifest Engine Package blocks are not contiguous"
			blocks+=("$block")
			counts+=(0)
			docker_refs+=("")
			ghcr_refs+=("")
		fi
		counts[index]=$((counts[index] + 1))
		case "$ref" in
		docker.io/*)
			[ -z "${docker_refs[$index]}" ] || die "release manifest has duplicate Docker Hub Engine Package candidate"
			docker_refs[index]="$ref"
			;;
		ghcr.io/*)
			[ -z "${ghcr_refs[$index]}" ] || die "release manifest has duplicate GHCR Engine Package candidate"
			ghcr_refs[index]="$ref"
			;;
		*) die "release manifest has an unsupported Engine Package registry" ;;
		esac
	done <"$refs_file"

	[ "${#blocks[@]}" -gt 0 ] || die "release manifest has no Engine Packages"
	for index in "${!blocks[@]}"; do
		block="${blocks[$index]}"
		[ "${counts[$index]}" -eq 2 ] || die "release manifest must contain exactly two candidates per Engine Package"
		ref="${docker_refs[$index]:-}"
		[[ "$ref" =~ ^docker\.io/yyhuni/(lunafox-engine-runtime-[a-z0-9-]+)@sha256:([a-f0-9]{64})$ ]] || die "release manifest has an invalid Docker Hub Engine Package ref"
		repository="${BASH_REMATCH[1]}"
		digest="${BASH_REMATCH[2]}"
		[ "${ghcr_refs[$index]:-}" = "ghcr.io/yyhuni/${repository}@sha256:${digest}" ] || die "Engine Package registry candidates must preserve repository and digest identity"
		if [ "$REGISTRY" = dockerhub ]; then
			ENGINE_PACKAGE_SELECTED_REFS+=("$ref")
		else
			ENGINE_PACKAGE_SELECTED_REFS+=("${ghcr_refs[$index]}")
		fi
	done
}

write_engine_inventory() {
	local tmp ref
	[ "${#ENGINE_PACKAGE_SELECTED_REFS[@]}" -gt 0 ] || die "manifest has no Engine packages for selected registry"
	umask 077
	mkdir -p "$STATE_DIR"
	chmod 0700 "$STATE_DIR"
	tmp="$(mktemp "$STATE_DIR/.engine-inventory.XXXXXX")"
	printf '%s\n' 'enginePackages:' >"$tmp"
	for ref in "${ENGINE_PACKAGE_SELECTED_REFS[@]}"; do
		printf '  - refs:\n      - "%s"\n' "$ref" >>"$tmp"
	done
	mv -f "$tmp" "$ENGINE_INVENTORY_PATH"
	chmod 0600 "$ENGINE_INVENTORY_PATH"
}

manifest_release_version() {
	awk -F: '
    /^releaseVersion:[[:space:]]*/ {
      value = substr($0, index($0, ":") + 1)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      gsub(/^[\047\"]|[\047\"]$/, "", value)
      print value
      exit
    }
  ' "$MANIFEST_PATH"
}

validate_manifest_for_release() {
	local version component refs_count docker_ref ghcr_ref digest ref
	local -a refs
	version="$(manifest_release_version)"
	[ "$version" = "${RELEASE_TAG#v}" ] || die "release manifest version does not match the selected channel"
	[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] || die "release manifest version is invalid"
	if grep -Eq '0\.0\.0-dev|alpha\.46|SCHEMA_VERSION=2|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE|lunafox-installer|checksums\.txt' "$MANIFEST_PATH"; then
		die "release manifest contains retired or development identity"
	fi
	for component in server frontend nginx agent bootstrap; do
		refs=()
		while IFS= read -r ref; do
			[ -n "$ref" ] && refs+=("$ref")
		done < <(yaml_runtime_refs "$component" "$MANIFEST_PATH")
		refs_count="${#refs[@]}"
		[ "$refs_count" -eq 2 ] || die "release manifest must contain exactly two refs for runtime image $component"
		docker_ref="${refs[0]}"
		ghcr_ref="${refs[1]}"
		[[ "$docker_ref" =~ ^docker\.io/yyhuni/lunafox-${component}@sha256:([a-f0-9]{64})$ ]] || die "release manifest has an invalid Docker Hub ref for runtime image $component"
		digest="${BASH_REMATCH[1]}"
		[ "$ghcr_ref" = "ghcr.io/yyhuni/lunafox-${component}@sha256:${digest}" ] || die "runtime image registry candidates must preserve repository and digest identity: $component"
	done
	validate_engine_package_closure
}

parse_manifest() {
	local write_inventory="${1:-write}"
	local prefix="docker.io/yyhuni/"
	if [ "$REGISTRY" = ghcr ]; then prefix="ghcr.io/yyhuni/"; fi
	validate_manifest_for_release
	local component ref
	for component in server frontend nginx agent bootstrap; do
		ref="$(yaml_runtime_refs "$component" "$MANIFEST_PATH" | awk -v prefix="$prefix" 'index($0, prefix) == 1 { print; exit }')"
		[ -n "$ref" ] || die "release manifest has no $REGISTRY ref for runtime image $component"
		[[ "$ref" =~ ^${prefix//\//\/}lunafox-${component}@sha256:[a-f0-9]{64}$ ]] || die "runtime image ref is not digest-qualified: $component"
		case "$component" in
		server) SERVER_IMAGE_REF="$ref" ;; frontend) FRONTEND_IMAGE_REF="$ref" ;; nginx) NGINX_IMAGE_REF="$ref" ;; agent) AGENT_IMAGE_REF="$ref" ;; bootstrap) BOOTSTRAP_IMAGE_REF="$ref" ;;
		esac
	done
	if [ "$write_inventory" = write ]; then
		write_engine_inventory
	fi
}

random_hex() {
	local bytes="$1"
	local value
	if [ -r /dev/urandom ]; then
		value="$(od -An -N "$bytes" -tx1 /dev/urandom | tr -d ' \n')"
		[ "${#value}" -eq $((bytes * 2)) ] || die "OS CSPRNG returned an incomplete value"
		printf '%s' "$value"
		return
	fi
	die "OS CSPRNG /dev/urandom is unavailable"
}

expected_public_url() {
	local url
	url="https://$(public_url_host)"
	[ "$PUBLIC_PORT" = 443 ] || url="${url}:${PUBLIC_PORT}"
	printf '%s' "$url"
}

atomic_write_env() {
	[ ! -e "$ENV_FILE" ] || die ".env already exists; use install.sh --reset --confirm before creating a new deployment"
	umask 077
	mkdir -p "$STATE_DIR"
	chmod 0700 "$STATE_DIR"
	local tmp
	tmp="$(mktemp "$ROOT_DIR/.env.XXXXXX")"
	register_temp_file "$tmp"
	chmod 0600 "$tmp"
	local public_url
	public_url="$(expected_public_url)"
	cat >"$tmp" <<EOF
# Generated by LunaFox public deployment lifecycle. Do not edit while deployed.
RELEASE_CHANNEL=${CHANNEL}
RELEASE_VERSION=${RELEASE_TAG#v}
AGENT_VERSION=${RELEASE_TAG#v}
RELEASE_MANIFEST_PATH=${MANIFEST_REL}
RELEASE_MANIFEST_SHA256=${MANIFEST_SHA256}
ENGINE_INVENTORY_HOST_PATH=${ENGINE_INVENTORY_PATH}
ENGINE_INSTALL_REGISTRY=$(case "$REGISTRY" in dockerhub) printf docker.io ;; ghcr) printf ghcr.io ;; esac)
SERVER_IMAGE_REF=${SERVER_IMAGE_REF}
FRONTEND_IMAGE_REF=${FRONTEND_IMAGE_REF}
NGINX_IMAGE_REF=${NGINX_IMAGE_REF}
AGENT_IMAGE_REF=${AGENT_IMAGE_REF}
BOOTSTRAP_IMAGE_REF=${BOOTSTRAP_IMAGE_REF}
PUBLIC_HOST=${PUBLIC_HOST}
PUBLIC_URL=${public_url}
PUBLIC_PORT=${PUBLIC_PORT}
LOKI_PUSH_URL=${public_url}/loki/api/v1/push
JWT_SECRET=$(random_hex 32)
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=$(random_hex 24)
DB_NAME=lunafox
DB_SSLMODE=disable
DB_TIMEZONE=UTC
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=300
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
SERVER_PORT=8080
SERVER_GRPC_PORT=9090
GIN_MODE=release
LOG_LEVEL=info
LOKI_URL=http://loki:3100
WORDLISTS_BASE_PATH=/opt/lunafox/wordlists
ENGINE_PACKAGE_CACHE_ROOT=/opt/lunafox/engine-packages
FINGERPRINT_ARTIFACTS_BASE_PATH=/opt/lunafox/fingerprint-artifacts
LOGIN_VISUALS_BASE_PATH=/opt/lunafox/login-visuals
WORKFLOW_DEFINITIONS_ROOT=/usr/local/share/lunafox/workflows
LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox:rw
EOF
	mv -f "$tmp" "$ENV_FILE"
	chmod 0600 "$ENV_FILE"
}

load_existing_env() {
	[ -f "$ENV_FILE" ] || die ".env is missing; run install.sh on a clean host"
	[ "$(file_mode "$ENV_FILE")" = 600 ] || die ".env must have mode 0600"
	validate_env_file_structure "$ENV_FILE"

	local key value expected_url manifest_digest inventory_path inventory_mode
	local required_keys=(RELEASE_CHANNEL RELEASE_VERSION AGENT_VERSION RELEASE_MANIFEST_PATH RELEASE_MANIFEST_SHA256 ENGINE_INVENTORY_HOST_PATH ENGINE_INSTALL_REGISTRY SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF PUBLIC_HOST PUBLIC_URL PUBLIC_PORT LOKI_PUSH_URL JWT_SECRET DB_HOST DB_PORT DB_USER DB_PASSWORD DB_NAME DB_SSLMODE DB_TIMEZONE DB_MAX_OPEN_CONNS DB_MAX_IDLE_CONNS DB_CONN_MAX_LIFETIME REDIS_HOST REDIS_PORT REDIS_PASSWORD REDIS_DB SERVER_PORT SERVER_GRPC_PORT GIN_MODE LOG_LEVEL LOKI_URL WORDLISTS_BASE_PATH ENGINE_PACKAGE_CACHE_ROOT FINGERPRINT_ARTIFACTS_BASE_PATH LOGIN_VISUALS_BASE_PATH WORKFLOW_DEFINITIONS_ROOT LUNAFOX_SHARED_DATA_VOLUME_BIND)
	local allowed_keys=("${required_keys[@]}" JWT_ACCESS_EXPIRE JWT_REFRESH_EXPIRE NUCLEI_POC_WORKSPACE_ROOT HTTP_PROXY HTTPS_PROXY ALL_PROXY NO_PROXY http_proxy https_proxy all_proxy no_proxy)
	require_env_entries "$ENV_FILE" "${required_keys[@]}"
	while IFS= read -r key; do
		case " ${allowed_keys[*]} " in *" $key "*) ;; *) die ".env contains unsupported key: $key" ;; esac
	done < <(awk -F= '/^[A-Za-z_][A-Za-z0-9_]*=/{print $1}' "$ENV_FILE")

	value="$(env_value "$ENV_FILE" RELEASE_CHANNEL)"
	case "$value" in stable | canary) CHANNEL="$value" ;; *) die ".env has an invalid RELEASE_CHANNEL" ;; esac
	value="$(env_value "$ENV_FILE" RELEASE_VERSION)"
	[[ "$value" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]] || die ".env has an invalid RELEASE_VERSION"
	RELEASE_TAG="v$value"
	[ "$(env_value "$ENV_FILE" AGENT_VERSION)" = "$value" ] || die ".env AGENT_VERSION must match RELEASE_VERSION"
	MANIFEST_REL="$(env_value "$ENV_FILE" RELEASE_MANIFEST_PATH)"
	MANIFEST_SHA256="$(env_value "$ENV_FILE" RELEASE_MANIFEST_SHA256)"
	[[ "$MANIFEST_SHA256" =~ ^[a-f0-9]{64}$ ]] || die ".env has an invalid RELEASE_MANIFEST_SHA256"
	[ "$MANIFEST_REL" = "manifests/${RELEASE_TAG}.yaml" ] || die ".env has an invalid RELEASE_MANIFEST_PATH"
	resolve_manifest_path "$MANIFEST_REL"
	manifest_digest="$(sha256sum "$MANIFEST_PATH" | awk '{print $1}')"
	[ "$manifest_digest" = "$MANIFEST_SHA256" ] || die "stored release manifest digest does not match .env"

	PUBLIC_HOST="$(env_value "$ENV_FILE" PUBLIC_HOST)"
	validate_host "$PUBLIC_HOST"
	PUBLIC_PORT="$(env_value "$ENV_FILE" PUBLIC_PORT)"
	case "$PUBLIC_PORT" in '' | *[!0-9]*) die ".env PUBLIC_PORT must be numeric" ;; esac
	if [ "$PUBLIC_PORT" -lt 1 ] || [ "$PUBLIC_PORT" -gt 65535 ]; then
		die ".env PUBLIC_PORT is outside 1-65535"
	fi
	expected_url="$(expected_public_url)"
	[ "$(env_value "$ENV_FILE" PUBLIC_URL)" = "$expected_url" ] || die ".env PUBLIC_URL does not match PUBLIC_HOST/PUBLIC_PORT"
	[ "$(env_value "$ENV_FILE" LOKI_PUSH_URL)" = "$expected_url/loki/api/v1/push" ] || die ".env LOKI_PUSH_URL does not match PUBLIC_URL"

	[[ "$(env_value "$ENV_FILE" JWT_SECRET)" =~ ^[a-f0-9]{64}$ ]] || die ".env JWT_SECRET must be 64 lowercase hex characters"
	[[ "$(env_value "$ENV_FILE" DB_PASSWORD)" =~ ^[a-f0-9]{48}$ ]] || die ".env DB_PASSWORD must be 48 lowercase hex characters"
	[ "$(env_value "$ENV_FILE" LUNAFOX_SHARED_DATA_VOLUME_BIND)" = "lunafox_data:/opt/lunafox:rw" ] || die ".env shared data volume binding is invalid"

	ENGINE_INSTALL_REGISTRY="$(env_value "$ENV_FILE" ENGINE_INSTALL_REGISTRY)"
	case "$ENGINE_INSTALL_REGISTRY" in
	docker.io)
		REGISTRY=dockerhub
		ENGINE_INSTALL_REGISTRY=docker.io
		;;
	ghcr.io)
		REGISTRY=ghcr
		ENGINE_INSTALL_REGISTRY=ghcr.io
		;;
	*) die "stored .env has an unsupported Engine registry identity" ;;
	esac
	parse_manifest verify
	for key in SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF; do
		value="$(env_value "$ENV_FILE" "$key")"
		[ "$value" = "${!key}" ] || die ".env $key does not match the selected release manifest"
	done
	inventory_path="$(env_value "$ENV_FILE" ENGINE_INVENTORY_HOST_PATH)"
	[ "$inventory_path" = "$ENGINE_INVENTORY_PATH" ] || die ".env Engine inventory path is not project-owned"
	if [ ! -f "$inventory_path" ] || [ -L "$inventory_path" ]; then
		die "stored Engine inventory is missing or unsafe"
	fi
	inventory_mode="$(file_mode "$inventory_path")"
	[ "$inventory_mode" = 600 ] || die "Engine inventory must have mode 0600"
	if ! diff -u <(
		printf '%s\n' 'enginePackages:'
		for value in "${ENGINE_PACKAGE_SELECTED_REFS[@]}"; do
			printf '  - refs:\n      - "%s"\n' "$value"
		done
	) "$inventory_path" >/dev/null; then
		die "stored Engine inventory does not exactly match the selected release manifest"
	fi
	validate_manifest_for_release
}

run_probe_container() {
	local image="$1" name="$2"
	docker run --rm --pull=never --network none --name "$name" "$image" sh -c 'true' >/dev/null
}

provision_certificate_volume() {
	ensure_owned_external_volume "$SSL_VOLUME" || die "failed to create certificate volume"
	local has_cert has_key result
	if docker run --rm --network none -v "$SSL_VOLUME:/ssl:ro" busybox:1.36.1 sh -c '[ -s /ssl/fullchain.pem ]' >/dev/null 2>&1; then
		has_cert=0
	else
		result=$?
		[ "$result" -eq 1 ] || die "could not inspect certificate volume"
		has_cert=1
	fi
	if docker run --rm --network none -v "$SSL_VOLUME:/ssl:ro" busybox:1.36.1 sh -c '[ -s /ssl/privkey.pem ]' >/dev/null 2>&1; then
		has_key=0
	else
		result=$?
		[ "$result" -eq 1 ] || die "could not inspect certificate volume"
		has_key=1
	fi
	if [ "$has_cert" -eq 0 ] && [ "$has_key" -eq 0 ]; then
		validate_certificate
		return
	fi
	if [ "$has_cert" -ne 1 ] || [ "$has_key" -ne 1 ]; then
		die "certificate volume is incomplete; recover with install.sh --reset --confirm"
	fi
	local cert_tmp key_tmp conf_tmp
	cert_tmp="$(mktemp)"
	key_tmp="$(mktemp)"
	conf_tmp="$(mktemp)"
	register_temp_file "$cert_tmp"
	register_temp_file "$key_tmp"
	register_temp_file "$conf_tmp"
	cat >"$conf_tmp" <<EOF
[req]
  distinguished_name=dn
x509_extensions=v3_req
prompt=no
[v3_req]
subjectAltName=$(certificate_san)
[dn]
CN=${PUBLIC_HOST}
EOF
	openssl req -x509 -nodes -newkey rsa:2048 -sha256 -days 365 \
		-keyout "$key_tmp" -out "$cert_tmp" -config "$conf_tmp" -extensions v3_req >/dev/null 2>&1 || die "failed to generate self-signed certificate"
	docker run --rm -v "$SSL_VOLUME:/ssl" -v "$cert_tmp:/src/fullchain.pem:ro" -v "$key_tmp:/src/privkey.pem:ro" busybox:1.36.1 \
		sh -c 'umask 077; cp /src/fullchain.pem /ssl/fullchain.pem.tmp && cp /src/privkey.pem /ssl/privkey.pem.tmp && chmod 0644 /ssl/fullchain.pem.tmp && chmod 0600 /ssl/privkey.pem.tmp && mv /ssl/fullchain.pem.tmp /ssl/fullchain.pem && mv /ssl/privkey.pem.tmp /ssl/privkey.pem' \
		>/dev/null || die "failed to atomically write certificate volume"
	validate_certificate
}

validate_existing_certificate_volume() {
	docker volume inspect "$SSL_VOLUME" >/dev/null 2>&1 || die "certificate volume is missing; recover with install.sh --reset --confirm"
	validate_certificate
}

validate_certificate() {
	local cert_tmp key_tmp cert_mod key_mod
	cert_tmp="$(mktemp)"
	key_tmp="$(mktemp)"
	cert_mod="$(mktemp)"
	key_mod="$(mktemp)"
	register_temp_file "$cert_tmp"
	register_temp_file "$key_tmp"
	register_temp_file "$cert_mod"
	register_temp_file "$key_mod"
	if ! docker run --rm --network none -v "$SSL_VOLUME:/ssl:ro" -v "$cert_tmp:/out/fullchain.pem" -v "$key_tmp:/out/privkey.pem" busybox:1.36.1 \
		sh -c 'test -s /ssl/fullchain.pem && test -s /ssl/privkey.pem && cat /ssl/fullchain.pem > /out/fullchain.pem && cat /ssl/privkey.pem > /out/privkey.pem' >/dev/null; then
		die "certificate volume is incomplete; recover with install.sh --reset --confirm"
	fi
	openssl x509 -in "$cert_tmp" -noout -checkend 0 >/dev/null 2>&1 || die "certificate is invalid or expired; recover with install.sh --reset --confirm"
	openssl x509 -in "$cert_tmp" -noout -ext subjectAltName | grep -Fq "$(certificate_text_san)" || die "certificate SAN does not match configured public host"
	openssl x509 -noout -modulus -in "$cert_tmp" | openssl sha256 | awk '{print $2}' >"$cert_mod"
	openssl rsa -noout -modulus -in "$key_tmp" 2>/dev/null | openssl sha256 | awk '{print $2}' >"$key_mod"
	if [ ! -s "$cert_mod" ] || [ ! -s "$key_mod" ]; then
		die "certificate private key is unreadable; recover with install.sh --reset --confirm"
	fi
	cmp -s "$cert_mod" "$key_mod" || die "certificate private key does not match fullchain"
}

assert_clean_state() {
	local found=0 state_file
	[ -e "$ENV_FILE" ] && found=1
	[ -d "$STATE_DIR" ] && found=1
	state_file="$(mktemp)"
	register_temp_file "$state_file"
	if ! docker ps -a --filter "label=com.docker.compose.project=$PROJECT_NAME" --format '{{.ID}}' >"$state_file" 2>/dev/null; then
		die "could not inspect existing LunaFox containers"
	fi
	if [ -s "$state_file" ]; then found=1; fi
	for volume in "${OWNED_VOLUMES[@]}"; do
		if docker volume inspect "$volume" >/dev/null 2>&1; then
			found=1
		else
			case "$?" in 1) ;; *) die "could not inspect Docker volume: $volume" ;; esac
		fi
	done
	[ "$found" -eq 0 ] || die "existing LunaFox state was found; use install.sh --reset --confirm or uninstall.sh"
}

invalidate_receipt() {
	ensure_owned_external_volume "$RECEIPT_VOLUME" || die "failed to create completion receipt volume"
	if compose_environment_is_renderable; then
		compose_receipt_helper receipt-invalidator >/dev/null
	else
		invalidate_receipt_without_env
	fi
}

reset_state() {
	info "invalidating completion receipt before reset"
	invalidate_receipt
	if compose_environment_is_renderable; then
		compose down --remove-orphans >/dev/null 2>&1 || remove_project_resources_without_env
	else
		remove_project_resources_without_env
	fi
	remove_project_agent
	remove_owned_volumes
	remove_owned_host_state
}

remove_project_resources_without_env() {
	local containers container
	containers="$(docker ps -aq --filter "label=com.docker.compose.project=$PROJECT_NAME")" || die "could not inspect LunaFox containers"
	for container in $containers; do
		docker rm -f "$container" >/dev/null || die "could not remove LunaFox container"
	done
	if docker network inspect "$PROJECT_NETWORK" >/dev/null 2>&1; then
		[ "$(docker network inspect --format '{{index .Labels "com.docker.compose.project"}}' "$PROJECT_NETWORK" 2>/dev/null || true)" = "$PROJECT_NAME" ] || die "refusing to remove unowned network: $PROJECT_NETWORK"
		docker network rm "$PROJECT_NETWORK" >/dev/null || die "could not remove LunaFox network"
	fi
}

ensure_owned_external_volume() {
	local volume="$1" owner
	if docker volume inspect "$volume" >/dev/null 2>&1; then
		owner="$(docker volume inspect --format "{{index .Labels \"$EXTERNAL_VOLUME_OWNER_LABEL\"}}" "$volume" 2>/dev/null || true)"
		[ "$owner" = "$EXTERNAL_VOLUME_OWNER_VALUE" ] || die "refusing to use external volume without LunaFox ownership label: $volume"
		return
	fi
	docker volume create --label "$EXTERNAL_VOLUME_OWNER_LABEL=$EXTERNAL_VOLUME_OWNER_VALUE" "$volume" >/dev/null
}

ensure_project_volume() {
	local volume="$1" compose_volume="$2"
	if docker volume inspect "$volume" >/dev/null 2>&1; then
		require_project_volume "$volume" "$compose_volume"
		return
	fi
	docker volume create \
		--label "com.docker.compose.project=$PROJECT_NAME" \
		--label "com.docker.compose.volume=$compose_volume" \
		"$volume" >/dev/null || die "could not create project-owned volume: $volume"
}

require_project_volume() {
	local volume="$1" compose_volume="$2" owner declared
	docker volume inspect "$volume" >/dev/null 2>&1 || die "required project-owned volume is missing: $volume"
	owner="$(docker volume inspect --format "{{index .Labels \"com.docker.compose.project\"}}" "$volume" 2>/dev/null || true)"
	declared="$(docker volume inspect --format "{{index .Labels \"com.docker.compose.volume\"}}" "$volume" 2>/dev/null || true)"
	[ "$owner" = "$PROJECT_NAME" ] || die "refusing to use volume without LunaFox Compose ownership label: $volume"
	[ "$declared" = "$compose_volume" ] || die "refusing to use volume with an unexpected Compose declaration: $volume"
}

ensure_fresh_runtime_volumes() {
	ensure_project_volume lunafox_data lunafox_data
	ensure_project_volume lunafox_engine_execution lunafox_engine_execution
	ensure_project_volume lunafox_agent_state lunafox_agent_state
}

require_runtime_volumes() {
	require_project_volume lunafox_data lunafox_data
	require_project_volume lunafox_engine_execution lunafox_engine_execution
	require_project_volume lunafox_agent_state lunafox_agent_state
}

remove_project_agent() {
	if docker inspect "$AGENT_CONTAINER" >/dev/null 2>&1; then
		[ "$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$AGENT_CONTAINER" 2>/dev/null || true)" = "$PROJECT_NAME" ] || die "refusing to remove unowned Agent container: $AGENT_CONTAINER"
		docker rm -f "$AGENT_CONTAINER" >/dev/null || die "could not remove the Agent container"
	fi
}

require_project_agent() {
	docker inspect "$AGENT_CONTAINER" >/dev/null 2>&1 || die "LunaFox Agent container is missing; recover with install.sh --reset --confirm"
	[ "$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$AGENT_CONTAINER" 2>/dev/null || true)" = "$PROJECT_NAME" ] || die "refusing to use an unowned Agent container: $AGENT_CONTAINER"
	[ "$(docker inspect --format '{{index .Config.Labels "com.docker.compose.service"}}' "$AGENT_CONTAINER" 2>/dev/null || true)" = agent ] || die "Agent container has an unexpected Compose service label"
}

wait_for_project_agent() {
	local state
	require_project_agent
	for _ in $(seq 1 15); do
		state="$(docker inspect --format '{{.State.Running}}' "$AGENT_CONTAINER" 2>/dev/null || true)"
		[ "$state" = true ] && return
		sleep 1
	done
	docker logs "$AGENT_CONTAINER" >&2 || true
	die "LunaFox Agent container did not remain running"
}

start_project_agent() {
	local state
	require_project_agent
	state="$(docker inspect --format '{{.State.Running}}' "$AGENT_CONTAINER" 2>/dev/null || true)"
	if [ "$state" != true ]; then
		docker start "$AGENT_CONTAINER" >/dev/null || die "could not start LunaFox Agent container"
	fi
	wait_for_project_agent
}

restart_project_agent() {
	require_project_agent
	docker restart "$AGENT_CONTAINER" >/dev/null || die "could not restart LunaFox Agent container"
	wait_for_project_agent
}

stop_project_agent() {
	if ! docker inspect "$AGENT_CONTAINER" >/dev/null 2>&1; then
		return
	fi
	[ "$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$AGENT_CONTAINER" 2>/dev/null || true)" = "$PROJECT_NAME" ] || die "refusing to stop an unowned Agent container: $AGENT_CONTAINER"
	docker stop "$AGENT_CONTAINER" >/dev/null || die "could not stop LunaFox Agent container"
}

remove_owned_volumes() {
	local volume owner
	for volume in "${OWNED_VOLUMES[@]}"; do
		if ! docker volume inspect "$volume" >/dev/null 2>&1; then
			continue
		fi
		case "$volume" in
		"$SSL_VOLUME" | "$RECEIPT_VOLUME")
			owner="$(docker volume inspect --format "{{index .Labels \"$EXTERNAL_VOLUME_OWNER_LABEL\"}}" "$volume" 2>/dev/null || true)"
			[ "$owner" = "$EXTERNAL_VOLUME_OWNER_VALUE" ] || die "refusing to remove external volume without LunaFox ownership label: $volume"
			;;
		*)
			owner="$(docker volume inspect --format '{{index .Labels "com.docker.compose.project"}}' "$volume" 2>/dev/null || true)"
			[ "$owner" = "$PROJECT_NAME" ] || die "refusing to remove volume without LunaFox Compose ownership label: $volume"
			;;
		esac
		docker volume rm "$volume" >/dev/null || die "could not remove project-owned volume: $volume"
	done
}

remove_owned_host_state() {
	rm -rf -- "$STATE_DIR" || die "could not remove host state"
	rm -f -- "$ENV_FILE" || die "could not remove generated environment"
}

verify_runtime_health() {
	validate_existing_certificate_volume
	wait_for_compose_health
	wait_for_project_agent
	curl --fail --silent --show-error --insecure --max-time 10 "$(env_value "$ENV_FILE" PUBLIC_URL)/healthChecks/current" >/dev/null || die "public HTTPS health check failed"
}

wait_for_compose_health() {
	local services=("$@") service status state container_id
	if [ "${#services[@]}" -eq 0 ]; then
		services=(postgres redis loki server frontend nginx)
	fi
	for service in "${services[@]}"; do
		status=""
		state=""
		container_id=""
		for _ in $(seq 1 60); do
			container_id="$(compose ps -q "$service" 2>/dev/null || true)"
			if [ -n "$container_id" ]; then
				state="$(docker inspect -f '{{.State.Status}}' "$container_id" 2>/dev/null || true)"
				status="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container_id" 2>/dev/null || true)"
				if [ "$status" = healthy ]; then break; fi
				if [ "$state" = exited ] || [ "$state" = dead ]; then
					compose logs "$service" >&2 || true
					die "$service exited before becoming healthy"
				fi
			fi
			sleep 2
		done
		if [ "$status" != healthy ]; then
			compose ps >&2 || true
			die "$service did not become healthy"
		fi
	done
}

run_mount_capability_probe() {
	local agent_ref="$1" suffix result
	suffix="$$-$(date +%s)"
	require_project_volume lunafox_engine_execution lunafox_engine_execution
	require_project_volume lunafox_data lunafox_data
	docker pull "$agent_ref" >/dev/null || die "cannot pull selected Agent image"
	if docker run --rm --pull=never --network none --name "lunafox-mount-preflight-$suffix" \
		--entrypoint /usr/local/bin/lunafox-engine-mount-preflight \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v lunafox_engine_execution:/var/lib/lunafox/engine-execution \
		-v lunafox_data:/opt/lunafox \
		"$agent_ref" --profile capability --image-ref "$agent_ref" \
		--execution-volume lunafox_engine_execution --execution-root /var/lib/lunafox/engine-execution \
		--shared-volume lunafox_data --shared-root /opt/lunafox --run-id "public-install-$suffix" \
		>/dev/null; then
		result=0
	else
		result=$?
	fi
	[ "$result" -eq 0 ] || die "Docker volume-subpath and sibling Engine Container capability preflight failed"
}

pull_release_closure() {
	local ref engine_ref
	for ref in "$(env_value "$ENV_FILE" SERVER_IMAGE_REF)" "$(env_value "$ENV_FILE" FRONTEND_IMAGE_REF)" "$(env_value "$ENV_FILE" NGINX_IMAGE_REF)" "$(env_value "$ENV_FILE" AGENT_IMAGE_REF)" "$(env_value "$ENV_FILE" BOOTSTRAP_IMAGE_REF)"; do
		docker pull "$ref" >/dev/null || die "cannot pull immutable release artifact: $ref"
	done
	while IFS= read -r engine_ref; do
		[ -n "$engine_ref" ] || continue
		docker pull "$engine_ref" >/dev/null || die "cannot pull immutable Engine Package artifact: $engine_ref"
	done < <(awk '/^      - / { value=$0; sub(/^      - "?/, "", value); sub(/"?$/, "", value); print value }' "$ENGINE_INVENTORY_PATH")
}

bootstrap_release() {
	pull_release_closure
	# The bootstrap container creates the sibling Agent through the Docker API,
	# so Compose cannot infer its state volume from a service mount. All three
	# runtime volumes must already be project-owned before bootstrap inspects
	# them or creates the Agent.
	require_runtime_volumes
	compose up -d --no-build postgres redis loki server
	wait_for_compose_health postgres redis loki server
	compose up -d --no-build bootstrap
	local bootstrap_id state exit_code
	bootstrap_id="$(compose ps -q bootstrap)"
	[ -n "$bootstrap_id" ] || die "bootstrap container was not created"
	for _ in $(seq 1 120); do
		state="$(docker inspect -f '{{.State.Status}}' "$bootstrap_id" 2>/dev/null || true)"
		[ "$state" = exited ] && break
		sleep 2
	done
	state="$(docker inspect -f '{{.State.Status}}' "$bootstrap_id" 2>/dev/null || true)"
	exit_code="$(docker inspect -f '{{.State.ExitCode}}' "$bootstrap_id" 2>/dev/null || true)"
	if [ "$state" != exited ] || [ "$exit_code" != 0 ]; then
		compose logs bootstrap >&2 || true
		die "bootstrap failed; inspect its logs, then use install.sh --reset --confirm for the supported recovery"
	fi
	compose up -d --no-build frontend nginx
	wait_for_compose_health frontend nginx
}

finalize_receipt() {
	compose_receipt_helper receipt-finalizer \
		-e "RECEIPT_RELEASE_VERSION=${RELEASE_TAG#v}" \
		-e "RECEIPT_MANIFEST_SHA256=$MANIFEST_SHA256" \
		-e "RECEIPT_REGISTRY=$REGISTRY" >/dev/null
}

verify_receipt_and_health() {
	compose_receipt_helper receipt-verifier \
		-e "RECEIPT_RELEASE_VERSION=${RELEASE_TAG#v}" \
		-e "RECEIPT_MANIFEST_SHA256=$MANIFEST_SHA256" \
		-e "RECEIPT_REGISTRY=$REGISTRY" >/dev/null || die "deployment has no valid completion receipt"
	verify_runtime_health
}

ensure_no_port_conflict() {
	local port="$1"
	local published
	if ! published="$(docker ps --format '{{.Names}} {{.Ports}}')"; then
		die "could not inspect existing published ports"
	fi
	if awk -v port=":${port}->" '$0 ~ port { found = 1 } END { exit found ? 0 : 1 }' <<<"$published"; then
		die "host port $port is already published by another container"
	fi
}

install() {
	preflight_mutating
	validate_host "$PUBLIC_HOST"
	hydrate_release_channel_metadata
	require_channel
	parse_manifest verify
	if [ "$RESET" -eq 1 ]; then
		reset_state
	else
		assert_clean_state
	fi
	parse_manifest
	ensure_no_port_conflict "$PUBLIC_PORT"
	ensure_fresh_runtime_volumes
	run_mount_capability_probe "$AGENT_IMAGE_REF"
	atomic_write_env
	validate_compose_config
	provision_certificate_volume
	ensure_owned_external_volume "$RECEIPT_VOLUME"
	bootstrap_release
	verify_runtime_health
	finalize_receipt
	compose_receipt_helper receipt-verifier \
		-e "RECEIPT_RELEASE_VERSION=${RELEASE_TAG#v}" \
		-e "RECEIPT_MANIFEST_SHA256=$MANIFEST_SHA256" \
		-e "RECEIPT_REGISTRY=$REGISTRY" >/dev/null || die "new completion receipt could not be verified"
	info "deployment completed: $(env_value "$ENV_FILE" PUBLIC_URL)"
}

start_or_restart() {
	preflight_mutating
	load_existing_env
	validate_compose_config
	validate_existing_certificate_volume
	require_runtime_volumes
	run_mount_capability_probe "$(env_value "$ENV_FILE" AGENT_IMAGE_REF)"
	if [ "$COMMAND" = restart ]; then
		compose restart postgres redis loki server frontend nginx
	else
		compose up -d --no-build postgres redis loki server frontend nginx
	fi
	wait_for_compose_health postgres redis loki server frontend nginx
	if [ "$COMMAND" = restart ]; then
		restart_project_agent
	else
		start_project_agent
	fi
	verify_receipt_and_health
	info "deployment is complete and healthy"
}

stop() {
	preflight_mutating
	[ -f "$ENV_FILE" ] || die "no LunaFox deployment configuration exists"
	load_existing_env
	validate_compose_config
	stop_project_agent
	compose stop
	info "resident Compose services stopped; configuration, certificates, data, and completion receipt were preserved"
}

status() {
	require_docker_access
	load_existing_env
	validate_compose_config
	verify_receipt_and_health
	compose ps
}

uninstall() {
	preflight_mutating
	if [ "$PURGE" -eq 1 ]; then
		info "invalidating completion receipt before purge"
		invalidate_receipt
	fi
	if compose_environment_is_renderable; then
		# A valid .env lets Compose remove the exact declared resource graph. If it
		# cannot do so, fall back to labels rather than leaving an unrecoverable
		# partially deleted deployment.
		compose down --remove-orphans >/dev/null 2>&1 || remove_project_resources_without_env
	else
		warn ".env is missing or invalid; removing only resources carrying LunaFox ownership labels"
		remove_project_resources_without_env
	fi
	remove_project_agent
	if [ "$PURGE" -eq 1 ]; then
		remove_owned_volumes
		remove_owned_host_state
		info "deployment and project-owned persistent state were purged"
	else
		info "deployment resources removed; data, configuration, certificates, and receipt were preserved"
	fi
}

main() {
	parse_command "$@"
	[ -f "$COMPOSE_FILE" ] || die "root compose.yaml is missing"
	case "$COMMAND" in
	install) install ;;
	start | restart) start_or_restart ;;
	stop) stop ;;
	uninstall) uninstall ;;
	status) status ;;
	esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
	main "$@"
fi
