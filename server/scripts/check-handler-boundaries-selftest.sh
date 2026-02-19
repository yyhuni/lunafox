#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-handler-boundaries.sh"

CASE1_FILE="$ROOT_DIR/internal/modules/catalog/handler/_selftest_invalid_export.go"
CASE2_FILE="$ROOT_DIR/internal/modules/scan/handler/_selftest_invalid_export.go"

cleanup() {
	rm -f "$CASE1_FILE" "$CASE2_FILE"
}
trap cleanup EXIT

if [[ ! -x "$CHECK_SCRIPT" ]]; then
	echo "❌ 未找到可执行守卫脚本: $CHECK_SCRIPT"
	exit 1
fi

echo "[1/2] 校验非 New*Handler 导出函数可被拦截..."
cat >"$CASE1_FILE" <<'CASE1'
package handler

func RegisterCatalogHandlerRoutes() {}
CASE1

if bash "$CHECK_SCRIPT" >/tmp/handler-guard-selftest-case1.log 2>&1; then
	echo "❌ 自测失败：非法导出函数未被拦截"
	cat /tmp/handler-guard-selftest-case1.log
	exit 1
fi
rm -f "$CASE1_FILE"

echo "[2/2] 校验通用导出 helper 函数可被拦截..."
cat >"$CASE2_FILE" <<'CASE2'
package handler

func BuildScanHandler() {}
CASE2

if bash "$CHECK_SCRIPT" >/tmp/handler-guard-selftest-case2.log 2>&1; then
	echo "❌ 自测失败：导出 helper 函数未被拦截"
	cat /tmp/handler-guard-selftest-case2.log
	exit 1
fi

echo "✅ handler 边界守卫自测通过（两类违规均可被正确拦截）"
