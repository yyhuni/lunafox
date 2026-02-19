#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_ROOT="$ROOT_DIR/internal/modules"

if [[ ! -d "$MODULE_ROOT" ]]; then
	echo "ℹ️ 未找到模块目录，跳过 repository 边界检查: $MODULE_ROOT"
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

mutation_files="$(find "$MODULE_ROOT" -type f -path '*/repository/*_mutation.go' | sort || true)"
if [[ -n "$mutation_files" ]]; then
	append_violation "禁止 repository 使用 *_mutation.go 命名" "$mutation_files"
fi

types_files="$(find "$MODULE_ROOT" -type f -path '*/repository/types.go' | sort || true)"
if [[ -n "$types_files" ]]; then
	append_violation "禁止 repository 使用泛名 types.go（请改为具体资源文件）" "$types_files"
fi

# Mapper files must use resource prefix: <resource>_mapper.go and have sibling <resource>.go
MAPPER_FILES=()
while IFS= read -r file; do
	MAPPER_FILES+=("$file")
done < <(find "$MODULE_ROOT" -type f -path '*/repository/*_mapper.go' | sort)

if [[ ${#MAPPER_FILES[@]} -gt 0 ]]; then
	invalid_mapper_files=""
	for file in "${MAPPER_FILES[@]}"; do
		file_dir="$(dirname "$file")"
		file_name="$(basename "$file")"
		resource_name="${file_name%_mapper.go}"
		resource_file="$file_dir/$resource_name.go"

		if [[ ! -f "$resource_file" ]]; then
			if [[ -n "$invalid_mapper_files" ]]; then
				invalid_mapper_files+=$'\n'
			fi
			invalid_mapper_files+="$file -> 缺少对应资源文件: $resource_file"
		fi
	done

	if [[ -n "$invalid_mapper_files" ]]; then
		append_violation "禁止聚合式泛名 mapper：*_mapper.go 必须对应同名 <resource>.go" "$invalid_mapper_files"
	fi
fi

QUERY_FILES=()
while IFS= read -r file; do
	QUERY_FILES+=("$file")
done < <(find "$MODULE_ROOT" -type f -path '*/repository/*_query.go' | sort)

if [[ ${#QUERY_FILES[@]} -gt 0 ]]; then
	output="$(rg -n --no-heading -e 'func[[:space:]]+\(.*\)[[:space:]]+(Create|Update|Delete|Bulk[A-Za-z0-9_]*|SoftDelete|Mark[A-Za-z0-9_]*|Save|Cancel[A-Za-z0-9_]*|Fail[A-Za-z0-9_]*|Unlock[A-Za-z0-9_]*|Unlink[A-Za-z0-9_]*|Add[A-Za-z0-9_]*)\(' "${QUERY_FILES[@]}" || true)"
	if [[ -n "$output" ]]; then
		append_violation "禁止 *_query.go 中出现写操作方法" "$output"
	fi
fi

COMMAND_FILES=()
while IFS= read -r file; do
	COMMAND_FILES+=("$file")
done < <(find "$MODULE_ROOT" -type f -path '*/repository/*_command.go' | sort)

if [[ ${#COMMAND_FILES[@]} -gt 0 ]]; then
	output="$(rg -n --no-heading -e 'func[[:space:]]+\(.*\)[[:space:]]+(Find[A-Za-z0-9_]*|Get[A-Za-z0-9_]*|List|Exists[A-Za-z0-9_]*|Count[A-Za-z0-9_]*|Stream[A-Za-z0-9_]*|ScanRow|Pull[A-Za-z0-9_]*)\(' "${COMMAND_FILES[@]}" || true)"
	if [[ -n "$output" ]]; then
		append_violation "禁止 *_command.go 中出现查询方法" "$output"
	fi
fi

# each module should provide repository/persistence as persistence home
missing_persistence=""
for mdir in "$MODULE_ROOT"/*; do
	[[ -d "$mdir" ]] || continue
	if [[ -d "$mdir/repository" && ! -d "$mdir/repository/persistence" ]]; then
		if [[ -n "$missing_persistence" ]]; then
			missing_persistence+=$'\n'
		fi
		missing_persistence+="$mdir/repository/persistence"
	fi
done
if [[ -n "$missing_persistence" ]]; then
	append_violation "每个模块必须存在 repository/persistence 目录" "$missing_persistence"
fi

if [[ -n "$VIOLATIONS" ]]; then
	echo "❌ repository 边界检查失败"
	echo "$VIOLATIONS"
	exit 1
fi

echo "✅ repository 边界检查通过"
