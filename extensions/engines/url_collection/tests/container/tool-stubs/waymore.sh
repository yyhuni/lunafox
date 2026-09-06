#!/bin/sh
set -eu
output=
previous=
for argument in "$@"; do
  if [ "$previous" = "-oU" ]; then output="$argument"; fi
  previous="$argument"
done
test -n "$output"
printf '%s\n' "$@" > /workspace/waymore.argv
printf '%s\n' 'https://example.com/archive' > "$output"
