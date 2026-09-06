#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/fingerprint-detection-runtime-engine}"
tools_dir=/opt/lunafox-tools/bin
versions_file=/opt/lunafox-tools/VERSIONS

fail() {
  echo "fingerprint detection runtime image conformance failed: $*" >&2
  exit 1
}

[ -r "$versions_file" ] || fail "missing version manifest"
[ -x "$engine_path" ] || fail "engine executable is missing or not executable at $engine_path"
[ -x "$tools_dir/observer-ward" ] || fail "observer-ward is not executable at $tools_dir/observer-ward"
command -v observer-ward >/dev/null 2>&1 || fail "observer-ward is not on PATH"

[ "$(sed -n 's/^observer_ward=//p' "$versions_file")" = "v2026.6.28-lunafox.1" ] || \
  fail "Observer Ward version marker mismatch"
[ "$(sed -n 's/^observer_ward_commit=//p' "$versions_file")" = "65801cf6d4b3dd4bb07ea7a1cf6e849713c42ea6" ] || \
  fail "Observer Ward commit marker mismatch"

version_output="$(observer-ward --version 2>&1 || true)"
case "$version_output" in
  *"observer_ward v2026.6.28-lunafox.1"*) ;;
  *) fail "Observer Ward version output did not report v2026.6.28-lunafox.1: ${version_output:-<empty>}" ;;
esac

# Verify the pinned payload can start in the target runtime image; the wrapper
# above intentionally intercepts --version and is not sufficient evidence.
"$tools_dir/observer_ward.real" --help >/dev/null 2>&1 || \
  fail "pinned Observer Ward binary cannot start"

# The runtime image must not carry a second process broker or Docker fallback.
if command -v docker >/dev/null 2>&1; then
  fail "runtime image must not include Docker CLI fallback"
fi

echo "fingerprint detection runtime image conformance passed"
