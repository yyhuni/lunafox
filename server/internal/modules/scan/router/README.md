# scan/router

## Structure
- Public entry: `routes.go` (`RegisterScanRoutes`)
- Other files contain package-private sub-route registrars

## Files
- `scans.go`: scan resource routes
- `task_progress_logs.go`: task-progress-log resource routes

## Route Matrix (mounted with `/v1` prefix)
The matrix below documents the current route registration. Several entries still use legacy HTTP shapes (`:id`, action-like paths, and custom pagination); they must not be copied as target style for new or migrated AIP-governed boundaries. Use `openspec/changes/standardize-project-boundary-contracts/` for canonical resource names, custom methods, async operation/task mapping, and pagination rules.

- `GET /v1/scans`
- `POST /v1/scans:batchCreate`
- `POST /v1/scans:batchDelete`
- `POST /v1/scans:batchStop`
- `POST /v1/scans:quickCreate`
- `GET /v1/scanStatistics`
- `GET /v1/scans/:scan`
- `POST /v1/scans/:scan:stop`
- `DELETE /v1/scans/:scan`
- `GET /v1/scans/:scan/taskProgressLogs`

Batch stop accepts `{ "names": ["scans/12", "..."] }` with 1–100 unique
canonical Scan resource names and returns `stoppedCount`, `skippedCount`,
and `revokedTaskCount`. Terminal selections are successful skips; malformed,
duplicate, missing, or deleted names are request errors.

## Constraints
- Register business routes through `RegisterScanRoutes`
- Normal scan creation uses `POST /v1/scans:batchCreate` for one or more target/organization scopes; do not reintroduce `POST /v1/scans` as a create route.
- Gin stores collection custom methods through an internal `/scans:customMethod` dispatch route; client-facing paths remain the canonical AIP custom methods listed above.
- Task progress log writes are runtime data-plane operations and are not exposed through HTTP routes.
