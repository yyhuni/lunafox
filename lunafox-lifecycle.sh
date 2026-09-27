#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
#
# Shared implementation for the LunaFox public Compose lifecycle scripts.
# Each root script validates its own arguments and execs this file with an
# action name, so the behaviour below is one implementation, not seven.
#
# Constraints that are not visible from the code:
# - The deployment root is the directory holding this helper, which makes an
#   invocation from any working directory equivalent.
# - Compose is always called with an explicit project name, project directory,
#   env file, and file stack. compose.override.yaml is appended only when the
#   upgrader installed it as a regular file; passing -f disables Compose's own
#   override discovery, so dropping it would silently revert confirmed images.
# - The host environment is cleared of every key that could outrank the
#   persistent .env, because Compose gives shell variables precedence.
# - Mutating actions compete for the same host-visible lock directory as the
#   Compose upgrader, so an Upgrade Operation and a lifecycle action cannot
#   mutate the deployment concurrently.

if [ -z "${BASH_VERSION:-}" ]; then
	exec /usr/bin/env bash "$0" "$@"
fi
set -euo pipefail

LUNAFOX_PROJECT_NAME=lunafox
LUNAFOX_COMPOSE_FILE=compose.yaml
LUNAFOX_ENV_FILE=.env
LUNAFOX_ENV_TEMPLATE=.env.example
LUNAFOX_OVERRIDE_FILE=compose.override.yaml
LUNAFOX_LOCK_DIR=.lunafox-lifecycle.lock
LUNAFOX_LOCK_METADATA=owner
LUNAFOX_LOCK_SCHEMA=1
LUNAFOX_CF_STATE_DIR=.lunafox-cf-acceleration
LUNAFOX_CF_STATE_FILE=state
LUNAFOX_CF_OVERLAY_FILE=compose.cf-acceleration.yaml
LUNAFOX_CF_INVENTORY_FILE=engine-inventory.yaml
LUNAFOX_CF_STATE_SCHEMA=1
LUNAFOX_CF_REGISTRY=docker.lunafox.cc.cd
LUNAFOX_THIRD_PARTY_POLICY_FILE=third-party-image-policy.json
LUNAFOX_ENGINE_INVENTORY_FILE=engine-inventory.yaml
LUNAFOX_MIN_COMPOSE_VERSION=2.24.0
LUNAFOX_INSTALL_TIMEOUT_SECONDS=900
LUNAFOX_DAILY_TIMEOUT_SECONDS=300
LUNAFOX_PREHEAT_MIN_TIMEOUT_SECONDS=300
LUNAFOX_PREHEAT_MAX_TIMEOUT_SECONDS=3600
LUNAFOX_POLL_SECONDS=5
# Waiting must stay visible: the heartbeat is independent of the poll cadence so
# readiness probing never has to slow down to keep the user informed.
LUNAFOX_HEARTBEAT_SECONDS=15
LUNAFOX_PROBE_TIMEOUT_SECONDS=20

# The one-shot tasks, healthy core services, and healthcheck-free resident
# services of the supported Compose graph. Unknown services are handled
# tolerantly below so a newer deployment snapshot still reaches a verdict.
LUNAFOX_ONESHOT_SERVICES="engine-preheater config-init agent-preflight migrate bootstrap cert-init"
LUNAFOX_CORE_SERVICES="postgres redis loki server frontend nginx"
LUNAFOX_AUX_SERVICES="agent upgrader alloy"

# Keys that would otherwise let a caller's shell override the persistent .env
# or redirect Compose at another project. The ready and preheat timeout values
# are deliberately not in this list: they are documented per-invocation
# overrides, never persistent deployment configuration.
LUNAFOX_OVERRIDING_KEYS="RELEASE_REGISTRY RELEASE_CHANNEL RELEASE_METADATA_BASE_URL \
RELEASE_VERSION AGENT_VERSION PUBLIC_HOST PUBLIC_PORT DATABASE_MODE COMPOSE_PROFILES \
DB_HOST DB_PORT DB_USER DB_NAME DB_SSLMODE DB_PASSWORD JWT_SECRET \
	SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF \
	ENGINE_INSTALL_REGISTRY ENGINE_INSTALL_CF_ACCELERATION ENGINE_INVENTORY_HOST_PATH LUNAFOX_SHARED_DATA_VOLUME_BIND \
	COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_ENV_FILES COMPOSE_PATH_SEPARATOR"

READY_TIMEOUT=""
UNINSTALL_PURGE=0
UNINSTALL_CONFIRM=0
LOCK_HELD=0
CF_ACCELERATION_REQUESTED=0
CF_ACCELERATION_ENABLED=0
INSTALL_PUBLIC_HOST=""
INSTALL_PUBLIC_PORT=""
INSTALL_PUBLIC_HOST_SET=0
INSTALL_PUBLIC_PORT_SET=0
LUNAFOX_CF_OVERLAY_USED=""
BASE_COMPOSE_ARGS=()
COMPOSE_ARGS=()

# ---------------------------------------------------------------- output ----

note() {
	printf 'LunaFox: %s\n' "$*"
}

progress() {
	printf 'LunaFox: %s\n' "$*"
}

warn() {
	printf 'LunaFox: WARNING %s\n' "$*" >&2
}

diagnose() {
	printf 'LunaFox:   %s\n' "$*" >&2
}

fail() {
	printf 'LunaFox: FAILED %s\n' "$*" >&2
	exit 1
}

usage_failure() {
	printf 'LunaFox: %s\n' "$*" >&2
	exit 2
}

# Purpose wording for progress and success lines. Anything unmapped keeps its
# raw name, so a new service is never hidden behind a vague phrase.
service_purpose() {
	case "$1" in
	config-init | cert-init | agent-preflight | migrate | bootstrap) printf 'first-start tasks' ;;
	postgres) printf 'the database' ;;
	redis) printf 'the cache' ;;
	loki) printf 'the log store' ;;
	alloy) printf 'the log collector' ;;
	server) printf 'the Server' ;;
	frontend) printf 'the web interface' ;;
	nginx) printf 'the HTTPS endpoint' ;;
	agent) printf 'the resident Agent' ;;
	upgrader) printf 'the upgrader' ;;
	*) printf '%s' "$1" ;;
	esac
}

format_duration() {
	local total="$1"
	if [ "$total" -ge 60 ]; then
		printf '%dm %ds' "$((total / 60))" "$((total % 60))"
	else
		printf '%ds' "$total"
	fi
}

# Prints when the message changes or the heartbeat interval elapses, so a long
# wait keeps reporting without flooding the terminal every poll.
LAST_PROGRESS_MESSAGE=""
LAST_PROGRESS_ELAPSED=0
progress_heartbeat() {
	local message="$1" elapsed="$2"
	if [ "$message" != "$LAST_PROGRESS_MESSAGE" ] || [ "$((elapsed - LAST_PROGRESS_ELAPSED))" -ge "$LUNAFOX_HEARTBEAT_SECONDS" ]; then
		progress "$message (${elapsed}s elapsed)"
		LAST_PROGRESS_MESSAGE="$message"
		LAST_PROGRESS_ELAPSED="$elapsed"
	fi
}

# ------------------------------------------------------------- utilities ----

file_mode() {
	stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required but was not found on PATH"
}

version_at_least() {
	local actual="$1" required="$2" index actual_part required_part
	local -a actual_parts required_parts
	local old_ifs="$IFS"
	IFS=.
	# Compared field by field so the check does not depend on GNU sort.
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

in_list() {
	local needle="$1" haystack="$2" item
	for item in $haystack; do
		[ "$item" = "$needle" ] && return 0
	done
	return 1
}

# ------------------------------------------------------------------ root ----

# The helper lives beside the root scripts, so its own resolved directory is the
# deployment root regardless of the caller's working directory.
resolve_root() {
	local source="${BASH_SOURCE[0]}" directory
	directory="$(cd "$(dirname "$source")" && pwd)" || fail "could not resolve the deployment root from $source"
	[ -f "$directory/$LUNAFOX_COMPOSE_FILE" ] ||
		fail "$directory does not look like a LunaFox deployment; $LUNAFOX_COMPOSE_FILE is missing"
	LUNAFOX_ROOT="$directory"
	LUNAFOX_ENV_PATH="$directory/$LUNAFOX_ENV_FILE"
	LUNAFOX_ENV_TEMPLATE_PATH="$directory/$LUNAFOX_ENV_TEMPLATE"
	LUNAFOX_OVERRIDE_PATH="$directory/$LUNAFOX_OVERRIDE_FILE"
	LUNAFOX_COMPOSE_PATH="$directory/$LUNAFOX_COMPOSE_FILE"
	LUNAFOX_LOCK_PATH="$directory/$LUNAFOX_LOCK_DIR"
	LUNAFOX_CF_STATE_PATH="$directory/$LUNAFOX_CF_STATE_DIR"
	LUNAFOX_CF_STATE_FILE_PATH="$LUNAFOX_CF_STATE_PATH/$LUNAFOX_CF_STATE_FILE"
	LUNAFOX_CF_OVERLAY_PATH="$LUNAFOX_CF_STATE_PATH/$LUNAFOX_CF_OVERLAY_FILE"
	LUNAFOX_CF_INVENTORY_PATH="$LUNAFOX_CF_STATE_PATH/$LUNAFOX_CF_INVENTORY_FILE"
	LUNAFOX_THIRD_PARTY_POLICY_PATH="$directory/$LUNAFOX_THIRD_PARTY_POLICY_FILE"
	LUNAFOX_ENGINE_INVENTORY_PATH="$directory/$LUNAFOX_ENGINE_INVENTORY_FILE"
}

# -------------------------------------------------------- host environment ----

sanitize_host_environment() {
	local key
	for key in $LUNAFOX_OVERRIDING_KEYS; do
		unset "$key" 2>/dev/null || true
	done
	export LC_ALL=C
}

# ---------------------------------------------------------------- compose ----

# Builds the base Compose argv once and reuses it for every call. The version
# override is appended only when the upgrader installed it as a regular file.
# CF state is deliberately not added here: it has to pass its own integrity
# checks against this base graph before it can affect any Compose command.
base_compose_file_args() {
	BASE_COMPOSE_ARGS=(
		--project-name "$LUNAFOX_PROJECT_NAME"
		--project-directory "$LUNAFOX_ROOT"
		--env-file "$LUNAFOX_ENV_PATH"
		-f "$LUNAFOX_COMPOSE_PATH"
	)
	if [ -n "$LUNAFOX_OVERRIDE_USED" ]; then
		BASE_COMPOSE_ARGS+=(-f "$LUNAFOX_OVERRIDE_PATH")
	fi
}

compose_file_args() {
	COMPOSE_ARGS=("${BASE_COMPOSE_ARGS[@]}")
	if [ -n "$LUNAFOX_CF_OVERLAY_USED" ]; then
		COMPOSE_ARGS+=(-f "$LUNAFOX_CF_OVERLAY_PATH")
	fi
}

lf_compose() {
	docker compose "${COMPOSE_ARGS[@]}" "$@"
}

lf_base_compose() {
	docker compose "${BASE_COMPOSE_ARGS[@]}" "$@"
}

# Docker access is capability-based on purpose: Docker Desktop, OrbStack, and
# native Linux endpoints are all accepted when the graph can actually run.
require_docker_access() {
	require_command docker
	docker info >/dev/null 2>&1 ||
		fail "the current user cannot reach the Docker daemon; grant Docker access and retry"
	[ "$(docker info --format '{{.OSType}}' 2>/dev/null || true)" = linux ] ||
		fail "the Docker daemon must run Linux containers; switch to Linux container mode and retry"
	if ! docker compose version >/dev/null 2>&1; then
		fail "the Docker Compose v2 plugin is required (docker compose); the legacy docker-compose binary is not supported"
	fi
	local compose_version
	compose_version="$(docker compose version --short 2>/dev/null | sed 's/^v//')"
	if [ -z "$compose_version" ] || ! version_at_least "$compose_version" "$LUNAFOX_MIN_COMPOSE_VERSION"; then
		fail "Docker Compose $LUNAFOX_MIN_COMPOSE_VERSION or newer is required; found ${compose_version:-unknown}"
	fi
}

# The deployment mounts the Docker socket into the agent, preflight, alloy, and
# upgrader services, so a daemon without the default socket path cannot run it.
# The deployment mounts the Docker socket into the agent, preflight, alloy, and
# upgrader services, so the socket the endpoint actually uses must be mountable.
# The path comes from Docker itself: rejecting by context name or brand would
# break Docker Desktop and OrbStack users whose capabilities are fine.
docker_endpoint_socket() {
	local endpoint
	endpoint="$(docker context inspect --format '{{(index .Endpoints "docker").Host}}' 2>/dev/null || true)"
	case "$endpoint" in
	unix://*) printf '%s' "${endpoint#unix://}" ;;
	'') printf '/var/run/docker.sock' ;;
	*) printf '' ;;
	esac
}

require_docker_socket() {
	local path
	path="$(docker_endpoint_socket)"
	if [ -n "$path" ] && [ ! -S "$path" ]; then
		fail "the Docker endpoint socket $path is missing or is not a socket; the deployment mounts the Docker socket and cannot run without it"
	fi
	if [ -n "$path" ] && [ "$path" != /var/run/docker.sock ] && [ ! -S /var/run/docker.sock ]; then
		# A capable endpoint is admitted here; say exactly what the shipped
		# compose.yaml still needs, because it mounts the default path.
		warn "this deployment mounts /var/run/docker.sock, but the Docker endpoint uses $path; add a compose.override.yaml that mounts $path or enable the daemon's default socket path"
	fi
}

# A disposable named volume proves the daemon supports the named volumes the
# graph declares without pulling an image or touching deployment state.
require_named_volume_capability() {
	local probe
	probe="lunafox-preflight-$PPID-$(date +%s)"
	docker volume create "$probe" >/dev/null 2>&1 ||
		fail "the Docker daemon cannot create named volumes; the deployment cannot persist data"
	docker volume rm "$probe" >/dev/null 2>&1 ||
		fail "the Docker daemon created $probe but could not remove it; verify daemon health before retrying"
}

require_curl() {
	command -v curl >/dev/null 2>&1 ||
		fail "curl is required to verify the public HTTPS endpoint; install curl and retry"
}

# --------------------------------------------------------------- env file ----

# .env is the only interpolation source, so it is never sourced or eval'd. The
# file is read through Compose itself (config --environment) or through a
# literal KEY=VALUE match.
env_value() {
	local key="$1" line
	[ -f "$LUNAFOX_ENV_PATH" ] || return 1
	while IFS= read -r line || [ -n "$line" ]; do
		case "$line" in
		"$key"=*)
			printf '%s' "${line#"$key"=}"
			return 0
			;;
		esac
	done <"$LUNAFOX_ENV_PATH"
	return 1
}

