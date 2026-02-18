#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-wiring-conventions.sh"
TMP_MODULE_DIR="$ROOT_DIR/internal/bootstrap/wiring/demoguard"
TMP_EXPORTS_FILE="$TMP_MODULE_DIR/exports.go"
TMP_ASSERTIONS_FILE="$TMP_MODULE_DIR/wiring_demoguard_adapter_assertions.go"

cleanup() {
  rm -rf "$TMP_MODULE_DIR"
}
trap cleanup EXIT

if [[ ! -x "$CHECK_SCRIPT" ]]; then
  echo "❌ 未找到可执行守卫脚本: $CHECK_SCRIPT"
  exit 1
fi

echo "[1/3] 校验缺少 adapter 断言文件可被拦截..."
mkdir -p "$TMP_MODULE_DIR"
cat > "$TMP_EXPORTS_FILE" <<'CASE1'
package demoguard

func NewDemoguardStoreAdapter(repo any) any { return repo }
CASE1

if bash "$CHECK_SCRIPT" >/tmp/wiring-guard-selftest-case1.log 2>&1; then
  echo "❌ 自测失败：缺少断言文件未被拦截"
  cat /tmp/wiring-guard-selftest-case1.log
  exit 1
fi
rm -rf "$TMP_MODULE_DIR"

echo "[2/3] 校验 exports 命名缺少模块前缀可被拦截..."
mkdir -p "$TMP_MODULE_DIR"
cat > "$TMP_EXPORTS_FILE" <<'CASE2'
package demoguard

func NewStoreAdapter(repo any) any { return repo }
CASE2
cat > "$TMP_ASSERTIONS_FILE" <<'CASE2_ASSERT'
package demoguard

var _ any = nil
CASE2_ASSERT

if bash "$CHECK_SCRIPT" >/tmp/wiring-guard-selftest-case2.log 2>&1; then
  echo "❌ 自测失败：exports 命名缺少模块前缀未被拦截"
  cat /tmp/wiring-guard-selftest-case2.log
  exit 1
fi
rm -rf "$TMP_MODULE_DIR"

echo "[3/3] 校验 Adapter 导出返回具体 struct 可被拦截..."
mkdir -p "$TMP_MODULE_DIR"
cat > "$TMP_EXPORTS_FILE" <<'CASE3'
package demoguard

type demoStoreAdapter struct{}

func NewDemoguardStoreAdapter(repo any) *demoStoreAdapter { return nil }
CASE3
cat > "$TMP_ASSERTIONS_FILE" <<'CASE3_ASSERT'
package demoguard

var _ any = nil
CASE3_ASSERT

if bash "$CHECK_SCRIPT" >/tmp/wiring-guard-selftest-case3.log 2>&1; then
  echo "❌ 自测失败：Adapter 返回具体 struct 未被拦截"
  cat /tmp/wiring-guard-selftest-case3.log
  exit 1
fi

echo "✅ wiring 规范守卫自测通过（3 类违规均可被正确拦截）"
