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
	local database_mode="${4:-embedded}" db_host="${5:-}" db_port="${6:-5432}" db_sslmode="${7:-}"
	mkdir -p "$config_dir"
	LUNAFOX_CONFIG_DIR="$config_dir" \
		DB_PASSWORD_INPUT_FILE="$db_input" \
		JWT_SECRET_INPUT_FILE="$jwt_input" \
		DATABASE_MODE="$database_mode" \
		DB_HOST="$db_host" \
		DB_PORT="$db_port" \
		DB_SSLMODE="$db_sslmode" \
		bash "$SCRIPT"
}

expect_fresh_failure() {
	local label="$1" database_mode="$2" db_host="$3" db_port="$4" db_sslmode="$5" db_input="$6" expected="$7"
	local config_dir="$TMP_DIR/failure-$label" output
	if output="$(run_init "$config_dir" "$db_input" "$EMPTY_JWT" "$database_mode" "$db_host" "$db_port" "$db_sslmode" 2>&1)"; then
		fail "$label was accepted"
	fi
	case "$output" in
	*"$expected"*) ;;
	*) fail "$label did not produce the expected error: $output" ;;
	esac
	if find "$config_dir" -mindepth 1 -print -quit | grep -q .; then
		fail "$label wrote configuration state before validation completed"
	fi
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
[ "$(cat "$GENERATED_DIR/database-mode")" = embedded ] || fail "default database mode was not persisted as embedded"
[ "$(mode_of "$GENERATED_DIR/database-mode")" = 600 ] || fail "database mode is not 0600"

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
[ "$(cat "$CUSTOM_DIR/database-mode")" = embedded ] || fail "custom embedded initialization did not persist its mode"

expect_fresh_failure invalid-mode invalid '' 5432 '' "$CUSTOM_DB" "DATABASE_MODE must be embedded or external"
expect_fresh_failure external-host external '' 5432 require "$CUSTOM_DB" "DB_HOST is required"
expect_fresh_failure external-ssl external database.example 5432 '' "$CUSTOM_DB" "DB_SSLMODE is required"
expect_fresh_failure unknown-ssl external database.example 5432 trust "$CUSTOM_DB" "DB_SSLMODE must be"
expect_fresh_failure nonnumeric-port embedded '' port '' "$CUSTOM_DB" "DB_PORT must be an integer"
expect_fresh_failure zero-port embedded '' 0 '' "$CUSTOM_DB" "DB_PORT must be an integer"
expect_fresh_failure oversized-port external database.example 65536 require "$CUSTOM_DB" "DB_PORT must be an integer"
expect_fresh_failure external-password external database.example 5432 require "$EMPTY_DB" "DB_PASSWORD is required"

for sslmode in disable allow prefer require verify-ca verify-full; do
	SSL_DIR="$TMP_DIR/ssl-$sslmode"
	run_init "$SSL_DIR" "$CUSTOM_DB" "$EMPTY_JWT" external database.example 5432 "$sslmode" >/dev/null
	[ "$(cat "$SSL_DIR/database-mode")" = external ] || fail "external mode was not persisted for sslmode=$sslmode"
done

EXTERNAL_DIR="$TMP_DIR/external"
run_init "$EXTERNAL_DIR" "$CUSTOM_DB" "$EMPTY_JWT" external 2001:db8::1 6543 require >/dev/null
external_jwt="$(cat "$EXTERNAL_DIR/jwt-secret")"
[[ "$external_jwt" =~ ^[a-f0-9]{64}$ ]] || fail "external initialization changed automatic JWT generation"
run_init "$EXTERNAL_DIR" "$EMPTY_DB" "$EMPTY_JWT" external 2001:db8::1 6543 require >/dev/null
cmp -s "$CUSTOM_DB" "$EXTERNAL_DIR/db-password" || fail "repeated external initialization did not reuse its database password"
[ "$(cat "$EXTERNAL_DIR/jwt-secret")" = "$external_jwt" ] || fail "repeated external initialization changed its JWT secret"

if output="$(run_init "$GENERATED_DIR" "$EMPTY_DB" "$EMPTY_JWT" external database.example 5432 require 2>&1)"; then
	fail "database mode switch was accepted"
fi
case "$output" in
*"DATABASE_MODE conflicts with the persisted value"*) ;;
*) fail "database mode conflict did not produce an actionable error" ;;
esac
[ "$(cat "$GENERATED_DIR/database-mode")" = embedded ] || fail "database mode conflict changed the persisted mode"
[ "$(cat "$GENERATED_DIR/db-password")" = "$db_generated" ] || fail "database mode conflict changed the persisted password"

LEGACY_EMBEDDED_DIR="$TMP_DIR/legacy-embedded"
mkdir -p "$LEGACY_EMBEDDED_DIR"
cp "$CUSTOM_DB" "$LEGACY_EMBEDDED_DIR/db-password"
cp "$CUSTOM_JWT" "$LEGACY_EMBEDDED_DIR/jwt-secret"
chmod 0600 "$LEGACY_EMBEDDED_DIR/db-password" "$LEGACY_EMBEDDED_DIR/jwt-secret"
run_init "$LEGACY_EMBEDDED_DIR" "$EMPTY_DB" "$EMPTY_JWT" >/dev/null
[ "$(cat "$LEGACY_EMBEDDED_DIR/database-mode")" = embedded ] || fail "legacy deployment did not adopt embedded mode"
cmp -s "$CUSTOM_DB" "$LEGACY_EMBEDDED_DIR/db-password" || fail "legacy embedded adoption changed the database password"

LEGACY_EXTERNAL_DIR="$TMP_DIR/legacy-external"
mkdir -p "$LEGACY_EXTERNAL_DIR"
cp "$CUSTOM_DB" "$LEGACY_EXTERNAL_DIR/db-password"
cp "$CUSTOM_JWT" "$LEGACY_EXTERNAL_DIR/jwt-secret"
chmod 0600 "$LEGACY_EXTERNAL_DIR/db-password" "$LEGACY_EXTERNAL_DIR/jwt-secret"
if output="$(run_init "$LEGACY_EXTERNAL_DIR" "$EMPTY_DB" "$EMPTY_JWT" external database.example 5432 require 2>&1)"; then
	fail "legacy deployment adopted external mode without an explicit password"
fi
[ ! -e "$LEGACY_EXTERNAL_DIR/database-mode" ] || fail "failed legacy external adoption persisted a mode"
run_init "$LEGACY_EXTERNAL_DIR" "$CUSTOM_DB" "$EMPTY_JWT" external database.example 5432 require >/dev/null
[ "$(cat "$LEGACY_EXTERNAL_DIR/database-mode")" = external ] || fail "legacy deployment did not adopt external mode"

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

chmod 0644 "$EXTERNAL_DIR/database-mode"
if run_init "$EXTERNAL_DIR" "$EMPTY_DB" "$EMPTY_JWT" external database.example 5432 require >/dev/null 2>&1; then
	fail "permissive persisted database mode was accepted"
fi

printf 'config-init self-test passed\n'
