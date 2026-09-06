#!/bin/sh
set -eu
input=
output=
previous=
for argument in "$@"; do
  if [ "$previous" = "-i" ]; then input="$argument"; fi
  if [ "$previous" = "-o" ]; then output="$argument"; fi
  previous="$argument"
done
test -n "$input"
test -n "$output"
printf '%s\n' "$@" > /workspace/uro.argv
cat "$input" > "$output"
