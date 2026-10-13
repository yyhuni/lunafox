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
Endpoint 的 evidence truncation flags；响应不包含漏洞摘要或页码，但每次请求（含翻页）
都携带封顶总数 `totalSize` 与 `totalSizeCapped`：命中数不超过 10000 时 `totalSize`
为精确值，超过时固定为 10000 且 `totalSizeCapped=true`。无效输入
统一映射为 `400 INVALID_ARGUMENT`，数据库搜索超时映射为可识别的
`DEADLINE_EXCEEDED` gateway response。

### 全局搜索全量导出边界

全量导出使用两段式 HTTP 契约：受 JWT 保护的 `POST /v1/assets:export` 只接受已提交的
`q` 与 `assetType`，返回短时 `downloadUrl` 和过期时间；浏览器随后访问
`GET /v1/assets/exportFiles/current?downloadToken=...`。下载 URL 只携带不可推导、一次性
消费的 grant，不接受主 JWT、原始搜索条件或 page 参数。缺失、过期、重放、会话版本失效或
绑定条件不匹配必须返回非 2xx，且不得打开导出快照。

成功响应是原生 CSV attachment，使用 UTF-8 BOM、固定后端列顺序
`id,target_id,url,host,location,title,status_code,content_length,content_type,webserver,tech,response_body,response_headers,vhost,created_at`、标准 CSV 转义、`|` 分隔 tech、空值为空字符串和 UTC RFC3339Nano 时间；响应同时提供精确 `Content-Length` 与 `X-Accel-Buffering: no`。

行数/编码字节超限时必须在发送 attachment header 前返回资源错误。CSV 已开始发送后若读取、
编码、提交、超时或写入失败，handler 必须中断响应，不得向 CSV 追加 JSON，也不得把部分
文件报告为完整成功。页面只能提示下载已发起，不能承诺磁盘保存或浏览器临时文件清理。
