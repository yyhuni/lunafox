#!/bin/sh
set -eu
output=
previous=
for argument in "$@"; do
	if [ "$previous" = "-o" ]; then output="$argument"; fi
	previous="$argument"
done
test -n "$output"
printf '%s\n' "$@" > /workspace/subfinder.argv
printf '%s\n' 'api.example.com' > "$output"
