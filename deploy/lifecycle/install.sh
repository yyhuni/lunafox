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
Usage: ./install.sh [--public-host <host>] [--public-port <port>] [--help]

Start this LunaFox deployment with one `docker compose up -d` and wait until it
is fully ready, including the resident Agent and the public HTTPS endpoint. The
Compose-managed image-preheat gate runs first for every published Engine Runtime
supported by this host and the selected Compose image closure.

`--public-host` and `--public-port` set the persistent public address for this
and later lifecycle commands. Other configuration remains in .env; review
RELEASE_REGISTRY and DATABASE_MODE before the first start.

An existing .env is never replaced. Explicit public-address options update only
their corresponding keys. When .env is missing and this directory has no
LunaFox data yet, .env is created from .env.example.

Options:
  --public-host <host>  set PUBLIC_HOST to a hostname or IP address
  --public-port <port>  set PUBLIC_PORT to a port from 1 to 65535 (default 443)
  --help             show this help

Environment:
  LUNAFOX_READY_TIMEOUT_SECONDS   ready wait in seconds (default 900)
  LUNAFOX_PREHEAT_TIMEOUT_SECONDS image-preheat deadline in seconds (default 900; 300-3600)
USAGE
}

usage_failure() {
	printf 'LunaFox: %s; run ./install.sh --help\n' "$*" >&2
	exit 2
}

validate_public_host() {
	[[ "$1" =~ ^[a-zA-Z0-9.:-]+$ ]] ||
		usage_failure "--public-host must be a hostname or IP address"
}

validate_public_port() {
	[[ "$1" =~ ^[1-9][0-9]{0,4}$ ]] ||
		usage_failure "--public-port must be a decimal port from 1 to 65535"
	if [ "$1" -gt 65535 ]; then
		usage_failure "--public-port must be a decimal port from 1 to 65535"
	fi
}

PUBLIC_HOST_SET=0
PUBLIC_PORT_SET=0
INSTALL_ARGS=(install)
while [ "$#" -gt 0 ]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
		;;
	--cf-acceleration)
		usage_failure "--cf-acceleration was removed; installs pull directly from the official registries"
		;;
	--public-host)
		shift
		[ "$#" -gt 0 ] || usage_failure "--public-host requires a value"
		case "$1" in --*) usage_failure "--public-host requires a value" ;; esac
		[ "$PUBLIC_HOST_SET" = 0 ] || usage_failure "install.sh accepts --public-host at most once"
		validate_public_host "$1"
		PUBLIC_HOST_SET=1
		INSTALL_ARGS+=(--public-host "$1")
		;;
	--public-port)
		shift
		[ "$#" -gt 0 ] || usage_failure "--public-port requires a value"
		case "$1" in --*) usage_failure "--public-port requires a value" ;; esac
		[ "$PUBLIC_PORT_SET" = 0 ] || usage_failure "install.sh accepts --public-port at most once"
		validate_public_port "$1"
		PUBLIC_PORT_SET=1
		INSTALL_ARGS+=(--public-port "$1")
		;;
	*)
		usage_failure "install.sh does not accept $1"
		;;
	esac
	shift
done

exec "$HELPER" "${INSTALL_ARGS[@]}"
