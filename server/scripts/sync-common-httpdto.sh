#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT_DIR/scripts/templates/common_http.go.tmpl"

MODULES=(agent asset catalog identity scan security snapshot)

if [[ ! -f "$TEMPLATE" ]]; then
  echo "❌ 未找到模板文件: $TEMPLATE"
  exit 1
fi

for module in "${MODULES[@]}"; do
  target="$ROOT_DIR/internal/modules/$module/dto/common_http.go"
  if [[ ! -f "$target" ]]; then
    echo "❌ 未找到目标文件: $target"
    exit 1
  fi
  cp "$TEMPLATE" "$target"
  gofmt -w "$target"
  echo "同步完成: $target"
done

echo "✅ common_http.go 模板同步完成"
