# asset/handler

The asset module handler layer is organized by resource subdirectories:

- `health.go`: health check (module-level)
- `asset_statistics.go`: protected Overview current and history report views
- `website/`
- `endpoint/`
- `subdomain/`
- `directory/`
- `host_port/`
- `screenshot/`

Each resource subdirectory follows a fixed responsibility split:

- `handler.go`: handler struct and constructor
- `read.go`: read APIs
- `write.go`: write APIs (excluding projection ingest)
- `export.go`: export APIs (only for resources that support export)

## Conventions

- Resource handler canonical list methods are named `List`; parent scope is expressed by the route and parsed parameters.
- AIP-style batch routes and their backend call paths use `Batch*` names; do not reintroduce legacy batch aliases or mixed terminology.
- Projection ingest routes are handled by snapshot handlers, not by `asset/handler/*/write.go`.
- Do not add new asset handler APIs using the legacy hyphenated action path style; use AIP-style batch or projection-ingest methods.
- Asset handlers should keep only asset-facing APIs such as query, export, create, and delete.
- Overview history requires exactly one `days` query parameter from 1 to 30. Handler validation must reject omitted, repeated, and malformed values before application dispatch; it must not provide a silent default.
- Website List and flattened `GET /v1/websites/:website` share one output mapper. The latter is the single reviewed migration exception for a Website whose canonical response name remains `targets/:target/websites/:website`; do not add a nested GET alias. Optional Screenshot summaries are metadata-only exact `target_id + url` read projections.

全局搜索 handler 位于 `search/`，只处理 `GET /v1/assets:search`。成功响应复用现有
Website/Endpoint read shape：`tech` 是数组，响应头和响应体是完整字符串，并保留
Endpoint 的 evidence truncation flags；响应不包含漏洞摘要、total 或页码。无效输入
统一映射为 `400 INVALID_ARGUMENT`，数据库搜索超时映射为可识别的
`DEADLINE_EXCEEDED` gateway response。
