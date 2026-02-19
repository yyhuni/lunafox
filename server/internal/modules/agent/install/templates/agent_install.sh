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
NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-off}"
AGENT_IMAGE_REF="{{.AgentImageRef}}"
AGENT_VERSION="{{.AgentVersion}}"
DEFAULT_WORKER_TOKEN="{{.WorkerToken}}"
WORKER_IMAGE_REF="{{.WorkerImageRef}}"
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
API_KEY="$(printf '%s' "$RESPONSE" | sed -n 's/.*"apiKey"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

if [ -z "$API_KEY" ]; then
	API_KEY="${RESPONSE//$'\r'/}"
	API_KEY="${API_KEY//$'\n'/}"
fi

if [ -z "$API_KEY" ]; then
	echo "Failed to obtain API key" >&2
	exit 1
fi

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
$DOCKER_CMD run -d --restart unless-stopped --name lunafox-agent \
	"${NETWORK_ARGS[@]}" \
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
