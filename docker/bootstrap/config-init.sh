#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

CONFIG_DIR="${LUNAFOX_CONFIG_DIR:-/var/lib/lunafox-config}"
DB_PASSWORD_INPUT_FILE="${DB_PASSWORD_INPUT_FILE:-/run/secrets/db-password-input}"
JWT_SECRET_INPUT_FILE="${JWT_SECRET_INPUT_FILE:-/run/secrets/jwt-secret-input}"
DB_PASSWORD_FILE="$CONFIG_DIR/db-password"
JWT_SECRET_FILE="$CONFIG_DIR/jwt-secret"

fail() {
	printf 'LunaFox configuration initialization failed: %s\n' "$*" >&2
	exit 1
}

file_mode() {
	stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

validate_single_line_file() {
	local path="$1" label="$2" allow_empty="$3" size
	if [ ! -e "$path" ]; then
		[ "$allow_empty" = yes ] && return 0
		fail "$label is missing"
	fi
	[ ! -L "$path" ] && [ -f "$path" ] || fail "$label must be a regular file"
	size="$(wc -c <"$path" | tr -d '[:space:]')"
	[ "$size" -le 65536 ] || fail "$label exceeds 65536 bytes"
	if [ "$allow_empty" != yes ] && [ "$size" -eq 0 ]; then
		fail "$label is empty"
	fi
	[ "$(wc -l <"$path" | tr -d '[:space:]')" -eq 0 ] || fail "$label must contain exactly one line"
}

validate_persisted_secret() {
	local path="$1" label="$2"
	if [ ! -e "$path" ] && [ ! -L "$path" ]; then
		return 0
	fi
	validate_single_line_file "$path" "$label" no
	[ "$(file_mode "$path")" = 600 ] || fail "$label must have mode 0600"
}

input_has_value() {
	[ -f "$1" ] && [ -s "$1" ]
}

verify_existing_input() {
	local persisted="$1" input="$2" label="$3"
	if [ -e "$persisted" ] && input_has_value "$input" && ! cmp -s "$persisted" "$input"; then
		fail "$label input conflicts with the persisted value; restore the original input or use an explicit rotation procedure"
	fi
}

persist_input() {
	local input="$1" target="$2" label="$3" tmp
	tmp="$(mktemp "$CONFIG_DIR/.${label}.XXXXXX")"
	chmod 0600 "$tmp"
	if ! cp "$input" "$tmp"; then
		rm -f "$tmp"
		fail "could not stage $label"
	fi
	chmod 0600 "$tmp"
	if ! mv -f "$tmp" "$target"; then
		rm -f "$tmp"
		fail "could not persist $label"
	fi
}

persist_random_hex() {
	local bytes="$1" target="$2" label="$3" tmp
	tmp="$(mktemp "$CONFIG_DIR/.${label}.XXXXXX")"
	chmod 0600 "$tmp"
	if ! openssl rand -hex "$bytes" | tr -d '\n' >"$tmp"; then
		rm -f "$tmp"
		fail "could not generate $label"
	fi
	[ -s "$tmp" ] || {
		rm -f "$tmp"
		fail "generated $label is empty"
	}
	if ! mv -f "$tmp" "$target"; then
		rm -f "$tmp"
		fail "could not persist $label"
	fi
}

[ -d "$CONFIG_DIR" ] && [ ! -L "$CONFIG_DIR" ] || fail "configuration directory must be a directory"
chmod 0700 "$CONFIG_DIR"
umask 077

validate_single_line_file "$DB_PASSWORD_INPUT_FILE" "DB_PASSWORD input" yes
validate_single_line_file "$JWT_SECRET_INPUT_FILE" "JWT_SECRET input" yes
validate_persisted_secret "$DB_PASSWORD_FILE" "persisted database password"
validate_persisted_secret "$JWT_SECRET_FILE" "persisted JWT secret"
verify_existing_input "$DB_PASSWORD_FILE" "$DB_PASSWORD_INPUT_FILE" "DB_PASSWORD"
verify_existing_input "$JWT_SECRET_FILE" "$JWT_SECRET_INPUT_FILE" "JWT_SECRET"

if [ ! -e "$DB_PASSWORD_FILE" ]; then
	if input_has_value "$DB_PASSWORD_INPUT_FILE"; then
		persist_input "$DB_PASSWORD_INPUT_FILE" "$DB_PASSWORD_FILE" "db-password"
	else
		persist_random_hex 24 "$DB_PASSWORD_FILE" "db-password"
	fi
fi
if [ ! -e "$JWT_SECRET_FILE" ]; then
	if input_has_value "$JWT_SECRET_INPUT_FILE"; then
		persist_input "$JWT_SECRET_INPUT_FILE" "$JWT_SECRET_FILE" "jwt-secret"
	else
		persist_random_hex 32 "$JWT_SECRET_FILE" "jwt-secret"
	fi
fi

validate_persisted_secret "$DB_PASSWORD_FILE" "persisted database password"
validate_persisted_secret "$JWT_SECRET_FILE" "persisted JWT secret"
printf 'LunaFox configuration secrets are ready.\n'
