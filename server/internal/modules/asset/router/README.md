# asset/router

## Structure
- Public entry: `routes.go` (`RegisterAssetRoutes`)
- Health check entry: `health.go` (`RegisterHealthRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `assets.go`: website/subdomain/directory routes
- `asset_statistics.go`: current and daily Overview asset statistics report routes
- `endpoints.go`: endpoint routes
- `host_ports.go`: host-port routes
- `screenshots.go`: screenshot routes
- `public.go`: public routes without authentication

## Constraints
- Keep exactly one public module entry function (except cross-module bootstrap entries such as health)
- Keep sub-route registration helpers private (lowercase) to avoid leaking implementation details
- Public screenshot `*/blob` routes are protocol-native media download routes registered in the boundary canonical media allowlist.
- `GET /websites/:website` is the one registered flattened Website Get migration exception. Its response name remains nested (`targets/:target/websites/:website`), and no nested compatibility route may be introduced.
- Projection ingest routes use AIP-style `:batchIngest` methods and delegate to snapshot handlers:
  - `/targets/:target/websites:batchIngest` -> `WebsiteSnapshotHandler.BatchIngest`
  - `/targets/:target/endpoints:batchIngest` -> `EndpointSnapshotHandler.BatchIngest`
  - `/targets/:target/directories:batchIngest` -> `DirectorySnapshotHandler.BatchIngest`
  - `/targets/:target/hostPorts:batchIngest` -> `HostPortSnapshotHandler.BatchIngest`
  - `/targets/:target/screenshots:batchIngest` -> `ScreenshotSnapshotHandler.BatchIngest`
- Overview statistics report views are protected fixed projections:
  - `GET /v1/assetStatistics`
  - `GET /v1/assetStatistics/history?days={1..30}`
  The history is reconstructed from surviving current-state records and is not a historical snapshot.

## Target subdomain list contract

- `GET /v1/targets/:target/subdomains` 是已迁移的后端分页业务列表，只接受 `pageSize`、`pageToken`、`filter`、`orderBy`。
- 旧查询别名 `page`、`sort`、`sortBy`、`sortOrder`、`keyword` 必须返回 400，不保留兼容分页路径。
- `filter` 只开放 `dnsName` 普通包含匹配，普通搜索由前端编译为 `dnsName="..."`。
- `orderBy` 只开放 `dnsName` 和 `createdAt`；默认业务排序是 `createdAt desc`，SQL 侧追加 `id` 稳定兜底。
- `pageToken` 是 URL-safe opaque token，application 层绑定 `targetId`、规范化后的 `filter`、`orderBy` 和 `pageSize`；不能跨 target 或查询形态复用。
- 公开搜索/排序字段必须有索引依据：`idx_subdomain_target_created_at_id`、`idx_subdomain_target_dns_name_id`、`idx_subdomain_dns_name_trgm`。
- scan 子域名结果页不属于该契约；不要把 target 子域名 token/query 语义套到 scan 路径。

## 全局资产搜索路由

全局搜索唯一入口是 `GET /v1/assets:search`，由 `registerGlobalAssetSearchRoutes`
注册到受保护路由。它是 custom method，不保留 `/v1/assets/search/`、尾斜杠或
export 别名；查询参数统一使用 `q`、`assetType`、`pageSize`、`pageToken` 的
camelCase 形式。该路由不携带 Target parent，handler 只负责 HTTP 绑定、错误映射和
Website/Endpoint DTO 映射，类型分流由 asset application 负责。
