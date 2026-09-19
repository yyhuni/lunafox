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
Usage: ./uninstall.sh [--help] [--purge --confirm]

Remove this deployment's containers, orphan containers of this Compose project,
and the project network. Named volumes, .env, certificates, upgrade state, and
compose.override.yaml are preserved together with this release directory, so
./install.sh or ./start.sh can restore the deployment.

  --purge --confirm   additionally delete the LunaFox named volumes declared by
                      the current compose.yaml after verifying their ownership,
                      then reset a persisted compose.override.yaml. Data in
                      those volumes cannot be recovered; .env and this release
                      directory are still preserved.

--purge without --confirm fails before anything is removed.
USAGE
}

PURGE=0
CONFIRM=0
for argument in "$@"; do
	case "$argument" in
	-h | --help)
		usage
		exit 0
		;;
	--purge) PURGE=1 ;;
	--confirm) CONFIRM=1 ;;
	*)
		printf 'LunaFox: uninstall.sh does not accept %s; run ./uninstall.sh --help\n' "$argument" >&2
		exit 2
		;;
	esac
done

if [ "$PURGE" = 1 ] && [ "$CONFIRM" != 1 ]; then
	printf 'LunaFox: FAILED --purge permanently deletes the LunaFox volumes and resets the persisted version override; re-run with ./uninstall.sh --purge --confirm\n' >&2
	exit 2
fi

if [ "$PURGE" = 1 ]; then
	exec "$HELPER" uninstall --purge --confirm
fi
exec "$HELPER" uninstall
