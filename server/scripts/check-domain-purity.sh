#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
	echo "ℹ️ 未找到模块目录，跳过 domain 纯度检查: $MODULE_ROOT"
	exit 0
fi

DOMAIN_FILES=()
while IFS= read -r file; do DOMAIN_FILES+=("$file"); done < <(find "$MODULE_ROOT" -type f -path '*/domain/*.go' ! -name '*_test.go' | sort)

if [[ ${#DOMAIN_FILES[@]} -eq 0 ]]; then
	echo "ℹ️ 未发现 domain 非测试文件，跳过 domain 纯度检查"
	exit 0
fi

VIOLATIONS=""
append_violation() {
	local title="$1"
	local body="$2"
	if [[ -n "$VIOLATIONS" ]]; then
		VIOLATIONS+=$'\n\n'
	fi
	VIOLATIONS+="$title"
	VIOLATIONS+=$'\n'
	VIOLATIONS+="$body"
}

output="$(rg -n --no-heading -e 'gorm:"' "${DOMAIN_FILES[@]}" || true)"
if [[ -n "$output" ]]; then
	append_violation "禁止 domain 层出现 gorm 标签" "$output"
fi

output="$(rg -n --no-heading -e 'internal/modules/.+/(application|repository|dto|handler|router)' "${DOMAIN_FILES[@]}" || true)"
if [[ -n "$output" ]]; then
	append_violation "禁止 domain 层反向依赖 application/repository/dto/handler/router" "$output"
fi

if [[ -n "$VIOLATIONS" ]]; then
	echo "❌ domain 纯度检查失败"
	echo "$VIOLATIONS"
	exit 1
fi

echo "✅ domain 纯度检查通过"
