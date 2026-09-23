#!/usr/bin/env bash
set -euo pipefail

# Compose owns resident containers. Initialization writes a restricted credential
# file shared with the Agent; secrets never cross the host environment.
: "${ENGINE_INSTALL_INVENTORY_PATH:?ENGINE_INSTALL_INVENTORY_PATH is required}"
: "${ENGINE_INSTALL_CF_ACCELERATION:?ENGINE_INSTALL_CF_ACCELERATION is required}"
: "${FINGERPRINT_BOOTSTRAP_PATH:?FINGERPRINT_BOOTSTRAP_PATH is required}"
: "${WORDLISTS_SOURCE_PATH:?WORDLISTS_SOURCE_PATH is required}"
: "${AGENT_VERSION:?AGENT_VERSION is required}"

case "$ENGINE_INSTALL_CF_ACCELERATION" in
true)
	[ -z "${ENGINE_INSTALL_REGISTRY:-}" ] || {
		echo 'ENGINE_INSTALL_REGISTRY must be empty when ENGINE_INSTALL_CF_ACCELERATION=true' >&2
		exit 1
	}
	;;
false)
	case "${ENGINE_INSTALL_REGISTRY:-}" in
	docker.io | ghcr.io) ;;
	*)
		echo 'ENGINE_INSTALL_REGISTRY must be docker.io or ghcr.io when CF acceleration is disabled' >&2
		exit 1
		;;
	esac
	;;
*)
	echo 'ENGINE_INSTALL_CF_ACCELERATION must be exactly true or false' >&2
	exit 1
	;;
esac

server engine-bootstrap
server fingerprint-bootstrap "$FINGERPRINT_BOOTSTRAP_PATH"
server wordlist-bootstrap "$WORDLISTS_SOURCE_PATH/manifest.json" "$WORDLISTS_SOURCE_PATH"
server agent-bootstrap /var/lib/lunafox-agent/bootstrap.json lunafox-agent "$AGENT_VERSION"
