#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

adapters="$(find "$ROOT_DIR/internal/modules" -path '*/dto/common_http.go' -type f | sort)"

if [[ -n "$adapters" ]]; then
	echo "❌ module-local HTTP DTO adapters are retired"
	echo "以下文件应删除，并改为直接导入 server/internal/modules/httpdto："
	printf '%s\n' "$adapters" | sed 's/^/ - /'
	exit 1
fi

echo "✅ module-local HTTP DTO adapters are absent"
