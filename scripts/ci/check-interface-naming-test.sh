#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHECK_SCRIPT="${REPO_ROOT}/scripts/ci/check-interface-naming.sh"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

write_file() {
  local path="$1"
  mkdir -p "$(dirname "${path}")"
  cat > "${path}"
}

run_expect_pass() {
  local name="$1"
  local root="$2"

  if ! output="$(bash "${CHECK_SCRIPT}" "${root}" 2>&1)"; then
    echo "expected ${name} to pass"
    echo "${output}"
    return 1
  fi
}

run_expect_fail() {
  local name="$1"
  local root="$2"
  local expected="$3"

  if output="$(bash "${CHECK_SCRIPT}" "${root}" 2>&1)"; then
    echo "expected ${name} to fail"
    return 1
  fi

  if [[ "${output}" != *"${expected}"* ]]; then
    echo "expected ${name} output to contain: ${expected}"
    echo "actual output:"
    echo "${output}"
    return 1
  fi
}

GOOD_ROOT="${WORK_DIR}/good"
write_file "${GOOD_ROOT}/server/internal/sample/good.go" <<'GO'
package sample

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type payload struct {
	RequestID string `json:"requestId"`
}

func ok(c *gin.Context, logger *zap.Logger, agentID int) {
	logger.Info("ok", zap.String("request.id", "req-1"), zap.Int("agent.id", agentID))
	c.JSON(200, gin.H{"requestId": "req-1"})
	_ = fmt.Errorf("invalid field scanId")
	_ = c.Param("scanId")
}
GO
write_file "${GOOD_ROOT}/server/internal/modules/catalog/repository/persistence/subfinder_provider_settings.go" <<'GO'
package persistence

type ProviderSettings struct {
	APIKey string `json:"api_key,omitempty"`
}
GO
write_file "${GOOD_ROOT}/server/internal/modules/agent/application/loki_log_query_service.go" <<'GO'
package application

import "fmt"

func selector() string {
	return fmt.Sprintf("{agent_id=%q,container_name=%q}", "1", "agent")
}
GO
write_file "${GOOD_ROOT}/server/internal/modules/scan/repository/sql.go" <<'GO'
package repository

const query = "SELECT * FROM scan_task WHERE target_id = ?"
GO
write_file "${GOOD_ROOT}/frontend/services/endpoint.service.ts" <<'TS'
export async function ok() {
  // GET /api/targets/{target_id}/endpoints/
  return {
    params: {
      pageSize: 10,
      organizationId: 1,
    },
  }
}
TS
write_file "${GOOD_ROOT}/frontend/types/organization.types.ts" <<'TS'
export interface OrganizationsResponse<T = unknown> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}
TS

BAD_JSON_ROOT="${WORK_DIR}/bad-json"
write_file "${BAD_JSON_ROOT}/server/internal/sample/bad_json.go" <<'GO'
package sample

type payload struct {
	RequestID string `json:"request_id"`
}
GO

BAD_LOG_ROOT="${WORK_DIR}/bad-log"
write_file "${BAD_LOG_ROOT}/server/internal/sample/bad_log.go" <<'GO'
package sample

import "go.uber.org/zap"

func bad(logger *zap.Logger) {
	logger.Info("bad", zap.String("request_id", "req-1"))
}
GO

BAD_CONTEXT_ROOT="${WORK_DIR}/bad-context"
write_file "${BAD_CONTEXT_ROOT}/server/internal/sample/bad_context.go" <<'GO'
package sample

import "github.com/gin-gonic/gin"

func bad(c *gin.Context) string {
	value, _ := c.Get("requestId")
	_, _ = c.Get("agentId")
	return value.(string)
}
GO

BAD_ERROR_ROOT="${WORK_DIR}/bad-error"
write_file "${BAD_ERROR_ROOT}/server/internal/grpc/sample/bad_error.go" <<'GO'
package sample

import "fmt"

func bad() error {
	return fmt.Errorf("missing scan_id")
}
GO

BAD_PATH_ROOT="${WORK_DIR}/bad-path"
write_file "${BAD_PATH_ROOT}/server/internal/modules/sample/handler/bad_path.go" <<'GO'
package handler

import "github.com/gin-gonic/gin"

func register(router *gin.RouterGroup) {
	router.GET("/:scan_id", func(c *gin.Context) {})
}
GO

BAD_FRONTEND_SNAKE_ROOT="${WORK_DIR}/bad-frontend-snake"
write_file "${BAD_FRONTEND_SNAKE_ROOT}/frontend/services/scheduled-scan.service.ts" <<'TS'
export async function bad() {
  const params = {
    pageSize: 10,
    target_id: 1,
  }
  return params
}
TS


BAD_CAMEL_ZAP_ROOT="${WORK_DIR}/bad-camel-zap"
write_file "${BAD_CAMEL_ZAP_ROOT}/worker/internal/sample/bad_camel_zap.go" <<'GO'
package sample

import "go.uber.org/zap"

func bad(logger *zap.Logger) {
	logger.Info("bad", zap.Int("taskId", 1), zap.Int("queueLength", 2))
}
GO

GOOD_OTEL_ZAP_ROOT="${WORK_DIR}/good-otel-zap"
write_file "${GOOD_OTEL_ZAP_ROOT}/server/internal/sample/good_otel_zap.go" <<'GO'
package sample

import "go.uber.org/zap"

func good(logger *zap.Logger) {
	logger.Info("ok",
		zap.Int("http.response.status_code", 200),
		zap.Int64("http.server.request.duration_ms", 12),
		zap.String("user_agent.original", "curl/8.0"),
	)
}
GO

BAD_FRONTEND_LEGACY_ROOT="${WORK_DIR}/bad-frontend-legacy"
write_file "${BAD_FRONTEND_LEGACY_ROOT}/frontend/types/organization.types.ts" <<'TS'
export interface OrganizationsResponse<T = unknown> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
  count?: number
}
TS

run_expect_pass "good fixture" "${GOOD_ROOT}"
run_expect_pass "good otel zap fixture" "${GOOD_OTEL_ZAP_ROOT}"
run_expect_fail "bad json fixture" "${BAD_JSON_ROOT}" "snake_case JSON"
run_expect_fail "bad log fixture" "${BAD_LOG_ROOT}" "bare snake_case zap"
run_expect_fail "bad camel zap fixture" "${BAD_CAMEL_ZAP_ROOT}" "camelCase zap"
run_expect_fail "bad context fixture" "${BAD_CONTEXT_ROOT}" "raw Gin context key"
run_expect_fail "bad error fixture" "${BAD_ERROR_ROOT}" "snake_case error field"
run_expect_fail "bad path fixture" "${BAD_PATH_ROOT}" "snake_case path param"
run_expect_fail "bad frontend snake fixture" "${BAD_FRONTEND_SNAKE_ROOT}" "frontend boundary snake_case"
run_expect_fail "bad frontend legacy fixture" "${BAD_FRONTEND_LEGACY_ROOT}" "frontend legacy pagination compatibility"

echo "interface naming checks: ok"
