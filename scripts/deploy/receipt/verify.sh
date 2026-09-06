#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
set -eu

: "${RECEIPT_RELEASE_VERSION:?RECEIPT_RELEASE_VERSION is required}"
: "${RECEIPT_MANIFEST_SHA256:?RECEIPT_MANIFEST_SHA256 is required}"
: "${RECEIPT_REGISTRY:?RECEIPT_REGISTRY is required}"
receipt="${RECEIPT_PATH:-/receipt/receipt.json}"
case "$receipt" in
/*) ;;
*)
	echo "completion receipt path must be absolute" >&2
	exit 1
	;;
esac
case "$RECEIPT_RELEASE_VERSION" in '' | *[!A-Za-z0-9.+-]*)
	echo "completion receipt release expectation is invalid" >&2
	exit 1
	;;
esac
case "$RECEIPT_MANIFEST_SHA256" in *[!a-f0-9]*)
	echo "completion receipt manifest expectation is invalid" >&2
	exit 1
	;;
esac
[ "${#RECEIPT_MANIFEST_SHA256}" -eq 64 ] || {
	echo "completion receipt manifest expectation is invalid" >&2
	exit 1
}
case "$RECEIPT_REGISTRY" in dockerhub | ghcr) ;; *)
	echo "completion receipt registry expectation is invalid" >&2
	exit 1
	;;
esac
[ -f "$receipt" ] && [ ! -L "$receipt" ] || {
	echo "completion receipt is missing" >&2
	exit 1
}
receipt_mode="$(stat -c '%a' "$receipt" 2>/dev/null || stat -f '%Lp' "$receipt" 2>/dev/null || true)"
[ "$receipt_mode" = 600 ] || {
	echo "completion receipt permissions are invalid" >&2
	exit 1
}
[ "$(tail -c 1 "$receipt" | od -An -t x1 | tr -d ' \n')" = 0a ] || {
	echo "completion receipt must end with a newline" >&2
	exit 1
}
line_count="$(wc -l <"$receipt" | tr -d ' ')"
[ "$line_count" -eq 1 ] || {
	echo "completion receipt must contain exactly one JSON record" >&2
	exit 1
}
line="$(sed -n '1p' "$receipt")"
prefix="$(printf '{"schemaVersion":1,"state":"complete","releaseVersion":"%s","manifestSha256":"%s","registry":"%s","completedAt":"' "$RECEIPT_RELEASE_VERSION" "$RECEIPT_MANIFEST_SHA256" "$RECEIPT_REGISTRY")"
case "$line" in
"$prefix"*) timestamp="${line#"$prefix"}" ;;
*)
	echo "completion receipt JSON schema is invalid" >&2
	exit 1
	;;
esac
timestamp="$(printf '%s' "$timestamp" | sed 's/"}$//')"
[ "$line" = "$prefix$timestamp\"}" ] || {
	echo "completion receipt JSON fields are invalid" >&2
	exit 1
}
printf '%s\n' "$timestamp" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$' || {
	echo "completion receipt timestamp is invalid" >&2
	exit 1
}
