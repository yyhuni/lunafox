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
LUNAFOX_MIN_COMPOSE_VERSION=2.24.0
LUNAFOX_INSTALL_TIMEOUT_SECONDS=900
LUNAFOX_DAILY_TIMEOUT_SECONDS=300
LUNAFOX_POLL_SECONDS=5
# Waiting must stay visible: the heartbeat is independent of the poll cadence so
# readiness probing never has to slow down to keep the user informed.
LUNAFOX_HEARTBEAT_SECONDS=15
LUNAFOX_PROBE_TIMEOUT_SECONDS=20

# The one-shot tasks, healthy core services, and healthcheck-free resident
# services of the supported Compose graph. Unknown services are handled
# tolerantly below so a newer deployment snapshot still reaches a verdict.
LUNAFOX_ONESHOT_SERVICES="config-init agent-preflight migrate bootstrap cert-init"
LUNAFOX_CORE_SERVICES="postgres redis loki server frontend nginx"
LUNAFOX_AUX_SERVICES="agent upgrader alloy"

# Keys that would otherwise let a caller's shell override the persistent .env
# or redirect Compose at another project. LUNAFOX_READY_TIMEOUT_SECONDS is
# deliberately not in this list: it is a documented per-invocation override.
LUNAFOX_OVERRIDING_KEYS="RELEASE_REGISTRY RELEASE_CHANNEL RELEASE_METADATA_BASE_URL \
RELEASE_VERSION AGENT_VERSION PUBLIC_HOST PUBLIC_PORT DATABASE_MODE COMPOSE_PROFILES \
DB_HOST DB_PORT DB_USER DB_NAME DB_SSLMODE DB_PASSWORD JWT_SECRET \
SERVER_IMAGE_REF FRONTEND_IMAGE_REF NGINX_IMAGE_REF AGENT_IMAGE_REF BOOTSTRAP_IMAGE_REF \
ENGINE_INSTALL_REGISTRY ENGINE_INVENTORY_HOST_PATH LUNAFOX_SHARED_DATA_VOLUME_BIND \
COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_ENV_FILES COMPOSE_PATH_SEPARATOR"

READY_TIMEOUT=""
UNINSTALL_PURGE=0
UNINSTALL_CONFIRM=0
LOCK_HELD=0

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

# Builds the Compose argv once and reuses it for every call. The override is
# appended only when the upgrader installed it as a regular file.
compose_file_args() {
	COMPOSE_ARGS=(
		--project-name "$LUNAFOX_PROJECT_NAME"
		--project-directory "$LUNAFOX_ROOT"
		--env-file "$LUNAFOX_ENV_PATH"
		-f "$LUNAFOX_COMPOSE_PATH"
	)
	if [ -n "$LUNAFOX_OVERRIDE_USED" ]; then
		COMPOSE_ARGS+=(-f "$LUNAFOX_OVERRIDE_PATH")
	fi
}

lf_compose() {
	docker compose "${COMPOSE_ARGS[@]}" "$@"
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

register_cleanup_path() {
	CLEANUP_PATHS="$CLEANUP_PATHS
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
PROBE_WAITING=""
PROBE_CORE_PENDING=""
PROBE_AUX_PENDING=""
PROBE_ONESHOT_PENDING=""
PROBE_UNKNOWN_PENDING=""

# Translates one docker compose ps row into the readiness dimensions. Explicit
# failures are separated from "still starting" so the caller can return
# immediately on an unrecoverable state instead of burning the whole window.
classify_service_row() {
	local service="$1" state="$2" health="$3" exit_code="$4"
	case "$state" in
	'') state=created ;;
	esac
	if in_list "$service" "$LUNAFOX_ONESHOT_SERVICES"; then
		case "$state" in
		exited)
			if [ "$exit_code" != 0 ]; then
				[ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service task exited with code ${exit_code:-unknown}"
			fi
			return 0
			;;
		dead)
			[ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service task failed"
			return 0
			;;
		esac
		PROBE_ONESHOT_PENDING="$service"
		return 0
	fi
	if in_list "$service" "$LUNAFOX_CORE_SERVICES"; then
		if [ "$state" = exited ] || [ "$state" = dead ]; then
			[ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service stopped"
			return 0
		fi
		case "$health" in
		healthy) return 0 ;;
		unhealthy)
			[ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service is unhealthy"
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
			[ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service exited"
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
		[ "$exit_code" = 0 ] || { [ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service exited with code ${exit_code:-unknown}"; }
		;;
	running)
		case "$health" in
		unhealthy) [ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service is unhealthy" ;;
		starting | '') PROBE_UNKNOWN_PENDING="$service" ;;
		esac
		;;
	dead) [ -n "$PROBE_FAILURE" ] || PROBE_FAILURE="the $service service failed" ;;
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
	PROBE_WAITING=""
	PROBE_CORE_PENDING=""
	PROBE_AUX_PENDING=""
	PROBE_ONESHOT_PENDING=""
	PROBE_UNKNOWN_PENDING=""

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

report_ready_failure() {
	local reason="$1" stage="$2" hint="$3"
	fail "$reason
LunaFox:   stage: $stage
LunaFox:   deployment: $LUNAFOX_ROOT (kept as-is; containers, volumes, and configuration were not changed)
LunaFox:   inspect logs: $hint
LunaFox:   recheck with: ./status.sh"
}

# Blocks until every readiness condition passes, an unrecoverable condition
# appears, or the window expires. Nothing is stopped or rolled back on failure.
wait_for_ready() {
	local started="$SECONDS" last_signature="" signature agent_note https_note
	local elapsed
	READY_ELAPSED=0
	LAST_PROGRESS_MESSAGE=""
	LAST_PROGRESS_ELAPSED=0
	while :; do
		elapsed=$((SECONDS - started))
		collect_probe
		if [ -n "$PROBE_FAILURE" ]; then
			report_ready_failure "$PROBE_FAILURE" "waiting for the deployment to become ready" "./logs.sh"
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
			report_ready_failure "the deployment did not become ready within $READY_TIMEOUT seconds" \
				"${PROBE_WAITING:-all readiness conditions}" "./logs.sh"
		fi
		if [ -n "$PROBE_WAITING" ]; then
			progress_heartbeat "$PROBE_WAITING" "$elapsed"
		fi
		sleep "$LUNAFOX_POLL_SECONDS"
	done
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
	local description="$1"
	shift
	progress "$description"
	if ! lf_compose up -d "$@"; then
		report_ready_failure "docker compose up -d${1:+ $*} failed" "$description" "./logs.sh"
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
	require_docker_access
	require_docker_socket
	require_curl
	require_named_volume_capability
	sanitize_host_environment
	validate_override_file
	compose_file_args
	acquire_lock install

	classify_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	missing_env | mode_conflict | collision)
		fail "$LUNAFOX_DEPLOYMENT_DETAIL; resolve it manually before retrying, because a first start is the only case that may create $LUNAFOX_ENV_FILE"
		;;
	existing) progress "an existing LunaFox deployment was detected; its configuration and volumes are preserved" ;;
	repairable) progress "an existing LunaFox deployment was detected without containers; they will be recreated" ;;
	fresh) progress "preparing a first start" ;;
	esac
	if [ ! -e "$LUNAFOX_ENV_PATH" ]; then
		create_env_from_template "$LUNAFOX_ENV_TEMPLATE_PATH"
	fi
	enforce_private_env_mode
	require_renderable_configuration
	check_public_port
	compose_up_once "starting LunaFox with docker compose up -d (a first start pulls images and initializes the database, so it can take a few minutes)" || return 1
	wait_for_ready || return 1
	print_success_summary
}

