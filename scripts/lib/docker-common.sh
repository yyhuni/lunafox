#!/usr/bin/env bash

compose_plugin_path() {
	local paths=()
	local user_home=""
	if [ -n "${SUDO_USER:-}" ]; then
		user_home="$(eval echo "~$SUDO_USER")"
	fi
	if [ -d "$HOME/.docker/cli-plugins" ]; then
		paths+=("$HOME/.docker/cli-plugins")
	fi
	if [ -n "$user_home" ] && [ -d "$user_home/.docker/cli-plugins" ]; then
		paths+=("$user_home/.docker/cli-plugins")
	fi
	for p in /usr/local/lib/docker/cli-plugins /usr/libexec/docker/cli-plugins /usr/lib/docker/cli-plugins; do
		if [ -d "$p" ]; then
			paths+=("$p")
		fi
	done
	local joined=""
	for p in "${paths[@]}"; do
		joined="${joined:+$joined:}$p"
	done
	echo "$joined"
}

ensure_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		error "未检测到 docker 命令，请先安装 Docker。"
		exit 1
	fi
	DOCKER_BIN="$(command -v docker)"
	DOCKER_PREFIX=()
	if "$DOCKER_BIN" info >/dev/null 2>&1; then
		return 0
	fi
	if command -v sudo >/dev/null 2>&1; then
		if sudo "$DOCKER_BIN" info >/dev/null 2>&1; then
			DOCKER_PREFIX=(sudo)
			return 0
		fi
		error "当前环境无法使用 sudo 访问 Docker（可能被禁用或需要权限）。"
	else
		error "未检测到 sudo，无法提升权限访问 Docker。"
	fi
	error "Docker 守护进程未运行或无权限访问。请确认 Docker 已启动且当前用户有权限访问 Docker socket。"
	exit 1
}

detect_compose() {
	if [ -z "${DOCKER_BIN:-}" ]; then
		DOCKER_BIN="$(command -v docker || true)"
	fi
	if [ -z "${DOCKER_BIN:-}" ]; then
		error "未检测到 docker 可执行文件"
		exit 1
	fi
	local plugin_path
	plugin_path="$(compose_plugin_path)"
	if [ -n "$plugin_path" ]; then
		COMPOSE_ENV=(env DOCKER_CLI_PLUGIN_PATH="$plugin_path")
	else
		COMPOSE_ENV=()
	fi
	if command -v docker-compose >/dev/null 2>&1; then
		COMPOSE_CMD=()
		if [ -n "${DOCKER_PREFIX+x}" ] && [ "${#DOCKER_PREFIX[@]}" -gt 0 ]; then
			COMPOSE_CMD+=("${DOCKER_PREFIX[@]}")
		fi
		COMPOSE_CMD+=("$(command -v docker-compose)")
		return 0
	fi
	for p in ${plugin_path//:/ }; do
		if [ -x "$p/docker-compose" ]; then
			COMPOSE_CMD=()
			if [ -n "${DOCKER_PREFIX+x}" ] && [ "${#DOCKER_PREFIX[@]}" -gt 0 ]; then
				COMPOSE_CMD+=("${DOCKER_PREFIX[@]}")
			fi
			if [ "${#COMPOSE_ENV[@]}" -gt 0 ]; then
				COMPOSE_CMD+=("${COMPOSE_ENV[@]}")
			fi
			COMPOSE_CMD+=("$DOCKER_BIN" compose)
			return 0
		fi
	done
	error "未检测到 docker compose，请先安装。"
	exit 1
}

run_docker() {
	if [ -z "${DOCKER_BIN:-}" ]; then
		error "Docker 未初始化，请先执行 ensure_docker"
		exit 1
	fi
	if [ -n "${DOCKER_PREFIX+x}" ] && [ "${#DOCKER_PREFIX[@]}" -gt 0 ]; then
		"${DOCKER_PREFIX[@]}" "$DOCKER_BIN" "$@"
		return 0
	fi
	"$DOCKER_BIN" "$@"
}

run_compose() {
	if [ -z "${COMPOSE_CMD+x}" ] || [ "${#COMPOSE_CMD[@]}" -eq 0 ]; then
		error "Compose 未初始化，请先执行 detect_compose"
		exit 1
	fi
	"${COMPOSE_CMD[@]}" "$@"
}

read_env_value() {
	local env_file="$1"
	local key="$2"
	if [ -z "$env_file" ] || [ -z "$key" ] || [ ! -f "$env_file" ]; then
		return 0
	fi

	awk -F= -v target="$key" '
    $1 == target {
      value = substr($0, index($0, "=") + 1)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      gsub(/^["'\"'"'"']|["'\"'"'"']$/, "", value)
      print value
      exit
    }
  ' "$env_file"
}

is_valid_port() {
	local port="$1"
	if [[ ! "$port" =~ ^[0-9]+$ ]]; then
		return 1
	fi
	if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
		return 1
	fi
	return 0
}

resolve_public_port_from_env() {
	local env_file="$1"
	local port
	port="$(read_env_value "$env_file" "PUBLIC_PORT")"

	if ! is_valid_port "$port"; then
		error "无法解析 PUBLIC_PORT，请检查配置文件: $env_file"
		return 1
	fi
	echo "$port"
}
