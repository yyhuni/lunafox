# snapshot/router

## Structure
- Public entry: `snapshots.go` (`RegisterScanSnapshotRoutes`)
- This route group handles scan snapshot writes (`bulk-upsert` / `bulk-create`) and query/export APIs.

## Route Matrix (mounted with `/api` prefix)

### Scan-scoped snapshots (`/api/scans/:id/*`)
- `POST /api/scans/:id/websites/bulk-upsert`
- `POST /api/scans/:id/subdomains/bulk-upsert`
- `POST /api/scans/:id/endpoints/bulk-upsert`
- `POST /api/scans/:id/directories/bulk-upsert`
- `POST /api/scans/:id/host-ports/bulk-upsert`
- `POST /api/scans/:id/screenshots/bulk-upsert`
- `POST /api/scans/:id/vulnerabilities/bulk-create`
- Plus corresponding `GET` list/export routes

### Global vulnerability snapshot queries
- `GET /api/vulnerability-snapshots`
- `GET /api/vulnerability-snapshots/:id`

## Mapping from asset upsert routes

Target-scoped upsert routes in `asset/router` are delegated to snapshot handlers:

- `/api/targets/:id/websites/bulk-upsert` -> `WebsiteSnapshotHandler.BulkUpsert`
- `/api/targets/:id/endpoints/bulk-upsert` -> `EndpointSnapshotHandler.BulkUpsert`
- `/api/targets/:id/directories/bulk-upsert` -> `DirectorySnapshotHandler.BulkUpsert`
- `/api/targets/:id/host-ports/bulk-upsert` -> `HostPortSnapshotHandler.BulkUpsert`
- `/api/targets/:id/screenshots/bulk-upsert` -> `ScreenshotSnapshotHandler.BulkUpsert`

## Runtime notes
Worker runtime writes no longer enter via HTTP `/api/worker/*` routes. Runtime batch upserts are handled by the gRPC runtime data proxy and reuse the same snapshot application services.
