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
AGENT_CONTROL_URL="{{.AgentControlURL}}"
LOKI_PUSH_URL="{{.LokiPushURL}}"
REQUIRE_DOCKER_NETWORK="{{.RequireDockerNetwork}}"
NETWORK_NAME="${LUNAFOX_AGENT_DOCKER_NETWORK:-{{.DockerNetworkDefault}}}"
AGENT_IMAGE_REF="{{.AgentImageRef}}"
AGENT_VERSION="{{.AgentVersion}}"
LOCAL_AGENT_CONFIG="${LUNAFOX_AGENT_USE_LOCAL_LIMITS:-}"
SHARED_DATA_VOLUME_BIND="{{.SharedDataVolumeBind}}"
SHARED_DATA_VOLUME="lunafox_data"
ENGINE_EXECUTION_VOLUME="lunafox_engine_execution"
STATE_VOLUME="lunafox_agent_state"
SHARED_DATA_TARGET="/opt/lunafox"
ENGINE_EXECUTION_ROOT="/var/lib/lunafox/engine-execution"

validate_inputs() {
	local data_volume="" data_target="" data_mode="" data_extra=""
	IFS=':' read -r data_volume data_target data_mode data_extra <<<"$SHARED_DATA_VOLUME_BIND"
	if [ -z "${data_volume:-}" ] || [ -z "${data_target:-}" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND must be '${SHARED_DATA_VOLUME}:${SHARED_DATA_TARGET}[:rw]'" >&2
		exit 1
	fi
	if [ -n "${data_extra:-}" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND contains unsupported fields." >&2
		exit 1
	fi
	if [ "$data_volume" != "$SHARED_DATA_VOLUME" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND source must be ${SHARED_DATA_VOLUME}." >&2
		exit 1
	fi
	if [ "$data_target" != "$SHARED_DATA_TARGET" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND target must be ${SHARED_DATA_TARGET}" >&2
		exit 1
	fi
	if [ -n "${data_mode:-}" ] && [ "$data_mode" != "rw" ]; then
		echo "LUNAFOX_SHARED_DATA_VOLUME_BIND mode must be rw." >&2
		exit 1
	fi
	if ! [[ "$ENGINE_EXECUTION_VOLUME" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]; then
		echo "LUNAFOX_ENGINE_EXECUTION_VOLUME must be a Docker named volume." >&2
		exit 1
	fi
	if ! [[ "$STATE_VOLUME" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]]; then
		echo "STATE_VOLUME must be a Docker named volume." >&2
		exit 1
	fi
	if [ -z "${AGENT_CONTROL_URL:-}" ]; then
		echo "AGENT_CONTROL_URL is required." >&2
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
echo "Agent control URL: $AGENT_CONTROL_URL"
echo "Loki push URL: $LOKI_PUSH_URL"
echo "Network: $NETWORK_NAME"
echo "Data bind: $SHARED_DATA_VOLUME_BIND"
echo "Engine execution volume: $ENGINE_EXECUTION_VOLUME"

force_local_images=0
case "$AGENT_IMAGE_REF" in
*:dev)
	force_local_images=1
	;;
esac
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
		# Internal profile must stay in the deployment network so service DNS name `server`
		# resolves consistently; fallback to default bridge would break that contract.
		if [ -z "$NETWORK_NAME" ] || [ "$NETWORK_NAME" = "off" ] || [ "$NETWORK_NAME" = "none" ]; then
			echo "LUNAFOX_AGENT_DOCKER_NETWORK is required for internal profile." >&2
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
			# External profile keeps network optional, but explicit network input
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

	register_payload=$(printf '{"token":"%s","observedHostname":"%s","agentVersion":"%s"' "$TOKEN" "$HOSTNAME" "$AGENT_VERSION")
	if [ -n "$max_tasks_value" ]; then
		register_payload=$(printf '%s,"maxTasks":%s,"cpuThreshold":%s,"memThreshold":%s,"diskThreshold":%s' \
			"$register_payload" "$max_tasks_value" "$cpu_threshold_value" "$mem_threshold_value" "$disk_threshold_value")
	fi
	register_payload=$(printf '%s}' "$register_payload")

	response=$(curl "${curl_opts[@]}" \
		-X POST "$REGISTER_URL/v1/agents:register" \
		-H "Content-Type: application/json" \
		-d "$register_payload" 2>&1) || {
		echo "Registration failed: $response" >&2
		exit 1
	}

	AGENT_ID="$(printf '%s' "$response" | sed -n 's/.*"agentId"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n1)"
	AGENT_INSTANCE_ID="$(printf '%s' "$response" | sed -n 's/.*"instanceId"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
	AGENT_DISPLAY_NAME="$(printf '%s' "$response" | sed -n 's/.*"displayName"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
	AGENT_AUTHENTICATION_TOKEN="$(printf '%s' "$response" | sed -n 's/.*"agentAuthenticationToken"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"

	if [ -z "$AGENT_AUTHENTICATION_TOKEN" ]; then
		AGENT_AUTHENTICATION_TOKEN="${response//$'\r'/}"
		AGENT_AUTHENTICATION_TOKEN="${AGENT_AUTHENTICATION_TOKEN//$'\n'/}"
	fi

	if [ -z "$AGENT_AUTHENTICATION_TOKEN" ]; then
		echo "Failed to obtain authentication token" >&2
		exit 1
	fi
	if [ -z "${AGENT_ID:-}" ]; then
		echo "Failed to obtain agent ID from registration response" >&2
		exit 1
	fi
	if [ -z "${AGENT_INSTANCE_ID:-}" ]; then
		echo "Failed to obtain instance ID from registration response" >&2
		exit 1
	fi
	if [ -z "${AGENT_DISPLAY_NAME:-}" ]; then
		echo "Failed to obtain display name from registration response" >&2
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

}

ensure_named_volume_identity() {
	local expected="$1" created observed
	created="$($DOCKER_CMD volume create "$expected")"
	if [ "$created" != "$expected" ]; then
		echo "Docker volume create identity drift: expected '$expected', got '$created'." >&2
		exit 1
	fi
	observed="$($DOCKER_CMD volume inspect --format '{{"{{.Name}}"}}' "$expected")"
	if [ "$observed" != "$expected" ]; then
		echo "Docker volume inspect identity drift: expected '$expected', got '$observed'." >&2
		exit 1
	fi
}

ensure_named_volumes() {
	ensure_named_volume_identity "$SHARED_DATA_VOLUME"
	ensure_named_volume_identity "$ENGINE_EXECUTION_VOLUME"
	ensure_named_volume_identity "$STATE_VOLUME"
}

run_engine_mount_preflight() {
	local probe_id
	probe_id="install-$(date +%s)-$$"
	echo "Verifying Engine named-volume subpath support..."
	$DOCKER_CMD run --rm \
		--name "lunafox-agent-mount-preflight-${probe_id}" \
		--network none \
		--entrypoint /usr/local/bin/lunafox-engine-mount-preflight \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "${ENGINE_EXECUTION_VOLUME}:${ENGINE_EXECUTION_ROOT}" \
		-v "${SHARED_DATA_VOLUME}:${SHARED_DATA_TARGET}" \
		"$AGENT_IMAGE_REF" \
		--profile capability \
		--image-ref "$AGENT_IMAGE_REF" \
		--execution-volume "$ENGINE_EXECUTION_VOLUME" \
		--execution-root "$ENGINE_EXECUTION_ROOT" \
		--shared-volume "$SHARED_DATA_VOLUME" \
		--shared-root "$SHARED_DATA_TARGET" \
		--run-id "$probe_id"
}

resolve_logging_args() {
	LOGGING_ARGS=()
	LOGGING_ARGS=(
		--log-driver=loki
		--log-opt "loki-url=$LOKI_PUSH_URL"
		--log-opt "loki-tls-insecure-skip-verify=true"
		--log-opt "no-file=false"
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

if ! validate_loki_push_url; then
	exit 1
fi

if ! ensure_loki_plugin; then
	echo "Failed to install Loki Docker plugin, exiting." >&2
	exit 1
fi
probe_loki_push_url

ensure_images
ensure_named_volumes
run_engine_mount_preflight
register_agent

$DOCKER_CMD rm -f lunafox-agent >/dev/null 2>&1 || true

resolve_logging_args

$DOCKER_CMD run -d --restart unless-stopped --name lunafox-agent \
	"${NETWORK_ARGS[@]}" \
	"${LOGGING_ARGS[@]}" \
	--hostname "$HOSTNAME" \
	-e AGENT_CONTROL_URL="$AGENT_CONTROL_URL" \
	-e AGENT_AUTHENTICATION_TOKEN="$AGENT_AUTHENTICATION_TOKEN" \
	-e AGENT_INSTANCE_ID="$AGENT_INSTANCE_ID" \
	-e AGENT_DISPLAY_NAME="$AGENT_DISPLAY_NAME" \
	-e LUNAFOX_AGENT_CONTAINER_NAME="lunafox-agent" \
	-e LUNAFOX_SHARED_DATA_VOLUME_BIND="$SHARED_DATA_VOLUME_BIND" \
	-e LUNAFOX_ENGINE_EXECUTION_VOLUME="$ENGINE_EXECUTION_VOLUME" \
	-e LUNAFOX_ENGINE_EXECUTION_ROOT="$ENGINE_EXECUTION_ROOT" \
	-e AGENT_VERSION="$AGENT_VERSION" \
	-e AGENT_HOSTNAME="$HOSTNAME" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$SHARED_DATA_VOLUME_BIND" \
	-v "${ENGINE_EXECUTION_VOLUME}:${ENGINE_EXECUTION_ROOT}" \
	-v "${STATE_VOLUME}:/var/lib/lunafox-agent" \
	"$AGENT_IMAGE_REF" >/dev/null

echo "Agent installed and running (container: lunafox-agent)"
