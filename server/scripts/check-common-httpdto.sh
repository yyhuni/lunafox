#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT_DIR/scripts/templates/common_http.go.tmpl"
MODULES=(agent asset catalog identity scan security snapshot)

if [[ ! -f "$TEMPLATE" ]]; then
  echo "❌ 未找到模板文件: $TEMPLATE"
  exit 1
fi

VIOLATIONS=""
append_violation() {
  local body="$1"
  if [[ -n "$VIOLATIONS" ]]; then
    VIOLATIONS+=$'\n'
  fi
  VIOLATIONS+="$body"
}

for module in "${MODULES[@]}"; do
  target="$ROOT_DIR/internal/modules/$module/dto/common_http.go"

  if [[ ! -f "$target" ]]; then
    append_violation "- 缺失文件: $target"
    continue
  fi

  if ! cmp -s "$TEMPLATE" "$target"; then
    append_violation "- 模板不一致: $target"
  fi
done

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ common_http.go 一致性检查失败"
  echo "以下文件与模板不一致（或缺失）："
  echo "$VIOLATIONS"
  echo
  echo "可执行: bash ./scripts/sync-common-httpdto.sh 进行同步"
  exit 1
fi

echo "✅ common_http.go 一致性检查通过"
