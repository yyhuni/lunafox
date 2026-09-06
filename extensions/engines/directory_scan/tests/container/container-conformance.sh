#!/bin/sh
set -eu

engine_path="${1:-/opt/lunafox-engine/bin/directory-scan-engine}"
tools_dir="${2:-/opt/lunafox-tools/bin}"
versions_file="${3:-/opt/lunafox-tools/VERSIONS}"
expected_config_home="${4:-/run/lunafox/ffuf-config}"

fail() {
  echo "directory scan runtime image conformance failed: $*" >&2
  exit 1
}

[ -r "$versions_file" ] || fail "missing version manifest"
[ -x "$engine_path" ] || fail "engine executable is missing or not executable at $engine_path"
[ -x "$tools_dir/ffuf" ] || fail "ffuf is not executable at $tools_dir/ffuf"
command -v ffuf >/dev/null 2>&1 || fail "ffuf is not on PATH"
[ "${XDG_CONFIG_HOME:-}" = "$expected_config_home" ] || fail "XDG_CONFIG_HOME is not isolated"
[ -d "$XDG_CONFIG_HOME/ffuf" ] || fail "isolated FFUF config directory is missing"
[ -z "$(find "$XDG_CONFIG_HOME/ffuf" -mindepth 1 -type f -print -quit)" ] || \
  fail "default FFUF config home is not empty"

marker="$(sed -n 's/^ffuf=//p' "$versions_file")"
[ "$marker" = "v2.2.1" ] || fail "ffuf marker is ${marker:-<missing>}, expected v2.2.1"
version_output="$("$tools_dir/ffuf" -V 2>&1)" || fail "ffuf -V failed"
case "$version_output" in
  *"ffuf version: 2.2.1"*) ;;
  *) fail "ffuf version output did not report 2.2.1: ${version_output:-<empty>}" ;;
esac

# Executability alone can be supplied by emulation. Match the ELF machine field
# to the selected container platform as independent architecture evidence.
machine="$(uname -m)"
elf_machine="$(od -An -tx1 -j 18 -N 2 "$tools_dir/ffuf" | tr -d ' \n')"
case "$machine:$elf_machine" in
  x86_64:3e00|aarch64:b700) ;;
  *) fail "ffuf architecture mismatch: runtime=$machine elf_machine=${elf_machine:-<missing>}" ;;
esac

# FFUF seeds pinned built-in calibration/scraper resources on first start. It
# must not gain an image- or host-provided default command configuration.
[ ! -e "$XDG_CONFIG_HOME/ffuf/ffufrc" ] || fail "default ffufrc must be absent"

if command -v docker >/dev/null 2>&1; then
  fail "runtime image must not include Docker CLI fallback"
fi

echo "directory scan runtime image conformance passed"
