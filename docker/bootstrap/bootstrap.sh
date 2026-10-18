#!/usr/bin/env bash
set -euo pipefail

# Compose owns resident containers. Initialization writes a restricted credential
# file shared with the Agent; secrets never cross the host environment.
: "${ENGINE_INSTALL_INVENTORY_PATH:?ENGINE_INSTALL_INVENTORY_PATH is required}"

# CF acceleration was removed; a legacy deployment may still carry the old
# key. Degrade to direct registry pulls with one warning instead of failing.
if [ -n "${ENGINE_INSTALL_CF_ACCELERATION:-}" ]; then
	echo 'ENGINE_INSTALL_CF_ACCELERATION is deprecated and ignored; pulling directly from the official registries' >&2
fi
: "${FINGERPRINT_BOOTSTRAP_PATH:?FINGERPRINT_BOOTSTRAP_PATH is required}"
: "${WORDLISTS_SOURCE_PATH:?WORDLISTS_SOURCE_PATH is required}"
: "${AGENT_VERSION:?AGENT_VERSION is required}"

# Registry selection controls transport only. Engine bootstrap still validates
# signatures, immutable digests, package compatibility and platform bindings.
if [[ ! "${ENGINE_INSTALL_REGISTRY:-}" =~ ^[a-z0-9]([a-z0-9.-]*[a-z0-9])?(:[0-9]+)?$ ]]; then
	echo 'ENGINE_INSTALL_REGISTRY must be a Registry authority (host[:port], without scheme or path)' >&2
	exit 1
fi

server engine-bootstrap
server fingerprint-bootstrap "$FINGERPRINT_BOOTSTRAP_PATH"
server wordlist-bootstrap "$WORDLISTS_SOURCE_PATH/manifest.json" "$WORDLISTS_SOURCE_PATH"
server agent-bootstrap /var/lib/lunafox-agent/bootstrap.json lunafox-agent "$AGENT_VERSION"
