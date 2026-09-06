#!/bin/sh
set -eu

if [ ! -e /workspace/cwebp.argv ]; then
  printf '%s\n' "$@" > /workspace/cwebp.argv
fi

output=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      output="${2:-}"
      shift 2
      ;;
    *) shift ;;
  esac
done

[ -n "$output" ]
printf '%s' 'UklGRiIAAABXRUJQVlA4IBYAAADQAQCdASoBAAEAAUAmJaQAA3AA/vuUAAA=' | base64 -d > "$output"
