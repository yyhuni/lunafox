#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MANIFEST_FILE="${1:-$ROOT_DIR/release.manifest.yaml}"
ENV_FILE="${2:-$ROOT_DIR/docker/.env}"

fail() {
	echo "✗ $*" >&2
	exit 1
}

read_yaml_value() {
	local file="$1"
	local key="$2"
	awk -v key="$key" '
		BEGIN { FS=":" }
		$0 ~ "^[[:space:]]*"key"[[:space:]]*:" {
			value=substr($0, index($0, ":")+1)
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
			gsub(/^["'"'"']|["'"'"']$/, "", value)
			print value
			exit
		}
	' "$file"
}

read_env_value() {
	local file="$1"
	local key="$2"
	awk -F= -v key="$key" '$1 == key { print substr($0, index($0, "=")+1); exit }' "$file" | tr -d '\r'
}

require_non_empty() {
	local value="$1"
	local name="$2"
	if [ -z "$value" ]; then
		fail "缺少必填字段: $name"
	fi
}

normalize_version() {
	local value="$1"
	value="$(echo "$value" | xargs)"
	value="${value#v}"
	value="${value#V}"
	echo "$value"
}

validate_semver() {
	local value="$1"
	local name="$2"
	if ! echo "$value" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+([+-][-0-9A-Za-z.+_]+)?$'; then
		fail "$name 格式非法: $value"
	fi
}

validate_digest_ref() {
	local value="$1"
	local name="$2"
	if ! echo "$value" | grep -Eq '^.+@sha256:[a-f0-9]{64}$'; then
		fail "$name 必须是 digest 引用（@sha256:...）: $value"
	fi
}

if [ ! -f "$MANIFEST_FILE" ]; then
	fail "未找到 release manifest: $MANIFEST_FILE"
fi

release_version="$(read_yaml_value "$MANIFEST_FILE" "releaseVersion")"
agent_version="$(read_yaml_value "$MANIFEST_FILE" "agentVersion")"
worker_version="$(read_yaml_value "$MANIFEST_FILE" "workerVersion")"
agent_image_ref="$(read_yaml_value "$MANIFEST_FILE" "agentImageRef")"
worker_image_ref="$(read_yaml_value "$MANIFEST_FILE" "workerImageRef")"

require_non_empty "$release_version" "releaseVersion"
require_non_empty "$agent_version" "agentVersion"
require_non_empty "$worker_version" "workerVersion"
require_non_empty "$agent_image_ref" "agentImageRef"
require_non_empty "$worker_image_ref" "workerImageRef"

release_version="$(normalize_version "$release_version")"
agent_version="$(normalize_version "$agent_version")"
worker_version="$(normalize_version "$worker_version")"

validate_semver "$release_version" "releaseVersion"
validate_semver "$agent_version" "agentVersion"
validate_semver "$worker_version" "workerVersion"

if [ "$release_version" != "$agent_version" ] || [ "$agent_version" != "$worker_version" ]; then
	fail "版本不一致: releaseVersion=$release_version agentVersion=$agent_version workerVersion=$worker_version"
fi

validate_digest_ref "$agent_image_ref" "agentImageRef"
validate_digest_ref "$worker_image_ref" "workerImageRef"

if [ -f "$ENV_FILE" ]; then
	env_release_version="$(normalize_version "$(read_env_value "$ENV_FILE" "RELEASE_VERSION")")"
	env_agent_version="$(normalize_version "$(read_env_value "$ENV_FILE" "AGENT_VERSION")")"
	env_worker_version="$(normalize_version "$(read_env_value "$ENV_FILE" "WORKER_VERSION")")"
	env_agent_image_ref="$(read_env_value "$ENV_FILE" "AGENT_IMAGE_REF")"
	env_worker_image_ref="$(read_env_value "$ENV_FILE" "WORKER_IMAGE_REF")"

	require_non_empty "$env_release_version" "docker/.env: RELEASE_VERSION"
	require_non_empty "$env_agent_version" "docker/.env: AGENT_VERSION"
	require_non_empty "$env_worker_version" "docker/.env: WORKER_VERSION"
	require_non_empty "$env_agent_image_ref" "docker/.env: AGENT_IMAGE_REF"
	require_non_empty "$env_worker_image_ref" "docker/.env: WORKER_IMAGE_REF"

	if [ "$release_version" != "$env_release_version" ]; then
		fail "manifest 与 docker/.env RELEASE_VERSION 不一致: $release_version != $env_release_version"
	fi
	if [ "$agent_version" != "$env_agent_version" ]; then
		fail "manifest 与 docker/.env AGENT_VERSION 不一致: $agent_version != $env_agent_version"
	fi
	if [ "$worker_version" != "$env_worker_version" ]; then
		fail "manifest 与 docker/.env WORKER_VERSION 不一致: $worker_version != $env_worker_version"
	fi
	if [ "$agent_image_ref" != "$env_agent_image_ref" ]; then
		fail "manifest 与 docker/.env AGENT_IMAGE_REF 不一致"
	fi
	if [ "$worker_image_ref" != "$env_worker_image_ref" ]; then
		fail "manifest 与 docker/.env WORKER_IMAGE_REF 不一致"
	fi
fi

echo "✓ release contract 验证通过"
