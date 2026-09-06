#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/port-scan-runtime-engine}"
tools_dir=/opt/lunafox-tools/bin
versions_file=/opt/lunafox-tools/VERSIONS

fail() {
	echo "port scan runtime image conformance failed: $*" >&2
	exit 1
}

[ -r "$versions_file" ] || fail "missing version manifest"
[ -x "$engine_path" ] || fail "engine executable is missing or not executable at $engine_path"
[ -x "$tools_dir/naabu" ] || fail "naabu is not executable at $tools_dir/naabu"
command -v naabu >/dev/null 2>&1 || fail "naabu is not on PATH"

marker="$(sed -n 's/^naabu=//p' "$versions_file")"
[ "$marker" = "v2.4.0" ] || fail "naabu marker is ${marker:-<missing>}, expected v2.4.0"
expected_version="${marker#v}"

version_output=""
for flag in --version -version; do
	version_output="$("$tools_dir/naabu" "$flag" 2>&1 || true)"
	case "$version_output" in
	*"Current Version: $expected_version"*) break ;;
	esac
done
case "$version_output" in
*"Current Version: $expected_version"*) ;;
*) fail "naabu version output did not report $expected_version: ${version_output:-<empty>}" ;;
esac

# A port image must not acquire a capability/profile override or a second
# process broker. The real connect/passive argv contract is locked in Go tests.
if command -v docker >/dev/null 2>&1; then
	fail "runtime image must not include Docker CLI fallback"
fi

echo "port scan runtime image conformance passed"
