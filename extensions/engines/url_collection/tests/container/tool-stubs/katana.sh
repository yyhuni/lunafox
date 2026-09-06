#!/bin/sh
set -eu
output=
previous=
for argument in "$@"; do
  if [ "$previous" = "-o" ]; then output="$argument"; fi
  previous="$argument"
done
test -n "$output"
printf '%s\n' "$@" > /workspace/katana.argv
printf '%s\n' 'https://example.com/crawl' > "$output"
