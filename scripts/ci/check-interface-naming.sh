#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
SCAN_ROOT="${1:-${REPO_ROOT}}"

if ! command -v rg >/dev/null 2>&1; then
	echo "rg is required" >&2
	exit 1
fi

if [[ ! -d "${SCAN_ROOT}" ]]; then
	echo "scan root not found: ${SCAN_ROOT}" >&2
	exit 1
fi

cd "${SCAN_ROOT}"

failures=0

FRONTEND_A_CLASS_FILES=(
	frontend/lib/api-client.ts
	frontend/lib/response-parser.ts
	frontend/hooks/_shared/pagination.ts
	frontend/hooks/_shared/use-stable-pagination-info.ts
	frontend/hooks/use-notification-sse.ts
	frontend/hooks/use-targets/queries.ts
	frontend/hooks/use-endpoints.ts
	frontend/hooks/use-vulnerabilities/queries.ts
	frontend/hooks/use-organizations.ts
	frontend/hooks/_shared/targets-helpers.ts
	frontend/services/scheduled-scan.service.ts
	frontend/services/organization.service.ts
	frontend/services/endpoint.service.ts
	frontend/types/notification.types.ts
	frontend/types/command.types.ts
	frontend/types/endpoint.types.ts
	frontend/types/tool.types.ts
	frontend/types/subdomain.types.ts
	frontend/types/common.types.ts
	frontend/types/organization.types.ts
	frontend/types/target.types.ts
)

FRONTEND_B_CLASS_FILES=(
	frontend/services/worker.service.ts
	frontend/hooks/use-workers.ts
	frontend/types/worker.types.ts
)

FRONTEND_LEGACY_RESPONSE_FILES=(
	frontend/hooks/use-organizations.ts
	frontend/types/organization.types.ts
	frontend/types/target.types.ts
)

run_rg_existing_files() {
	local pattern="$1"
	shift
	local files=()
	local file

	for file in "$@"; do
		if [[ -f "${file}" ]]; then
			files+=("${file}")
		fi
	done

	if [[ ${#files[@]} -eq 0 ]]; then
		return 1
	fi

	rg -n -P "${pattern}" "${files[@]}" | rg -v '^[^:]+:[0-9]+:[[:space:]]*(//|/\*|\*)'
}

run_check() {
	local title="$1"
	local hint="$2"
	shift 2

	local output=""
	local status=0

	set +e
	output="$($@ 2>&1)"
	status=$?
	set -e

	if [[ ${status} -gt 1 ]]; then
		echo "❌ ${title}"
		echo "${output}"
		echo
		failures=$((failures + 1))
		return
	fi

	if [[ -n "${output}" ]]; then
		echo "❌ ${title}"
		echo "${hint}"
		echo "${output}"
		echo
		failures=$((failures + 1))
		return
	fi

	echo "✅ ${title}"
}

run_check \
	"snake_case JSON" \
	"JSON 字段应使用 camelCase；数据库列名、generated 文件和 provider 占位符除外。" \
	rg -n 'json:"[a-z0-9]+_[a-z0-9_]+(?:,[^"]*)?"' . \
	-g '*.go' \
	-g '!**/generated/**' \
	-g '!**/*.pb.go' \
	-g '!server/cmd/server/migrations/**' \
	-g '!server/internal/modules/catalog/repository/persistence/subfinder_provider_settings.go'

run_check \
	"bare snake_case zap" \
	"结构化日志字段不要使用裸 snake_case，请改为语义字段名或点分命名空间。" \
	rg -n 'zap\.[A-Za-z0-9_]+\("[a-z0-9]+_[a-z0-9_]+"' . \
	-g '*.go' \
	-g '!**/*_test.go' \
	-g '!**/generated/**'

run_check \
	"camelCase zap" \
	"结构化日志字段不要使用 camelCase key，请改为语义化点分命名或 OTel 语义字段。" \
	rg -n 'zap\.[A-Za-z0-9_]+\("[^"\n]*[A-Z][^"\n]*"' . \
	-g 'server/**/*.go' \
	-g 'worker/**/*.go' \
	-g '!**/*_test.go' \
	-g '!**/generated/**' \
	-g '!**/*.pb.go'

run_check \
	"raw Gin context key" \
	"requestId / userClaims / agentId / agent 必须通过 middleware accessor 访问。" \
	rg -n '\.(Get|Set|MustGet)\("(requestId|userClaims|agentId|agent)"' . \
	-g '*.go' \
	-g '!server/internal/middleware/context.go' \
	-g '!**/generated/**'

run_check \
	"snake_case error field" \
	"中间件与 gRPC 边界错误字段请使用 camelCase，例如 requestId / scanId / targetId / itemsJson。" \
	rg -n 'request_id|scan_id|target_id|items_json|agent_id' . \
	-g 'server/internal/grpc/**/*.go' \
	-g 'server/internal/middleware/**/*.go' \
	-g '!**/*_test.go'

run_check \
	"snake_case path param" \
	"显式命名 path param 请使用 camelCase，例如 :scanId / :targetId。" \
	rg -n '\.(GET|POST|PUT|PATCH|DELETE|Any|Group)\([^)]*:[a-z0-9]+_[a-z0-9_]+' . \
	-g '*.go' \
	-g '!**/generated/**'

run_check \
	"frontend boundary snake_case" \
	"前端 A 类边界代码只允许 camelCase；只检查审计范围内可执行代码，并显式排除 B 类 worker 链路文件。" \
	run_rg_existing_files '(?:\bpage_size\b|\btotal_pages\b|\bcreated_at\b|\bread_at\b|\bis_read\b|\btotal_count\b|\btarget_id\b|\borganization_id\b)' "${FRONTEND_A_CLASS_FILES[@]}"

run_check \
	"frontend legacy pagination compatibility" \
	"前端 A 类边界不要继续保留 count/next/previous 兼容字段或 response.count fallback。" \
	run_rg_existing_files '(?:\bresponse\.count\b|count\?:|next\?:|previous\?:)' "${FRONTEND_LEGACY_RESPONSE_FILES[@]}"

if [[ ${failures} -ne 0 ]]; then
	echo "interface naming checks failed" >&2
	exit 1
fi

echo "interface naming checks: ok"
