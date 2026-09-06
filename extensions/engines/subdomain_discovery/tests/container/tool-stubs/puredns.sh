#!/bin/sh
set -eu
test "${1:-}" = resolve
input="${2:-}"
output=
previous=
for argument in "$@"; do
	if [ "$previous" = "--write" ]; then output="$argument"; fi
	previous="$argument"
done
test -f "$input"
test -n "$output"
printf '%s\n' "$@" > /workspace/puredns.argv
cp "$input" "$output"
