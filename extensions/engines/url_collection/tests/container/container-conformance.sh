#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/url-collection-engine}"
[ -x "$engine_path" ] || exit 1
[ -r /opt/lunafox-tools/VERSIONS ] || exit 1
grep -Fx 'waymore=v8.9' /opt/lunafox-tools/VERSIONS >/dev/null
grep -Fx 'katana=v1.6.1' /opt/lunafox-tools/VERSIONS >/dev/null
grep -Fx 'uro=v1.0.2' /opt/lunafox-tools/VERSIONS >/dev/null
grep -Fx 'httpx=v1.10.0' /opt/lunafox-tools/VERSIONS >/dev/null
for tool in waymore katana uro httpx; do
  command -v "$tool" >/dev/null 2>&1 || exit 1
done
waymore --help >/dev/null
uro --help >/dev/null

uro_input=$(mktemp)
uro_output="${uro_input}.output"
trap 'rm -f "$uro_input" "$uro_output"' EXIT
printf '%s\n' 'https://example.com/api.php?id=1' > "$uro_input"
uro -i "$uro_input" -o "$uro_output"
test -f "$uro_output"
grep -Fx 'https://example.com/api.php?id=1' "$uro_output" >/dev/null