action_start() {
	require_ready_timeout "$LUNAFOX_DAILY_TIMEOUT_SECONDS"
	require_docker_access
	require_docker_socket
	require_curl
	sanitize_host_environment
	validate_override_file
	compose_file_args
	acquire_lock start
	require_mutable_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment yet; run ./install.sh for the first start"
		;;
	esac
	enforce_private_env_mode
	require_renderable_configuration
	check_public_port
	compose_up_once "starting the existing deployment with docker compose up -d" || return 1
	wait_for_ready || return 1
	print_success_summary
}

action_restart() {
	require_ready_timeout "$LUNAFOX_DAILY_TIMEOUT_SECONDS"
	require_docker_access
	require_docker_socket
	require_curl
	sanitize_host_environment
	validate_override_file
	compose_file_args
	acquire_lock restart
	require_mutable_deployment
	case "$LUNAFOX_DEPLOYMENT_STATE" in
	fresh)
		fail "this directory has no LunaFox deployment yet; run ./install.sh for the first start"
		;;
	esac
	enforce_private_env_mode
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
	compose_file_args
	require_env_regular_file "$LUNAFOX_ENV_PATH" ".env" || fail "$LUNAFOX_ENV_FILE is missing; there is no deployment to read logs from"
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
		printf 'LunaFox: SUCCESS the deployment and its LunaFox volumes were removed.\n'
		printf 'LunaFox:   %s and the release directory were preserved.\n' "$LUNAFOX_ENV_FILE"
		return 0
	fi
	printf 'LunaFox: SUCCESS the deployment containers and network were removed.\n'
	printf 'LunaFox:   preserved: named volumes, %s, certificates, upgrade state, and this release directory.\n' "$LUNAFOX_ENV_FILE"
	printf 'LunaFox:   restore the deployment with: ./install.sh\n'
	printf 'LunaFox:   delete the preserved data with: ./uninstall.sh --purge --confirm\n'
}

# ------------------------------------------------------------ dispatching ---

usage() {
	cat <<'USAGE'
Usage: lunafox-lifecycle.sh <action>

This helper implements the LunaFox public Compose lifecycle actions. Run the
root scripts instead of calling it directly:

  ./install.sh     first start, repeatable, keeps existing data
  ./start.sh       start an existing deployment
  ./restart.sh     recreate the deployment containers
  ./stop.sh        stop every resident service, keep all data
  ./status.sh      read-only readiness summary
  ./logs.sh        follow Docker Compose logs
  ./uninstall.sh   remove containers and the project network
USAGE
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
		[ "$#" -eq 0 ] || usage_failure "install does not accept additional arguments"
		action_install
		;;
	start)
		[ "$#" -eq 0 ] || usage_failure "start does not accept additional arguments"
		action_start
		;;
	restart)
		[ "$#" -eq 0 ] || usage_failure "restart does not accept additional arguments"
		action_restart
		;;
	stop)
		[ "$#" -eq 0 ] || usage_failure "stop does not accept additional arguments"
		action_stop
		;;
	status)
		[ "$#" -eq 0 ] || usage_failure "status does not accept additional arguments"
		action_status
		;;
	logs)
		action_logs "$@"
		;;
	uninstall)
		while [ "$#" -gt 0 ]; do
			case "$1" in
			--purge) UNINSTALL_PURGE=1 ;;
			--confirm) UNINSTALL_CONFIRM=1 ;;
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
