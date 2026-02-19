#!/usr/bin/env bash
set -euo pipefail

EXPECTED_SCHEMA_VERSION="${EXPECTED_SCHEMA_VERSION:-2}"

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

require_digest_ref_list() {
	local file="$1"
	local key="$2"
	local value
	local -a parts=()
	local part=""
	local trimmed=""
	local found=0

	value="$(read_env_value "$file" "$key")"
	if [ -z "$value" ]; then
		error "[$file] missing required key: $key"
		exit 1
	fi

	IFS=',' read -r -a parts <<<"$value"
	for part in "${parts[@]}"; do
		trimmed="$(printf '%s' "$part" | xargs)"
		if [ -z "$trimmed" ]; then
			continue
		fi
		if ! printf '%s' "$trimmed" | grep -Eq '^.+@sha256:[a-f0-9]{64}$'; then
			error "[$file] invalid digest image ref in $key: $trimmed"
			exit 1
		fi
		found=1
	done

	if [ "$found" -ne 1 ]; then
		error "[$file] invalid digest image ref list for $key: $value"
		exit 1
	fi
}

forbid_key() {
	local file="$1"
	local key="$2"
	# Hard-cut guard: reject legacy single-value keys in schema v2.
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

	require_key "$file" "IMAGE_REGISTRY"
	require_key "$file" "IMAGE_NAMESPACE"
	require_digest_ref_list "$file" "AGENT_IMAGE_REFS"
	require_digest_ref_list "$file" "WORKER_IMAGE_REFS"
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
