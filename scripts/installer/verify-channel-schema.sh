#!/usr/bin/env bash
set -euo pipefail

EXPECTED_SCHEMA_VERSION="${EXPECTED_SCHEMA_VERSION:-3}"

error() {
	echo "✗ $*" >&2
}

read_env_value() {
	local file="$1"
	local key="$2"
	awk -F= -v key="$key" '$1 == key { print substr($0, index($0, "=") + 1); exit }' "$file" | tr -d '\r'
}

require_key() {
	local file="$1"
	local key="$2"
	local value
	value="$(read_env_value "$file" "$key")"
	if [ -z "$value" ]; then
		error "[$file] missing required key: $key"
		exit 1
	fi
}

require_release_manifest() {
	local file="$1"
	local key="$2"
	local value

	value="$(read_env_value "$file" "$key")"
	if [ -z "$value" ]; then
		error "[$file] missing required key: $key"
		exit 1
	fi

	if ! printf '%s' "$value" | grep -Eq '^manifests/[A-Za-z0-9._-]+\.yaml$'; then
		error "[$file] invalid $key: $value (expected manifests/<name>.yaml)"
		exit 1
	fi
}

forbid_key() {
	local file="$1"
	local key="$2"
	# Hard-cut guard: reject legacy keys in schema v3.
	if grep -Eq "^${key}=" "$file"; then
		error "[$file] forbidden legacy key: $key"
		exit 1
	fi
}

validate_file() {
	local file="$1"
	if [ ! -f "$file" ]; then
		error "file not found: $file"
		exit 1
	fi

	require_key "$file" "SCHEMA_VERSION"
	local schema
	schema="$(read_env_value "$file" "SCHEMA_VERSION")"
	if [ "$schema" != "$EXPECTED_SCHEMA_VERSION" ]; then
		error "[$file] incompatible SCHEMA_VERSION: $schema (expected: $EXPECTED_SCHEMA_VERSION)"
		exit 1
	fi

	require_key "$file" "VERSION"
	require_key "$file" "GITHUB_BASE_URL"
	require_key "$file" "GITEE_BASE_URL"

	local platforms=(
		"LINUX_AMD64"
		"LINUX_ARM64"
		"DARWIN_AMD64"
		"DARWIN_ARM64"
	)
	local platform=""
	for platform in "${platforms[@]}"; do
		require_key "$file" "${platform}_ASSET"
		require_key "$file" "${platform}_SHA256"
	done

	require_release_manifest "$file" "RELEASE_MANIFEST"
	local version release_manifest expected_release_manifest
	version="$(read_env_value "$file" "VERSION")"
	release_manifest="$(read_env_value "$file" "RELEASE_MANIFEST")"
	expected_release_manifest="manifests/${version}.yaml"
	if [ "$release_manifest" != "$expected_release_manifest" ]; then
		error "[$file] RELEASE_MANIFEST mismatch: expected ${expected_release_manifest}, got ${release_manifest}"
		exit 1
	fi

	forbid_key "$file" "IMAGE_REGISTRY"
	forbid_key "$file" "IMAGE_NAMESPACE"
	forbid_key "$file" "AGENT_IMAGE_REFS"
	forbid_key "$file" "WORKER_IMAGE_REFS"
	forbid_key "$file" "AGENT_IMAGE_REF"
	forbid_key "$file" "WORKER_IMAGE_REF"
}

if [ "$#" -lt 1 ]; then
	error "usage: $0 <channel-env> [channel-env...]"
	exit 1
fi

for file in "$@"; do
	validate_file "$file"
done

echo "✓ channel schema validated (${EXPECTED_SCHEMA_VERSION})"
