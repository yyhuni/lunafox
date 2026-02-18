#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
  echo "ℹ️ 未找到模块目录，跳过 handler 边界检查: $MODULE_ROOT"
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

HANDLER_FILES=()
while IFS= read -r file; do HANDLER_FILES+=("$file"); done < <(
  find "$MODULE_ROOT" -type f \( -path '*/handler/*.go' -o -path '*/handler/*/*.go' \) ! -name '*_test.go' | sort
)

if [[ ${#HANDLER_FILES[@]} -eq 0 ]]; then
  echo "ℹ️ 未发现 handler 非测试文件，跳过 handler 边界检查"
  exit 0
fi

# Rule: handler top-level exported functions must be constructor-only (New*Handler)
violating_functions="$(
  rg -n --no-heading '^func[[:space:]]+[A-Z][A-Za-z0-9_]*\(' "${HANDLER_FILES[@]}" \
    | rg -v '^[^:]+:[0-9]+:func[[:space:]]+New[A-Za-z0-9_]*Handler\(' || true
)"
if [[ -n "$violating_functions" ]]; then
  append_violation "禁止 handler 暴露除 New*Handler 之外的顶层导出函数" "$violating_functions"
fi

# Rule: agent handler filenames must end with _handler.go or _mapper.go
agent_handler_files="$(printf '%s\n' "${HANDLER_FILES[@]}" | rg '/internal/modules/agent/handler/' || true)"
if [[ -n "$agent_handler_files" ]]; then
  invalid_handler_files="$(
    printf '%s\n' "$agent_handler_files" | rg -v '/[^/]+(_handler|_mapper)\.go$' || true
  )"
  if [[ -n "$invalid_handler_files" ]]; then
    append_violation "agent handler 文件命名必须为 *_handler.go 或 *_mapper.go" "$invalid_handler_files"
  fi
fi

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ handler 边界检查失败"
  echo "$VIOLATIONS"
  exit 1
fi

echo "✅ handler 边界检查通过"
