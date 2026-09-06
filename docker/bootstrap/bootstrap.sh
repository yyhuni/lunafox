#!/usr/bin/env bash
set -euo pipefail

# This public-source bootstrap entrypoint is the only actor that creates the
# first internal Agent. Host lifecycle scripts receive only its exit status
# and logs, never the runtime-created Agent bearer credential.

die() {
	printf 'LunaFox bootstrap error: %s\n' "$*" >&2
	exit 1
}

require_env() {
	local key="$1" value="${!1:-}"
	[ -n "$value" ] || die "$key is required"
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

require_env BOOTSTRAP_NETWORK
require_env AGENT_IMAGE_REF
require_env AGENT_VERSION
require_env AGENT_CONTROL_URL
require_env LOKI_PUSH_URL
require_env ENGINE_INSTALL_INVENTORY_PATH
require_env ENGINE_INSTALL_REGISTRY
require_env FINGERPRINT_BOOTSTRAP_PATH
require_env WORDLISTS_SOURCE_PATH
require_env LUNAFOX_AGENT_STATE_VOLUME
require_env LUNAFOX_ENGINE_EXECUTION_VOLUME
require_env LUNAFOX_ENGINE_EXECUTION_ROOT
require_env LUNAFOX_SHARED_DATA_VOLUME_BIND
require_command docker
require_command jq

case "$AGENT_IMAGE_REF" in
docker.io/yyhuni/lunafox-agent@sha256:[a-f0-9][a-f0-9]* | ghcr.io/yyhuni/lunafox-agent@sha256:[a-f0-9][a-f0-9]*) ;;
*) die "AGENT_IMAGE_REF must be a yyhuni digest-qualified Agent image" ;;
esac
case "$AGENT_VERSION" in
*[!0-9A-Za-z.+-]* | "") die "AGENT_VERSION must be canonical semantic version text" ;;
esac
case "$LUNAFOX_AGENT_STATE_VOLUME" in
lunafox_agent_state) ;;
*) die "LUNAFOX_AGENT_STATE_VOLUME must be lunafox_agent_state" ;;
esac
case "$LUNAFOX_ENGINE_EXECUTION_VOLUME" in
lunafox_engine_execution) ;;
*) die "LUNAFOX_ENGINE_EXECUTION_VOLUME must be lunafox_engine_execution" ;;
esac
case "$LUNAFOX_ENGINE_EXECUTION_ROOT" in
/var/lib/lunafox/engine-execution) ;;
*) die "LUNAFOX_ENGINE_EXECUTION_ROOT must be /var/lib/lunafox/engine-execution" ;;
esac
case "$LUNAFOX_SHARED_DATA_VOLUME_BIND" in
lunafox_data:/opt/lunafox | lunafox_data:/opt/lunafox:rw) ;;
*) die "LUNAFOX_SHARED_DATA_VOLUME_BIND must use lunafox_data:/opt/lunafox" ;;
esac

agent_name="lunafox-agent"
credentials_dir="$(mktemp -d /run/lunafox-bootstrap.XXXXXX)"
credentials_path="$credentials_dir/agent.json"
trap 'rm -rf -- "$credentials_dir"' EXIT

agent_hostname="${BOOTSTRAP_AGENT_HOSTNAME:-lunafox-agent}"

docker info >/dev/null 2>&1 || die "cannot access Docker through the mounted socket"
docker network inspect "$BOOTSTRAP_NETWORK" >/dev/null || die "bootstrap network is unavailable: $BOOTSTRAP_NETWORK"
for volume in lunafox_data "$LUNAFOX_ENGINE_EXECUTION_VOLUME" "$LUNAFOX_AGENT_STATE_VOLUME"; do
	docker volume inspect "$volume" >/dev/null || die "required Docker volume is unavailable: $volume"
done
if docker inspect "$agent_name" >/dev/null 2>&1; then
	die "bootstrap Agent container already exists; reset is required before a fresh install"
fi

server engine-bootstrap
server fingerprint-bootstrap "$FINGERPRINT_BOOTSTRAP_PATH"
server wordlist-bootstrap "$WORDLISTS_SOURCE_PATH/manifest.json" "$WORDLISTS_SOURCE_PATH"

