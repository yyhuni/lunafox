#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
  echo "ℹ️ 未找到模块目录，跳过 legacy 层检查: $MODULE_ROOT"
  exit 0
fi

legacy_service="$(find "$MODULE_ROOT" -type d -name service | sort || true)"
legacy_model="$(find "$MODULE_ROOT" -type d -name model | sort || true)"

if [[ -n "$legacy_service" || -n "$legacy_model" ]]; then
  echo "❌ legacy 层检查失败"
  [[ -n "$legacy_service" ]] && echo "仍存在 service 目录:" && echo "$legacy_service"
  [[ -n "$legacy_model" ]] && echo "仍存在 model 目录:" && echo "$legacy_model"
  exit 1
fi

echo "✅ legacy 层检查通过（无 modules/**/service 与 modules/**/model）"
