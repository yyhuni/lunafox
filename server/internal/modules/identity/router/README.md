# identity/router

## Structure
- Public entry: `routes.go` (`RegisterIdentityRoutes`)
- Other files contain package-private sub-route registrars (`register*Routes`)

## Files
- `auth_routes.go`: auth routes
- `user_routes.go`: user routes
- `organization_routes.go`: organization routes

## Constraints
- Expose only the module entry function
- Use sub-registration helpers only within this package

## Organization List Query Contract

- `GET /v1/organizations` 的规范集合查询参数是 `pageSize`、`pageToken`、`filter`、`orderBy`。
- `filter` 只支持 `displayName="..."` 或等价受控名称包含搜索，映射到组织业务名称 `organization.name`；其它字段必须以参数无效语义拒绝。
- `orderBy` 只支持 `displayName` 和 `createdAt`，方向为 `asc` 或 `desc`；未传时使用 `createdAt desc`，数据库稳定排序为 `organization.created_at DESC, organization.id DESC`。
- 描述、目标数、选择列和操作列没有组织主列表后端排序契约，前端不得展示排序表头。
- 后端执行顺序固定为：先排除软删除记录，再应用 `filter`，再应用 `orderBy` 和稳定 `id` 兜底排序，最后分页。
- `pageToken` 是 URL-safe opaque token，绑定规范化后的 `filter`、`orderBy`、`pageSize` 查询形态；查询形态变化时旧 token 必须被拒绝。
- 组织列表公开搜索/排序字段必须有索引依据：`idx_org_name_trgm_active` 支撑名称包含搜索，`idx_org_name_id_active` 支撑名称排序，`idx_org_created_at_id_active` 支撑默认排序和添加时间排序。
- 旧查询别名 `page`、`sort`、`sortBy`、`sortOrder`、`keyword` 已 hard cut；不要重新引入并行兼容路径。
