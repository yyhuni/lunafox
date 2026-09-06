#!/bin/sh
set -eu
mode=active
output=
previous=
for argument in "$@"; do
	if [ "$previous" = "-o" ]; then output="$argument"; fi
	if [ "$argument" = "-passive" ]; then mode=passive; fi
	previous="$argument"
done
test -n "$output"
printf '%s\n' "$@" > "/workspace/naabu_${mode}.argv"
printf '%s\n' '{"host":"example.com","ip":"192.0.2.10","port":443}' > "$output"
