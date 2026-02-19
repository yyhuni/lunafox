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
COMPOSE_DEV="$DOCKER_DIR/docker-compose.dev.yml"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/docker-common.sh"
normalize_exit_codes

usage() {
	cat <<'USAGE'
用法:
  ./scripts/dev/restart.sh

说明:
  重启开发模式服务。
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

	if [ ! -f "$ENV_FILE" ]; then
		error "未找到 docker/.env，请先运行 ./scripts/dev/install.sh 或 ./install.sh。"
		exit 1
	fi

	if [ ! -f "$COMPOSE_DEV" ]; then
		error "未找到 compose 文件: $COMPOSE_DEV"
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
	run_compose --env-file "$ENV_FILE" -f "$COMPOSE_DEV" restart
	restart_local_agent
	success "开发服务已重启"
}

prewarm_frontend() {
	if ! command -v curl >/dev/null 2>&1; then
		info "未检测到 curl，跳过前端预热"
		return
	fi

	local public_port
	public_port="$(resolve_public_port_from_env "$ENV_FILE")"
	local base_url="https://localhost:${public_port}"
	local paths=("/zh/login" "/zh/dashboard/")
	local max_attempts=30
	local interval=2
	local timeout=6

	info "预热前端页面（开发模式，端口 ${public_port}）..."
	for path in "${paths[@]}"; do
		local url="${base_url}${path}"
		local warmed=0
		for _ in $(seq 1 "$max_attempts"); do
			local status
			status="$(curl -ksS -o /dev/null -w "%{http_code}" --max-time "$timeout" "$url" || echo "000")"
			if [ "$status" -ge 200 ] && [ "$status" -lt 500 ]; then
				success "预热完成: ${path}"
				warmed=1
				break
			fi
			sleep "$interval"
		done
		if [ "$warmed" -eq 0 ]; then
			warn "预热未完成: ${path}（可稍后访问触发编译）"
		fi
	done
}

main() {
	parse_args "$@"
	check_system
	restart_services
	prewarm_frontend
}

main "$@"
