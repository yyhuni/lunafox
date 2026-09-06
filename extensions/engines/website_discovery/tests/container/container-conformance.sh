#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/website-discovery-runtime-engine}"
tools_dir=/opt/lunafox-tools/bin
versions_file=/opt/lunafox-tools/VERSIONS
smoke_input=/tmp/lunafox-httpx-smoke-input.txt
smoke_output=/tmp/lunafox-httpx-smoke-output.jsonl

fail() {
	echo "website discovery runtime image conformance failed: $*" >&2
	exit 1
}

[ -r "$versions_file" ] || fail "missing version manifest"
[ -x "$engine_path" ] || fail "engine executable is missing or not executable at $engine_path"
[ -x "$tools_dir/httpx" ] || fail "httpx is not executable at $tools_dir/httpx"
command -v httpx >/dev/null 2>&1 || fail "httpx is not on PATH"

marker="$(sed -n 's/^httpx=//p' "$versions_file")"
[ "$marker" = "v1.8.1" ] || fail "httpx marker is ${marker:-<missing>}, expected v1.8.1"

version_output=""
for flag in --version -version; do
	version_output="$("$tools_dir/httpx" "$flag" 2>&1 || true)"
	case "$version_output" in
	*v1.8.1*) break ;;
	esac
done
case "$version_output" in
*v1.8.1*) ;;
*) fail "httpx version output did not contain v1.8.1: ${version_output:-<empty>}" ;;
esac

# Execute a no-network empty-list probe. This proves the image payload can
# start httpx without turning the image build into a real website scan.
: >"$smoke_input"
if ! httpx -list "$smoke_input" -json -silent -no-color -o "$smoke_output" >/dev/null 2>&1; then
	fail "httpx empty-list smoke failed"
fi
[ -f "$smoke_output" ] || fail "httpx empty-list smoke produced no output file"

if command -v docker >/dev/null 2>&1; then
	fail "runtime image must not include Docker CLI fallback"
fi

echo "website discovery runtime image conformance passed"
