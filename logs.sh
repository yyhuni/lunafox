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
Usage: ./logs.sh [--help] [SERVICE...] [COMPOSE LOGS OPTIONS]

Follow Docker Compose logs. Without arguments it shows the last 200 lines of
every service and keeps following. Name services or pass Docker Compose logs
options to narrow the output, for example:

  ./logs.sh server
  ./logs.sh --since 10m --no-log-prefix agent

Logs are read-only and never change the deployment.
USAGE
}

for argument in "$@"; do
	case "$argument" in
	-h | --help)
		usage
		exit 0
		;;
	esac
done

exec "$HELPER" logs "$@"
