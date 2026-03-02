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
DOCKER_DIR="$ROOT_DIR/docker"
ENV_FILE="$DOCKER_DIR/.env"
COMPOSE_PROD="$DOCKER_DIR/docker-compose.yml"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/docker-common.sh"
normalize_exit_codes

PROFILE_ARGS=(--profile local-db)

usage() {
	cat <<'USAGE'
用法:
  ./restart.sh

说明:
  仅重启生产模式服务。
  开发模式请使用 ./scripts/dev/restart.sh
USAGE
}

parse_args() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
		-h | --help)
			usage
			exit 0
			;;
		--dev)
			usage_error "请使用 ./scripts/dev/restart.sh"
			;;
		*)
			usage_error "不支持的参数: $1"
			;;
		esac
		shift
	done
}

check_system() {
	info "系统环境校验..."
	ensure_docker
	detect_compose

	if [ ! -f "$ENV_FILE" ]; then
		error "未找到 docker/.env，请先运行 ./install.sh。"
		exit 1
	fi

	for required in IMAGE_TAG RELEASE_VERSION AGENT_VERSION WORKER_VERSION AGENT_IMAGE_REF WORKER_IMAGE_REF LUNAFOX_SHARED_DATA_VOLUME_BIND JWT_SECRET WORKER_TOKEN DB_HOST DB_PASSWORD REDIS_HOST; do
		if ! grep -q "^${required}=" "$ENV_FILE"; then
			error "docker/.env 缺少 ${required}"
			exit 1
		fi
	done

	if [ ! -f "$COMPOSE_PROD" ]; then
		error "未找到 compose 文件: $COMPOSE_PROD"
		exit 1
	fi

	info "使用 compose 命令: ${COMPOSE_CMD[*]}"
	success "环境校验通过"
}

restart_local_agent() {
	local agent_containers=()
	while IFS= read -r container_name; do
		[ -n "$container_name" ] || continue
		agent_containers+=("$container_name")
	done < <(run_docker ps -a --format "{{.Names}}" | grep -E '^lunafox-agent($|-)' || true)

	if [ "${#agent_containers[@]}" -eq 0 ]; then
		return
	fi

	info "检测到本地 Agent 容器，正在重启: ${agent_containers[*]}"
	run_docker restart "${agent_containers[@]}" >/dev/null
	success "本地 Agent 已重启"
}

restart_services() {
	run_compose --env-file "$ENV_FILE" -f "$COMPOSE_PROD" "${PROFILE_ARGS[@]}" restart
	restart_local_agent
	success "生产服务已重启"
}

main() {
	parse_args "$@"
	check_system
	restart_services
}

main "$@"