require_env_regular_file() {
	local path="$1" label="$2"
	if [ -L "$path" ]; then
		fail "$label must be a regular file, but it is a symbolic link"
	fi
	if [ ! -e "$path" ]; then
		return 1
	fi
	[ -f "$path" ] || fail "$label must be a regular file"
	return 0
}

# Creates .env only for a directory that has no LunaFox persistence yet. The
# hard link gives the target its full content atomically and fails when a
# concurrent writer won the race, which is why it is not a plain mv.
create_env_from_template() {
	local template="$1" temporary
	require_env_regular_file "$template" "$LUNAFOX_ENV_TEMPLATE" || fail "$LUNAFOX_ENV_TEMPLATE is missing; restore the deployment template and retry"
	temporary="$(mktemp "$LUNAFOX_ROOT/.env.template.XXXXXX")" || fail "could not create a temporary file beside .env"
	register_cleanup_path "$temporary"
	cp "$template" "$temporary" || fail "could not stage $LUNAFOX_ENV_FILE from $LUNAFOX_ENV_TEMPLATE"
	chmod 0600 "$temporary" || fail "could not restrict the staged $LUNAFOX_ENV_FILE"
	if ! ln "$temporary" "$LUNAFOX_ENV_PATH" 2>/dev/null; then
		fail "$LUNAFOX_ENV_FILE appeared while this command was starting; resolve the concurrent run before retrying"
	fi
	rm -f "$temporary"
	CLEANUP_PATHS="" # the staged file no longer exists
	note "created $LUNAFOX_ENV_FILE from $LUNAFOX_ENV_TEMPLATE"
}

validate_install_public_host() {
	[[ "$1" =~ ^[a-zA-Z0-9.:-]+$ ]] ||
		usage_failure "--public-host must be a hostname or IP address"
}

validate_install_public_port() {
	[[ "$1" =~ ^[1-9][0-9]{0,4}$ ]] ||
		usage_failure "--public-port must be a decimal port from 1 to 65535"
	if [ "$1" -gt 65535 ]; then
		usage_failure "--public-port must be a decimal port from 1 to 65535"
	fi
}

require_single_env_key() {
	local key="$1" count
	count="$(awk -v key="$key" '$0 ~ ("^" key "=") { count++ } END { print count + 0 }' "$LUNAFOX_ENV_PATH")"
	[ "$count" = 1 ] || fail "$LUNAFOX_ENV_FILE must contain exactly one $key entry before install can update it"
}

# Render the replacement record-by-record so a user-owned file without a final
# newline stays byte-identical everywhere except the requested key's value.
render_install_public_address() {
	local line line_terminated carriage_return
	while :; do
		line=""
		if IFS= read -r line; then
			line_terminated=1
		else
			line_terminated=0
			[ -n "$line" ] || break
		fi

		case "$line" in
		PUBLIC_HOST=*)
			if [ "$INSTALL_PUBLIC_HOST_SET" = 1 ]; then
				carriage_return=""
				case "$line" in *$'\r') carriage_return=$'\r' ;; esac
				printf 'PUBLIC_HOST=%s%s' "$INSTALL_PUBLIC_HOST" "$carriage_return"
			else
				printf '%s' "$line"
			fi
			;;
		PUBLIC_PORT=*)
			if [ "$INSTALL_PUBLIC_PORT_SET" = 1 ]; then
				carriage_return=""
				case "$line" in *$'\r') carriage_return=$'\r' ;; esac
				printf 'PUBLIC_PORT=%s%s' "$INSTALL_PUBLIC_PORT" "$carriage_return"
			else
				printf '%s' "$line"
			fi
			;;
		*) printf '%s' "$line" ;;
		esac
		if [ "$line_terminated" = 1 ]; then
			printf '\n'
		fi
	done <"$LUNAFOX_ENV_PATH"
}

# The root parser already checks this syntax before Docker access. Repeating it
# here keeps direct helper invocation from bypassing the same persistent-config
# boundary, while the atomic replacement preserves every unrelated user setting.
persist_install_public_address() {
	local temporary
	[ "$INSTALL_PUBLIC_HOST_SET" = 1 ] || [ "$INSTALL_PUBLIC_PORT_SET" = 1 ] || return 0
	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || fail "$LUNAFOX_ENV_FILE is missing; run ./install.sh for a first start"
	if [ "$INSTALL_PUBLIC_HOST_SET" = 1 ]; then
		require_single_env_key PUBLIC_HOST
	fi
	if [ "$INSTALL_PUBLIC_PORT_SET" = 1 ]; then
		require_single_env_key PUBLIC_PORT
	fi
	temporary="$(mktemp "$LUNAFOX_ROOT/.env.public-address.XXXXXX")" ||
		fail "could not stage the public address update"
	register_cleanup_path "$temporary"
	if ! render_install_public_address >"$temporary"; then
		fail "could not stage the public address update"
	fi
	chmod 0600 "$temporary" || fail "could not restrict the staged public address update"
	if [ -L "$LUNAFOX_ENV_PATH" ]; then
		fail "$LUNAFOX_ENV_FILE changed into a symbolic link while updating the public address"
	fi
	[ -f "$LUNAFOX_ENV_PATH" ] || fail "$LUNAFOX_ENV_FILE disappeared while updating the public address"
	mv -f "$temporary" "$LUNAFOX_ENV_PATH" || fail "could not persist the public address update"
	if [ "$INSTALL_PUBLIC_HOST_SET" = 1 ] && [ "$INSTALL_PUBLIC_PORT_SET" = 1 ]; then
		note "updated PUBLIC_HOST and PUBLIC_PORT from install arguments"
	elif [ "$INSTALL_PUBLIC_HOST_SET" = 1 ]; then
		note "updated PUBLIC_HOST from an install argument"
	else
		note "updated PUBLIC_PORT from an install argument"
	fi
}

# --------------------------------------------------------------- override ----

# A symlinked, non-regular, or unreadable override is never silently ignored:
# the deployment would start with unconfirmed images.
validate_override_file() {
	LUNAFOX_OVERRIDE_USED=""
	if [ -L "$LUNAFOX_OVERRIDE_PATH" ]; then
		fail "$LUNAFOX_OVERRIDE_FILE is a symbolic link; the upgrader installs it as a regular file, so restore or remove the link before retrying"
	fi
	if [ ! -e "$LUNAFOX_OVERRIDE_PATH" ]; then
		return 0
	fi
	[ -f "$LUNAFOX_OVERRIDE_PATH" ] || fail "$LUNAFOX_OVERRIDE_FILE must be a regular file"
	[ -r "$LUNAFOX_OVERRIDE_PATH" ] || fail "$LUNAFOX_OVERRIDE_FILE is not readable; fix its permissions before retrying"
	LUNAFOX_OVERRIDE_USED=1
}

# Compose still needs the confirmed overlay for `down`; only a completed purge
# may clear it, so a failed final removal must not be reported as a clean reset.
reset_persisted_override() {
	if [ -z "$LUNAFOX_OVERRIDE_USED" ]; then
		if [ -e "$LUNAFOX_OVERRIDE_PATH" ] || [ -L "$LUNAFOX_OVERRIDE_PATH" ]; then
			fail "the deployment containers, network, and LunaFox volumes were removed, but $LUNAFOX_OVERRIDE_FILE appeared after validation; remove it manually after confirming that no lifecycle command or upgrade is running"
		fi
		return 0
	fi
	if [ -L "$LUNAFOX_OVERRIDE_PATH" ]; then
		fail "the deployment containers, network, and LunaFox volumes were removed, but $LUNAFOX_OVERRIDE_FILE changed into a symbolic link after validation; remove it manually after confirming that no lifecycle command or upgrade is running"
	fi
	[ -e "$LUNAFOX_OVERRIDE_PATH" ] || return 0
	[ -f "$LUNAFOX_OVERRIDE_PATH" ] ||
		fail "the deployment containers, network, and LunaFox volumes were removed, but $LUNAFOX_OVERRIDE_FILE changed after validation; remove it manually after confirming that no lifecycle command or upgrade is running"
	if ! rm -f "$LUNAFOX_OVERRIDE_PATH"; then
		fail "the deployment containers, network, and LunaFox volumes were removed, but could not remove the persisted version override $LUNAFOX_OVERRIDE_FILE; remove it manually after fixing the host filesystem condition"
	fi
}

# ---------------------------------------------------- CF acceleration ----

# The policy is deliberately rendered here instead of parsed with a best-effort
# JSON matcher. The public package is release-frozen; byte equality makes a
# malformed, extra, stale, or edited policy fail before any Compose mutation
# without adding jq, Node, or Python as an operator prerequisite.
render_expected_third_party_policy() {
	cat <<'POLICY'
{
  "schemaVersion": 1,
  "provenanceClaim": "lunafox-reviewed-content",
  "entries": [
    {
      "service": "postgres",
      "profiles": [
        "embedded"
      ],
      "registry": "docker.io",
      "repository": "library/postgres",
      "digest": "sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675",
      "contentAudit": "reviewed",
      "publisherSignatureVerification": "not-approved",
      "evidence": "release-frozen Docker Official Image digest reviewed by LunaFox"
    },
    {
      "service": "redis",
      "profiles": [
        "embedded",
        "external"
      ],
      "registry": "docker.io",
      "repository": "library/redis",
      "digest": "sha256:2afba59292f25f5d1af200496db41bea2c6c816b059f57ae74703a50a03a27d0",
      "contentAudit": "reviewed",
      "publisherSignatureVerification": "not-approved",
      "evidence": "release-frozen Docker Official Image digest reviewed by LunaFox"
    },
    {
      "service": "loki",
      "profiles": [
        "embedded",
        "external"
      ],
      "registry": "docker.io",
      "repository": "grafana/loki",
      "digest": "sha256:3c8fd3570dd9219951a60d3f919c7f31923d10baee578b77bc26c4a0b32d092d",
      "contentAudit": "reviewed",
      "publisherSignatureVerification": "not-approved",
      "evidence": "release-frozen Grafana image digest reviewed by LunaFox"
    },
    {
      "service": "alloy",
      "profiles": [
        "embedded",
        "external"
      ],
      "registry": "docker.io",
      "repository": "grafana/alloy",
      "digest": "sha256:b8ec653c44235fbe910879145dac3597d66b0aaecf60bcbbe82580767771a839",
      "contentAudit": "reviewed",
      "publisherSignatureVerification": "not-approved",
      "evidence": "release-frozen Grafana image digest reviewed by LunaFox"
    }
  ]
}
POLICY
}

validate_third_party_policy() {
	local expected
	require_env_regular_file "$LUNAFOX_THIRD_PARTY_POLICY_PATH" "$LUNAFOX_THIRD_PARTY_POLICY_FILE" ||
		fail "$LUNAFOX_THIRD_PARTY_POLICY_FILE is missing; restore the release package before enabling Cloudflare acceleration"
	[ -r "$LUNAFOX_THIRD_PARTY_POLICY_PATH" ] ||
		fail "$LUNAFOX_THIRD_PARTY_POLICY_FILE is not readable; restore the release package before enabling Cloudflare acceleration"
	require_command cmp
	expected="$(mktemp "$LUNAFOX_ROOT/.lunafox-third-party-policy.XXXXXX")" ||
		fail "could not stage the third-party image policy validation"
	register_cleanup_path "$expected"
	render_expected_third_party_policy >"$expected" || fail "could not render the third-party image policy validation"
	if ! cmp -s "$expected" "$LUNAFOX_THIRD_PARTY_POLICY_PATH"; then
		fail "$LUNAFOX_THIRD_PARTY_POLICY_FILE does not match the release-frozen LunaFox-reviewed digest policy"
	fi
}

cf_internal_file_is_valid() {
	local path="$1" label="$2"
	if [ -L "$path" ]; then
		fail "$label must be a regular file, but it is a symbolic link"
	fi
	[ -f "$path" ] || fail "$label is missing or is not a regular file"
	[ -r "$path" ] || fail "$label is not readable"
	[ "$(file_mode "$path")" = 600 ] || fail "$label must have mode 0600"
}

cf_state_is_present() {
	[ -e "$LUNAFOX_CF_STATE_PATH" ] || [ -L "$LUNAFOX_CF_STATE_PATH" ]
}

require_cf_state_directory() {
	if [ -L "$LUNAFOX_CF_STATE_PATH" ]; then
		fail "$LUNAFOX_CF_STATE_DIR must be a directory, but it is a symbolic link"
	fi
	[ -d "$LUNAFOX_CF_STATE_PATH" ] || fail "$LUNAFOX_CF_STATE_DIR is not a directory"
	[ "$(file_mode "$LUNAFOX_CF_STATE_PATH")" = 700 ] ||
		fail "$LUNAFOX_CF_STATE_DIR must have mode 0700"
}

cf_state_value() {
	awk -F= -v key="$1" '$1 == key { print substr($0, length(key) + 2); exit }' "$LUNAFOX_CF_STATE_FILE_PATH"
}

validate_cf_state_shape() {
	if ! awk '
		BEGIN {
			split("schema enabled database_mode server_ref frontend_ref nginx_ref agent_ref bootstrap_ref postgres_ref redis_ref loki_ref alloy_ref", keys, " ")
			for (key_index in keys) allowed[keys[key_index]] = 1
		}
		{
			separator = index($0, "=")
			if (separator < 2) invalid = 1
			key = substr($0, 1, separator - 1)
			if (!(key in allowed) || seen[key]++) invalid = 1
		}
		END {
			for (key_index in keys) if (!seen[keys[key_index]]) invalid = 1
			exit invalid ? 1 : 0
		}
	' "$LUNAFOX_CF_STATE_FILE_PATH"; then
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE has an unrecognised or incomplete state shape"
	fi
}

render_cf_state() {
	printf 'schema=%s\n' "$LUNAFOX_CF_STATE_SCHEMA"
	printf 'enabled=true\n'
	printf 'database_mode=%s\n' "$CF_DATABASE_MODE"
	printf 'server_ref=%s\n' "$CF_SERVER_REF"
	printf 'frontend_ref=%s\n' "$CF_FRONTEND_REF"
	printf 'nginx_ref=%s\n' "$CF_NGINX_REF"
	printf 'agent_ref=%s\n' "$CF_AGENT_REF"
	printf 'bootstrap_ref=%s\n' "$CF_BOOTSTRAP_REF"
	printf 'postgres_ref=%s\n' "$CF_POSTGRES_REF"
	printf 'redis_ref=%s\n' "$CF_REDIS_REF"
	printf 'loki_ref=%s\n' "$CF_LOKI_REF"
	printf 'alloy_ref=%s\n' "$CF_ALLOY_REF"
}

