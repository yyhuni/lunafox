#!/bin/sh
set -eu

printf '%s\n' "$@" > /workspace/httpx.argv

candidate_file=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -list)
      candidate_file="${2:-}"
      shift 2
      ;;
    *) shift ;;
  esac
done

[ -n "$candidate_file" ]

index=0
while IFS= read -r candidate; do
  [ -n "$candidate" ]
  index=$((index + 1))
  png_path="/workspace/httpx-fixture-$index.png"
  printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAAEElEQVR4nGJiZGIGBAAA//8AFgAJ1oG2VgAAAABJRU5ErkJggg==' | base64 -d > "$png_path"
  printf '{"input":"%s","status_code":200,"screenshot_path":"%s"}\n' "$candidate" "$png_path"
done < "$candidate_file"

[ "$index" -gt 0 ]
