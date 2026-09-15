#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCRIPT="$ROOT_DIR/docker/bootstrap/cert-init.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
	echo "Compose certificate initialization self-test failed: $*" >&2
	exit 1
}

expect_failure() {
	local label="$1"
	shift
	if "$@" >"$TMP_DIR/$label.log" 2>&1; then
		fail "$label unexpectedly succeeded"
	fi
}

certificate_dir="$TMP_DIR/certificates"
PUBLIC_HOST=localhost LUNAFOX_CERTIFICATE_DIRECTORY="$certificate_dir" bash "$SCRIPT" >/dev/null 2>&1
[ "$(stat -f '%Lp' "$certificate_dir/fullchain.pem" 2>/dev/null || stat -c '%a' "$certificate_dir/fullchain.pem")" = 600 ] || fail "certificate mode is not 0600"
[ "$(stat -f '%Lp' "$certificate_dir/privkey.pem" 2>/dev/null || stat -c '%a' "$certificate_dir/privkey.pem")" = 600 ] || fail "private key mode is not 0600"
openssl x509 -in "$certificate_dir/fullchain.pem" -noout -text | grep -Fq 'DNS:localhost' || fail "certificate SAN is missing"
first_digest="$(openssl dgst -sha256 -r "$certificate_dir/fullchain.pem" | awk '{print $1}')"
PUBLIC_HOST=localhost LUNAFOX_CERTIFICATE_DIRECTORY="$certificate_dir" bash "$SCRIPT" >/dev/null
second_digest="$(openssl dgst -sha256 -r "$certificate_dir/fullchain.pem" | awk '{print $1}')"
[ "$first_digest" = "$second_digest" ] || fail "successful retry replaced the certificate"

expect_failure wrong-host env PUBLIC_HOST=other.invalid LUNAFOX_CERTIFICATE_DIRECTORY="$certificate_dir" bash "$SCRIPT"

partial_dir="$TMP_DIR/partial"
mkdir -p "$partial_dir"
printf '%s\n' invalid >"$partial_dir/fullchain.pem"
chmod 0600 "$partial_dir/fullchain.pem"
expect_failure partial env PUBLIC_HOST=localhost LUNAFOX_CERTIFICATE_DIRECTORY="$partial_dir" bash "$SCRIPT"

permissive_dir="$TMP_DIR/permissive"
cp -R "$certificate_dir" "$permissive_dir"
chmod 0644 "$permissive_dir/fullchain.pem"
expect_failure permissive env PUBLIC_HOST=localhost LUNAFOX_CERTIFICATE_DIRECTORY="$permissive_dir" bash "$SCRIPT"

symlink_dir="$TMP_DIR/symlink"
mkdir -p "$symlink_dir"
ln -s "$certificate_dir/fullchain.pem" "$symlink_dir/fullchain.pem"
cp "$certificate_dir/privkey.pem" "$symlink_dir/privkey.pem"
expect_failure symlink env PUBLIC_HOST=localhost LUNAFOX_CERTIFICATE_DIRECTORY="$symlink_dir" bash "$SCRIPT"

echo "Compose certificate initialization self-test passed"
