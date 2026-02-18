#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
  echo "ℹ️ 未找到模块目录，跳过命名规范检查: $MODULE_ROOT"
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

for bad in commands.go ports.go types_alias.go; do
  matches="$(find "$MODULE_ROOT" -type f -path '*/application/*' -name "$bad" | sort || true)"
  if [[ -n "$matches" ]]; then
    append_violation "禁止 application 使用泛名文件: $bad" "$matches"
  fi
done

handler_generic_files="$(find "$MODULE_ROOT" -type f \( -path '*/handler/types.go' -o -path '*/handler/helpers.go' -o -path '*/handler/ws_types.go' \) | sort || true)"
if [[ -n "$handler_generic_files" ]]; then
  append_violation "禁止 handler 使用泛名文件（types.go/helpers.go/ws_types.go）" "$handler_generic_files"
fi

service_matches="$(find "$MODULE_ROOT" -type f -path '*/application/service.go' | sort || true)"
if [[ -n "$service_matches" ]]; then
  append_violation "禁止 application 使用无资源前缀 service.go" "$service_matches"
fi

hostport_files="$(find "$ROOT_DIR/internal" -type f -name '*hostport*' | sort || true)"
if [[ -n "$hostport_files" ]]; then
  append_violation "禁止文件名使用 hostport（应统一为 host_port）" "$hostport_files"
fi

FACADE_FILES=()
while IFS= read -r file; do FACADE_FILES+=("$file"); done < <(find "$MODULE_ROOT" -type f -path '*/application/facade_*.go' | sort)

if [[ ${#FACADE_FILES[@]} -gt 0 ]]; then
  output="$(rg -n --no-heading -e 'func[[:space:]]+[A-Za-z0-9_]+(FromDTO|fromDTO|ToDTO|toDTO)\(' "${FACADE_FILES[@]}" || true)"
  if [[ -n "$output" ]]; then
    append_violation "禁止 facade_*.go 中定义 DTO 映射函数" "$output"
  fi
fi

CONTRACT_FILES=()
while IFS= read -r file; do CONTRACT_FILES+=("$file"); done < <(find "$MODULE_ROOT" -type f -path '*/application/contracts.go' | sort)
if [[ ${#CONTRACT_FILES[@]} -gt 0 ]]; then
	service_structs="$(rg -n --no-heading -e '^type[[:space:]]+[A-Za-z0-9_]*Service[[:space:]]+struct' "${CONTRACT_FILES[@]}" || true)"
	if [[ -n "$service_structs" ]]; then
		append_violation "禁止 contracts.go 定义 *Service struct" "$service_structs"
	fi

	service_methods="$(rg -n --no-heading -e '^func[[:space:]]*\([^)]*Service\)[[:space:]]+[A-Za-z0-9_]+' "${CONTRACT_FILES[@]}" || true)"
	if [[ -n "$service_methods" ]]; then
		append_violation "禁止 contracts.go 定义 Service 方法实现" "$service_methods"
	fi
fi

APP_FILES=()
while IFS= read -r file; do APP_FILES+=("$file"); done < <(find "$MODULE_ROOT" -type f -path '*/application/*.go' ! -name '*_test.go' | sort)
if [[ ${#APP_FILES[@]} -gt 0 ]]; then
  gorm_imports="$(rg -n --no-heading 'gorm\.io/gorm' "${APP_FILES[@]}" || true)"
  if [[ -n "$gorm_imports" ]]; then
    append_violation "禁止 application 非测试文件 import gorm.io/gorm（请在 repository/adapter 层处理）" "$gorm_imports"
  fi
fi

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ 命名规范检查失败"
  echo "$VIOLATIONS"
  exit 1
fi

echo "✅ 命名规范检查通过"
