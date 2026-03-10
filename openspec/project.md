# Project Context

## Purpose
LunaFox is a web-based security operations platform for asset discovery, scanning orchestration, vulnerability management, and distributed agent execution.  
Core goals are:
- Provide a unified control plane for scan scheduling and result aggregation.
- Offer operator-facing system observability and configuration pages.
- Keep backend module boundaries strict and maintainable under active feature growth.

## Tech Stack
- Frontend: Next.js 15, React 19, TypeScript, TanStack Query, next-intl, Tailwind CSS.
- Backend: Go 1.26, Gin, GORM, PostgreSQL, Redis.
- Tooling: pnpm, Vitest, Playwright scripts, ESLint, Go test, golangci-lint.

## Project Conventions

### Code Style
- Frontend uses TypeScript with camelCase naming and typed service/hook layers.
- API JSON uses camelCase fields.
- Structured logs use semantic field names; HTTP observability fields align with OpenTelemetry naming where applicable.
- Backend context data must be accessed through middleware helpers instead of raw Gin context key strings.
- Explicitly named path params use camelCase; generic resource routes may keep `:id`.
- Provider-constrained external labels (for example Loki / Prometheus labels) keep snake_case.
- Backend Go code follows strict package layering and explicit DTO boundaries.
- Keep changes small and explicit; avoid broad refactors when delivering focused features.

#### Boundary Naming Matrix
- Frontend audited HTTP boundary files use `camelCase` for request params, request bodies, response DTOs, and pagination metadata.
- Backend API JSON uses `camelCase` field names such as `requestId`, `pageSize`, and `totalPages`.
- Backend context data is accessed via middleware helpers; canonical keys are `requestId`, `userClaims`, `agentId`, and `agent`.
- Structured logs use semantic field names or dot namespaces such as `request.id` and `agent.id`; avoid raw `snake_case` log keys in application log bodies.
- Loki / Prometheus style labels remain provider-facing contracts and may use `snake_case`; prefer mapping selected low-cardinality fields at collection time instead of renaming application log fields.
- Explicit path params use `camelCase` such as `:scanId` and `:targetId`; generic `:id` routes remain allowed.
- Route-example comments, `.proto`, generated files, DB / SQL naming, and provider-constrained labels may keep upstream naming and stay outside frontend HTTP boundary enforcement.
- Documented legacy-hold frontend files are excluded from frontend boundary CI until their backend contract is confirmed.

### Architecture Patterns
- Frontend data access pattern: `services/*` for HTTP calls, `hooks/*` for React Query state, presentational components for rendering.
- Backend follows DDD-like modular boundaries: `router -> handler -> application -> repository -> domain`.
- Route registration is centralized in `server/internal/bootstrap/routes.go`.

### Testing Strategy
- Frontend: unit/component tests via Vitest, with route smoke and interaction scripts for broad UI regression checks.
- Backend: `go test ./...`, plus architecture guard scripts in `server/scripts`.
- For cross-layer changes, add or update tests at the seam where behavior changed (handler/service/hook).

### Git Workflow
- Work in focused feature branches, keep commits scoped by concern.
- Avoid rewriting unrelated history; preserve existing in-progress work.
- Prefer additive changes with clear rollback path for operational features.

## Domain Context
- The platform manages targets, scans, assets, vulnerabilities, and agent runtime operations.
- System settings pages surface operational status (logs, workers, disk, database health).
- Database health page is operator-facing and should express production-ready health semantics, not only demo metrics.

## Important Constraints
- Frontend requests backend through `/api/*` rewrite in Next.js.
- Most API routes are protected by JWT middleware.
- Trailing slash conventions are used in many existing endpoints.
- Some frontend pages support mock mode via `NEXT_PUBLIC_USE_MOCK`; production behavior must remain correct when mock is disabled.
- Health semantics must distinguish true unavailability from threshold-based degradation.

## External Dependencies
- PostgreSQL for primary persisted data and operational telemetry.
- Redis for cache/coordination.
- Optional distributed agents/workers for scan execution and telemetry backflow.
