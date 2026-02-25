#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
	SCRIPT_PATH="$0"
	case "$SCRIPT_PATH" in
	/* | */*) ;;
	*) SCRIPT_PATH="./$SCRIPT_PATH" ;;
	esac
	exec /usr/bin/env bash "$SCRIPT_PATH" "$@"
fi
set -e

TOKEN="{{.Token}}"
REGISTER_URL="{{.RegisterURL}}"
AGENT_SERVER_URL="{{.AgentServerURL}}"
LOKI_PUSH_URL="{{.LokiPushURL}}"
NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-off}"
AGENT_IMAGE_REF="{{.AgentImageRef}}"
AGENT_VERSION="{{.AgentVersion}}"
DEFAULT_WORKER_TOKEN="{{.WorkerToken}}"
WORKER_IMAGE_REF="{{.WorkerImageRef}}"
LOKI_PLUGIN_REFS=()
LOCAL_AGENT_CONFIG="${LUNAFOX_AGENT_USE_LOCAL_LIMITS:-}"
SHARED_DATA_VOLUME_BIND="{{.SharedDataVolumeBind}}"

IFS=':' read -r DATA_VOLUME DATA_TARGET _ <<<"$SHARED_DATA_VOLUME_BIND"
if [ -z "${DATA_VOLUME:-}" ] || [ -z "${DATA_TARGET:-}" ]; then
	echo "LUNAFOX_SHARED_DATA_VOLUME_BIND must be '<named-volume>:/opt/lunafox[:mode]'" >&2
	exit 1
fi
if ! [[ "$DATA_VOLUME" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]; then
	echo "LUNAFOX_SHARED_DATA_VOLUME_BIND source must be Docker named volume (not host path)." >&2
	exit 1
fi
if [ "$DATA_TARGET" != "/opt/lunafox" ]; then
	echo "LUNAFOX_SHARED_DATA_VOLUME_BIND target must be /opt/lunafox" >&2
	exit 1
fi

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "$1 is required" >&2
		exit 1
	fi
}

require_cmd curl

echo "Configuration:"
echo "Register URL: $REGISTER_URL"
echo "Agent server URL: $AGENT_SERVER_URL"
echo "Loki push URL: $LOKI_PUSH_URL"
echo "Network: $NETWORK_NAME"
echo "Data bind: $SHARED_DATA_VOLUME_BIND"

force_local_images=0
if [ "$AGENT_VERSION" = "dev" ]; then
	force_local_images=1
fi

curl_opts=("-fsSL" "--connect-timeout" "10" "--max-time" "30" "-k")

if ! command -v docker >/dev/null 2>&1; then
	echo "Docker is required. Install it first: https://docs.docker.com/engine/install/" >&2
	exit 1
fi

DOCKER_CMD="docker"
if ! docker info >/dev/null 2>&1; then
	if sudo docker info >/dev/null 2>&1; then
		DOCKER_CMD="sudo docker"
	else
		echo "Docker daemon is not running or is not accessible." >&2
		echo "Please start Docker and ensure the current user can access the Docker daemon." >&2
		exit 1
	fi
fi

validate_loki_push_url() {
	case "$LOKI_PUSH_URL" in
	https://*) ;;
	*)
		echo "LOKI_PUSH_URL must be a complete https URL, got: $LOKI_PUSH_URL" >&2
		return 1
		;;
	esac
	return 0
}

ensure_loki_plugin() {
	local existing_line existing_name existing_enabled arch ref

	existing_line="$($DOCKER_CMD plugin ls --format '{{"{{.Name}}"}} {{"{{.Enabled}}"}}' | awk '$1 ~ /^loki(:|$)/ {print $0; exit}')"
	if [ -n "$existing_line" ]; then
		existing_name="${existing_line%% *}"
		existing_enabled="${existing_line##* }"
		if [ "$existing_enabled" = "true" ]; then
			return 0
		fi
		echo "Enabling Loki Docker plugin ($existing_name)..."
		if $DOCKER_CMD plugin enable "$existing_name" >/dev/null 2>&1; then
			return 0
		fi
	fi

	arch="$(detect_docker_arch)"
	build_loki_plugin_refs "$arch"

	echo "Installing Loki Docker plugin..."
	for ref in "${LOKI_PLUGIN_REFS[@]}"; do
		if [ -z "$ref" ]; then
			continue
		fi
		echo "Trying Loki Docker plugin ref: $ref"
		if $DOCKER_CMD plugin install "$ref" --alias loki --grant-all-permissions; then
			return 0
		fi
	done

	if $DOCKER_CMD plugin ls --format '{{"{{.Name}}"}}' | grep -Eq '^loki(:|$)'; then
		return 0
	fi

	echo "Warning: failed to install Loki Docker plugin (${LOKI_PLUGIN_REFS[*]})." >&2
	echo "Warning: agent container will start without Loki log driver." >&2
	return 1
}

