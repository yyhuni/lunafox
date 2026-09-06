#!/bin/sh
set -eu

list_path=
previous=
for argument in "$@"; do
  if [ "$previous" = "-l" ]; then
    list_path="$argument"
  fi
  previous="$argument"
done

test -n "$list_path"
printf '%s\n' "$@" > /workspace/observer_ward.argv

while IFS= read -r candidate; do
  [ -n "$candidate" ] || continue
  # Conformance candidates are canonical URLs without JSON metacharacters.
  printf '%s\n' "{\"input_target\":\"$candidate\",\"target\":\"$candidate\",\"success\":true,\"matched\":[{\"base_url\":\"$candidate\",\"result\":{\"status\":200,\"name\":[\"observer-ward-stub\"]}}]}"
done < "$list_path"
