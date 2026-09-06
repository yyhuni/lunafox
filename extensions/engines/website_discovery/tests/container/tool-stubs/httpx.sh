#!/bin/sh
set -eu
output=
previous=
for argument in "$@"; do
	if [ "$previous" = "-o" ]; then output="$argument"; fi
	previous="$argument"
done
test -n "$output"
printf '%s\n' "$@" > /workspace/httpx.argv
printf '%s\n' '{"input":"https://example.com","url":"https://example.com","host":"example.com","title":"Example","status_code":200}' > "$output"