detect_docker_arch() {
	local raw_arch
	raw_arch="$($DOCKER_CMD info --format '{{"{{.Architecture}}"}}' 2>/dev/null || true)"
	case "$raw_arch" in
	aarch64 | arm64)
		echo "arm64"
		;;
	x86_64 | amd64)
		echo "amd64"
		;;
	*)
		echo ""
		;;
	esac
}

build_loki_plugin_refs() {
	local arch="$1"
	LOKI_PLUGIN_REFS=()
	if [ -n "$arch" ]; then
		LOKI_PLUGIN_REFS+=("grafana/loki-docker-driver:3.6.7-${arch}")
	fi
}

probe_loki_push_url() {
	# /loki/api/v1/push is a write endpoint; GET may return 404/405 even when reachable.
	# Treat this as a connectivity probe only and avoid failing on HTTP status codes.
	if ! curl -ksSL --connect-timeout 10 --max-time 30 -o /dev/null "$LOKI_PUSH_URL"; then
		echo "Warning: LOKI_PUSH_URL connectivity probe failed (network/TLS): $LOKI_PUSH_URL" >&2
	fi
}

NETWORK_ARGS=()
if [ -n "$NETWORK_NAME" ] && [ "$NETWORK_NAME" != "off" ] && [ "$NETWORK_NAME" != "none" ]; then
	if $DOCKER_CMD network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
		NETWORK_ARGS=(--network "$NETWORK_NAME")
	else
		echo "Docker network '$NETWORK_NAME' not found, using default bridge." >&2
	fi
fi

image_exists() {
	$DOCKER_CMD image inspect "$1" >/dev/null 2>&1
}

if [ -z "${WORKER_TOKEN:-}" ]; then
	WORKER_TOKEN="$DEFAULT_WORKER_TOKEN"
fi

if [ -z "${WORKER_TOKEN:-}" ]; then
	echo "WORKER_TOKEN is required (export WORKER_TOKEN=...)" >&2
	exit 1
fi

HOSTNAME="${AGENT_HOSTNAME:-$(hostname)}"

echo "Installing LunaFox Agent $AGENT_VERSION..."
echo "Registering agent..."
MAX_TASKS_VALUE=""
CPU_THRESHOLD_VALUE=""
MEM_THRESHOLD_VALUE=""
DISK_THRESHOLD_VALUE=""

if [ "$LOCAL_AGENT_CONFIG" = "1" ] || [ "$LOCAL_AGENT_CONFIG" = "true" ]; then
	MAX_TASKS_VALUE="${LUNAFOX_AGENT_MAX_TASKS:-10}"
	CPU_THRESHOLD_VALUE="${LUNAFOX_AGENT_CPU_THRESHOLD:-80}"
	MEM_THRESHOLD_VALUE="${LUNAFOX_AGENT_MEM_THRESHOLD:-80}"
	DISK_THRESHOLD_VALUE="${LUNAFOX_AGENT_DISK_THRESHOLD:-85}"
fi

REGISTER_PAYLOAD=$(printf '{"token":"%s","hostname":"%s","version":"%s"' "$TOKEN" "$HOSTNAME" "$AGENT_VERSION")
if [ -n "$MAX_TASKS_VALUE" ]; then
	REGISTER_PAYLOAD=$(printf '%s,"maxTasks":%s,"cpuThreshold":%s,"memThreshold":%s,"diskThreshold":%s' \
		"$REGISTER_PAYLOAD" "$MAX_TASKS_VALUE" "$CPU_THRESHOLD_VALUE" "$MEM_THRESHOLD_VALUE" "$DISK_THRESHOLD_VALUE")