load_cf_state() {
	local expected
	require_cf_state_directory
	cf_internal_file_is_valid "$LUNAFOX_CF_STATE_FILE_PATH" "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE"
	cf_internal_file_is_valid "$LUNAFOX_CF_OVERLAY_PATH" "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_OVERLAY_FILE"
	if ! preheat_capable_deployment; then
		cf_internal_file_is_valid "$LUNAFOX_CF_INVENTORY_PATH" "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_INVENTORY_FILE"
	fi
	validate_cf_state_shape

	CF_STATE_SCHEMA_VALUE="$(cf_state_value schema)"
	CF_STATE_ENABLED_VALUE="$(cf_state_value enabled)"
	CF_DATABASE_MODE="$(cf_state_value database_mode)"
	CF_SERVER_REF="$(cf_state_value server_ref)"
	CF_FRONTEND_REF="$(cf_state_value frontend_ref)"
	CF_NGINX_REF="$(cf_state_value nginx_ref)"
	CF_AGENT_REF="$(cf_state_value agent_ref)"
	CF_BOOTSTRAP_REF="$(cf_state_value bootstrap_ref)"
	CF_POSTGRES_REF="$(cf_state_value postgres_ref)"
	CF_REDIS_REF="$(cf_state_value redis_ref)"
	CF_LOKI_REF="$(cf_state_value loki_ref)"
	CF_ALLOY_REF="$(cf_state_value alloy_ref)"

	[ "$CF_STATE_SCHEMA_VALUE" = "$LUNAFOX_CF_STATE_SCHEMA" ] ||
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE has an unsupported schema"
	[ "$CF_STATE_ENABLED_VALUE" = true ] ||
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE enabled must be exactly true"
	case "$CF_DATABASE_MODE" in
	embedded | external) ;;
	*) fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE has an unsupported database mode" ;;
	esac
	for field in CF_SERVER_REF CF_FRONTEND_REF CF_NGINX_REF CF_AGENT_REF CF_BOOTSTRAP_REF CF_REDIS_REF CF_LOKI_REF CF_ALLOY_REF; do
		[ -n "${!field}" ] || fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE has an empty $field"
	done
	case "$CF_DATABASE_MODE" in
	embedded) [ -n "$CF_POSTGRES_REF" ] || fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE must select PostgreSQL for embedded mode" ;;
	external) [ -z "$CF_POSTGRES_REF" ] || fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE must not select PostgreSQL for external mode" ;;
	esac
	expected="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-state.XXXXXX")" || fail "could not stage Cloudflare state validation"
	register_cleanup_path "$expected"
	render_cf_state >"$expected" || fail "could not render Cloudflare state validation"
	if ! cmp -s "$expected" "$LUNAFOX_CF_STATE_FILE_PATH"; then
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE is not a canonical Cloudflare acceleration state"
	fi
}

set_base_service_image() {
	local service="$1" image="$2"
	[ -n "$image" ] || fail "the rendered Compose service $service has an empty image"
	case "$service" in
	config-init)
		[ -z "${BASE_CONFIG_INIT_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for config-init"
		BASE_CONFIG_INIT_IMAGE="$image"
		;;
	postgres)
		[ -z "${BASE_POSTGRES_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for postgres"
		BASE_POSTGRES_IMAGE="$image"
		;;
	redis)
		[ -z "${BASE_REDIS_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for redis"
		BASE_REDIS_IMAGE="$image"
		;;
	loki)
		[ -z "${BASE_LOKI_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for loki"
		BASE_LOKI_IMAGE="$image"
		;;
	server)
		[ -z "${BASE_SERVER_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for server"
		BASE_SERVER_IMAGE="$image"
		;;
	frontend)
		[ -z "${BASE_FRONTEND_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for frontend"
		BASE_FRONTEND_IMAGE="$image"
		;;
	bootstrap)
		[ -z "${BASE_BOOTSTRAP_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for bootstrap"
		BASE_BOOTSTRAP_IMAGE="$image"
		;;
	migrate)
		[ -z "${BASE_MIGRATE_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for migrate"
		BASE_MIGRATE_IMAGE="$image"
		;;
	agent-preflight)
		[ -z "${BASE_AGENT_PREFLIGHT_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for agent-preflight"
		BASE_AGENT_PREFLIGHT_IMAGE="$image"
		;;
	nginx)
		[ -z "${BASE_NGINX_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for nginx"
		BASE_NGINX_IMAGE="$image"
		;;
	cert-init)
		[ -z "${BASE_CERT_INIT_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for cert-init"
		BASE_CERT_INIT_IMAGE="$image"
		;;
	agent)
		[ -z "${BASE_AGENT_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for agent"
		BASE_AGENT_IMAGE="$image"
		;;
	upgrader)
		[ -z "${BASE_UPGRADER_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for upgrader"
		BASE_UPGRADER_IMAGE="$image"
		;;
	alloy)
		[ -z "${BASE_ALLOY_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for alloy"
		BASE_ALLOY_IMAGE="$image"
		;;
	engine-preheater)
		[ -z "${BASE_ENGINE_PREHEATER_IMAGE:-}" ] || fail "the rendered Compose graph has duplicate image entries for engine-preheater"
		BASE_ENGINE_PREHEATER_IMAGE="$image"
		;;
	*) fail "Cloudflare acceleration cannot map the unexpected Compose service $service" ;;
	esac
}

rendered_compose_service_images() {
	awk '
		/^services:[[:space:]]*$/ { active = 1; next }
		active && /^[^[:space:]]/ { exit }
		active && /^  [A-Za-z0-9][A-Za-z0-9_.-]*:[[:space:]]*$/ {
			service = $1
			sub(/:$/, "", service)
			next
		}
		active && /^    image:[[:space:]]+/ {
			image = $0
			sub(/^    image:[[:space:]]*/, "", image)
			gsub(/^[\047\"]|[\047\"]$/, "", image)
			if (service == "" || image == "") exit 1
			print service "\t" image
		}
	' "$1"
}

validate_first_party_base_image() {
	local service="$1" component="$2" reference="$3"
	if [[ ! "$reference" =~ ^(docker\.io|ghcr\.io)/yyhuni/lunafox-${component}@sha256:[a-f0-9]{64}$ ]]; then
		fail "the rendered Compose image for $service is not the fixed first-party $component release identity"
	fi
}

validate_exact_third_party_image() {
	local service="$1" expected="$2" actual="$3"
	[ "$actual" = "$expected" ] ||
		fail "the rendered Compose image for $service does not match the release-frozen third-party policy"
}

capture_cf_image_closure() {
	local rendered images service image required_services
	BASE_CONFIG_INIT_IMAGE=""
	BASE_POSTGRES_IMAGE=""
	BASE_REDIS_IMAGE=""
	BASE_LOKI_IMAGE=""
	BASE_SERVER_IMAGE=""
	BASE_FRONTEND_IMAGE=""
	BASE_BOOTSTRAP_IMAGE=""
	BASE_MIGRATE_IMAGE=""
	BASE_AGENT_PREFLIGHT_IMAGE=""
	BASE_NGINX_IMAGE=""
	BASE_CERT_INIT_IMAGE=""
	BASE_AGENT_IMAGE=""
	BASE_UPGRADER_IMAGE=""
	BASE_ALLOY_IMAGE=""
	BASE_ENGINE_PREHEATER_IMAGE=""
	rendered="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-rendered-compose.XXXXXX")" || fail "could not stage the rendered Compose graph"
	register_cleanup_path "$rendered"
	if ! lf_base_compose config >"$rendered" 2>&1; then
		fail "the base Compose configuration cannot be rendered for Cloudflare acceleration"
	fi
	images="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-service-images.XXXXXX")" || fail "could not inspect the rendered Compose image closure"
	register_cleanup_path "$images"
	if ! rendered_compose_service_images "$rendered" >"$images"; then
		fail "the rendered Compose image closure has an unsupported shape"
	fi
	while IFS=$'\t' read -r service image; do
		[ -n "$service" ] || continue
		set_base_service_image "$service" "$image"
	done <"$images"

	required_services="config-init redis loki server frontend bootstrap migrate agent-preflight nginx cert-init agent upgrader alloy"
	if [ "$CF_DATABASE_MODE" = embedded ]; then
		required_services="config-init postgres redis loki server frontend bootstrap migrate agent-preflight nginx cert-init agent upgrader alloy"
	fi
	[ -n "$BASE_ENGINE_PREHEATER_IMAGE" ] && required_services="$required_services engine-preheater"
	for service in $required_services; do
		case "$service" in
		config-init) image="$BASE_CONFIG_INIT_IMAGE" ;;
		postgres) image="$BASE_POSTGRES_IMAGE" ;;
		redis) image="$BASE_REDIS_IMAGE" ;;
		loki) image="$BASE_LOKI_IMAGE" ;;
		server) image="$BASE_SERVER_IMAGE" ;;
		frontend) image="$BASE_FRONTEND_IMAGE" ;;
		bootstrap) image="$BASE_BOOTSTRAP_IMAGE" ;;
		migrate) image="$BASE_MIGRATE_IMAGE" ;;
		agent-preflight) image="$BASE_AGENT_PREFLIGHT_IMAGE" ;;
		nginx) image="$BASE_NGINX_IMAGE" ;;
		cert-init) image="$BASE_CERT_INIT_IMAGE" ;;
		agent) image="$BASE_AGENT_IMAGE" ;;
		upgrader) image="$BASE_UPGRADER_IMAGE" ;;
		alloy) image="$BASE_ALLOY_IMAGE" ;;
		engine-preheater) image="$BASE_ENGINE_PREHEATER_IMAGE" ;;
		esac
		[ -n "$image" ] || fail "the rendered Compose graph is missing required service image $service"
	done

	validate_first_party_base_image config-init bootstrap "$BASE_CONFIG_INIT_IMAGE"
	validate_first_party_base_image server server "$BASE_SERVER_IMAGE"
	validate_first_party_base_image frontend frontend "$BASE_FRONTEND_IMAGE"
	validate_first_party_base_image bootstrap bootstrap "$BASE_BOOTSTRAP_IMAGE"
	validate_first_party_base_image migrate bootstrap "$BASE_MIGRATE_IMAGE"
	validate_first_party_base_image agent-preflight agent "$BASE_AGENT_PREFLIGHT_IMAGE"
	validate_first_party_base_image nginx nginx "$BASE_NGINX_IMAGE"
	validate_first_party_base_image cert-init bootstrap "$BASE_CERT_INIT_IMAGE"
	validate_first_party_base_image agent agent "$BASE_AGENT_IMAGE"
	if [ -n "$BASE_ENGINE_PREHEATER_IMAGE" ]; then
		validate_first_party_base_image engine-preheater agent "$BASE_ENGINE_PREHEATER_IMAGE"
	fi
	validate_first_party_base_image upgrader bootstrap "$BASE_UPGRADER_IMAGE"
	[ "$BASE_CONFIG_INIT_IMAGE" = "$BASE_BOOTSTRAP_IMAGE" ] || fail "the first-party config-init image does not match bootstrap"
	[ "$BASE_MIGRATE_IMAGE" = "$BASE_BOOTSTRAP_IMAGE" ] || fail "the first-party migrate image does not match bootstrap"
	[ "$BASE_CERT_INIT_IMAGE" = "$BASE_BOOTSTRAP_IMAGE" ] || fail "the first-party cert-init image does not match bootstrap"
	[ "$BASE_UPGRADER_IMAGE" = "$BASE_BOOTSTRAP_IMAGE" ] || fail "the first-party upgrader image does not match bootstrap"
	[ "$BASE_AGENT_PREFLIGHT_IMAGE" = "$BASE_AGENT_IMAGE" ] || fail "the first-party agent-preflight image does not match agent"
	if [ -n "$BASE_ENGINE_PREHEATER_IMAGE" ]; then
		[ "$BASE_ENGINE_PREHEATER_IMAGE" = "$BASE_AGENT_IMAGE" ] || fail "the first-party engine-preheater image does not match agent"
	fi

	if [ "$CF_DATABASE_MODE" = embedded ]; then
		validate_exact_third_party_image postgres "docker.io/library/postgres@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675" "$BASE_POSTGRES_IMAGE"
	fi
	validate_exact_third_party_image redis "docker.io/library/redis@sha256:2afba59292f25f5d1af200496db41bea2c6c816b059f57ae74703a50a03a27d0" "$BASE_REDIS_IMAGE"
	validate_exact_third_party_image loki "docker.io/grafana/loki@sha256:3c8fd3570dd9219951a60d3f919c7f31923d10baee578b77bc26c4a0b32d092d" "$BASE_LOKI_IMAGE"
	validate_exact_third_party_image alloy "docker.io/grafana/alloy@sha256:b8ec653c44235fbe910879145dac3597d66b0aaecf60bcbbe82580767771a839" "$BASE_ALLOY_IMAGE"
}

resolve_cf_database_mode() {
	CF_DATABASE_MODE="$(compose_environment_value DATABASE_MODE || true)"
	case "$CF_DATABASE_MODE" in
	embedded | external) ;;
	*) fail "DATABASE_MODE must be exactly embedded or external before enabling Cloudflare acceleration" ;;
	esac
	local profile
	profile="$(compose_environment_value COMPOSE_PROFILES || true)"
	[ "$profile" = "$CF_DATABASE_MODE" ] ||
		fail "COMPOSE_PROFILES must equal DATABASE_MODE before enabling Cloudflare acceleration"
}

discover_cf_image_closure() {
	validate_third_party_policy
	resolve_cf_database_mode
	capture_cf_image_closure
}

# Modern deployment snapshots carry this digest and let the Compose preheater
# own all registry traffic. Historical fixed snapshots have no such binding and
# retain their legacy host-side CF preparation path for compatibility.
preheat_capable_deployment() {
	[ -f "$LUNAFOX_ENV_PATH" ] || return 1
	grep -Eq '^LUNAFOX_PREHEAT_MANIFEST_DIGEST=sha256:[a-f0-9]{64}$' "$LUNAFOX_ENV_PATH" || return 1
	[ -f "$LUNAFOX_ROOT/preheat-manifest.json" ] || return 1
}

repository_from_digest_ref() {
	local reference="$1" remainder
	remainder="${reference#*/}"
	printf '%s' "${remainder%@*}"
}

digest_from_digest_ref() {
	printf '%s' "${1##*@}"
}

first_party_docker_reference() {
	local reference="$1" repository digest
	repository="$(repository_from_digest_ref "$reference")"
	digest="$(digest_from_digest_ref "$reference")"
	printf 'docker.io/%s@%s' "$repository" "$digest"
}

first_party_ghcr_reference() {
	local reference="$1" repository digest
	repository="$(repository_from_digest_ref "$reference")"
	digest="$(digest_from_digest_ref "$reference")"
	printf 'ghcr.io/%s@%s' "$repository" "$digest"
}

cloudflare_reference() {
	local reference="$1" repository digest
	repository="$(repository_from_digest_ref "$reference")"
	digest="$(digest_from_digest_ref "$reference")"
	printf '%s/%s@%s' "$LUNAFOX_CF_REGISTRY" "$repository" "$digest"
}

