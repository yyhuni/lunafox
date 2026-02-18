# asset/router

## Structure
- Public entry: `routes.go` (`RegisterAssetRoutes`)
- Health check entry: `health.go` (`RegisterHealthRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `assets.go`: website/subdomain/directory routes
- `endpoints.go`: endpoint routes
- `host_ports.go`: host-port routes
- `screenshots.go`: screenshot routes
- `public.go`: public routes without authentication

## Constraints
- Keep exactly one public module entry function (except cross-module bootstrap entries such as health)
- Keep sub-route registration helpers private (lowercase) to avoid leaking implementation details
- Delegate `bulk-upsert` routes to snapshot handlers:
  - `/targets/:id/websites/bulk-upsert` -> `WebsiteSnapshotHandler`
  - `/targets/:id/endpoints/bulk-upsert` -> `EndpointSnapshotHandler`
  - `/targets/:id/directories/bulk-upsert` -> `DirectorySnapshotHandler`
  - `/targets/:id/host-ports/bulk-upsert` -> `HostPortSnapshotHandler`
  - `/targets/:id/screenshots/bulk-upsert` -> `ScreenshotSnapshotHandler`
