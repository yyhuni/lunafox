#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECK_SCRIPT="$ROOT_DIR/scripts/check-repository-boundaries.sh"
TMP_MODULE="$ROOT_DIR/internal/modules/.repo_guard_selftest_tmp"

cleanup() {
	rm -rf "$TMP_MODULE"
}
trap cleanup EXIT

if [[ ! -x "$CHECK_SCRIPT" ]]; then
	echo "❌ 未找到可执行守卫脚本: $CHECK_SCRIPT"
	exit 1
fi

echo "[1/3] 校验 *_mutation.go 命名违规可被拦截..."
mkdir -p "$TMP_MODULE/demo/repository"
cat >"$TMP_MODULE/demo/repository/demo_mutation.go" <<'CASE1'
package repository
CASE1

if bash "$CHECK_SCRIPT" >/tmp/repo-guard-selftest-case1.log 2>&1; then
	echo "❌ 自测失败：*_mutation.go 违规未被拦截"
	cat /tmp/repo-guard-selftest-case1.log
	exit 1
fi
rm -rf "$TMP_MODULE"

echo "[2/3] 校验 query 文件写操作违规可被拦截..."
mkdir -p "$TMP_MODULE/demo/repository"
cat >"$TMP_MODULE/demo/repository/demo_query.go" <<'CASE2'
package repository

func (r *DemoRepository) Create() {}
CASE2

if bash "$CHECK_SCRIPT" >/tmp/repo-guard-selftest-case2.log 2>&1; then
	echo "❌ 自测失败：query 写操作违规未被拦截"
	cat /tmp/repo-guard-selftest-case2.log
	exit 1
fi
rm -rf "$TMP_MODULE"

echo "[3/3] 校验 command 文件查询操作违规可被拦截..."
mkdir -p "$TMP_MODULE/demo/repository"
cat >"$TMP_MODULE/demo/repository/demo_command.go" <<'CASE3'
package repository

func (r *DemoRepository) FindByID() {}
CASE3

if bash "$CHECK_SCRIPT" >/tmp/repo-guard-selftest-case3.log 2>&1; then
	echo "❌ 自测失败：command 查询操作违规未被拦截"
	cat /tmp/repo-guard-selftest-case3.log
	exit 1
fi

echo "✅ repository 守卫自测通过"
