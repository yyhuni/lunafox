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
RUNTIME_GRPC_URL="{{.RuntimeGRPCURL}}"
LOKI_PUSH_URL="{{.LokiPushURL}}"
REQUIRE_DOCKER_NETWORK="{{.RequireDockerNetwork}}"
NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-{{.DockerNetworkDefault}}}"
AGENT_IMAGE_REF="{{.AgentImageRef}}"
AGENT_VERSION="{{.AgentVersion}}"
WORKER_IMAGE_REF="{{.WorkerImageRef}}"
LOCAL_AGENT_CONFIG="${LUNAFOX_AGENT_USE_LOCAL_LIMITS:-}"
SHARED_DATA_VOLUME_BIND="{{.SharedDataVolumeBind}}"
RUNTIME_VOLUME="lunafox_runtime"
RUNTIME_SOCKET_PATH="/run/lunafox/worker-runtime.sock"

validate_inputs() {
	local data_volume="" data_target=""
	IFS=':' read -r data_volume data_target _ <<<"$SHARED_DATA_VOLUME_BIND"
	if [ -z "${data_volume:-}" ] || [ -z "${data_target:-}" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND must be '<named-volume>:/opt/lunafox[:mode]'" >&2
		exit 1
	fi
	if ! [[ "$data_volume" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND source must be Docker named volume (not host path)." >&2
		exit 1
	fi
	if [ "$data_target" != "/opt/lunafox" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND target must be /opt/lunafox" >&2
		exit 1
	fi
	if ! [[ "$RUNTIME_VOLUME" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]; then
		echo "LUNAFOX_RUNTIME_VOLUME must be a Docker named volume." >&2
		exit 1
	fi
	if [ -z "${RUNTIME_GRPC_URL:-}" ]; then
		echo "RUNTIME_GRPC_URL is required." >&2
		exit 1
	fi
}

validate_inputs

require_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "$1 is required" >&2
		exit 1
	fi
}

require_cmd curl

echo "Configuration:"
echo "Register URL: $REGISTER_URL"
echo "Runtime URL: $RUNTIME_GRPC_URL"
echo "Loki push URL: $LOKI_PUSH_URL"
echo "Network: $NETWORK_NAME"
echo "Data bind: $SHARED_DATA_VOLUME_BIND"
echo "Runtime volume: $RUNTIME_VOLUME"

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

$DOCKER_CMD volume create "$RUNTIME_VOLUME" >/dev/null

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

	echo "Failed to install Loki Docker plugin (${LOKI_PLUGIN_REFS[*]})." >&2
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

image_exists() {
	$DOCKER_CMD image inspect "$1" >/dev/null 2>&1
}

resolve_network_args() {
	NETWORK_ARGS=()
	if [ "$REQUIRE_DOCKER_NETWORK" = "1" ]; then
		# Local profile must stay in the compose network so service DNS name `server`
		# resolves consistently; fallback to default bridge would break that contract.
		if [ -z "$NETWORK_NAME" ] || [ "$NETWORK_NAME" = "off" ] || [ "$NETWORK_NAME" = "none" ]; then
			echo "LUNAFOX_AGENT_DOCKER_NETWORK is required for local profile." >&2
			exit 1
		fi
		if ! $DOCKER_CMD network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
			echo "Docker network '$NETWORK_NAME' not found. Create it or set LUNAFOX_AGENT_DOCKER_NETWORK to an existing network." >&2
			exit 1
		fi
		NETWORK_ARGS=(--network "$NETWORK_NAME")
	elif [ -n "$NETWORK_NAME" ] && [ "$NETWORK_NAME" != "off" ] && [ "$NETWORK_NAME" != "none" ]; then
		if $DOCKER_CMD network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
			NETWORK_ARGS=(--network "$NETWORK_NAME")
		else
			# Remote profile keeps network optional, but explicit network input
			# must be honored strictly to avoid silent topology mismatch.
			echo "Docker network '$NETWORK_NAME' not found. Create it or set LUNAFOX_AGENT_DOCKER_NETWORK to an existing network." >&2
			exit 1
		fi
	fi
}

register_agent() {
	local max_tasks_value="" cpu_threshold_value="" mem_threshold_value="" disk_threshold_value=""
	local register_payload response

	echo "Registering agent..."
	if [ "$LOCAL_AGENT_CONFIG" = "1" ] || [ "$LOCAL_AGENT_CONFIG" = "true" ]; then
		max_tasks_value="${LUNAFOX_AGENT_MAX_TASKS:-10}"
		cpu_threshold_value="${LUNAFOX_AGENT_CPU_THRESHOLD:-80}"
		mem_threshold_value="${LUNAFOX_AGENT_MEM_THRESHOLD:-80}"
		disk_threshold_value="${LUNAFOX_AGENT_DISK_THRESHOLD:-85}"
	fi

	register_payload=$(printf '{"token":"%s","hostname":"%s","version":"%s"' "$TOKEN" "$HOSTNAME" "$AGENT_VERSION")
	if [ -n "$max_tasks_value" ]; then
		register_payload=$(printf '%s,"maxTasks":%s,"cpuThreshold":%s,"memThreshold":%s,"diskThreshold":%s' \
			"$register_payload" "$max_tasks_value" "$cpu_threshold_value" "$mem_threshold_value" "$disk_threshold_value")
	fi
	register_payload=$(printf '%s}' "$register_payload")

	response=$(curl "${curl_opts[@]}" \
		-X POST "$REGISTER_URL/api/agent/register" \
		-H "Content-Type: application/json" \
		-d "$register_payload" 2>&1) || {
		echo "Registration failed: $response" >&2
		exit 1
	}

	AGENT_ID="$(printf '%s' "$response" | sed -n 's/.*"agentId"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n1)"
	API_KEY="$(printf '%s' "$response" | sed -n 's/.*"apiKey"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

	if [ -z "$API_KEY" ]; then
		API_KEY="${response//$'\r'/}"
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
}

ensure_images() {
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
}

resolve_logging_args() {
	LOGGING_ARGS=()
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
}

resolve_network_args

HOSTNAME="${AGENT_HOSTNAME:-$(hostname)}"

echo "Installing LunaFox Agent $AGENT_VERSION..."
register_agent

if ! validate_loki_push_url; then
	exit 1
fi

if ! ensure_loki_plugin; then
	echo "Failed to install Loki Docker plugin, exiting." >&2
	exit 1
fi
probe_loki_push_url

ensure_images

$DOCKER_CMD rm -f lunafox-agent >/dev/null 2>&1 || true

resolve_logging_args

$DOCKER_CMD run -d --restart unless-stopped --name lunafox-agent \
	"${NETWORK_ARGS[@]}" \
	"${LOGGING_ARGS[@]}" \
	--hostname "$HOSTNAME" \
	-e RUNTIME_GRPC_URL="$RUNTIME_GRPC_URL" \
	-e API_KEY="$API_KEY" \
	-e WORKER_IMAGE_REF="$WORKER_IMAGE_REF" \
	-e LUNAFOX_SHARED_DATA_VOLUME_BIND="$SHARED_DATA_VOLUME_BIND" \
	-e LUNAFOX_RUNTIME_SOCKET="$RUNTIME_SOCKET_PATH" \
	-e AGENT_VERSION="$AGENT_VERSION" \
	-e AGENT_HOSTNAME="$HOSTNAME" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$SHARED_DATA_VOLUME_BIND" \
	-v "${RUNTIME_VOLUME}:/run/lunafox" \
	"$AGENT_IMAGE_REF" >/dev/null

echo "Agent installed and running (container: lunafox-agent)"
