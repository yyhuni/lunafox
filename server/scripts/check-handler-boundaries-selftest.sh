#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-handler-boundaries.sh"

CASE1_FILE="$ROOT_DIR/internal/modules/catalog/handler/_selftest_invalid_export.go"
CASE2_FILE="$ROOT_DIR/internal/modules/scan/handler/_selftest_invalid_export.go"
CASE3_FILE="$ROOT_DIR/internal/modules/agent/handler/_selftest_concrete_service_handler.go"
CASE4_FILE="$ROOT_DIR/internal/modules/agent/handler/_selftest_repository_handler.go"

cleanup() {
	rm -f "$CASE1_FILE" "$CASE2_FILE" "$CASE3_FILE" "$CASE4_FILE"
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
rm -f "$CASE2_FILE"

echo "[3/3] 校验 handler 直接依赖 concrete application service 可被拦截..."
cat >"$CASE3_FILE" <<'CASE3'
package handler

import agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"

type badHandlerDependency struct {
	svc *agentapp.AgentControlLifecycleService
}
CASE3

if bash "$CHECK_SCRIPT" >/tmp/handler-guard-selftest-case3.log 2>&1; then
	echo "❌ 自测失败：handler 直接依赖 concrete service 未被拦截"
	cat /tmp/handler-guard-selftest-case3.log
	exit 1
fi
rm -f "$CASE3_FILE"

echo "[4/4] 校验 handler 直接依赖 domain repository 接口可被拦截..."
cat >"$CASE4_FILE" <<'CASE4'
package handler

import agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"

type badRepositoryDependency struct {
	repo agentdomain.AgentRepository
}
CASE4

if bash "$CHECK_SCRIPT" >/tmp/handler-guard-selftest-case4.log 2>&1; then
	echo "❌ 自测失败：handler 直接依赖 domain repository 接口未被拦截"
	cat /tmp/handler-guard-selftest-case4.log
	exit 1
fi

echo "✅ handler 边界守卫自测通过（四类违规均可被正确拦截）"
