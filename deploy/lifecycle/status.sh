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
Usage: ./status.sh [--help]

Print a read-only readiness summary: overall state, public address, one-shot
tasks, core service health, auxiliary services, the resident Agent, and the
public HTTPS endpoint. It never waits and never changes the deployment.

The exit status is 0 only for a fully ready deployment, so the command can be
used directly by monitoring and automation.
USAGE
}

for argument in "$@"; do
	case "$argument" in
	-h | --help)
		usage
		exit 0
		;;
	*)
		printf 'LunaFox: status.sh does not accept %s; run ./status.sh --help\n' "$argument" >&2
		exit 2
		;;
	esac
done

exec "$HELPER" status
