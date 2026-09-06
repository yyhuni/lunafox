#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "$ROOT_DIR/scripts/check-layer-dependencies.sh"
bash "$ROOT_DIR/scripts/check-domain-purity.sh"

echo "✅ DDD 边界检查通过"
