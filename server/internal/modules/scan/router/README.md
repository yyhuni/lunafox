# scan/router

## Structure
- Public entry: `routes.go` (`RegisterScanRoutes`)
- Worker entry: `worker_routes.go` (`RegisterWorkerScanRoutes`)
- Other files contain package-private sub-route registrars

## Files
- `scans.go`: scan resource routes
- `scan_logs.go`: scan-log resource routes

## Route Matrix (mounted with `/api` prefix)
- `GET /api/scans`
- `POST /api/scans`
- `GET /api/scans/stats`
- `GET /api/scans/:id`
- `DELETE /api/scans/:id`
- `DELETE /api/scans/:id/permanent`
- `POST /api/scans/:id/stoppages`
- `POST /api/scans/deletions`
- `GET /api/scans/:id/logs`
- `POST /api/scans/:id/logs`
- `GET /api/worker/scans/:id/target`

## Constraints
- Register business routes through `RegisterScanRoutes`
- Keep worker routes in a dedicated entry to avoid mixing with business auth groups
