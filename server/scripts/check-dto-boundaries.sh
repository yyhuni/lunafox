#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
  echo "ℹ️ 未找到模块目录，跳过 DTO 边界检查: $MODULE_ROOT"
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

LEGACY_DTO_MODEL_FILES="$(find "$MODULE_ROOT" -type f \( -path '*/dto/models.go' -o -path '*/dto/*_models.go' \) | sort || true)"
if [[ -n "$LEGACY_DTO_MODEL_FILES" ]]; then
  append_violation "禁止 DTO 使用旧命名（models.go 或 *_models.go）" "$LEGACY_DTO_MODEL_FILES"
fi

DTO_FILES=()
while IFS= read -r file; do
  DTO_FILES+=("$file")
done < <(find "$MODULE_ROOT" -type f \( -path "*/dto/dto.go" -o -path "*/dto/*_dto.go" \) | sort)

if [[ ${#DTO_FILES[@]} -gt 0 ]]; then
  output="$(rg -n --no-heading -e "github.com/yyhuni/lunafox/server/internal/dto" "${DTO_FILES[@]}" || true)"
  if [[ -n "$output" ]]; then
    append_violation "禁止 modules/*/dto DTO 文件依赖 shared 业务 DTO（server/internal/dto）" "$output"
  fi

  output="$(rg -n --no-heading -e "type[[:space:]]+[A-Za-z0-9_]+[[:space:]]*=[[:space:]]*shared\." "${DTO_FILES[@]}" || true)"
  if [[ -n "$output" ]]; then
    append_violation "禁止 modules/*/dto DTO 文件通过 shared 别名复用 DTO" "$output"
  fi
fi

SNAPSHOT_DTO_FILES=()
if [[ -d "$MODULE_ROOT/snapshot/dto" ]]; then
  while IFS= read -r file; do
    SNAPSHOT_DTO_FILES+=("$file")
  done < <(find "$MODULE_ROOT/snapshot/dto" -type f -name "*.go" | sort)
fi

if [[ ${#SNAPSHOT_DTO_FILES[@]} -gt 0 ]]; then
  output="$(rg -n --no-heading -e "github.com/yyhuni/lunafox/server/internal/modules/(asset|security)/dto" "${SNAPSHOT_DTO_FILES[@]}" || true)"
  if [[ -n "$output" ]]; then
    append_violation "禁止 snapshot/dto 直接依赖 asset/security dto" "$output"
  fi
fi

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ DTO 边界检查失败"
  echo "DTO 文件（dto.go 或 *_dto.go）仅允许模块私有业务 DTO；snapshot/dto 禁止直接依赖 asset/security dto；禁止沿用 *_models.go 旧命名。"
  echo
  echo "$VIOLATIONS"
  exit 1
fi

echo "✅ DTO 边界检查通过（DTO 文件无 shared 业务别名，snapshot/dto 无 asset/security 依赖，且未使用 *_models.go 旧命名）"
