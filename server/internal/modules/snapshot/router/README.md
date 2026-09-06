# snapshot/router

## Structure
- Public entry: `snapshots.go` (`RegisterScanSnapshotRoutes`)
- This route group handles scan snapshot batch ingest/create writes and query/export APIs.
- Do not use these legacy paths as the target style for new boundaries. The target boundary standard is AIP-style resource names, explicit resource path variables, and `:batchIngest` / `:batchReconcile` or AIP batch methods with exceptions registered in `openspec/changes/standardize-project-boundary-contracts/`.

## Route Matrix (mounted with `/v1` prefix)

### Current scan-scoped snapshots (`/v1/scans/:scan/*`)
- `POST /v1/scans/:scan/websites:batchIngest`
- `POST /v1/scans/:scan/subdomains:batchIngest`
- `POST /v1/scans/:scan/endpoints:batchIngest`
- `POST /v1/scans/:scan/directories:batchIngest`
- `POST /v1/scans/:scan/hostPorts:batchIngest`
- `POST /v1/scans/:scan/screenshots:batchIngest`
- `POST /v1/scans/:scan/vulnerabilities:batchCreate`
- Plus corresponding `GET` list/export routes

### Migrated scan subdomain list query

`GET /v1/scans/:scan/subdomains` is migrated to the standard backend business-list query contract:

- Canonical query parameters: `pageSize`, `pageToken`, `filter`, `orderBy`.
- Hard-cut legacy aliases: `page`, `sort`, `sortBy`, `sortOrder`, `keyword` are rejected before repository access.
- Supported filter field: `dnsName` only, mapped to `subdomain_snapshot.dns_name` as controlled contains matching.
- Supported `orderBy` fields: `dnsName` and `createdAt`; default is `createdAt desc`.
- Stable SQL ordering always appends `id`: `dns_name ASC/DESC, id ASC/DESC` or `created_at ASC/DESC, id ASC/DESC`.
- `pageToken` is opaque and bound to `scanId`, normalized `filter`, normalized `orderBy`, and `pageSize`; tokens must not be reused across scans or query shapes.
- Baseline PostgreSQL indexes backing public search/sort fields are `idx_subdomain_snap_dns_name_trgm`, `idx_subdomain_snap_scan_created_at_id`, and `idx_subdomain_snap_scan_dns_name_id`.

Other scan snapshot list routes in this module remain on their existing query contracts until their own migration batch lands; do not expose sorting/filter affordances for them by copying the subdomain behavior.

### Global vulnerability snapshot queries
- `GET /v1/vulnerabilitySnapshots`
- `GET /v1/vulnerabilitySnapshots/:id`

## Current mapping from asset ingest routes

Target-scoped upsert routes in `asset/router` are delegated to snapshot handlers:

- `/v1/targets/:target/websites:batchIngest` -> `WebsiteSnapshotHandler.BatchIngest`
- `/v1/targets/:target/endpoints:batchIngest` -> `EndpointSnapshotHandler.BatchIngest`
- `/v1/targets/:target/directories:batchIngest` -> `DirectorySnapshotHandler.BatchIngest`
- `/v1/targets/:target/hostPorts:batchIngest` -> `HostPortSnapshotHandler.BatchIngest`
- `/v1/targets/:target/screenshots:batchIngest` -> `ScreenshotSnapshotHandler.BatchIngest`

## Runtime notes
Worker runtime writes no longer enter via HTTP `/v1/worker/*` routes. Runtime batch upserts are handled by the gRPC runtime data-plane service and reuse the same snapshot application services.
