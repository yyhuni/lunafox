# security/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

security 模块补充规则：

- **入口聚合**：对外用例入口单点收敛在 `facade_vulnerability.go`。
- **Facade 边界**：当前 facade 仍是受控局部例外，但重复的 vulnerability/target not-found 映射统一收口到 `security_facade_helpers.go`。
- **列表命名**：canonical collection list 使用 `List`；带父资源或跨集合作用域的变体显式写 scope，例如 `ListByTarget`。
- **统计命名**：漏洞统计用例统一使用 `VulnerabilityStatistics` 术语，不混用 `StatsSummary`。
- **批量边界**：HTTP batch update 入口、facade/store 端口与 repository 实现统一使用 `Batch*` 命名；不保留旧批量命名别名。
- **服务编排**：当前编排集中在 `facade_vulnerability.go`，这是该模块在现有复杂度下的受控局部例外，不代表 server application 的默认模式；复杂度提升时再拆出明确命名的 `*_service.go`。
- **adapter 例外**：当前无 transport-facing 或 cross-module bridge 型 adapter service 例外。
- **端口拆分**：端口统一按 vulnerability 资源命名，`vulnerability_raw_output_codec_ports.go` 与 `vulnerability_raw_output_codec.go` 必须成对维护（port + default implementation）。
- **模型命名**：`application` 文件统一 `vulnerability_*` 前缀，跨边界模型使用 `vulnerability_item_models.go`，类型别名使用 `vulnerability_aliases.go`。
- **历史迁移**：无历史聚合文件，保持资源化命名不回退。

## 漏洞列表查询契约

- 全局漏洞 `GET /v1/vulnerabilities` 和目标漏洞 `GET /v1/targets/:target/vulnerabilities` 已迁移到 canonical collection query，只接受 `pageSize/pageToken/filter/orderBy`。
- 旧参数 `page/sort/sortBy/sortOrder/keyword`，以及独立 `severity/isReviewed` 查询参数必须 hard cut；严重程度和审查状态只能进入 `filter`。
- 默认搜索字段与结构化 `url` 条件均是精确原始 URL 字符串匹配；结构化筛选白名单是 `url/severity/source/vulnType/isReviewed`，外部字段必须使用 lowerCamelCase，不接受 `vuln_type/reviewed/created_at` 等数据库字段名。
- Website 详情可向 Target 列表 filter 加入且仅加入一次 mandatory `websiteUrl=="<完整 Website URL>"`。application 只从该原始值派生临时的同源、有效端口与路径段 Scope，拒绝非 `==`、重复或 OR 分支中的条件；随后在普通筛选、`totalSize`、排序和分页前应用该 Scope。这只是读取时投影：Vulnerability 行保留完整原始 URL identity，不新增 Website foreign key、关系写入或写入阶段 lookup，且派生 Scope 不会改写 URL。
- 审查状态是 tab 语义，对应 `isReviewed` filter；它只适用于资产漏洞列表，不应下沉成普通筛选按钮。
- 后端排序白名单只有 `createdAt`，默认 `createdAt desc`，repository 必须追加 `id` 作为稳定 tie-breaker；不得开放 severity、URL、source、vulnType、cvssScore 排序。
- `pageToken` 必须绑定 global/target scope、targetId、normalized filter、normalized orderBy 和 pageSize；查询形状变化必须拒绝旧 token。
- `severity/source/vulnType` options 由后端按 global 或 target scope 聚合，不能从当前页 rows 推导完整菜单。
