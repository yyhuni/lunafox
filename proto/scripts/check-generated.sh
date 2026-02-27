#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

before_status="$(cd "${REPO_ROOT}" && git status --porcelain -- \
  contracts/gen/lunafox/runtime/v1)"

bash "${SCRIPT_DIR}/gen-go.sh"

after_status="$(cd "${REPO_ROOT}" && git status --porcelain -- \
  contracts/gen/lunafox/runtime/v1)"

if [[ "${before_status}" != "${after_status}" ]]; then
  echo "runtime proto generated artifacts are not up to date:" >&2
  echo "${after_status}" >&2
  exit 1
fi

echo "runtime proto generated artifacts are up to date"
