#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INTERNAL_ROOT="$ROOT_DIR/internal"

if [[ ! -d "$INTERNAL_ROOT" ]]; then
  echo "ℹ️ 未找到 worker internal 目录，跳过命名规范检查: $INTERNAL_ROOT"
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

generic_files="$(find "$INTERNAL_ROOT" -type f \( \
  -name 'types.go' \
  -o -name 'common.go' \
  -o -name 'helpers.go' \
  -o -name '*_types.go' \
  -o -name '*_common.go' \
\) | sort || true)"
if [[ -n "$generic_files" ]]; then
  append_violation "禁止 worker/internal 使用泛名文件（types/common/helpers 及 *_types.go）" "$generic_files"
fi

generic_ports="$(find "$INTERNAL_ROOT" -type f -name 'ports.go' | sort || true)"
if [[ -n "$generic_ports" ]]; then
  append_violation "禁止 worker/internal 使用泛名 ports.go（请使用资源化 *_ports.go）" "$generic_ports"
fi

if [[ -n "$VIOLATIONS" ]]; then
  echo "❌ Worker 命名规范检查失败"
  echo "$VIOLATIONS"
  exit 1
fi

echo "✅ Worker 命名规范检查通过"
