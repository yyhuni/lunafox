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
receipt_dir="$(dirname "$receipt")"
case "$RECEIPT_RELEASE_VERSION" in '' | *[!A-Za-z0-9.+-]*) exit 1 ;; esac
case "$RECEIPT_MANIFEST_SHA256" in *[!a-f0-9]*) exit 1 ;; esac
[ "${#RECEIPT_MANIFEST_SHA256}" -eq 64 ] || exit 1
case "$RECEIPT_REGISTRY" in dockerhub | ghcr) ;; *) exit 1 ;; esac

mkdir -p "$receipt_dir"
umask 077
tmp="$receipt_dir/.receipt.$$"
trap 'rm -f "$tmp"' EXIT HUP INT TERM
printf '{"schemaVersion":1,"state":"complete","releaseVersion":"%s","manifestSha256":"%s","registry":"%s","completedAt":"%s"}\n' \
	"$RECEIPT_RELEASE_VERSION" "$RECEIPT_MANIFEST_SHA256" "$RECEIPT_REGISTRY" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$tmp"
chmod 0600 "$tmp"
mv -f "$tmp" "$receipt"
