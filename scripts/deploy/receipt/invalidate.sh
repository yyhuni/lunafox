#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-only
set -eu

receipt="${RECEIPT_PATH:-/receipt/receipt.json}"
case "$receipt" in
/*) ;;
*)
	echo "completion receipt path must be absolute" >&2
	exit 1
	;;
esac
receipt_dir="$(dirname "$receipt")"
mkdir -p "$receipt_dir"
tmp="$receipt_dir/.receipt.invalid.$$"
umask 077
trap 'rm -f "$tmp"' EXIT HUP INT TERM
printf '%s\n' '{"schemaVersion":1,"state":"invalid"}' >"$tmp"
chmod 0600 "$tmp"
mv -f "$tmp" "$receipt"
