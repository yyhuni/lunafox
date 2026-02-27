#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
PROTO_DIR="${REPO_ROOT}/proto"
PROTO_FILE="lunafox/runtime/v1/runtime.proto"
GEN_ROOT="${REPO_ROOT}/contracts/gen"
GEN_RUNTIME_DIR="${GEN_ROOT}/lunafox/runtime/v1"

for bin in protoc protoc-gen-go protoc-gen-go-grpc; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "missing required tool: ${bin}" >&2
    exit 1
  fi
done

mkdir -p "${GEN_RUNTIME_DIR}"

protoc \
  --proto_path="${PROTO_DIR}" \
  --go_out="${GEN_ROOT}" \
  --go_opt=paths=source_relative \
  --go-grpc_out="${GEN_ROOT}" \
  --go-grpc_opt=paths=source_relative \
  "${PROTO_FILE}"

echo "generated runtime gRPC contracts"
