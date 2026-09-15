#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/docker/bootstrap/config-init.sh"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/lunafox-config-init.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
	printf 'config-init self-test failed: %s\n' "$*" >&2
	exit 1
}

mode_of() {
	stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

run_init() {
	local config_dir="$1" db_input="$2" jwt_input="$3"
	mkdir -p "$config_dir"
	LUNAFOX_CONFIG_DIR="$config_dir" \
		DB_PASSWORD_INPUT_FILE="$db_input" \
		JWT_SECRET_INPUT_FILE="$jwt_input" \
		bash "$SCRIPT"
}

EMPTY_DB="$TMP_DIR/empty-db"
EMPTY_JWT="$TMP_DIR/empty-jwt"
: >"$EMPTY_DB"
: >"$EMPTY_JWT"
GENERATED_DIR="$TMP_DIR/generated"
output="$(run_init "$GENERATED_DIR" "$EMPTY_DB" "$EMPTY_JWT")"
[ "$output" = "LunaFox configuration secrets are ready." ] || fail "unexpected success output"
db_generated="$(cat "$GENERATED_DIR/db-password")"
jwt_generated="$(cat "$GENERATED_DIR/jwt-secret")"
[[ "$db_generated" =~ ^[a-f0-9]{48}$ ]] || fail "database password was not generated as 48 lowercase hex characters"
[[ "$jwt_generated" =~ ^[a-f0-9]{64}$ ]] || fail "JWT secret was not generated as 64 lowercase hex characters"
[ "$(mode_of "$GENERATED_DIR")" = 700 ] || fail "configuration directory mode is not 0700"
[ "$(mode_of "$GENERATED_DIR/db-password")" = 600 ] || fail "database password mode is not 0600"
[ "$(mode_of "$GENERATED_DIR/jwt-secret")" = 600 ] || fail "JWT secret mode is not 0600"

run_init "$GENERATED_DIR" "$EMPTY_DB" "$EMPTY_JWT" >/dev/null
[ "$(cat "$GENERATED_DIR/db-password")" = "$db_generated" ] || fail "database password changed on repeated initialization"
[ "$(cat "$GENERATED_DIR/jwt-secret")" = "$jwt_generated" ] || fail "JWT secret changed on repeated initialization"

CUSTOM_DB="$TMP_DIR/custom-db"
CUSTOM_JWT="$TMP_DIR/custom-jwt"
printf '%s' 'operator database $ecret #1' >"$CUSTOM_DB"
printf '%s' 'existing-jwt-secret-for-upgrade' >"$CUSTOM_JWT"
CUSTOM_DIR="$TMP_DIR/custom"
run_init "$CUSTOM_DIR" "$CUSTOM_DB" "$CUSTOM_JWT" >/dev/null
cmp -s "$CUSTOM_DB" "$CUSTOM_DIR/db-password" || fail "operator database password was not preserved"
cmp -s "$CUSTOM_JWT" "$CUSTOM_DIR/jwt-secret" || fail "existing JWT secret was not adopted"

printf '%s' 'different-database-password' >"$TMP_DIR/conflicting-db"
if output="$(run_init "$CUSTOM_DIR" "$TMP_DIR/conflicting-db" "$CUSTOM_JWT" 2>&1)"; then
	fail "conflicting database password was accepted"
fi
case "$output" in
*"input conflicts with the persisted value"*) ;;
*) fail "conflicting database password did not produce an actionable error" ;;
esac
case "$output" in
*'operator database'* | *'different-database-password'*) fail "failure output exposed a secret" ;;
esac
cmp -s "$CUSTOM_DB" "$CUSTOM_DIR/db-password" || fail "conflict changed the persisted database password"

chmod 0644 "$CUSTOM_DIR/jwt-secret"
if run_init "$CUSTOM_DIR" "$CUSTOM_DB" "$CUSTOM_JWT" >/dev/null 2>&1; then
	fail "permissive persisted secret was accepted"
fi
chmod 0600 "$CUSTOM_DIR/jwt-secret"
rm -f "$CUSTOM_DIR/jwt-secret"
ln -s "$CUSTOM_JWT" "$CUSTOM_DIR/jwt-secret"
if run_init "$CUSTOM_DIR" "$CUSTOM_DB" "$CUSTOM_JWT" >/dev/null 2>&1; then
	fail "symlinked persisted secret was accepted"
fi

printf 'config-init self-test passed\n'