fi
REGISTER_PAYLOAD=$(printf '%s}' "$REGISTER_PAYLOAD")

RESPONSE=$(curl "${curl_opts[@]}" \
	-X POST "$REGISTER_URL/api/agent/register" \
	-H "Content-Type: application/json" \
	-d "$REGISTER_PAYLOAD" 2>&1) || {
	echo "Registration failed: $RESPONSE" >&2
	exit 1
}
AGENT_ID="$(printf '%s' "$RESPONSE" | sed -n 's/.*"agentId"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n1)"
API_KEY="$(printf '%s' "$RESPONSE" | sed -n 's/.*"apiKey"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

if [ -z "$API_KEY" ]; then
	API_KEY="${RESPONSE//$'\r'/}"
	API_KEY="${API_KEY//$'\n'/}"
fi

if [ -z "$API_KEY" ]; then
	echo "Failed to obtain API key" >&2
	exit 1
fi

if [ -z "${AGENT_ID:-}" ]; then
	echo "Failed to obtain agent ID from registration response" >&2
	exit 1
fi

if ! validate_loki_push_url; then
	exit 1
fi

USE_LOKI_DRIVER=1
if ! ensure_loki_plugin; then
	USE_LOKI_DRIVER=0
fi
probe_loki_push_url

echo "Pulling agent image..."
if [ "$force_local_images" -eq 1 ]; then
	if image_exists "$AGENT_IMAGE_REF"; then
		echo "Using local agent image: $AGENT_IMAGE_REF"
	else
		echo "Local agent image not found for dev mode: $AGENT_IMAGE_REF" >&2
		exit 1
	fi
else
	if image_exists "$AGENT_IMAGE_REF"; then
		echo "Local agent image exists, skip pull: $AGENT_IMAGE_REF"
	else
		$DOCKER_CMD pull "$AGENT_IMAGE_REF"
	fi
fi

echo "Pulling worker image..."
if [ "$force_local_images" -eq 1 ]; then
	if image_exists "$WORKER_IMAGE_REF"; then
		echo "Using local worker image: $WORKER_IMAGE_REF"
	else
		echo "Local worker image not found for dev mode: $WORKER_IMAGE_REF" >&2
		exit 1
	fi
else
	if image_exists "$WORKER_IMAGE_REF"; then
		echo "Local worker image exists, skip pull: $WORKER_IMAGE_REF"
	else
		$DOCKER_CMD pull "$WORKER_IMAGE_REF"
	fi
fi

$DOCKER_CMD rm -f lunafox-agent >/dev/null 2>&1 || true

LOGGING_ARGS=()
if [ "$USE_LOKI_DRIVER" -eq 1 ]; then
	LOGGING_ARGS=(
		--log-driver=loki
		--log-opt "loki-url=$LOKI_PUSH_URL"
		--log-opt "loki-tls-insecure-skip-verify=true"
		--log-opt "no-file=true"
		--log-opt "mode=non-blocking"
		--log-opt "loki-batch-size=1048576"
		--log-opt "max-buffer-size=5m"
		--log-opt "loki-retries=3"
		--log-opt "loki-external-labels=agent_id=$AGENT_ID,container_name=lunafox-agent"
	)
fi

$DOCKER_CMD run -d --restart unless-stopped --name lunafox-agent \
	"${NETWORK_ARGS[@]}" \
	"${LOGGING_ARGS[@]}" \
	--hostname "$HOSTNAME" \
	-e SERVER_URL="$AGENT_SERVER_URL" \
	-e API_KEY="$API_KEY" \
	-e WORKER_TOKEN="$WORKER_TOKEN" \
	-e WORKER_IMAGE_REF="$WORKER_IMAGE_REF" \
	-e LUNAFOX_SHARED_DATA_VOLUME_BIND="$SHARED_DATA_VOLUME_BIND" \
	-e AGENT_VERSION="$AGENT_VERSION" \
	-e AGENT_HOSTNAME="$HOSTNAME" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$SHARED_DATA_VOLUME_BIND" \
	"$AGENT_IMAGE_REF" >/dev/null

echo "Agent installed and running (container: lunafox-agent)"
