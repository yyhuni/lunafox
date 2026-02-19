#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
	echo "ℹ️ 未找到模块目录，跳过 router 边界检查: $MODULE_ROOT"
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

expected_entries_for_module() {
	local module="$1"
	case "$module" in
	agent)
		echo "RegisterAgentRoutes"
		;;
	asset)
		echo "RegisterAssetRoutes RegisterHealthRoutes"
		;;
	catalog)
		echo "RegisterCatalogRoutes"
		;;
	identity)
		echo "RegisterIdentityRoutes"
		;;
	scan)
		echo "RegisterScanRoutes RegisterWorkerScanRoutes"
		;;
	security)
		echo "RegisterSecurityRoutes"
		;;
	snapshot)
		echo "RegisterScanSnapshotRoutes"
		;;
	*)
		echo ""
		;;
	esac
}

contains() {
	local target="$1"
	shift
	for item in "$@"; do
		if [[ "$item" == "$target" ]]; then
			return 0
		fi
	done
	return 1
}

while IFS= read -r module_dir; do
	module="$(basename "$module_dir")"
	router_dir="$module_dir/router"

	expected_raw="$(expected_entries_for_module "$module")"
	if [[ -z "$expected_raw" ]]; then
		append_violation "router 守卫未配置模块（请显式声明公开入口）" "$module_dir"
		continue
	fi

	ROUTER_FILES=()
	while IFS= read -r file; do ROUTER_FILES+=("$file"); done < <(
		find "$router_dir" -type f -name '*.go' ! -name '*_test.go' | sort
	)

	if [[ ${#ROUTER_FILES[@]} -eq 0 ]]; then
		append_violation "router 目录不能为空" "$router_dir"
		continue
	fi

	EXPORTED=()
	while IFS= read -r name; do
		[[ -z "$name" ]] && continue
		EXPORTED+=("$name")
	done < <(
		rg -n --no-heading '^func[[:space:]]+Register[A-Za-z0-9_]*Routes\(' "${ROUTER_FILES[@]}" |
			sed -E 's/.*func[[:space:]]+(Register[A-Za-z0-9_]*Routes)\(.*/\1/' |
			sort -u
	)

	if [[ ${#EXPORTED[@]} -eq 0 ]]; then
		append_violation "router 必须至少暴露一个 Register*Routes 入口" "$router_dir"
		continue
	fi

	read -r -a EXPECTED <<<"$expected_raw"

	unexpected=""
	for fn in "${EXPORTED[@]}"; do
		if ! contains "$fn" "${EXPECTED[@]}"; then
			if [[ -n "$unexpected" ]]; then
				unexpected+=$'\n'
			fi
			unexpected+="$router_dir -> 非法公开入口: $fn"
		fi
	done
	if [[ -n "$unexpected" ]]; then
		append_violation "router 出现未授权的公开入口函数（请改为小写私有 register*Routes）" "$unexpected"
	fi

	missing=""
	for fn in "${EXPECTED[@]}"; do
		if ! contains "$fn" "${EXPORTED[@]}"; then
			if [[ -n "$missing" ]]; then
				missing+=$'\n'
			fi
			missing+="$router_dir -> 缺少必需公开入口: $fn"
		fi
	done
	if [[ -n "$missing" ]]; then
		append_violation "router 缺少必需公开入口函数" "$missing"
	fi
done < <(find "$MODULE_ROOT" -mindepth 1 -maxdepth 1 -type d -exec test -d '{}/router' ';' -print | sort)

if [[ -n "$VIOLATIONS" ]]; then
	echo "❌ router 边界检查失败"
	echo "$VIOLATIONS"
	exit 1
fi

echo "✅ router 边界检查通过"
