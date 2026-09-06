#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-layer-dependencies.sh"
CATALOG_WIRING_DIR="$ROOT_DIR/internal/bootstrap/wiring/catalog"

CASE1_FILE="$CATALOG_WIRING_DIR/_selftest_mappers.go"
CASE2_FILE="$CATALOG_WIRING_DIR/_selftest_model_to_domain.go"
CASE3_FILE="$CATALOG_WIRING_DIR/_selftest_persistence_import.go"

cleanup() {
	rm -f "$CASE1_FILE" "$CASE2_FILE" "$CASE3_FILE"
}
trap cleanup EXIT

if [[ ! -x "$CHECK_SCRIPT" ]]; then
	echo "❌ 未找到可执行守卫脚本: $CHECK_SCRIPT"
	exit 1
fi

if [[ ! -d "$CATALOG_WIRING_DIR" ]]; then
	echo "❌ 未找到目录: $CATALOG_WIRING_DIR"
	exit 1
fi

echo "[1/3] 校验 catalog wiring *mappers.go 违规可被拦截..."
cat >"$CASE1_FILE" <<'CASE1'
package catalogwiring
CASE1

if bash "$CHECK_SCRIPT" >/tmp/layer-guard-selftest-case1.log 2>&1; then
	echo "❌ 自测失败：catalog wiring *mappers.go 违规未被拦截"
	cat /tmp/layer-guard-selftest-case1.log
	exit 1
fi
rm -f "$CASE1_FILE"

echo "[2/3] 校验 catalog wiring ModelToDomain/DomainToModel 违规可被拦截..."
cat >"$CASE2_FILE" <<'CASE2'
package catalogwiring

func demoModelToDomain() {}
CASE2

if bash "$CHECK_SCRIPT" >/tmp/layer-guard-selftest-case2.log 2>&1; then
	echo "❌ 自测失败：catalog wiring 映射函数违规未被拦截"
	cat /tmp/layer-guard-selftest-case2.log
	exit 1
fi
rm -f "$CASE2_FILE"

echo "[3/3] 校验 catalog wiring 依赖 persistence 违规可被拦截..."
cat >"$CASE3_FILE" <<'CASE3'
package catalogwiring

import _ "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
CASE3

if bash "$CHECK_SCRIPT" >/tmp/layer-guard-selftest-case3.log 2>&1; then
	echo "❌ 自测失败：catalog wiring persistence 依赖违规未被拦截"
	cat /tmp/layer-guard-selftest-case3.log
	exit 1
fi

echo "✅ layer 依赖守卫自测通过（catalog wiring 三类违规均可被正确拦截）"
