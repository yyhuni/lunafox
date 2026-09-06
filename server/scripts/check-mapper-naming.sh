#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

TARGETS=()
if [[ -d "$ROOT_DIR/internal/modules" ]]; then
	TARGETS+=("$ROOT_DIR/internal/modules")
fi
if [[ -d "$ROOT_DIR/internal/bootstrap" ]]; then
	TARGETS+=("$ROOT_DIR/internal/bootstrap")
fi

if [[ ${#TARGETS[@]} -eq 0 ]]; then
	echo "ℹ️ 未找到守卫目标目录，跳过 mapper 命名检查"
	exit 0
fi

violations="$(rg -n --no-heading \
	--glob '!**/*_test.go' \
	--glob '*.go' \
	-e 'func[[:space:]]+(to|new)[A-Za-z0-9_]+Response\(' \
	"${TARGETS[@]}" || true)"

if [[ -n "$violations" ]]; then
	echo "❌ Mapper 命名守卫检查失败"
	echo "禁止命名: to*Response / new*Response（请改为 *Output）"
	echo
	echo "$violations"
	exit 1
fi

echo "✅ Mapper 命名守卫检查通过"
