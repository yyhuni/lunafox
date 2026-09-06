# security/handler

该目录承载 security 模块的 HTTP handler。

- canonical collection list handler 使用 `List`；父资源或跨集合列表使用 `ListBy<Scope>` / `ListAcross<Scope>`。
- AIP-style batch route handler 与后端调用链统一使用 `Batch*` 命名；不再保留旧批量命名别名。
- vulnerability statistics handler 统一使用完整 `VulnerabilityStatistics` 术语，不混用 `StatsSummary`。
- 漏洞列表 handler 只绑定 `pageSize/pageToken/filter/orderBy`；`page/sort/sortBy/sortOrder/keyword/severity/isReviewed` 必须在进入 facade 前直接返回 bad request。全局和目标漏洞的 `filterOptions` 路由必须位于 `:vulnerability` 动态路由之前，避免静态路径被当作资源 ID。
