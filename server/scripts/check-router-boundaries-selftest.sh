#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-router-boundaries.sh"

CASE1_FILE="$ROOT_DIR/internal/modules/catalog/router/_selftest_public_routes.go"
CASE2_DIR="$ROOT_DIR/internal/modules/_selftest_router"
CASE2_FILE="$CASE2_DIR/router/routes.go"

cleanup() {
  rm -f "$CASE1_FILE"
  rm -rf "$CASE2_DIR"
}
trap cleanup EXIT

if [[ ! -x "$CHECK_SCRIPT" ]]; then
  echo "❌ 未找到可执行守卫脚本: $CHECK_SCRIPT"
  exit 1
fi

echo "[1/2] 校验未授权公开入口可被拦截..."
cat > "$CASE1_FILE" <<'CASE1'
package router

func RegisterCatalogLegacyRoutes() {}
CASE1

if bash "$CHECK_SCRIPT" >/tmp/router-guard-selftest-case1.log 2>&1; then
  echo "❌ 自测失败：未授权公开入口未被拦截"
  cat /tmp/router-guard-selftest-case1.log
  exit 1
fi
rm -f "$CASE1_FILE"

echo "[2/2] 校验未配置模块的 router 可被拦截..."
mkdir -p "$CASE2_DIR/router"
cat > "$CASE2_FILE" <<'CASE2'
package router

func RegisterSelftestRoutes() {}
CASE2

if bash "$CHECK_SCRIPT" >/tmp/router-guard-selftest-case2.log 2>&1; then
  echo "❌ 自测失败：未配置模块 router 未被拦截"
  cat /tmp/router-guard-selftest-case2.log
  exit 1
fi

echo "✅ router 边界守卫自测通过（两类违规均可被正确拦截）"
