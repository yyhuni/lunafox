#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/screenshot-runtime-engine}"
tools_dir=/opt/lunafox-tools/bin
versions_file=/opt/lunafox-tools/VERSIONS
root=/tmp/lunafox-screenshot-conformance
input="$root/input.txt"
output="$root/httpx.jsonl"
workspace="$root/workspace"
server_pid=""

fail() {
  echo "screenshot runtime image conformance failed: $*" >&2
  exit 1
}

cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

[ -x "$engine_path" ] || fail "engine executable is missing or not executable"
[ -r "$versions_file" ] || fail "missing version manifest"
[ "$(sed -n 's/^httpx=//p' "$versions_file")" = "v1.10.0" ] || fail "HTTPX version marker mismatch"
[ "$(sed -n 's/^chromium=//p' "$versions_file")" = "v131.0.6778" ] || fail "Chromium version marker mismatch"
[ "$(sed -n 's/^cwebp=//p' "$versions_file")" = "v1.6.0" ] || fail "cwebp version marker mismatch"

for tool in httpx chromium cwebp; do
  [ -x "$tools_dir/$tool" ] || fail "$tool is not executable"
  command -v "$tool" >/dev/null 2>&1 || fail "$tool is not on PATH"
done
httpx --version 2>&1 | grep -F 'v1.10.0' >/dev/null || fail "HTTPX binary version mismatch"
chromium --version 2>&1 | grep -F '131.0.6778.85' >/dev/null || fail "Chromium binary version mismatch"
cwebp -version 2>&1 | grep -F '1.6.0' >/dev/null || fail "cwebp binary version mismatch"

mkdir -p "$root/site" "$workspace"
printf '%s\n' '<!doctype html><title>LunaFox Screenshot</title><main>offline conformance</main>' > "$root/site/index.html"
python3 -m http.server 18080 --bind 127.0.0.1 --directory "$root/site" >/dev/null 2>&1 &
server_pid=$!
i=0
while ! (python3 -c 'import urllib.request; urllib.request.urlopen("http://127.0.0.1:18080/", timeout=1).read()' >/dev/null 2>&1); do
  i=$((i + 1))
  [ "$i" -lt 20 ] || fail "local fixture server did not start"
  sleep 0.1
done

printf '%s\n' 'http://127.0.0.1:18080/' > "$input"
httpx -list "$input" -json -ss -no-screenshot-full-page -system-chrome \
  -screenshot-timeout 15s -sid 1s -threads 1 -retries 0 \
  -ho window-size=1280,720 -ho force-device-scale-factor=1 \
  -ob -esb -ehb -srd "$workspace" > "$output" 2>/dev/null || fail "offline HTTPX screenshot failed"

png_path="$(sed -n 's/.*"screenshot_path":"\([^"]*\)".*/\1/p' "$output" | head -n 1)"
[ -n "$png_path" ] || fail "HTTPX did not report a screenshot path"
[ -s "$png_path" ] || fail "HTTPX screenshot PNG is missing or empty"
cwebp -quiet -resize 800 0 -q 72 "$png_path" -o "$root/output.webp" >/dev/null 2>&1 || fail "cwebp conversion failed"
[ -s "$root/output.webp" ] || fail "cwebp output is empty"
magic="$(dd if="$root/output.webp" bs=1 count=4 2>/dev/null)"
[ "$magic" = "RIFF" ] || fail "cwebp output is not a RIFF WebP"

echo "screenshot runtime image conformance passed"
