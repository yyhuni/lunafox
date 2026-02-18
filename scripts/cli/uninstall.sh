#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
  SCRIPT_PATH="$0"
  case "$SCRIPT_PATH" in
    /*|*/*) ;;
    *) SCRIPT_PATH="./$SCRIPT_PATH" ;;
  esac
  exec /usr/bin/env bash "$SCRIPT_PATH" "$@"
fi
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCKER_DIR="$ROOT_DIR/docker"
ENV_FILE="$DOCKER_DIR/.env"
COMPOSE_DEV="$DOCKER_DIR/docker-compose.dev.yml"
COMPOSE_PROD="$DOCKER_DIR/docker-compose.yml"

# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/ui-common.sh"
# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/lib/docker-common.sh"
normalize_exit_codes

PURGE_ALL=1
RUNTIME_ENV_FILE="$ENV_FILE"
TEMP_ENV_FILE=""

cleanup_temp_env() {
  if [ -n "$TEMP_ENV_FILE" ] && [ -f "$TEMP_ENV_FILE" ]; then
    rm -f "$TEMP_ENV_FILE"
    info "已清理临时卸载配置: $TEMP_ENV_FILE"
  fi
}

trap cleanup_temp_env EXIT

usage() {
  cat <<'USAGE'
用法:
  ./uninstall.sh             # 完全删除（默认）
  ./uninstall.sh --keep-data # 仅卸载服务，保留数据

说明:
  默认行为是完全删除（容器、网络、数据卷、证书、.env）。
  如需保留数据，请显式使用 --keep-data。
USAGE
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --keep-data)
        PURGE_ALL=0
        ;;
      -h|--help)
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

remove_local_agent_containers() {
  local agent_containers=()
  while IFS= read -r container_name; do
    [ -n "$container_name" ] || continue
    agent_containers+=("$container_name")
  done < <(run_docker ps -a --format "{{.Names}}" | grep -E '^lunafox-agent($|-)' || true)

  if [ "${#agent_containers[@]}" -eq 0 ]; then
    return
  fi

  info "移除本地 Agent 容器: ${agent_containers[*]}"
  run_docker rm -f "${agent_containers[@]}" >/dev/null
}

uninstall_services() {
  info "正在卸载服务（容器与网络）..."

  if [ ! -f "$ENV_FILE" ]; then
    TEMP_ENV_FILE="$(mktemp "${TMPDIR:-/tmp}/lunafox-uninstall-env.XXXXXX")"
    cat >"$TEMP_ENV_FILE" <<'EOF'
IMAGE_TAG=cleanup
JWT_SECRET=cleanup
WORKER_TOKEN=cleanup
DB_HOST=127.0.0.1
DB_PASSWORD=cleanup
REDIS_HOST=127.0.0.1
DB_NAME=lunafox
DB_USER=postgres
DB_PORT=5432
PUBLIC_PORT=8083
EOF
    RUNTIME_ENV_FILE="$TEMP_ENV_FILE"
    warn "未找到 docker/.env，已生成临时卸载配置"
  else
    RUNTIME_ENV_FILE="$ENV_FILE"
  fi

  if [ -f "$COMPOSE_PROD" ]; then
    run_compose --env-file "$RUNTIME_ENV_FILE" -f "$COMPOSE_PROD" --profile local-db down --remove-orphans
  fi
  if [ -f "$COMPOSE_DEV" ]; then
    run_compose --env-file "$RUNTIME_ENV_FILE" -f "$COMPOSE_DEV" down --remove-orphans
  fi

  remove_local_agent_containers
  success "服务已卸载"
}

purge_everything() {
  info "开始完全删除（可能耗时数秒）..."

  local volumes=(
    "lunafox_data"
    "lunafox_ssl"
    "postgres_data"
    "go-mod-cache"
    "go-build-cache"
    "frontend_node_modules"
    "frontend_pnpm_store"
  )

  for volume in "${volumes[@]}"; do
    if run_docker volume rm "$volume" >/dev/null 2>&1; then
      info "已删除卷: $volume"
    fi
  done

  if run_docker network rm lunafox_network >/dev/null 2>&1; then
    info "已删除网络: lunafox_network"
  fi

  if [ -f "$ENV_FILE" ]; then
    rm -f "$ENV_FILE"
    info "已删除文件: $ENV_FILE"
  fi

  if [ -f "$DOCKER_DIR/nginx/ssl/fullchain.pem" ]; then
    rm -f "$DOCKER_DIR/nginx/ssl/fullchain.pem"
    info "已删除证书: docker/nginx/ssl/fullchain.pem"
  fi
  if [ -f "$DOCKER_DIR/nginx/ssl/privkey.pem" ]; then
    rm -f "$DOCKER_DIR/nginx/ssl/privkey.pem"
    info "已删除证书: docker/nginx/ssl/privkey.pem"
  fi

  success "完全删除完成"
}

main() {
  parse_args "$@"
  check_system
  uninstall_services

  if [ "$PURGE_ALL" -eq 1 ]; then
    purge_everything
  else
    warn "已按 --keep-data 保留数据卷与配置"
  fi

  success "卸载流程完成"
}

main "$@"
