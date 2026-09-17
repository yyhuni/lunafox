#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-3.0-only
#
# Public LunaFox Compose lifecycle entry point. It validates this command's own
# arguments, then hands the action to the shared helper beside it. The helper
# owns the Docker, readiness, and locking behaviour for every root script.
if [ -z "${BASH_VERSION:-}" ]; then
	# A `sh install.sh` invocation must still reach Bash; fail clearly when the
	# host has no Bash at all instead of running under a different shell.
	if ! command -v bash >/dev/null 2>&1; then
		printf 'LunaFox: FAILED this script requires Bash 3.2 or newer; install bash and retry\n' >&2
		exit 1
	fi
	exec bash "$0" "$@"
fi
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELPER="$ROOT_DIR/lunafox-lifecycle.sh"

usage() {
	cat <<'USAGE'
Usage: ./start.sh [--help]

Start an existing LunaFox deployment with one `docker compose up -d` and wait up
to five minutes for the full readiness conditions.

This command never performs a first installation. Run ./install.sh when this
directory has no deployment yet.

Environment:
  LUNAFOX_READY_TIMEOUT_SECONDS   ready wait in seconds (default 300)
USAGE
}

for argument in "$@"; do
	case "$argument" in
	-h | --help)
		usage
		exit 0
		;;
	*)
		printf 'LunaFox: start.sh does not accept %s; run ./start.sh --help\n' "$argument" >&2
		exit 2
		;;
	esac
done

exec "$HELPER" start
