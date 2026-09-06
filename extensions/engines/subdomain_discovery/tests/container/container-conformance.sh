#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/subdomain-discovery-runtime-engine}"
tools_dir="${2:-/opt/lunafox-tools/bin}"
versions_file="${3:-/opt/lunafox-tools/VERSIONS}"

fail() {
	echo "subdomain runtime image conformance failed: $*" >&2
	exit 1
}

[ -r "$versions_file" ] || fail "missing version manifest"
[ -x "$engine_path" ] || fail "engine executable is missing or not executable at $engine_path"

assert_executable() {
	name="$1"
	path="$tools_dir/$name"
	[ -x "$path" ] || fail "$name is not executable at $path"
	command -v "$name" >/dev/null 2>&1 || fail "$name is not on PATH"
}

assert_file_executable() {
	name="$1"
	path="$tools_dir/$name"
	[ -x "$path" ] || fail "$name is not executable at $path"
}

assert_help_probe() {
	name="$1"
	path="$tools_dir/$name"
	if ! output="$("$path" --help 2>&1)"; then
		fail "$name help probe failed: ${output:-<empty>}"
	fi
	[ -n "$output" ] || fail "$name did not produce a help probe"
}

assert_version_marker() {
	name="$1"
	expected="$2"
	marker="$(sed -n "s/^${name}=//p" "$versions_file")"
	[ "$marker" = "$expected" ] || fail "$name marker is ${marker:-<missing>}, expected $expected"
}

assert_version_output() {
	name="$1"
	expected="$2"
	output=""
	for flag in --version -version; do
		output="$("$tools_dir/$name" "$flag" 2>&1 || true)"
		case "$output" in
		*"$expected"*) return 0 ;;
		esac
	done
	fail "$name version output did not contain $expected: ${output:-<empty>}"
}

assert_executable subfinder
assert_executable puredns
assert_executable massdns
assert_file_executable massdns.real
assert_help_probe massdns.real

assert_version_marker subfinder v2.12.0
assert_version_marker puredns v2.1.2-0.20260223162428-46bd4c963ec2
assert_version_marker massdns v1.1.0

# These checks exercise the public version probes where upstream supports them
# and the transparent wrappers for tools that do not.
assert_version_output subfinder v2.12.0
assert_version_output puredns v2.1.2
assert_version_output massdns v1.1.0

# PureDNS resolves MassDNS by PATH; fail the image build if that dependency is
# missing or accidentally shadowed by a host/Worker-provided executable.
resolved_massdns="$(command -v massdns)"
[ "$resolved_massdns" = "$tools_dir/massdns" ] || fail "puredns dependency resolved to $resolved_massdns"

echo "subdomain runtime image conformance passed"
