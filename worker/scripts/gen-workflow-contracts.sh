#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "${ROOT_DIR}"

SERVER_SCHEMA_DIR="${SERVER_SCHEMA_DIR:-../server/internal/workflow/schema}"
SERVER_PROFILE_DIR="${SERVER_PROFILE_DIR:-../server/internal/workflow/profile/profiles}"
SERVER_MANIFEST_DIR="${SERVER_MANIFEST_DIR:-../server/internal/workflow/manifest}"
DOCS_DIR="${DOCS_DIR:-../docs/config-reference}"
WORKER_SCHEMA_BASE_DIR="${WORKER_SCHEMA_BASE_DIR:-}"
MIRROR_SCHEMA_DIR="${MIRROR_SCHEMA_DIR:-}"

echo "Generating workflow contracts from code-first definitions..."
go generate ./internal/workflow/all

for workflow_dir in internal/workflow/*/; do
  if [[ -f "${workflow_dir}contract_definition.go" ]]; then
    workflow_name="$(basename "${workflow_dir%/}")"
    worker_schema_dir="${workflow_dir}generated"
    worker_manifest_dir="${workflow_dir}generated"
    if [[ -n "${WORKER_SCHEMA_BASE_DIR}" ]]; then
      worker_schema_dir="${WORKER_SCHEMA_BASE_DIR%/}/${workflow_name}"
    fi
    echo "Generating for ${workflow_name}..."
    cmd=(
      go run ./cmd/workflow-contract-gen
      -workflow "${workflow_name}" \
      -worker-schema-dir "${worker_schema_dir}" \
      -worker-manifest-dir "${worker_manifest_dir}" \
      -server-schema-dir "${SERVER_SCHEMA_DIR}" \
      -server-manifest-dir "${SERVER_MANIFEST_DIR}" \
      -server-profile-dir "${SERVER_PROFILE_DIR}" \
      -docs-dir "${DOCS_DIR}" \
      -typed-go-output "${workflow_dir}config_typed_generated.go" \
      -typed-go-package "${workflow_name}"
    )
    if [[ -n "${MIRROR_SCHEMA_DIR}" ]]; then
      cmd+=( -mirror-schema-dir "${MIRROR_SCHEMA_DIR}" )
    fi
    "${cmd[@]}"
  fi
done

echo "✓ Workflow contract generation completed"