run_id="bootstrap-$(date +%s)-$$"
docker run --rm --pull=never --network none \
	--name "lunafox-agent-mount-preflight-$run_id" \
	--entrypoint /usr/local/bin/lunafox-engine-mount-preflight \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$LUNAFOX_ENGINE_EXECUTION_VOLUME:$LUNAFOX_ENGINE_EXECUTION_ROOT" \
	-v "lunafox_data:/opt/lunafox" \
	"$AGENT_IMAGE_REF" \
	--profile capability \
	--image-ref "$AGENT_IMAGE_REF" \
	--execution-volume "$LUNAFOX_ENGINE_EXECUTION_VOLUME" \
	--execution-root "$LUNAFOX_ENGINE_EXECUTION_ROOT" \
	--shared-volume lunafox_data \
	--shared-root /opt/lunafox \
	--run-id "$run_id"

server agent-bootstrap "$credentials_path" "$agent_hostname" "$AGENT_VERSION"
agent_instance_id="$(jq -er '.instanceId | strings | select(length > 0)' "$credentials_path")" || die "bootstrap Agent credentials have no instanceId"
agent_display_name="$(jq -er '.displayName | strings | select(length > 0)' "$credentials_path")" || die "bootstrap Agent credentials have no displayName"
agent_authentication_token="$(jq -er '.agentAuthenticationToken | strings | select(test("^[a-f0-9]{8}$"))' "$credentials_path")" || die "bootstrap Agent credentials have an invalid bearer token"

docker pull "$AGENT_IMAGE_REF" >/dev/null
docker run -d --restart unless-stopped --name "$agent_name" \
	--network "$BOOTSTRAP_NETWORK" \
	--label com.docker.compose.project=lunafox \
	--label com.docker.compose.service=agent \
	--label org.opencontainers.image.title=lunafox-agent \
	--hostname "$agent_hostname" \
	--log-driver=loki \
	--log-opt "loki-url=$LOKI_PUSH_URL" \
	--log-opt "loki-tls-insecure-skip-verify=true" \
	--log-opt "no-file=false" \
	--log-opt "mode=non-blocking" \
	--log-opt "loki-batch-size=1048576" \
	--log-opt "max-buffer-size=5m" \
	--log-opt "loki-retries=3" \
	--log-opt "loki-external-labels=component=agent,container_name=lunafox-agent,compose_service=agent" \
	-e "AGENT_CONTROL_URL=$AGENT_CONTROL_URL" \
	-e "AGENT_AUTHENTICATION_TOKEN=$agent_authentication_token" \
	-e "AGENT_INSTANCE_ID=$agent_instance_id" \
	-e "AGENT_DISPLAY_NAME=$agent_display_name" \
	-e "AGENT_VERSION=$AGENT_VERSION" \
	-e "AGENT_HOSTNAME=$agent_hostname" \
	-e "LUNAFOX_AGENT_CONTAINER_NAME=$agent_name" \
	-e "LUNAFOX_SHARED_DATA_VOLUME_BIND=lunafox_data:/opt/lunafox:rw" \
	-e "LUNAFOX_ENGINE_EXECUTION_VOLUME=$LUNAFOX_ENGINE_EXECUTION_VOLUME" \
	-e "LUNAFOX_ENGINE_EXECUTION_ROOT=$LUNAFOX_ENGINE_EXECUTION_ROOT" \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v lunafox_data:/opt/lunafox \
	-v "$LUNAFOX_ENGINE_EXECUTION_VOLUME:$LUNAFOX_ENGINE_EXECUTION_ROOT" \
	-v "$LUNAFOX_AGENT_STATE_VOLUME:/var/lib/lunafox-agent" \
	"$AGENT_IMAGE_REF" >/dev/null

for _ in $(seq 1 15); do
	[ "$(docker inspect --format '{{.State.Running}}' "$agent_name" 2>/dev/null || true)" = true ] && exit 0
	sleep 1
done
docker logs "$agent_name" >&2 || true
die "bootstrap Agent container did not remain running"
