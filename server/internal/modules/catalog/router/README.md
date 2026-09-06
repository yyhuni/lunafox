# catalog/router

## Structure
- Public entry: `routes.go` (`RegisterCatalogRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `targets.go`: target routes
- `engines.go`: installed-package read-only engine catalog routes
- `workflows.go`: scan workflow management routes及其 parent-scoped Profile singleton
- `wordlists.go`: wordlist routes

## Target Routes
- `GET /v1/targets?pageSize=&pageToken=&filter=&orderBy=` returns a paginated target collection with `results`, `nextPageToken`, and `totalSize`.
- `POST /v1/targets` creates a target from `name`; the response exposes canonical `name` (`targets/{id}`) and user-facing `displayName`.
- `PATCH /v1/targets/:target` updates `displayName` guarded by `updateMask=displayName`.
- `POST /v1/targets:batchCreate` accepts up to 5,000 target names and an optional `organization` resource name.
- `POST /v1/targets:batchDelete` accepts target resource names and soft-deletes them.

## Engine Catalog Routes

- `GET /v1/engines` lists validated installed engine package summaries.
- `GET /v1/engines/:engine` returns configuration-facing manifest, locale, and schema metadata for one installed engine.
- These routes are a read-only installed-package projection. They do not restore the removed database-backed engine CRUD API; `POST`, `PUT`, `PATCH`, and `DELETE` engine routes remain unregistered.

## Scan Workflow Routes

- `POST /v1/scanWorkflows`、`GET /v1/scanWorkflows`、`GET /v1/scanWorkflows/:scanWorkflow` 和 `PATCH /v1/scanWorkflows/:scanWorkflow` 构成管理面；不注册 Delete 或 delete-like 路由。
- `GET /v1/scanWorkflows/:scanWorkflow/profile` 是唯一 Profile 读取接口。已移除 `scanWorkflowProfiles` collection，workflow List/Get 不返回 `configuration`。
- List 使用 `pageSize`、`pageToken`、`filter`，固定按内置优先、display name、资源名排序。Update 要求 ETag 和显式 update mask；内置 workflow 返回不可修改错误。

## Target Query Contract

- `GET /v1/targets` 的规范集合查询参数是 `pageSize`、`pageToken`、`filter`、`orderBy`。
- 端点不接受旧查询别名：`page`、`type`、`sort`、`sortBy`、`sortOrder`、`keyword`。
- `filter` 支持字段：`displayName`、`type`。普通文本搜索按 `displayName` 处理，语义是名称包含匹配。
- `type` 只支持精确枚举：`type=="domain"`、`type=="ip"`、`type=="cidr"`；多选类型应编译为 `(type=="domain" || type=="ip")`，再与名称搜索条件 AND。`type="domain"` 这类模糊匹配必须拒绝。
- `orderBy` 支持字段：`displayName`、`createdAt`、`lastScannedAt`，方向为 `asc` 或 `desc`。未传 `orderBy` 时使用业务默认 `createdAt desc`，数据库排序为 `created_at DESC, id DESC`。
- `lastScannedAt` 升序和降序都固定 `NULLS LAST`，未扫描目标排在已扫描目标之后。
- 后端执行顺序固定为：先排除软删除记录，再应用 `filter`，再应用 `orderBy` 和稳定 `id` 兜底排序，最后分页。
- 排序和筛选性能依据：名称包含搜索使用 `idx_target_name_trgm_active`；名称排序使用 `idx_target_name_id_active`；默认排序和添加时间排序使用 `idx_target_created_at_id_active`；最后扫描时间双方向使用 `idx_target_last_scanned_at_desc_id_active` 和 `idx_target_last_scanned_at_asc_id_active`；类型筛选结合默认排序使用 `idx_target_type_created_at_id_active`。新增目标列表搜索、筛选或排序字段时必须同步补白名单、稳定排序、索引或查询计划依据，以及迁移/模型契约测试。
- `pageToken` 是 URL-safe opaque token，绑定规范化后的 `filter`、`orderBy`、`pageSize` 查询形态；查询形态变化时旧 token 必须被拒绝。
- 响应中的 `totalSize` 表示筛选后的集合大小。

## Wordlist Text Routes
- `GET /v1/wordlists?pageSize=&pageToken=&filter=&orderBy=` returns a paginated wordlist collection with `results`, `nextPageToken`, and `totalSize`.
- `POST /v1/wordlists` accepts multipart upload and persists the uploaded basename as read-only `fileName`; callers do not provide a separate wordlist name field. The response derives canonical `name` as `wordlists/{id}`.
- `PATCH /v1/wordlists/:wordlist` updates only wordlist metadata fields `description` and `tags` guarded by `updateMask`; `name`, `displayName`, and `fileName` are immutable and rejected when present in the mask.
- `GET /v1/wordlistTags?pageSize=&pageToken=&filter=` returns derived read-only tag summaries with `name`, `displayName`, and `wordlistCount`.
- `GET /v1/wordlists/:wordlist/text` returns the `WordlistText` singleton resource.
- `PATCH /v1/wordlists/:wordlist/text?updateMask=content` updates the `WordlistText` singleton resource and returns that resource, not the parent wordlist.
- `GET /v1/wordlists/:wordlist/blob` is a protocol-native media download route registered in the boundary canonical media allowlist.
- Legacy `/wordlists/:id/content` and `PUT /wordlists/:wordlist/text` shapes are not registered.

## Wordlist Query Contract

- `GET /v1/wordlists` 的规范集合查询参数是 `pageSize`、`pageToken`、`filter`、`orderBy`。
- 端点不接受非规范查询参数：`page`、`sort`、`sortBy`、`sortOrder`、`keyword`。
- `filter` 支持字段：`fileName`、`description`、`tags`、`lineCount`、`fileSize`、`fileHash`。
- `tags` 只支持精确包含语义，例如 `tags=="fuzz"`；普通多选标签应编译成 `(tags=="fuzz" || tags=="subdomain")`，再与搜索条件 AND。
- `orderBy` 支持字段：`fileName`、`lineCount`、`fileSize`、`updatedAt`，方向为 `asc` 或 `desc`；省略方向时 `updatedAt` 使用 `desc`，其他字段使用 `asc`。
- 后端执行顺序固定为：先应用 `filter`，再应用 `orderBy` 和稳定 `id` 兜底排序，最后分页。
- 排序性能依据：`fileName` 使用 `unique_wordlist_file_name`；`lineCount` 使用 `idx_wordlist_line_count_id`；`fileSize` 使用 `idx_wordlist_file_size_id`；`updatedAt` 使用 `idx_wordlist_updated_at_id`。新增排序字段时必须同步补白名单、稳定排序、索引或查询计划依据，以及迁移/模型契约测试。
- 响应中的 `totalSize` 表示筛选后的集合大小，`nextPageToken` 是 URL-safe opaque token，绑定规范化后的 `filter`、`orderBy`、`pageSize` 查询形态；查询形态变化时旧 token 必须被拒绝。
- 当前页面 UI 仍使用页码展示，但请求层已经是 `pageToken`；前端只能跳到第一页、上一页或已获得 token 的页面，不得伪造未知页 token。

## Constraints
- External callers should only use `RegisterCatalogRoutes`
- Keep resource route registration split by resource, with private helper functions
