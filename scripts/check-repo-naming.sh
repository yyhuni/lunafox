#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

bash "$ROOT_DIR/server/scripts/check-naming-conventions.sh"
bash "$ROOT_DIR/server/scripts/check-mapper-naming.sh"
bash "$ROOT_DIR/server/scripts/check-router-boundaries.sh"
bash "$ROOT_DIR/server/scripts/check-handler-boundaries.sh"
bash "$ROOT_DIR/server/scripts/check-wiring-conventions.sh"
node "$ROOT_DIR/scripts/check-frontend-naming.mjs"

echo "✅ 全仓命名与边界守卫检查通过"
