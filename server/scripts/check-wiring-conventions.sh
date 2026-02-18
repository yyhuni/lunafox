#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WIRING_ROOT="$ROOT_DIR/internal/bootstrap/wiring"

if [[ ! -d "$WIRING_ROOT" ]]; then
  echo "ℹ️ 未找到 wiring 目录，跳过 wiring 规范检查: $WIRING_ROOT"
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

module_prefix() {
  local module="$1"
  case "$module" in
    asset) echo "Asset" ;;
    catalog) echo "Catalog" ;;
    identity) echo "Identity" ;;
    scan) echo "Scan" ;;
    scanlog) echo "ScanLog" ;;
    security) echo "Security" ;;
    snapshot) echo "Snapshot" ;;
    worker) echo "Worker" ;;
    *)
      local first="${module:0:1}"
      local rest="${module:1}"
      echo "${first^^}${rest}"
      ;;
  esac
}

MODULE_DIRS=()
while IFS= read -r dir; do MODULE_DIRS+=("$dir"); done < <(find "$WIRING_ROOT" -mindepth 1 -maxdepth 1 -type d | sort)

for module_dir in "${MODULE_DIRS[@]}"; do
  module="$(basename "$module_dir")"
  prefix="$(module_prefix "$module")"

  exports_file="$module_dir/exports.go"
  assertions_file="$module_dir/wiring_${module}_adapter_assertions.go"

  if [[ ! -f "$exports_file" ]]; then
    append_violation "wiring/$module 缺少 exports.go" "$exports_file"
    continue
  fi

  if [[ ! -f "$assertions_file" ]]; then
    append_violation "wiring/$module 缺少 adapter 断言文件" "$assertions_file"
  else
    assertions_count="$(rg -n --no-heading '^var[[:space:]]+_[[:space:]]+' "$assertions_file" | wc -l | tr -d '[:space:]')"
    if [[ "$assertions_count" == "0" ]]; then
      append_violation "wiring/$module adapter 断言文件不能为空" "$assertions_file"
    fi
  fi

  exported_non_new="$(
    rg -n --no-heading '^func[[:space:]]+[A-Z][A-Za-z0-9_]*\(' "$exports_file" \
      | rg -v '^([^:]+:)?[0-9]+:func[[:space:]]+New[A-Za-z0-9_]*\(' || true
  )"
  if [[ -n "$exported_non_new" ]]; then
    append_violation "wiring/$module exports.go 导出函数必须以 New 开头" "$exported_non_new"
  fi

  named_exports="$(rg -n --no-heading '^func[[:space:]]+New[A-Za-z0-9_]*(Adapter|ApplicationService|Codec)\(' "$exports_file" || true)"
  if [[ -n "$named_exports" ]]; then
    invalid_prefix="$(
      printf '%s\n' "$named_exports" \
        | rg -v "^([^:]+:)?[0-9]+:func[[:space:]]+New${prefix}[A-Za-z0-9_]*(Adapter|ApplicationService|Codec)\\(" || true
    )"
    if [[ -n "$invalid_prefix" ]]; then
      append_violation "wiring/$module exports 命名需使用模块前缀 New${prefix}*" "$invalid_prefix"
    fi
  fi

  adapter_concrete_return="$(rg -n --no-heading '^func[[:space:]]+New[A-Za-z0-9_]*Adapter\([^)]*\)[[:space:]]+\*[a-z][A-Za-z0-9_]*Adapter' "$exports_file" || true)"
  if [[ -n "$adapter_concrete_return" ]]; then
    append_violation "wiring/$module Adapter 导出函数应返回 application/domain 接口，而非具体 adapter struct" "$adapter_concrete_return"
  fi
done

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ wiring 规范检查失败"
  echo "$VIOLATIONS"
  exit 1
fi

echo "✅ wiring 规范检查通过"
