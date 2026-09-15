#!/usr/bin/env bash
# Shared by host entry points and embedded in generated remote installers.
LUNAFOX_LOKI_VERSION=3.6.7

lunafox_loki_reference() {
	case "$1" in
	amd64 | x86_64) printf 'grafana/loki-docker-driver:%s-amd64\n' "$LUNAFOX_LOKI_VERSION" ;;
	arm64 | aarch64) printf 'grafana/loki-docker-driver:%s-arm64\n' "$LUNAFOX_LOKI_VERSION" ;;
	*)
		printf 'Unsupported Docker architecture: %s\n' "$1" >&2
		return 1
		;;
	esac
}
