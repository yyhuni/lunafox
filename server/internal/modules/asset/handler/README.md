# asset/handler

The asset module handler layer is organized by resource subdirectories:

- `health.go`: health check (module-level)
- `website/`
- `endpoint/`
- `subdomain/`
- `directory/`
- `host_port/`
- `screenshot/`

Each resource subdirectory follows a fixed responsibility split:

- `handler.go`: handler struct and constructor
- `read.go`: read APIs
- `write.go`: write APIs (excluding `bulk-upsert`)
- `export.go`: export APIs (only for resources that support export)

## Conventions

- `bulk-upsert` is handled by snapshot handlers, not by `asset/handler/*/write.go`.
- Asset handlers should keep only asset-facing APIs such as query, export, create, and delete.
