#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$APP_ROOT" ]]; then
	echo "ℹ️ 未找到 application 模块目录，跳过 application composition 检查: $APP_ROOT"
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

while IFS= read -r facade_file; do
	direct_child_service_assembly="$(
		awk '
			/^func New[A-Za-z0-9_]*(Facade|ApplicationService)\(/ { in_constructor=1 }
			in_constructor && /^[[:space:]]*}/ { in_constructor=0 }
			in_constructor && /New[A-Za-z0-9_]*(Query|Command|Create|Lifecycle|Registration|Bridge|Snapshot|Service)Service\(/ { print FILENAME ":" FNR ":" $0 }
		' "$facade_file" || true
	)"
	if [[ -n "$direct_child_service_assembly" ]]; then
		append_violation "application facade/application-boundary constructor 不得直接组装子 service；请在 bootstrap/wiring composition root 中组装后注入" "$direct_child_service_assembly"
	fi
done < <(find "$APP_ROOT" -path "*/application/facade*.go" -type f | sort)

if [[ -n "$VIOLATIONS" ]]; then
	echo "❌ application composition 检查失败"
	echo "$VIOLATIONS"
	exit 1
fi

echo "✅ application composition 检查通过"