first_party_state_candidate_is_valid() {
	local base="$1" selected="$2" docker_reference ghcr_reference cloudflare
	docker_reference="$(first_party_docker_reference "$base")"
	ghcr_reference="$(first_party_ghcr_reference "$base")"
	cloudflare="$(cloudflare_reference "$base")"
	[ "$selected" = "$cloudflare" ] || [ "$selected" = "$docker_reference" ] || [ "$selected" = "$ghcr_reference" ]
}

third_party_state_candidate_is_valid() {
	local original="$1" selected="$2" cloudflare
	cloudflare="$(cloudflare_reference "$original")"
	[ "$selected" = "$cloudflare" ] || [ "$selected" = "$original" ]
}

yaml_engine_package_refs() {
	local inventory="$1"
	awk '
		function emit(value) {
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
			gsub(/^[\047\"]|[\047\"]$/, "", value)
			if (value == "") { invalid = 1; return }
			print block "\t" value
		}
		/^enginePackages:[[:space:]]*$/ { active = 1; next }
		!active { next }
		/^  - refs:[[:space:]]*\[/ {
			block++
			inside = 0
			values = $0
			sub(/^.*\[/, "", values)
			sub(/\][[:space:]]*$/, "", values)
			count = split(values, refs, ",")
			for (item_index = 1; item_index <= count; item_index++) emit(refs[item_index])
			next
		}
		/^  - refs:[[:space:]]*$/ { block++; inside = 1; next }
		/^  - / { invalid = 1; next }
		inside && /^      - / {
			value = $0
			sub(/^      - /, "", value)
			emit(value)
			next
		}
		/^[A-Za-z][A-Za-z0-9_-]*:[[:space:]]*/ { exit }
		/^[[:space:]]*$/ { next }
		{ invalid = 1 }
		END { if (!active || block == 0 || invalid) exit 1 }
	' "$inventory"
}

append_cf_engine_inventory_block() {
	local output="$1" docker_reference="$2" ghcr_reference="$3" count="$4" repository digest
	[ "$count" = 2 ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE must contain exactly Docker Hub and GHCR candidates for every Engine Package"
	if [[ ! "$docker_reference" =~ ^docker\.io/(yyhuni/lunafox-engine-runtime-[a-z0-9][a-z0-9._-]*)@(sha256:[a-f0-9]{64})$ ]]; then
		fail "$LUNAFOX_ENGINE_INVENTORY_FILE has an invalid Docker Hub Engine Package identity"
	fi
	repository="${BASH_REMATCH[1]}"
	digest="${BASH_REMATCH[2]}"
	[ "$ghcr_reference" = "ghcr.io/$repository@$digest" ] ||
		fail "$LUNAFOX_ENGINE_INVENTORY_FILE does not preserve the GHCR Engine Package identity"
	printf '  - refs:\n      - "%s/%s@%s"\n      - "%s"\n      - "%s"\n' \
		"$LUNAFOX_CF_REGISTRY" "$repository" "$digest" "$docker_reference" "$ghcr_reference" >>"$output" ||
		fail "could not render the accelerated Engine inventory"
}

render_cf_engine_inventory() {
	local output="$1" refs block reference current_block="" docker_reference="" ghcr_reference="" count=0 expected_block=1
	refs="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-engine-refs.XXXXXX")" || fail "could not stage the Engine inventory validation"
	register_cleanup_path "$refs"
	if ! yaml_engine_package_refs "$LUNAFOX_ENGINE_INVENTORY_PATH" >"$refs"; then
		fail "$LUNAFOX_ENGINE_INVENTORY_FILE has an unsupported Engine Package inventory shape"
	fi
	[ -s "$refs" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has no Engine Packages"
	printf 'enginePackages:\n' >"$output" || fail "could not render the accelerated Engine inventory"
	while IFS=$'\t' read -r block reference; do
		case "$block" in '' | *[!0-9]*) fail "$LUNAFOX_ENGINE_INVENTORY_FILE has an invalid Engine Package block" ;; esac
		[ -n "$reference" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has an empty Engine Package reference"
		if [ -z "$current_block" ]; then
			[ "$block" = "$expected_block" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has non-contiguous Engine Package blocks"
			current_block="$block"
		elif [ "$block" != "$current_block" ]; then
			append_cf_engine_inventory_block "$output" "$docker_reference" "$ghcr_reference" "$count"
			expected_block=$((expected_block + 1))
			[ "$block" = "$expected_block" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has non-contiguous Engine Package blocks"
			current_block="$block"
			docker_reference=""
			ghcr_reference=""
			count=0
		fi
		count=$((count + 1))
		case "$reference" in
		docker.io/*)
			[ -z "$docker_reference" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has duplicate Docker Hub Engine Package candidates"
			docker_reference="$reference"
			;;
		ghcr.io/*)
			[ -z "$ghcr_reference" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has duplicate GHCR Engine Package candidates"
			ghcr_reference="$reference"
			;;
		*) fail "$LUNAFOX_ENGINE_INVENTORY_FILE has an unsupported Engine Package registry" ;;
		esac
	done <"$refs"
	[ -n "$current_block" ] || fail "$LUNAFOX_ENGINE_INVENTORY_FILE has no Engine Packages"
	append_cf_engine_inventory_block "$output" "$docker_reference" "$ghcr_reference" "$count"
}

render_cf_overlay() {
	local output="$1" preheat_digest=""
	if preheat_capable_deployment; then
		preheat_digest="$(awk -F= '$1 == "LUNAFOX_PREHEAT_MANIFEST_DIGEST" {print $2}' "$LUNAFOX_ENV_PATH")"
	fi
	cat >"$output" <<EOF
services:
  config-init:
    image: ${CF_BOOTSTRAP_REF}
  redis:
    image: ${CF_REDIS_REF}
  loki:
    image: ${CF_LOKI_REF}
  server:
    image: ${CF_SERVER_REF}
    environment:
      AGENT_IMAGE_REF: ${CF_AGENT_REF}
      ENGINE_INSTALL_REGISTRY: ""
      ENGINE_INSTALL_CF_ACCELERATION: "true"
  frontend:
    image: ${CF_FRONTEND_REF}
  bootstrap:
    image: ${CF_BOOTSTRAP_REF}
    environment:
      ENGINE_INSTALL_REGISTRY: ""
      ENGINE_INSTALL_CF_ACCELERATION: "true"
  migrate:
    image: ${CF_BOOTSTRAP_REF}
  agent-preflight:
    image: ${CF_AGENT_REF}
    environment:
      AGENT_IMAGE_REF: ${CF_AGENT_REF}
  nginx:
    image: ${CF_NGINX_REF}
  cert-init:
    image: ${CF_BOOTSTRAP_REF}
  agent:
    image: ${CF_AGENT_REF}
  upgrader:
    image: ${CF_BOOTSTRAP_REF}
  alloy:
    image: ${CF_ALLOY_REF}
EOF
	if ! preheat_capable_deployment; then
		cat >>"$output" <<EOF
  bootstrap:
    volumes:
      - ./${LUNAFOX_CF_STATE_DIR}/${LUNAFOX_CF_INVENTORY_FILE}:/bootstrap/engine-inventory.yaml:ro
EOF
	fi
	if preheat_capable_deployment; then
		cat >>"$output" <<EOF
  engine-preheater:
    image: ${CF_AGENT_REF}
    command:
    - --manifest
    - /deployment/preheat-manifest.json
    - --release-manifest
    - /deployment/release.manifest.yaml
    - --runtime-composition
    - /deployment/runtime-composition.json
    - --compose
    - /deployment/compose.yaml
    - --third-party-policy
    - /deployment/third-party-image-policy.json
    - --manifest-digest
    - ${preheat_digest}
    - --profile
    - ${CF_DATABASE_MODE}
    - --cloudflare-acceleration
    - "true"
    environment:
      LUNAFOX_PREHEAT_TIMEOUT_SECONDS: \${LUNAFOX_PREHEAT_TIMEOUT_SECONDS:-900}
EOF
	fi
	if [ "$CF_DATABASE_MODE" = embedded ]; then
		{
			printf '  postgres:\n'
			printf '    image: %s\n' "$CF_POSTGRES_REF"
		} >>"$output" || fail "could not render the embedded Cloudflare overlay"
	fi
}

validate_cf_state_mapping() {
	[ "$CF_DATABASE_MODE" = "$CURRENT_CF_DATABASE_MODE" ] ||
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_STATE_FILE database mode does not match the current .env"
	first_party_state_candidate_is_valid "$BASE_SERVER_IMAGE" "$CF_SERVER_REF" || fail "Cloudflare state server_ref is not an approved same-digest candidate"
	first_party_state_candidate_is_valid "$BASE_FRONTEND_IMAGE" "$CF_FRONTEND_REF" || fail "Cloudflare state frontend_ref is not an approved same-digest candidate"
	first_party_state_candidate_is_valid "$BASE_NGINX_IMAGE" "$CF_NGINX_REF" || fail "Cloudflare state nginx_ref is not an approved same-digest candidate"
	first_party_state_candidate_is_valid "$BASE_AGENT_IMAGE" "$CF_AGENT_REF" || fail "Cloudflare state agent_ref is not an approved same-digest candidate"
	first_party_state_candidate_is_valid "$BASE_BOOTSTRAP_IMAGE" "$CF_BOOTSTRAP_REF" || fail "Cloudflare state bootstrap_ref is not an approved same-digest candidate"
	if [ "$CF_DATABASE_MODE" = embedded ]; then
		third_party_state_candidate_is_valid "$BASE_POSTGRES_IMAGE" "$CF_POSTGRES_REF" || fail "Cloudflare state postgres_ref is not an approved same-digest candidate"
	fi
	third_party_state_candidate_is_valid "$BASE_REDIS_IMAGE" "$CF_REDIS_REF" || fail "Cloudflare state redis_ref is not an approved same-digest candidate"
	third_party_state_candidate_is_valid "$BASE_LOKI_IMAGE" "$CF_LOKI_REF" || fail "Cloudflare state loki_ref is not an approved same-digest candidate"
	third_party_state_candidate_is_valid "$BASE_ALLOY_IMAGE" "$CF_ALLOY_REF" || fail "Cloudflare state alloy_ref is not an approved same-digest candidate"
}

validate_persisted_cf_acceleration() {
	local expected_overlay expected_inventory
	load_cf_state
	CF_STATE_DATABASE_MODE="$CF_DATABASE_MODE"
	CURRENT_CF_DATABASE_MODE="$(compose_environment_value DATABASE_MODE || true)"
	case "$CURRENT_CF_DATABASE_MODE" in
	embedded | external) ;;
	*) fail "DATABASE_MODE must be exactly embedded or external for a persisted Cloudflare acceleration state" ;;
	esac
	discover_cf_image_closure
	# The closure discovery intentionally reads the current .env. Reapply the
	# parsed state only after that check so the overlay is compared in state mode.
	CF_DATABASE_MODE="$CF_STATE_DATABASE_MODE"
	validate_cf_state_mapping
	expected_overlay="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-overlay.XXXXXX")" || fail "could not stage Cloudflare overlay validation"
	register_cleanup_path "$expected_overlay"
	render_cf_overlay "$expected_overlay" || fail "could not render Cloudflare overlay validation"
	if ! cmp -s "$expected_overlay" "$LUNAFOX_CF_OVERLAY_PATH"; then
		fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_OVERLAY_FILE does not match the validated Cloudflare state"
	fi
	if preheat_capable_deployment; then
		# The preheater consumes the release-bound manifest directly. A modern CF
		# overlay must not manufacture an Engine closure beside that canonical input.
		if [ -e "$LUNAFOX_CF_INVENTORY_PATH" ] || [ -L "$LUNAFOX_CF_INVENTORY_PATH" ]; then
			fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_INVENTORY_FILE is forbidden for a preheat-capable deployment"
		fi
	else
		expected_inventory="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-inventory.XXXXXX")" || fail "could not stage Cloudflare Engine inventory validation"
		register_cleanup_path "$expected_inventory"
		render_cf_engine_inventory "$expected_inventory"
		if ! cmp -s "$expected_inventory" "$LUNAFOX_CF_INVENTORY_PATH"; then
			fail "$LUNAFOX_CF_STATE_DIR/$LUNAFOX_CF_INVENTORY_FILE does not match the current Engine Package closure"
		fi
	fi
	CF_DATABASE_MODE="$CF_STATE_DATABASE_MODE"
}

activate_persisted_cf_acceleration() {
	CF_ACCELERATION_ENABLED=0
	LUNAFOX_CF_OVERLAY_USED=""
	if ! cf_state_is_present; then
		return 0
	fi
	validate_persisted_cf_acceleration
	CF_ACCELERATION_ENABLED=1
	LUNAFOX_CF_OVERLAY_USED=1
	note "using the persisted Cloudflare acceleration mapping"
}

verify_cf_first_party_identities() {
	local component reference
	require_command cosign
	for component in server frontend nginx agent bootstrap; do
		case "$component" in
		server) reference="$BASE_SERVER_IMAGE" ;;
		frontend) reference="$BASE_FRONTEND_IMAGE" ;;
		nginx) reference="$BASE_NGINX_IMAGE" ;;
		agent) reference="$BASE_AGENT_IMAGE" ;;
		bootstrap) reference="$BASE_BOOTSTRAP_IMAGE" ;;
		esac
		reference="$(first_party_ghcr_reference "$reference")"
		if ! cosign verify \
			--certificate-oidc-issuer https://token.actions.githubusercontent.com \
			--certificate-identity-regexp '^https://github\.com/yyhuni/lunafox/\.github/workflows/public-validate\.yml@refs/heads/main$' \
			"$reference" >/dev/null; then
			fail "GHCR signature verification failed for the first-party $component image; Cloudflare transport fallback is not permitted"
		fi
	done
}

is_cf_transport_failure_output() {
	local message
	message="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')"
	case "$message" in
	*"no such host"* | *"temporary failure in name resolution"* | *"network is unreachable"* | *"connection refused"* | *"connection reset"* | *"connection timed out"* | *"i/o timeout"* | *"tls handshake timeout"* | *"temporary error"* | *"too many requests"* | *"status code 429"* | *"bad gateway"* | *"service unavailable"* | *"gateway timeout"* | *"status code 500"* | *"status code 502"* | *"status code 503"* | *"status code 504"*)
		return 0
		;;
	*) return 1 ;;
	esac
}

pull_cf_candidate() {
	local reference="$1" output_file status
	output_file="$(mktemp "$LUNAFOX_ROOT/.lunafox-cf-pull.XXXXXX")" || fail "could not capture Docker pull output"
	register_cleanup_path "$output_file"
	if docker pull "$reference" >"$output_file" 2>&1; then
		# These helpers return the selected immutable reference on stdout. Keep
		# Docker's progress visible without letting it corrupt that return value.
		cat "$output_file" >&2
		CF_PULL_OUTPUT=""
		return 0
	else
		# Capture this in the branch: after `if` completes, `$?` is the `if`
		# construct's status rather than Docker's transport result.
		status=$?
	fi
	CF_PULL_OUTPUT="$(cat "$output_file")"
	cat "$output_file" >&2
	return "$status"
}

pull_accelerated_first_party_image() {
	local component="$1" base_reference="$2" cloudflare docker_reference ghcr_reference
	cloudflare="$(cloudflare_reference "$base_reference")"
	docker_reference="$(first_party_docker_reference "$base_reference")"
	ghcr_reference="$(first_party_ghcr_reference "$base_reference")"
	if pull_cf_candidate "$cloudflare"; then
		printf '%s' "$cloudflare"
		return 0
	fi
	if ! is_cf_transport_failure_output "$CF_PULL_OUTPUT"; then
		fail "Cloudflare returned a terminal policy, authentication, or integrity failure for $component; direct fallback is not permitted"
	fi
	warn "Cloudflare transport failed for $component; trying the same digest from Docker Hub"
	if pull_cf_candidate "$docker_reference"; then
		printf '%s' "$docker_reference"
		return 0
	fi
	if ! is_cf_transport_failure_output "$CF_PULL_OUTPUT"; then
		fail "Docker Hub returned a terminal failure for the verified $component image"
	fi
	warn "Docker Hub transport failed for $component; trying the verified GHCR identity"
	if pull_cf_candidate "$ghcr_reference"; then
		printf '%s' "$ghcr_reference"
		return 0
	fi
	fail "all verified first-party candidates are unavailable for $component"
}

pull_accelerated_third_party_image() {
	local service="$1" original_reference="$2" cloudflare
	cloudflare="$(cloudflare_reference "$original_reference")"
	if pull_cf_candidate "$cloudflare"; then
		printf '%s' "$cloudflare"
		return 0
	fi
	if ! is_cf_transport_failure_output "$CF_PULL_OUTPUT"; then
		fail "Cloudflare returned a terminal policy, authentication, or integrity failure for reviewed third-party image $service; direct fallback is not permitted"
	fi
	warn "Cloudflare transport failed for reviewed third-party image $service; trying its audited Docker Hub digest"
	if pull_cf_candidate "$original_reference"; then
		printf '%s' "$original_reference"
		return 0
	fi
	fail "the reviewed third-party image $service could not be pulled from its only permitted fallback"
}

prepare_new_cf_acceleration() {
	local staging
	if cf_state_is_present; then
		fail "$LUNAFOX_CF_STATE_DIR already exists; validate or remove its incomplete state manually before enabling Cloudflare acceleration"
	fi
	discover_cf_image_closure
	if preheat_capable_deployment; then
		note "using the release-bound Cloudflare candidate mapping; image pulls are owned by engine-preheater"
		CF_SERVER_REF="$(cloudflare_reference "$BASE_SERVER_IMAGE")"
		CF_FRONTEND_REF="$(cloudflare_reference "$BASE_FRONTEND_IMAGE")"
		CF_NGINX_REF="$(cloudflare_reference "$BASE_NGINX_IMAGE")"
		CF_AGENT_REF="$(cloudflare_reference "$BASE_AGENT_IMAGE")"
		CF_BOOTSTRAP_REF="$(cloudflare_reference "$BASE_BOOTSTRAP_IMAGE")"
		if [ "$CF_DATABASE_MODE" = embedded ]; then
			CF_POSTGRES_REF="$(cloudflare_reference "$BASE_POSTGRES_IMAGE")"
		else
			CF_POSTGRES_REF=""
		fi
		CF_REDIS_REF="$(cloudflare_reference "$BASE_REDIS_IMAGE")"
		CF_LOKI_REF="$(cloudflare_reference "$BASE_LOKI_IMAGE")"
		CF_ALLOY_REF="$(cloudflare_reference "$BASE_ALLOY_IMAGE")"
	else
		verify_cf_first_party_identities
		note "preparing the complete ${CF_DATABASE_MODE} Cloudflare image closure before Compose startup"
		CF_SERVER_REF="$(pull_accelerated_first_party_image server "$BASE_SERVER_IMAGE")"
		CF_FRONTEND_REF="$(pull_accelerated_first_party_image frontend "$BASE_FRONTEND_IMAGE")"
		CF_NGINX_REF="$(pull_accelerated_first_party_image nginx "$BASE_NGINX_IMAGE")"
		CF_AGENT_REF="$(pull_accelerated_first_party_image agent "$BASE_AGENT_IMAGE")"
		CF_BOOTSTRAP_REF="$(pull_accelerated_first_party_image bootstrap "$BASE_BOOTSTRAP_IMAGE")"
		if [ "$CF_DATABASE_MODE" = embedded ]; then
			CF_POSTGRES_REF="$(pull_accelerated_third_party_image postgres "$BASE_POSTGRES_IMAGE")"
		else
			CF_POSTGRES_REF=""
		fi
		CF_REDIS_REF="$(pull_accelerated_third_party_image redis "$BASE_REDIS_IMAGE")"
		CF_LOKI_REF="$(pull_accelerated_third_party_image loki "$BASE_LOKI_IMAGE")"
		CF_ALLOY_REF="$(pull_accelerated_third_party_image alloy "$BASE_ALLOY_IMAGE")"
	fi

	staging="$(mktemp -d "$LUNAFOX_ROOT/.lunafox-cf-stage.XXXXXX")" || fail "could not stage Cloudflare acceleration state"
	register_cleanup_directory "$staging"
	chmod 0700 "$staging" || fail "could not restrict staged Cloudflare acceleration state"
	if ! preheat_capable_deployment; then
		render_cf_engine_inventory "$staging/$LUNAFOX_CF_INVENTORY_FILE"
	fi
	render_cf_overlay "$staging/$LUNAFOX_CF_OVERLAY_FILE"
	render_cf_state >"$staging/$LUNAFOX_CF_STATE_FILE" || fail "could not write staged Cloudflare acceleration state"
	chmod 0600 "$staging/$LUNAFOX_CF_STATE_FILE" "$staging/$LUNAFOX_CF_OVERLAY_FILE" ||
		fail "could not restrict staged Cloudflare acceleration files"
	if ! preheat_capable_deployment; then
		chmod 0600 "$staging/$LUNAFOX_CF_INVENTORY_FILE" || fail "could not restrict staged Cloudflare Engine inventory"
	fi
	if [ -e "$LUNAFOX_CF_STATE_PATH" ] || [ -L "$LUNAFOX_CF_STATE_PATH" ]; then
		fail "$LUNAFOX_CF_STATE_DIR appeared while Cloudflare acceleration was being prepared; retry after confirming no other lifecycle command is running"
	fi
	if ! mv "$staging" "$LUNAFOX_CF_STATE_PATH"; then
		fail "could not atomically persist Cloudflare acceleration state"
	fi
	CLEANUP_DIRECTORIES=""
	CF_ACCELERATION_ENABLED=1
	LUNAFOX_CF_OVERLAY_USED=1
	note "persisted the Cloudflare acceleration mapping without changing .env"
}

reset_persisted_cf_acceleration() {
	[ "$CF_ACCELERATION_ENABLED" = 1 ] || return 0
	if ! rm -f "$LUNAFOX_CF_STATE_FILE_PATH" "$LUNAFOX_CF_OVERLAY_PATH" "$LUNAFOX_CF_INVENTORY_PATH"; then
		fail "the LunaFox volumes were removed, but could not remove the protected Cloudflare acceleration files; remove $LUNAFOX_CF_STATE_DIR manually after fixing the filesystem condition"
	fi
	if ! rmdir "$LUNAFOX_CF_STATE_PATH"; then
		fail "the LunaFox volumes were removed, but could not remove the protected Cloudflare acceleration directory; remove $LUNAFOX_CF_STATE_DIR manually after confirming no lifecycle command is running"
	fi
	CF_ACCELERATION_ENABLED=0
	LUNAFOX_CF_OVERLAY_USED=""
}

# ------------------------------------------------------------------ lock ----

# The lock is a directory, so mkdir is the atomic primitive and works identically
# on macOS and Linux without flock. It lives on the deployment bind mount rather
# than under .lunafox/upgrade, which the upgrade-state volume shadows.
lock_metadata_path() {
	printf '%s/%s' "$LUNAFOX_LOCK_PATH" "$LUNAFOX_LOCK_METADATA"
}

read_lock_field() {
	local field="$1" metadata line key value
	metadata="$(lock_metadata_path)"
	[ -f "$metadata" ] || return 1
	while IFS= read -r line || [ -n "$line" ]; do
		key="${line%%=*}"
		value="${line#*=}"
		[ "$key" = "$field" ] && {
			printf '%s' "$value"
			return 0
		}
	done <"$metadata"
	return 1
}

# Validates the on-disk shape before trusting any content. A malformed lock is
# treated as fail-closed: it is never removed or overwritten automatically.
# Returns non-zero instead of exiting so read-only callers can report it.
lock_shape_reason() {
	local metadata actual
	if [ -L "$LUNAFOX_LOCK_PATH" ]; then
		printf '%s is a symbolic link' "$LUNAFOX_LOCK_DIR"
		return 1
	fi
	[ -d "$LUNAFOX_LOCK_PATH" ] || {
		printf '%s must be a directory' "$LUNAFOX_LOCK_DIR"
		return 1
	}
	metadata="$(lock_metadata_path)"
	if [ -L "$metadata" ]; then
		printf '%s/%s is a symbolic link' "$LUNAFOX_LOCK_DIR" "$LUNAFOX_LOCK_METADATA"
		return 1
	fi
	[ -f "$metadata" ] || {
		printf '%s/%s is missing' "$LUNAFOX_LOCK_DIR" "$LUNAFOX_LOCK_METADATA"
		return 1
	}
	actual="$(file_mode "$metadata")"
	case "$actual" in
	600 | 644) ;;
	*)
		printf '%s/%s has mode %s instead of 0600 or 0644' "$LUNAFOX_LOCK_DIR" "$LUNAFOX_LOCK_METADATA" "$actual"
		return 1
		;;
	esac
	[ "$(read_lock_field schema || true)" = "$LUNAFOX_LOCK_SCHEMA" ] || {
		printf '%s has an unrecognised schema' "$LUNAFOX_LOCK_DIR"
		return 1
	}
	return 0
}

validate_lock_shape() {
	local reason
	if ! reason="$(lock_shape_reason)"; then
		fail "$reason; remove $LUNAFOX_LOCK_DIR manually after confirming that no lifecycle command or upgrade is running"
	fi
}

lock_holder_summary() {
	local owner operation command created fence
	owner="$(read_lock_field owner || true)"
	operation="$(read_lock_field operation_id || true)"
	command="$(read_lock_field command || true)"
	created="$(read_lock_field created_at || true)"
	fence="$(read_lock_field recovery_fence || true)"
	if [ "$fence" = true ]; then
		printf 'upgrade recovery fence (operation %s)' "${operation:-unknown}"
		return 0
	fi
	case "$owner" in
	upgrade) printf 'upgrade operation %s in progress' "${operation:-unknown}" ;;
	lifecycle) printf 'lifecycle command %s in progress' "${command:-unknown}" ;;
	*) printf 'an unrecognised owner' ;;
	esac
	if [ -n "$created" ]; then
		printf ' (started %s seconds ago)' "$(elapsed_since "$created")"
	fi
}

elapsed_since() {
	local then="$1" now
	now="$(date +%s)"
	case "$then" in *[!0-9]*)
		printf 'an unknown time'
		return 0
		;;
	esac
	if [ "$now" -ge "$then" ]; then
		printf '%s' "$((now - then))"
	else
		printf 'an unknown time'
	fi
}

acquire_lock() {
	local label="$1" operation="${2:-}" fence="${3:-false}" metadata temporary
	if [ -e "$LUNAFOX_LOCK_PATH" ] || [ -L "$LUNAFOX_LOCK_PATH" ]; then
		validate_lock_shape
		fail "another deployment action holds the lock: $(lock_holder_summary). Wait for it to finish, or follow the lock recovery steps in the deployment documentation"
	fi
	if ! mkdir "$LUNAFOX_LOCK_PATH" 2>/dev/null; then
		if [ -e "$LUNAFOX_LOCK_PATH" ] || [ -L "$LUNAFOX_LOCK_PATH" ]; then
			validate_lock_shape
			fail "another deployment action acquired the lock: $(lock_holder_summary)"
		fi
		fail "could not create $LUNAFOX_LOCK_DIR in the deployment root"
	fi
	chmod 0755 "$LUNAFOX_LOCK_PATH" 2>/dev/null || true
	metadata="$(lock_metadata_path)"
	temporary="$metadata.tmp.$$"
	if ! {
		printf 'schema=%s\n' "$LUNAFOX_LOCK_SCHEMA"
		printf 'owner=%s\n' "lifecycle"
		printf 'command=%s\n' "$label"
		printf 'operation_id=%s\n' "$operation"
		printf 'created_at=%s\n' "$(date +%s)"
		printf 'recovery_fence=%s\n' "$fence"
	} >"$temporary" 2>/dev/null; then
		rm -rf "$LUNAFOX_LOCK_PATH"
		fail "could not record lock metadata in $LUNAFOX_LOCK_DIR"
	fi
	chmod 0644 "$temporary" 2>/dev/null || true
	if ! mv -f "$temporary" "$metadata" 2>/dev/null; then
		rm -f "$temporary"
		rm -rf "$LUNAFOX_LOCK_PATH"
		fail "could not record lock metadata in $LUNAFOX_LOCK_DIR"
	fi
	LOCK_HELD=1
}

release_lock() {
	[ "$LOCK_HELD" = 1 ] || return 0
	LOCK_HELD=0
	rm -f "$(lock_metadata_path)" 2>/dev/null || true
	rmdir "$LUNAFOX_LOCK_PATH" 2>/dev/null || true
}

# ----------------------------------------------------------------- cleanup ---

CLEANUP_PATHS=""
CLEANUP_DIRECTORIES=""

register_cleanup_path() {
	CLEANUP_PATHS="$CLEANUP_PATHS
$1"
}

register_cleanup_directory() {
	CLEANUP_DIRECTORIES="$CLEANUP_DIRECTORIES
$1"
}

cleanup() {
	local path
	release_lock
	if [ -n "$CLEANUP_PATHS" ]; then
		while IFS= read -r path; do
			[ -n "$path" ] || continue
			rm -f "$path" 2>/dev/null || true
		done <<EOF
$CLEANUP_PATHS
EOF
	fi
	if [ -n "$CLEANUP_DIRECTORIES" ]; then
		while IFS= read -r path; do
			[ -n "$path" ] || continue
			rm -rf "$path" 2>/dev/null || true
		done <<EOF
$CLEANUP_DIRECTORIES
EOF
	fi
}

trap cleanup EXIT
# An interrupted wait is a controlled error, so the lock is released. A killed
# process leaves the lock behind on purpose: leftover locks are never reclaimed
# automatically, because a slow operation must not be mistaken for a dead one.
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

# ------------------------------------------------------- deployment state ----

owned_volumes() {
	docker volume ls --filter "label=com.docker.compose.project=$LUNAFOX_PROJECT_NAME" --format '{{.Name}}' 2>/dev/null || true
}

# A compose file may deliver a volume under a different name than its key
# (postgres_data is delivered as lunafox_postgres), so the key list alone does
# not identify Docker resources. The rendered `volumes:` section pairs each key
# with the name it is delivered as; Compose stays the single source of truth for
# both.
load_declared_volumes() {
	DECLARED_VOLUME_PAIRS="$(lf_compose config 2>/dev/null | awk -v project="$LUNAFOX_PROJECT_NAME" '
		/^volumes:$/ { inside = 1; next }
		/^[a-z]/ { inside = 0 }
		inside && /^  [A-Za-z0-9._-]+:$/ {
			if (key != "") print project "_" key, key
			key = $1
			sub(/:$/, "", key)
			next
		}
		inside && key != "" && /^    name: / { print $2, key; key = "" }
	')"
	DECLARED_VOLUME_NAMES="$(printf '%s\n' "$DECLARED_VOLUME_PAIRS" | awk 'NF { print $1 }')"
	DECLARED_VOLUME_KEYS="$(printf '%s\n' "$DECLARED_VOLUME_PAIRS" | awk 'NF { print $2 }')"
}

volume_has_ownership_label() {
	local name="$1"
	docker volume inspect --format '{{index .Labels "com.docker.compose.project"}}' "$name" 2>/dev/null | grep -Fxq "$LUNAFOX_PROJECT_NAME"
}

# The compose key a delivered volume belongs to, which is how a volume is tied
# back to the current compose file without trusting its name.
volume_compose_key() {
	local name="$1"
	docker volume inspect --format '{{index .Labels "com.docker.compose.volume"}}' "$name" 2>/dev/null || true
}

key_is_declared() {
	local needle="$1" key
	for key in $DECLARED_VOLUME_KEYS; do
		[ "$key" = "$needle" ] && return 0
	done
	return 1
}

name_is_collision() {
	local name="$1" declared
	for declared in $DECLARED_VOLUME_NAMES; do
		[ "$declared" = "$name" ] && return 0
	done
	return 1
}

project_containers_running() {
	local services
	services="$(lf_compose ps --services --filter status=running 2>/dev/null || true)"
	[ -n "$services" ]
}

# Classifies the deployment before any mutation so the scripts never repair a
# name collision, never rebuild .env over existing data, and never treat a
# repairable deployment as a conflict.
classify_deployment() {
	LUNAFOX_DEPLOYMENT_STATE=fresh
	local name key present=0 collision="" persisted_embedded=0

	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || LUNAFOX_ENV_PRESENT=0
	[ -f "$LUNAFOX_ENV_PATH" ] && LUNAFOX_ENV_PRESENT=1

	load_declared_volumes
	for name in $DECLARED_VOLUME_NAMES; do
		# A volume that exists under a name this deployment would use, but is not
		# owned by this Compose project, is a collision: adopting or removing it
		# could destroy another deployment's data.
		if docker volume inspect "$name" >/dev/null 2>&1 && ! volume_has_ownership_label "$name"; then
			collision="$collision $name"
		fi
	done
	if [ -n "$collision" ]; then
		LUNAFOX_DEPLOYMENT_STATE=collision
		LUNAFOX_DEPLOYMENT_DETAIL="these volumes already exist without LunaFox ownership labels:$collision"
		return 0
	fi

	for name in $(owned_volumes); do
		key="$(volume_compose_key "$name")"
		key_is_declared "$key" || continue
		present=1
		# postgres_data is the compose key of the embedded database volume.
		[ "$key" = postgres_data ] && persisted_embedded=1
	done

	if [ "$present" = 0 ]; then
		LUNAFOX_DEPLOYMENT_STATE=fresh
		return 0
	fi

	if [ "${LUNAFOX_ENV_PRESENT:-0}" = 0 ]; then
		LUNAFOX_DEPLOYMENT_STATE=missing_env
		LUNAFOX_DEPLOYMENT_DETAIL="this deployment already has LunaFox volumes, but $LUNAFOX_ENV_FILE is missing"
		return 0
	fi

	local mode
	mode="$(compose_environment_value DATABASE_MODE || true)"
	case "$mode" in
	embedded)
		if [ "$persisted_embedded" = 0 ]; then
			LUNAFOX_DEPLOYMENT_STATE=mode_conflict
			LUNAFOX_DEPLOYMENT_DETAIL="DATABASE_MODE=embedded, but the persisted deployment has no embedded database volume"
			return 0
		fi
		;;
	external)
		if [ "$persisted_embedded" = 1 ]; then
			LUNAFOX_DEPLOYMENT_STATE=mode_conflict
			LUNAFOX_DEPLOYMENT_DETAIL="DATABASE_MODE=external, but the persisted deployment owns an embedded database volume"
			return 0
		fi
		;;
	esac

	# Volumes without any container are the documented default-uninstall result,
	# so they stay repairable rather than conflicting.
	if project_containers_running; then
		LUNAFOX_DEPLOYMENT_STATE=existing
	else
		LUNAFOX_DEPLOYMENT_STATE=repairable
	fi
	return 0
}

require_mutable_deployment() {
	classify_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	collision | mode_conflict)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; resolve the conflict manually before retrying"
		;;
	missing_env)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; restore the original .env, because the template is only used for a first start"
		;;
	esac
}

# --------------------------------------------------------- compose values ----

# Compose owns .env parsing, so reading a published key through it never
# executes the file as shell code.
compose_environment_value() {
	local key="$1" line
	[ -f "$LUNAFOX_ENV_PATH" ] || return 1
	line="$(lf_compose config --environment 2>/dev/null | grep -E "^${key}=" | head -n 1 || true)"
	[ -n "$line" ] || return 1
	printf '%s' "${line#*=}"
}

require_renderable_configuration() {
	if ! lf_compose config --quiet 2>&1 | sed 's/^/LunaFox:   /' >&2; then
		fail "the Compose configuration is not valid; fix $LUNAFOX_ENV_FILE before retrying"
	fi
}

# ------------------------------------------------------------ env mode ------

env_mode_is_private() {
	local actual
	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || return 1
	actual="$(file_mode "$LUNAFOX_ENV_PATH")"
	[ "$actual" = 600 ]
}

enforce_private_env_mode() {
	local actual
	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || fail "$LUNAFOX_ENV_FILE is missing; run ./install.sh for a first start"
	chmod 0600 "$LUNAFOX_ENV_PATH" 2>/dev/null || true
	actual="$(file_mode "$LUNAFOX_ENV_PATH")"
	[ "$actual" = 600 ] ||
		fail ".env must have mode 0600 before the deployment starts; fix it with: chmod 600 $LUNAFOX_ENV_FILE"
}

# ----------------------------------------------------------- ready probe ----

SVC_ONESHOT_STATE=""
SVC_CORE_STATE=""
SVC_AUX_STATE=""
PROBE_FAILURE=""
# The service behind PROBE_FAILURE, when the probe identified one. The failure
# footer uses it to point at that service's logs instead of every service's.
PROBE_FAILURE_SERVICE=""
PROBE_WAITING=""
PROBE_CORE_PENDING=""
PROBE_AUX_PENDING=""
PROBE_ONESHOT_PENDING=""
PROBE_UNKNOWN_PENDING=""
PREHEAT_CONTAINER_STATE=""
PREHEAT_EXIT_CODE=""

# The first explicit failure wins: a later row must not overwrite the verdict
# the user will read. The service name is kept beside the message so the failure
# footer can point at that service's logs.
record_probe_failure() {
	local service="$1" message="$2"
	if [ -z "$PROBE_FAILURE" ]; then
		PROBE_FAILURE="$message"
		PROBE_FAILURE_SERVICE="$service"
	fi
}

# Translates one docker compose ps row into the readiness dimensions. Explicit
# failures are separated from "still starting" so the caller can return
# immediately on an unrecoverable state instead of burning the whole window.
classify_service_row() {
	local service="$1" state="$2" health="$3" exit_code="$4"
	case "$state" in
	'') state=created ;;
	esac
	if [ "$service" = engine-preheater ]; then
		PREHEAT_CONTAINER_STATE="$state"
		PREHEAT_EXIT_CODE="$exit_code"
	fi
	if in_list "$service" "$LUNAFOX_ONESHOT_SERVICES"; then
		case "$state" in
		exited)
			if [ "$exit_code" != 0 ]; then
				record_probe_failure "$service" "the $service task exited with code ${exit_code:-unknown}"
			fi
			return 0
			;;
		dead)
			record_probe_failure "$service" "the $service task failed"
			return 0
			;;
		esac
		PROBE_ONESHOT_PENDING="$service"
		return 0
	fi
	if in_list "$service" "$LUNAFOX_CORE_SERVICES"; then
		if [ "$state" = exited ] || [ "$state" = dead ]; then
			record_probe_failure "$service" "the $service service stopped"
			return 0
		fi
		case "$health" in
		healthy) return 0 ;;
		unhealthy)
			record_probe_failure "$service" "the $service service is unhealthy"
			return 0
			;;
		esac
		PROBE_CORE_PENDING="$service"
		return 0
	fi
	if in_list "$service" "$LUNAFOX_AUX_SERVICES"; then
		case "$state" in
		running) return 0 ;;
		exited | dead)
			record_probe_failure "$service" "the $service service exited"
			return 0
			;;
		esac
		PROBE_AUX_PENDING="$service"
		return 0
	fi
	# Unknown services are treated tolerantly so a newer deployment snapshot still
	# reaches a verdict: a clean exit or a running healthy container is ready.
	case "$state" in
	exited)
		[ "$exit_code" = 0 ] || record_probe_failure "$service" "the $service service exited with code ${exit_code:-unknown}"
		;;
	running)
		case "$health" in
		unhealthy) record_probe_failure "$service" "the $service service is unhealthy" ;;
		starting | '') PROBE_UNKNOWN_PENDING="$service" ;;
		esac
		;;
	dead) record_probe_failure "$service" "the $service service failed" ;;
	*) PROBE_UNKNOWN_PENDING="$service" ;;
	esac
	return 0
}

# Collects one read-only snapshot of the deployment used by the ready wait and
# by status.sh, so both report the same conclusion.
collect_probe() {
	local table line service state health exit_code
	SVC_ONESHOT_STATE=""
	SVC_CORE_STATE=""
	SVC_AUX_STATE=""
	PROBE_FAILURE=""
	PROBE_FAILURE_SERVICE=""
	PROBE_WAITING=""
	PROBE_CORE_PENDING=""
	PROBE_AUX_PENDING=""
	PROBE_ONESHOT_PENDING=""
	PROBE_UNKNOWN_PENDING=""
	PREHEAT_CONTAINER_STATE="not created"
	PREHEAT_EXIT_CODE=""

	table="$(lf_compose ps -a --format '{{.Service}}|{{.State}}|{{.Health}}|{{.ExitCode}}' 2>/dev/null || true)"
	if [ -z "$table" ]; then
		# No row at all means nothing was created here, which the callers report
		# as not-installed or stopped rather than as a failed start.
		SVC_ONESHOT_STATE="not created"
		SVC_CORE_STATE="not created"
		SVC_AUX_STATE="not created"
		PROBE_WAITING="no Compose services exist yet"
		return 0
	fi
	while IFS= read -r line; do
		[ -n "$line" ] || continue
		service="$(printf '%s' "$line" | cut -d'|' -f1)"
		state="$(printf '%s' "$line" | cut -d'|' -f2)"
		health="$(printf '%s' "$line" | cut -d'|' -f3)"
		exit_code="$(printf '%s' "$line" | cut -d'|' -f4)"
		classify_service_row "$service" "$state" "$health" "$exit_code"
	done <<EOF
$table
EOF

	if [ -n "$PROBE_FAILURE" ]; then
		# An explicit failure is reported on its own; a pending note would only
		# hide which check failed.
		PROBE_WAITING=""
	fi
	if [ -n "$PROBE_ONESHOT_PENDING" ]; then
		SVC_ONESHOT_STATE="waiting for $PROBE_ONESHOT_PENDING"
	else
		SVC_ONESHOT_STATE=complete
	fi
	if [ -n "$PROBE_CORE_PENDING" ]; then
		SVC_CORE_STATE="waiting for $PROBE_CORE_PENDING"
	else
		SVC_CORE_STATE=healthy
	fi
	if [ -n "$PROBE_AUX_PENDING" ]; then
		SVC_AUX_STATE="waiting for $PROBE_AUX_PENDING"
	elif [ -n "$PROBE_UNKNOWN_PENDING" ]; then
		SVC_AUX_STATE="waiting for $PROBE_UNKNOWN_PENDING"
	else
		SVC_AUX_STATE=running
	fi

	if [ -z "$PROBE_FAILURE" ] && [ -z "$PROBE_WAITING" ]; then
		if [ -n "$PROBE_ONESHOT_PENDING" ]; then
			PROBE_WAITING="waiting for first-start tasks to finish ($PROBE_ONESHOT_PENDING)"
		elif [ -n "$PROBE_CORE_PENDING" ]; then
			PROBE_WAITING="waiting for $(service_purpose "$PROBE_CORE_PENDING") to become healthy"
		elif [ -n "$PROBE_AUX_PENDING" ]; then
			PROBE_WAITING="waiting for $(service_purpose "$PROBE_AUX_PENDING") to start"
		elif [ -n "$PROBE_UNKNOWN_PENDING" ]; then
			PROBE_WAITING="waiting for $(service_purpose "$PROBE_UNKNOWN_PENDING") to become healthy"
		fi
	fi
	return 0
}

# A restart count signature identifies containers that keep cycling even while
# they briefly report themselves as running.
container_restart_signature() {
	local ids
	ids="$(lf_compose ps -q 2>/dev/null || true)"
	if [ -z "$ids" ]; then
		printf 'none'
		return 0
	fi
	# shellcheck disable=SC2086
	docker inspect --format '{{.Name}}={{.RestartCount}}' $ids 2>/dev/null | sort | tr '\n' ' '
}

probe_agent_ready() {
	local output
	if output="$(lf_compose exec -T server /usr/local/bin/server agent-ready 2>&1)"; then
		printf '%s' "$output"
		return 0
	fi
	printf '%s' "${output:-the Server did not report a resident Agent}"
	return 1
}

public_address() {
	local host port
	host="$(compose_environment_value PUBLIC_HOST || true)"
	port="$(compose_environment_value PUBLIC_PORT || true)"
	[ -n "$host" ] || host=localhost
	[ -n "$port" ] || port=443
	printf 'https://%s:%s' "$host" "$port"
}

# The public entry point is the only check that proves the certificate, the
# reverse proxy, and the login page all work for a real user.
probe_public_https() {
	local address code
	address="$(public_address)"
	code="$(curl -kfsS --max-time "$LUNAFOX_PROBE_TIMEOUT_SECONDS" -o /dev/null -w '%{http_code}' "$address/healthChecks/current" 2>/dev/null || true)"
	case "$code" in
	2*) ;;
	*)
		printf 'the HTTPS health endpoint is not reachable at %s/healthChecks/current' "$address"
		return 1
		;;
	esac
	code="$(curl -kfLsS --max-time "$LUNAFOX_PROBE_TIMEOUT_SECONDS" -o /dev/null -w '%{http_code}' "$address/" 2>/dev/null || true)"
	case "$code" in
	2*)
		printf 'reachable at %s' "$address"
		return 0
		;;
	*)
		printf 'the login page is not reachable at %s' "$address"
		return 1
		;;
	esac
}

require_ready_timeout() {
	local raw="${LUNAFOX_READY_TIMEOUT_SECONDS:-}" default_seconds="$1"
	if [ -z "$raw" ]; then
		READY_TIMEOUT="$default_seconds"
		return 0
	fi
	case "$raw" in
	'' | *[!0-9]*)
		usage_failure "LUNAFOX_READY_TIMEOUT_SECONDS must be a positive integer number of seconds"
		;;
	esac
	if [ "$raw" -le 0 ]; then
		usage_failure "LUNAFOX_READY_TIMEOUT_SECONDS must be a positive integer number of seconds"
	fi
	READY_TIMEOUT="$raw"
}

# The preheat deadline belongs to the Compose one-shot rather than lifecycle
# readiness. Validate the shell override before Docker access and then validate
# the effective Compose environment before `up`, so .env and direct shell
# callers share the same bounded contract.
require_preheat_timeout() {
	local raw="${LUNAFOX_PREHEAT_TIMEOUT_SECONDS:-}"
	if [ -z "$raw" ] && [ "${#COMPOSE_ARGS[@]}" -gt 0 ]; then
		raw="$(compose_environment_value LUNAFOX_PREHEAT_TIMEOUT_SECONDS || true)"
	fi
	[ -n "$raw" ] || return 0
	case "$raw" in
	'' | *[!0-9]*)
		usage_failure "LUNAFOX_PREHEAT_TIMEOUT_SECONDS must be a decimal integer from $LUNAFOX_PREHEAT_MIN_TIMEOUT_SECONDS to $LUNAFOX_PREHEAT_MAX_TIMEOUT_SECONDS"
		;;
	esac
	if [ "$raw" -lt "$LUNAFOX_PREHEAT_MIN_TIMEOUT_SECONDS" ] || [ "$raw" -gt "$LUNAFOX_PREHEAT_MAX_TIMEOUT_SECONDS" ]; then
		usage_failure "LUNAFOX_PREHEAT_TIMEOUT_SECONDS must be a decimal integer from $LUNAFOX_PREHEAT_MIN_TIMEOUT_SECONDS to $LUNAFOX_PREHEAT_MAX_TIMEOUT_SECONDS"
	fi
}

# A repairable deployment keeps the named volumes and the installed engines a
# failed first-start task could not replace, so only that combination gets one
# optional, last, data-loss-aware way out. A fresh directory and a failure that
# named no service stay free of destructive commands.
purge_hint_applies() {
	local service="$1"
	[ -n "$service" ] || return 1
	# Preheater failures concern only verified Docker cache. They never justify
	# offering the data-destructive recovery path used by data-writing one-shots.
	[ "$service" != engine-preheater ] || return 1
	[ "${LUNAFOX_DEPLOYMENT_STATE:-}" = repairable ] || return 1
	in_list "$service" "$LUNAFOX_ONESHOT_SERVICES"
}

# The failure block is the last thing a user reads, so it is ordered as the
# verdict, the stage it failed at, the preserved scene, and the next commands.
# Nothing is stopped or rolled back, and the footer never claims the deployment
# was not changed: compose up may already have created and started containers.
report_failure() {
	local conclusion="$1" stage="$2" service="${3:-}" preheat_still_running="${4:-0}"
	printf 'LunaFox: FAILED %s\n' "$conclusion" >&2
	printf 'LunaFox:   stage: %s\n' "$stage" >&2
	printf 'LunaFox:   deployment: %s (left in place; nothing was rolled back)\n' "$LUNAFOX_ROOT" >&2
	if [ "$preheat_still_running" = 1 ]; then
		printf 'LunaFox:   engine image preheat: still running; the readiness timeout did not cancel it\n' >&2
	fi
	if [ -n "$service" ]; then
		printf 'LunaFox:   inspect logs: ./logs.sh %s\n' "$service" >&2
	else
		printf 'LunaFox:   inspect logs: ./logs.sh\n' >&2
	fi
	printf 'LunaFox:   recheck with: ./status.sh\n' >&2
	if [ "$service" = engine-preheater ]; then
		printf 'LunaFox:   rerun only image preheating with: docker compose up -d --force-recreate engine-preheater\n' >&2
	fi
	if purge_hint_applies "$service"; then
		printf 'LunaFox:   if this preserved data can be discarded, the named volumes can be deleted with: ./uninstall.sh --purge --confirm\n' >&2
	fi
	exit 1
}

# Blocks until every readiness condition passes, an unrecoverable condition
# appears, or the window expires. Nothing is stopped or rolled back on failure.
wait_for_ready() {
	local started="$SECONDS" last_signature="" signature agent_note https_note
	local elapsed preheat_still_running
	READY_ELAPSED=0
	LAST_PROGRESS_MESSAGE=""
	LAST_PROGRESS_ELAPSED=0
	while :; do
		elapsed=$((SECONDS - started))
		collect_probe
		if [ -n "$PROBE_FAILURE" ]; then
			report_failure "$PROBE_FAILURE" "waiting for the deployment to become ready" "$PROBE_FAILURE_SERVICE"
		fi
		if [ -z "$PROBE_WAITING" ]; then
			if agent_note="$(probe_agent_ready)"; then
				if https_note="$(probe_public_https)"; then
					# A container that restarts right after reporting healthy must
					# not be reported as success, so the whole picture has to hold
					# twice with an unchanged restart count.
					signature="$(container_restart_signature)"
					if [ -n "$last_signature" ] && [ "$signature" = "$last_signature" ]; then
						READY_ELAPSED="$elapsed"
						return 0
					fi
					last_signature="$signature"
					progress_heartbeat "final check: every service stays up" "$elapsed"
				else
					progress_heartbeat "$https_note" "$elapsed"
				fi
			else
				progress_heartbeat "waiting for the resident Agent: $agent_note" "$elapsed"
			fi
		fi
		if [ "$elapsed" -ge "$READY_TIMEOUT" ]; then
			# A timeout names no single service, so it keeps the plain logs hint.
			preheat_still_running=0
			case "$PREHEAT_CONTAINER_STATE" in
			created | restarting | running) preheat_still_running=1 ;;
			esac
			report_failure "the deployment did not become ready within $READY_TIMEOUT seconds" \
				"${PROBE_WAITING:-all readiness conditions}" "" "$preheat_still_running"
		fi
		if [ -n "$PROBE_WAITING" ]; then
			progress_heartbeat "$PROBE_WAITING" "$elapsed"
		fi
		sleep "$LUNAFOX_POLL_SECONDS"
	done
}

# status.sh may read Docker logs but must never replay their arbitrary text.
# The preheater writes one stable JSON summary whose entry identities and result
# enums are already constrained by the release-bound manifest and the command.
is_sha256_digest() {
	printf '%s\n' "$1" | grep -Eq '^sha256:[a-f0-9]{64}$'
}

preheat_summary_line() {
	lf_compose logs --no-log-prefix --tail 200 engine-preheater 2>/dev/null |
		grep -F '"code":"preheat.summary"' |
		tail -n 1 || true
}

print_preheat_summary_entries() {
	local summary="$1" entries row entry result code attempts cache_hit
	entries="$(printf '%s' "$summary" | sed -n 's/^.*"entries":\[\(.*\)\],"manifestDigest".*$/\1/p')"
	[ -n "$entries" ] || return 1
	printf '%s\n' "$entries" | sed 's/},{/}\n{/g' | while IFS= read -r row; do
		entry="$(printf '%s' "$row" | sed -n 's/^.*"entry":"\([^"]*\)".*$/\1/p')"
		result="$(printf '%s' "$row" | sed -n 's/^.*"result":"\([^"]*\)".*$/\1/p')"
		code="$(printf '%s' "$row" | sed -n 's/^.*"code":"\([^"]*\)".*$/\1/p')"
		attempts="$(printf '%s' "$row" | sed -n 's/^.*"attempts":\([0-9][0-9]*\).*$/\1/p')"
		cache_hit="$(printf '%s' "$row" | sed -n 's/^.*"cacheHit":\([^,}]*\).*$/\1/p')"
		printf '%s\n' "$entry" | grep -Eq '^[a-z0-9][a-z0-9._/-]*@sha256:[a-f0-9]{64}$' || continue
		case "$result" in success | configuration | daemon | integrity | unavailable | deadline | cancelled) ;; *) continue ;; esac
		case "$code" in preheat.*) ;; *) continue ;; esac
		case "$attempts" in '' | *[!0-9]*) continue ;; esac
		case "$cache_hit" in true | false) ;; *) continue ;; esac
		printf 'LunaFox:     entry: %s (%s; attempts: %s; cache: %s)\n' "$entry" "$result" "$attempts" "$cache_hit"
	done
}

print_preheat_status() {
	local manifest_digest summary
	manifest_digest="$(compose_environment_value LUNAFOX_PREHEAT_MANIFEST_DIGEST || true)"
	[ -n "$manifest_digest" ] || return 0
	if ! is_sha256_digest "$manifest_digest"; then
		printf 'LunaFox:   engine image preheat: unavailable (the release manifest digest is invalid)\n'
		return 0
	fi
	case "$PREHEAT_CONTAINER_STATE" in
	exited)
		if [ "$PREHEAT_EXIT_CODE" = 0 ]; then
			printf 'LunaFox:   engine image preheat: complete (exit code 0)\n'
		else
			printf 'LunaFox:   engine image preheat: failed (exit code %s)\n' "${PREHEAT_EXIT_CODE:-unknown}"
		fi
		;;
	created | restarting | running)
		printf 'LunaFox:   engine image preheat: %s\n' "$PREHEAT_CONTAINER_STATE"
		;;
	*) printf 'LunaFox:   engine image preheat: not created\n' ;;
	esac
	printf 'LunaFox:     manifest digest: %s\n' "$manifest_digest"
	if [ "$PREHEAT_CONTAINER_STATE" != "not created" ]; then
		summary="$(preheat_summary_line)"
		if [ -n "$summary" ] && print_preheat_summary_entries "$summary"; then
			:
		else
			printf 'LunaFox:     entry results: no final summary has been recorded yet\n'
		fi
	fi
	printf 'LunaFox:     diagnostics: ./logs.sh engine-preheater\n'
	if [ "$PREHEAT_CONTAINER_STATE" = exited ] && [ "$PREHEAT_EXIT_CODE" != 0 ]; then
		printf 'LunaFox:     rerun: docker compose up -d --force-recreate engine-preheater\n'
	fi
}

# ------------------------------------------------------------- readiness ----

# The checklist reports the dimensions the successful wait actually verified, so
# the success block is checkable instead of a bare claim. It reuses the same probe
# state status.sh prints, which keeps both verdicts identical by construction.
print_success_summary() {
	local address mode database_note
	address="$(public_address)"
	mode="$(compose_environment_value DATABASE_MODE || true)"
	case "$mode" in
	external) database_note='database ready (external PostgreSQL)' ;;
	*) database_note='database ready (embedded PostgreSQL)' ;;
	esac
	printf 'LunaFox: readiness verified in %s\n' "$(format_duration "$READY_ELAPSED")"
	printf 'LunaFox:   [ok] first-start tasks completed\n'
	printf 'LunaFox:   [ok] %s\n' "$database_note"
	printf 'LunaFox:   [ok] core services healthy\n'
	printf 'LunaFox:   [ok] resident Agent claim-ready\n'
	printf 'LunaFox:   [ok] public HTTPS reachable\n'
	printf 'LunaFox: SUCCESS the LunaFox deployment is ready.\n'
	printf 'LunaFox:   address: %s\n' "$address"
	printf 'LunaFox:   sign in: admin / admin (change this password after the first sign-in)\n'
	printf 'LunaFox:   status:  ./status.sh\n'
	printf 'LunaFox:   logs:    ./logs.sh\n'
}

# --------------------------------------------------------------- actions ----

# The Compose output stays visible on purpose: the first start may pull images
# for minutes, and progress during that window can only come from Compose. The
# phase lines around it attribute that output to the start step so it is never
# mistaken for the readiness verdict, and a failure still shows Compose's own
# error verbatim.
compose_up_once() {
	local description="$1" conclusion
	shift
	progress "$description"
	if ! lf_compose up -d "$@"; then
		conclusion="docker compose up -d${1:+ $*} failed"
		# Compose may have created or started part of the graph before it failed,
		# so the probe — not the exit status — decides whether a service can be
		# named. A failure that identified no service keeps the generic wording.
		collect_probe
		if [ -n "$PROBE_FAILURE" ]; then
			conclusion="$PROBE_FAILURE"
		fi
		report_failure "$conclusion" "$description" "$PROBE_FAILURE_SERVICE"
	fi
	progress "containers are up; waiting until the deployment is actually ready"
	return 0
}

check_public_port() {
	local host port
	host="$(compose_environment_value PUBLIC_HOST || true)"
	port="$(compose_environment_value PUBLIC_PORT || true)"
	[ -n "$port" ] || return 0
	if project_containers_running; then
		return 0
	fi
	# The probe runs in a subshell so the connection closes with it; a failed
	# redirection on `exec` in the parent shell would end the script silently.
	if (exec 3<>"/dev/tcp/${host:-localhost}/$port") 2>/dev/null; then
		fail "port $port is already in use on ${host:-localhost}; set PUBLIC_PORT in $LUNAFOX_ENV_FILE to a free port and retry"
	fi
	return 0
}

action_install() {
	require_ready_timeout "$LUNAFOX_INSTALL_TIMEOUT_SECONDS"
	require_preheat_timeout
	require_docker_access
	require_docker_socket
	require_curl
	require_named_volume_capability
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	acquire_lock install

	classify_deployment
	# This sentence is printed as the start step and reused verbatim as the
	# failure stage, so it must match the classified state: only a directory with
	# no LunaFox data yet is a first start, and a reinstall is not an upgrade.
	local compose_description
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	missing_env | mode_conflict | collision)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; resolve it manually before retrying, because a first start is the only case that may create $LUNAFOX_ENV_FILE"
		;;
	fresh)
		compose_description="starting LunaFox with docker compose up -d (a first start pulls images and initializes the database, so it can take a few minutes)"
		;;
	repairable)
		compose_description="recreating the deployment containers with docker compose up -d (the named volumes and the installed engines are kept, and this is not an upgrade)"
		;;
	existing)
		compose_description="starting the existing deployment with docker compose up -d (its configuration and volumes are preserved)"
		;;
	esac
	if [ ! -e "$LUNAFOX_ENV_PATH" ]; then
		create_env_from_template "$LUNAFOX_ENV_TEMPLATE_PATH"
	fi
	enforce_private_env_mode
	persist_install_public_address
	if cf_state_is_present; then
		activate_persisted_cf_acceleration
	elif [ "$CF_ACCELERATION_REQUESTED" = 1 ]; then
		prepare_new_cf_acceleration
	fi
	compose_file_args
	require_preheat_timeout
	require_renderable_configuration
	check_public_port
	compose_up_once "$compose_description" || return 1
	wait_for_ready || return 1
	print_success_summary
}

action_start() {
	require_ready_timeout "$LUNAFOX_DAILY_TIMEOUT_SECONDS"
	require_preheat_timeout
	require_docker_access
	require_docker_socket
	require_curl
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	acquire_lock start
	require_mutable_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment yet; run ./install.sh for the first start"
		;;
	esac
	activate_persisted_cf_acceleration
	compose_file_args
	enforce_private_env_mode
	require_preheat_timeout
	require_renderable_configuration
	check_public_port
	compose_up_once "starting the existing deployment with docker compose up -d" || return 1
	wait_for_ready || return 1
	print_success_summary
}

action_restart() {
	require_ready_timeout "$LUNAFOX_DAILY_TIMEOUT_SECONDS"
	require_preheat_timeout
	require_docker_access
	require_docker_socket
	require_curl
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	acquire_lock restart
	require_mutable_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment yet; run ./install.sh for the first start"
		;;
	esac
	activate_persisted_cf_acceleration
	compose_file_args
	enforce_private_env_mode
	require_preheat_timeout
	require_renderable_configuration
	check_public_port
	compose_up_once "recreating the deployment containers with docker compose up -d --force-recreate" --force-recreate || return 1
	wait_for_ready || return 1
	print_success_summary
}

action_stop() {
	require_docker_access
	require_docker_socket
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	acquire_lock stop
	classify_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment to stop"
		;;
	missing_env | mode_conflict | collision)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; resolve it manually before retrying"
		;;
	esac
	activate_persisted_cf_acceleration
	compose_file_args
	enforce_private_env_mode
	require_renderable_configuration
	progress "stopping every resident service with docker compose stop"
	if ! lf_compose stop; then
		fail "docker compose stop failed; inspect the container state with ./status.sh"
	fi
	local remaining
	remaining="$(lf_compose ps --services --filter status=running 2>/dev/null || true)"
	if [ -n "$remaining" ]; then
		fail "these services are still running: $(printf '%s' "$remaining" | tr '\n' ' ')"
	fi
	printf 'LunaFox: SUCCESS the deployment is stopped; containers, networks, volumes, and configuration were preserved.\n'
	printf 'LunaFox:   resume with: ./start.sh\n'
}

action_status() {
	require_docker_access
	require_curl
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	# status is read-only: it never takes the mutation lock, never waits for a
	# state change, and never rewrites .env permissions.
	local overall issues agent_note https_note lock_reason
	LOCK_PRESENT=0

	classify_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		printf 'LunaFox deployment status: not-installed\n'
		printf 'LunaFox:   no LunaFox containers, volumes, or configuration were found\n'
		printf 'LunaFox:   start a deployment with: ./install.sh\n'
		return 1
		;;
	missing_env | collision | mode_conflict)
		printf 'LunaFox deployment status: degraded\n'
		printf 'LunaFox:   %s\n' "$LUNAFOX_DEPLOYMENT_DETAIL"
		return 1
		;;
	esac
	activate_persisted_cf_acceleration
	compose_file_args
	# status is read-only, so it never rewrites .env permissions and never treats
	# a permissive mode as a readiness failure: the documented direct Compose path
	# leaves the extracted file at its package mode. Mutating entries enforce 0600
	# before they change anything.
	if ! env_mode_is_private; then
		printf 'LunaFox:   warning: %s does not have mode 0600; fix it with: chmod 600 %s\n' "$LUNAFOX_ENV_FILE" "$LUNAFOX_ENV_FILE"
	fi
	printf 'LunaFox:   address: %s\n' "$(public_address)"

	collect_probe
	issues=""
	if project_containers_running; then
		overall=starting
	else
		overall=stopped
	fi

	# A leftover lock is reported, not reclaimed: clearing it is a documented
	# manual step because a slow operation must not look like a dead one.
	LOCK_PRESENT=0
	if [ -e "$LUNAFOX_LOCK_PATH" ] || [ -L "$LUNAFOX_LOCK_PATH" ]; then
		LOCK_PRESENT=1
		if lock_reason="$(lock_shape_reason)"; then
			printf 'LunaFox:   lock: %s\n' "$(lock_holder_summary)"
			printf 'LunaFox:     verify that no lifecycle command or upgrade is running, then remove %s manually\n' "$LUNAFOX_LOCK_DIR"
			[ -n "$issues" ] || issues="a deployment lock is present"
		else
			printf 'LunaFox:   lock: %s exists but its metadata cannot be trusted (%s)\n' "$LUNAFOX_LOCK_DIR" "$lock_reason"
			printf 'LunaFox:     verify that no lifecycle command or upgrade is running, then remove it manually\n'
			issues="a deployment lock with untrusted metadata is present"
		fi
	fi

	printf 'LunaFox:   one-shot tasks: %s\n' "$SVC_ONESHOT_STATE"
	print_preheat_status
	printf 'LunaFox:   core services: %s\n' "$SVC_CORE_STATE"
	printf 'LunaFox:   auxiliary services: %s\n' "$SVC_AUX_STATE"
	if [ -n "$PROBE_FAILURE" ]; then
		printf 'LunaFox:   failed check: %s\n' "$PROBE_FAILURE"
		issues="$PROBE_FAILURE"
	fi

	if [ -z "$PROBE_FAILURE" ] && [ -z "$PROBE_WAITING" ] && project_containers_running; then
		if agent_note="$(probe_agent_ready)"; then
			printf 'LunaFox:   resident Agent: ready\n'
			if https_note="$(probe_public_https)"; then
				printf 'LunaFox:   public HTTPS: %s\n' "$https_note"
				overall=ready
			else
				printf 'LunaFox:   public HTTPS: not reachable\n'
				printf 'LunaFox:     %s\n' "$https_note"
				issues="$https_note"
				overall=degraded
			fi
		else
			printf 'LunaFox:   resident Agent: not ready\n'
			printf 'LunaFox:     %s\n' "$agent_note"
			issues="$agent_note"
			overall=starting
		fi
	elif project_containers_running; then
		printf 'LunaFox:   resident Agent: not checked yet\n'
		printf 'LunaFox:   public HTTPS: not checked yet\n'
		[ -n "$issues" ] || issues="${PROBE_WAITING:-the deployment is starting}"
		overall=starting
	else
		printf 'LunaFox:   resident Agent: not checked; no service is running\n'
		printf 'LunaFox:   public HTTPS: not checked; no service is running\n'
		[ -n "$issues" ] || issues="the deployment is stopped"
		overall=stopped
	fi

	if [ "$overall" = ready ] && [ "$LOCK_PRESENT" = 1 ]; then
		# Services answer while an operation is in flight, but the deployment is
		# not in a steady state, so callers and monitoring must not see success.
		overall=starting
	fi

	printf 'LunaFox deployment status: %s\n' "$overall"
	if [ "$overall" = ready ]; then
		return 0
	fi
	printf 'LunaFox:   %s\n' "$issues"
	printf 'LunaFox:   inspect logs: ./logs.sh\n'
	return 1
}

action_logs() {
	require_docker_access
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || fail "$LUNAFOX_ENV_FILE is missing; there is no deployment to read logs from"
	activate_persisted_cf_acceleration
	compose_file_args
	require_renderable_configuration
	if [ "$#" -eq 0 ]; then
		exec docker compose "${COMPOSE_ARGS[@]}" logs --tail 200 --follow
	fi
	exec docker compose "${COMPOSE_ARGS[@]}" logs "$@"
}

action_uninstall() {
	require_docker_access
	require_docker_socket
	sanitize_host_environment
	validate_override_file
	base_compose_file_args
	compose_file_args
	if [ "$UNINSTALL_PURGE" = 1 ] && [ "$UNINSTALL_CONFIRM" != 1 ]; then
		usage_failure "--purge requires --confirm, because it permanently deletes the LunaFox volumes"
	fi
	acquire_lock uninstall
	classify_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment to remove"
		;;
	missing_env | mode_conflict | collision)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; resolve it manually before retrying"
		;;
	esac
	activate_persisted_cf_acceleration
	compose_file_args
	enforce_private_env_mode
	require_renderable_configuration

	local name key failed="" removable="" skipped=""
	if [ "$UNINSTALL_PURGE" = 1 ]; then
		load_declared_volumes
		for name in $(owned_volumes); do
			key="$(volume_compose_key "$name")"
			if ! key_is_declared "$key"; then
				# Owned by this project but not declared by the current compose file:
				# never delete a resource this release does not describe.
				skipped="$skipped $name"
				continue
			fi
			removable="$removable $name"
		done
		if [ -n "$skipped" ]; then
			printf 'LunaFox:   keeping volumes that the current compose.yaml does not declare:%s\n' "$skipped"
		fi
		if [ -n "$failed" ]; then
			fail "these declared volumes cannot be verified as LunaFox-owned, so nothing was removed:$failed"
		fi
	fi

	progress "removing this deployment's containers, orphan containers, and project network"
	if ! lf_compose down --remove-orphans; then
		fail "docker compose down failed; no volumes were removed"
	fi

	if [ "$UNINSTALL_PURGE" = 1 ]; then
		for name in $removable; do
			local holders
			holders="$(docker ps -a --filter "volume=$name" --format '{{.Names}}' 2>/dev/null || true)"
			if [ -n "$holders" ]; then
				fail "$name is still used by: $(printf '%s' "$holders" | tr '\n' ' '); it was preserved"
			fi
			docker volume rm "$name" >/dev/null 2>&1 || fail "could not remove the LunaFox volume $name; it was preserved"
		done
		reset_persisted_override
		reset_persisted_cf_acceleration
		if [ -n "$LUNAFOX_OVERRIDE_USED" ]; then
			printf 'LunaFox: SUCCESS the deployment, its LunaFox volumes, and %s were removed.\n' "$LUNAFOX_OVERRIDE_FILE"
		else
			printf 'LunaFox: SUCCESS the deployment and its LunaFox volumes were removed.\n'
		fi
		printf 'LunaFox:   %s and the release directory were preserved.\n' "$LUNAFOX_ENV_FILE"
		return 0
	fi
	printf 'LunaFox: SUCCESS the deployment containers and network were removed.\n'
	printf 'LunaFox:   preserved: named volumes, %s, certificates, upgrade state, %s, and this release directory.\n' "$LUNAFOX_ENV_FILE" "$LUNAFOX_OVERRIDE_FILE"
	printf 'LunaFox:   restore the deployment with: ./install.sh\n'
	printf 'LunaFox:   delete the preserved data with: ./uninstall.sh --purge --confirm\n'
}

# ------------------------------------------------------------ dispatching ---

usage() {
	cat <<'USAGE'
Usage: lunafox-lifecycle.sh <action>

This helper implements the LunaFox public Compose lifecycle actions. Run the
root scripts instead of calling it directly:

  ./install.sh [--public-host <host>] [--public-port <port>] [--cf-acceleration]
                   first start, repeatable, keeps existing data
  ./start.sh       start an existing deployment
  ./restart.sh     recreate the deployment containers
  ./stop.sh        stop every resident service, keep all data
  ./status.sh      read-only readiness summary
  ./logs.sh        follow Docker Compose logs
  ./uninstall.sh   remove containers and the project network
USAGE
}

parse_install_options() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
		--cf-acceleration)
			[ "$CF_ACCELERATION_REQUESTED" = 0 ] || usage_failure "install accepts --cf-acceleration at most once"
			CF_ACCELERATION_REQUESTED=1
			;;
		--public-host)
			shift
			[ "$#" -gt 0 ] || usage_failure "--public-host requires a value"
			case "$1" in --*) usage_failure "--public-host requires a value" ;; esac
			[ "$INSTALL_PUBLIC_HOST_SET" = 0 ] || usage_failure "install accepts --public-host at most once"
			validate_install_public_host "$1"
			INSTALL_PUBLIC_HOST="$1"
			INSTALL_PUBLIC_HOST_SET=1
			;;
		--public-port)
			shift
			[ "$#" -gt 0 ] || usage_failure "--public-port requires a value"
			case "$1" in --*) usage_failure "--public-port requires a value" ;; esac
			[ "$INSTALL_PUBLIC_PORT_SET" = 0 ] || usage_failure "install accepts --public-port at most once"
			validate_install_public_port "$1"
			INSTALL_PUBLIC_PORT="$1"
			INSTALL_PUBLIC_PORT_SET=1
			;;
		*) usage_failure "install does not accept additional arguments: $1" ;;
		esac
		shift
	done
}

reject_install_only_option() {
	case "$1" in
	--cf-acceleration | --public-host | --public-port)
		usage_failure "$1 is valid only with install"
		;;
	esac
}

main() {
	local action="${1:-}"
	[ -n "$action" ] || {
		usage >&2
		exit 2
	}
	shift
	resolve_root
	case "$action" in
	install)
		parse_install_options "$@"
		action_install
		;;
	start)
		reject_install_only_option "${1:-}"
		[ "$#" -eq 0 ] || usage_failure "start does not accept additional arguments"
		action_start
		;;
	restart)
		reject_install_only_option "${1:-}"
		[ "$#" -eq 0 ] || usage_failure "restart does not accept additional arguments"
		action_restart
		;;
	stop)
		reject_install_only_option "${1:-}"
		[ "$#" -eq 0 ] || usage_failure "stop does not accept additional arguments"
		action_stop
		;;
	status)
		reject_install_only_option "${1:-}"
		[ "$#" -eq 0 ] || usage_failure "status does not accept additional arguments"
		action_status
		;;
	logs)
		for argument in "$@"; do
			reject_install_only_option "$argument"
		done
		action_logs "$@"
		;;
	uninstall)
		while [ "$#" -gt 0 ]; do
			case "$1" in
			--purge) UNINSTALL_PURGE=1 ;;
			--confirm) UNINSTALL_CONFIRM=1 ;;
			--cf-acceleration | --public-host | --public-port) usage_failure "$1 is valid only with install" ;;
			*) usage_failure "uninstall does not accept: $1" ;;
			esac
			shift
		done
		action_uninstall
		;;
	-h | --help)
		usage
		exit 0
		;;
	*) usage_failure "unknown action: $action" ;;
	esac
}

main "$@"
