# security/repository

security 模块 repository 规范：

- `vulnerability.go`：仓储结构体、构造函数、筛选映射与公共常量。
- `vulnerability_query.go`：漏洞查询与统计查询方法。
- `vulnerability_command.go`：漏洞写操作方法（批量创建、删除、审核状态变更）。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。

## 漏洞列表索引与查询顺序

- `vulnerability_query.go` 的列表查询顺序必须是：先限定 scope（全局或 `target_id`），再应用 parsed `websiteUrl==` Scope（仅 Target Website detail 可用），再应用 `url/severity/source/vulnType/isReviewed` 白名单 filter，再应用 `createdAt` 白名单排序，最后分页；`totalSize` 统计筛选后的全集。Website Scope 只从原始 URL 派生同源、有效端口和路径段边界；它只读，不改写 URL，不新增 URL split fields、Website relation、foreign key 或 ingestion write。
- URL 查询必须使用完整原始字符串的精确 `=` 与既有自然键/B-tree 索引；不得退回 URL `ILIKE` 或 `%term%` 搜索。
- 默认排序必须有 `idx_vuln_created_at_id` 和 `idx_vuln_target_created_at_id` 支撑；目标范围筛选必须有 `target_id + severity/source/vuln_type + id` 组合索引。
- 审查 tab 必须有 `reviewed + created_at + id` 及 `target_id + reviewed + created_at + id` 索引依据。
- `ListFilterOptions` / `ListFilterOptionsByTargetID` 的 count 表示该 scope 全集下字段值出现次数，不随当前列表搜索、筛选、排序或分页联动。
