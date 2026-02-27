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
COMPOSE_DEV="$DOCKER_DIR/docker-compose.dev.yml"
COMPOSE_PROD="$DOCKER_DIR/docker-compose.yml"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/docker-common.sh"
normalize_exit_codes

ENV_FILE="$DOCKER_DIR/.env"
RUNTIME_ENV_FILE="$ENV_FILE"
TEMP_ENV_FILE=""

cleanup_temp_env() {
	if [ -n "$TEMP_ENV_FILE" ] && [ -f "$TEMP_ENV_FILE" ]; then
		rm -f "$TEMP_ENV_FILE"
		info "已清理临时停止配置: $TEMP_ENV_FILE"
	fi
}

trap cleanup_temp_env EXIT

usage() {
	cat <<'USAGE'
用法:
  ./stop.sh

说明:
  同时停止 LunaFox 的 prod/dev 服务。
USAGE
}

parse_args() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
		-h | --help)
			usage
			exit 0
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

	if [ ! -f "$COMPOSE_PROD" ] && [ ! -f "$COMPOSE_DEV" ]; then
		error "未找到可用 compose 文件（docker-compose.yml / docker-compose.dev.yml）"
		exit 1
	fi

	info "使用 compose 命令: ${COMPOSE_CMD[*]}"
	success "环境校验通过"
}

stop_local_agent() {
	local running_agents=()
	while IFS= read -r container_name; do
		[ -n "$container_name" ] || continue
		running_agents+=("$container_name")
	done < <(run_docker ps --format "{{.Names}}" | grep -E '^lunafox-agent($|-)' || true)

	if [ "${#running_agents[@]}" -eq 0 ]; then
		return
	fi

	info "检测到本地 Agent 容器，正在停止: ${running_agents[*]}"
	run_docker stop "${running_agents[@]}" >/dev/null
	success "本地 Agent 已停止"
}

stop_services() {
	if [ ! -f "$ENV_FILE" ]; then
		error "未找到 docker/.env，无法解析 PUBLIC_PORT。请先恢复配置文件后再执行停止。"
		exit 1
	fi
	RUNTIME_ENV_FILE="$ENV_FILE"

	if [ -f "$COMPOSE_PROD" ]; then
		run_compose --env-file "$RUNTIME_ENV_FILE" -f "$COMPOSE_PROD" --profile local-db down --remove-orphans
	fi
	if [ -f "$COMPOSE_DEV" ]; then
		run_compose --env-file "$RUNTIME_ENV_FILE" -f "$COMPOSE_DEV" down --remove-orphans
	fi

	stop_local_agent

	success "服务已停止"
}

main() {
	parse_args "$@"
	check_system
	stop_services
}

main "$@"
