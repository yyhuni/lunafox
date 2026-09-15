#!/usr/bin/env bash
set -euo pipefail

# Compose owns resident containers. Initialization writes a restricted credential
# file shared with the Agent; secrets never cross the host environment.
: "${ENGINE_INSTALL_INVENTORY_PATH:?ENGINE_INSTALL_INVENTORY_PATH is required}"
: "${ENGINE_INSTALL_REGISTRY:?ENGINE_INSTALL_REGISTRY is required}"
: "${FINGERPRINT_BOOTSTRAP_PATH:?FINGERPRINT_BOOTSTRAP_PATH is required}"
: "${WORDLISTS_SOURCE_PATH:?WORDLISTS_SOURCE_PATH is required}"
: "${AGENT_VERSION:?AGENT_VERSION is required}"

server engine-bootstrap
server fingerprint-bootstrap "$FINGERPRINT_BOOTSTRAP_PATH"
server wordlist-bootstrap "$WORDLISTS_SOURCE_PATH/manifest.json" "$WORDLISTS_SOURCE_PATH"
server agent-bootstrap /var/lib/lunafox-agent/bootstrap.json lunafox-agent "$AGENT_VERSION"
