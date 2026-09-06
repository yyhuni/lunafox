# asset/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

asset 模块补充规则：

- **入口聚合**：按资产类型拆分 facade 入口（`facade_website.go`、`facade_subdomain.go`、`facade_endpoint.go`、`facade_directory.go`、`facade_host_port.go`、`facade_screenshot.go`）。
- **Facade 边界**：facade 只保留入口委派和边界错误语义映射调用；重复的 target/resource not-found 映射统一收口到 `asset_facade_helpers.go`。
- **列表命名**：facade/query service 的父资源列表使用 `ListByTarget`；port/repository 保留 `ListByTargetID` 表达持久化查询键。
- **观测 URL**：Website、Endpoint、Directory 和 Screenshot 的 URL 是不可变原始证据。准入后不得 trim、百分号解码、大小写折叠、解析重建或用于宽松匹配；普通 URL 查询、自然键和详情身份均以完整存储字符串精确比较。Host/Target 校验与 Website Scope 只能创建临时派生视图，不能回写 URL。
- **导出边界**：target-scoped export/full-read 通过 facade/query service 的 domain callback（如 `ForEachByTarget`）流出，host-port IP 过滤使用显式 `ForEachByTargetAndIPs`，不向 handler、facade 或 application port 暴露 `*sql.Rows` / `ScanRow`。
- **host-port IP 聚合列表**：`GET /v1/targets/:target/hostPorts` 的列表语义是一行一个聚合 IP，不是 host-port 明细行。application 只接受 `pageSize/pageToken/filter/orderBy`，旧 `page/sort/sortBy/sortOrder/keyword` 由 handler hard cut；`pageToken` 必须绑定 `targetId + filter + orderBy + pageSize`。filter 只允许默认搜索分支中的 `ip` / `host` 和端口 facet `port`，端口多选是 OR/IN；Website 详情额外使用一次不可移除的 `host=="<website.host>"`，按归一化 host 精确匹配，不能放入 OR；普通 `host="..."` 仍是包含搜索。orderBy 只允许 `ip`、`createdAt`，默认 `createdAt desc`。聚合 `createdAt` 表示同一 IP 组内最早 `created_at`。当前 host-port 资产只保存 IPv4；IPv6 结果在 application 保存前跳过，不让混合批次里的有效 IPv4 因不支持 IPv6 而失败。
- **website 列表和 Get**：`GET /v1/targets/:target/websites` 是已迁移的后端分页业务列表，`GET /v1/websites/:website` 读取同一 Website resource shape。List/Get 都可投影由精确 `target_id + url` 命中的可选 Screenshot 摘要，不携带 image blob；这只是读模型，不改变 Website/Screenshot identity、schema 或结果写入。列表 application 只接受 `pageSize/pageToken/filter/orderBy`，旧 `page/sort/sortBy/sortOrder/keyword` 由 handler hard cut；`pageToken` 必须绑定 `targetId + filter + orderBy + pageSize`。filter 外部字段只允许 lowerCamelCase 的 `url/statusCode/tech/webserver/contentType/vhost`；普通 URL 文本和 `url` 条件均是精确原始字符串匹配，`tech` 多选是数组重叠语义，`vhost` 是布尔语义。orderBy 只允许 `statusCode/contentLength/createdAt`，默认 `createdAt desc`；不得开放 `url` 排序；`statusCode/contentLength` 排序的空值固定 `NULLS LAST`。不得接受 `status_code/content_length/content_type/created_at` 这类数据库 snake_case 字段作为外部 filter/orderBy。
- **endpoint URL 列表**：`GET /v1/targets/:target/endpoints` 是已迁移的后端分页业务列表。application 只接受 `pageSize/pageToken/filter/orderBy`，旧 `page/sort/sortBy/sortOrder/keyword` 由 handler hard cut；`pageToken` 必须绑定 `targetId + filter + orderBy + pageSize`。Website 详情可携带一次 mandatory `websiteUrl=="<full Website URL>"`；application 仅从原始 URL 派生临时的同源、有效端口与路径段 Scope，拒绝非 `==`、重复或 OR 分支，并在普通 filter/order/page stages 前提取该 Scope。它不改写持久化 URL，且只在关联视图收窄 `totalSize`、排序与分页。filter 其余外部字段只允许 lowerCamelCase 的 `url/statusCode/tech/webserver/contentType/vhost`；普通 URL 文本和 `url` 条件均是精确原始字符串匹配，`tech` 多选是数组重叠语义，`vhost` 是布尔语义。orderBy 只允许 `statusCode/contentLength/createdAt`，默认 `createdAt desc`；`statusCode/contentLength` 排序的空值固定 `NULLS LAST`。不得开放 `url` 排序，也不得接受 `status_code/content_length/content_type/created_at` 这类数据库 snake_case 字段作为外部 filter/orderBy。
- **directory 列表**：`GET /v1/targets/:target/directories` 是已迁移的后端分页业务列表。application 只接受 `pageSize/pageToken/filter/orderBy`，旧 `page/sort/sortBy/sortOrder/keyword` 由 handler hard cut；`pageToken` 必须绑定 `targetId + filter + orderBy + pageSize`。Website 详情的 mandatory `websiteUrl==` 使用与 Endpoint 相同的只读 Scope：从原始 URL 派生同源、有效端口与路径段边界，仅在关联视图的 `totalSize`、排序和分页前收窄结果，绝不改写 URL 或参与持久化身份。filter 其余外部字段只允许 `url/status/contentType`；普通 URL 文本和 `url` 条件均是精确原始字符串匹配，同一 facet 多值是 OR/IN，不同 facet 与搜索是 AND。orderBy 只允许 `status/contentLength/createdAt`，默认 `createdAt desc`；`status/contentLength` 排序空值固定 `NULLS LAST`。不得开放 `url/contentType` 排序，也不得接受 `content_length/content_type/created_at` 这类数据库 snake_case 字段作为外部 filter/orderBy。
- **screenshot 列表**：`GET /v1/targets/:target/screenshots` 是已迁移的后端分页图库列表。application 只接受 `pageSize/pageToken/filter/orderBy`，旧 `page/sort/sortBy/sortOrder/keyword` 由 handler hard cut；`pageToken` 必须绑定 `targetId + filter + orderBy + pageSize`。filter 外部字段只允许 `url/statusCode`；普通 URL 文本和 `url` 条件均是精确原始字符串匹配，`statusCode` 多选是 OR/IN，与 URL 搜索是 AND。orderBy 只允许 `statusCode/createdAt`，默认 `createdAt desc`；`statusCode` 排序空值固定 `NULLS LAST`。不得开放 `url/image/updatedAt` 排序，也不得接受 `status_code/created_at` 这类数据库 snake_case 字段作为外部 filter/orderBy；列表响应不得选择或返回 `image` blob，图片只走专用 blob endpoint。
- **服务编排**：读写流程按资源拆分为 `*_query.go` 与 `*_command.go`，作为 facade 背后的 query/command 编排实现。
- **Overview 统计**：`AssetStatisticsQueryService` 是独立的只读聚合用例，不并入资源 facade。历史窗口必须由 caller 显式提供且限制在 1 至 30 天；历史基于存活当前态记录的 `created_at` 重建，不能表述为库存快照。
- **写入 context**：result-ingest 使用的 Subdomain/HostPort/Website command 路径必须把同一个 caller context 同时传给 target authorization lookup 和最终 store write；target lookup 不得使用裸 DB 或替换为后台 context。
- **adapter 例外**：当前无 transport-facing 或 cross-module bridge 型 adapter service 例外。
- **端口拆分**：端口按资源+职责拆分为 `*_query_ports.go`、`*_command_ports.go`；跨资源依赖单独建模（如 `target_lookup_ports.go`）。
- **模型命名**：新增跨边界模型优先使用资源化 `*_item_models.go`、`*_query_inputs.go`；避免新增泛名模型文件。
- **历史迁移**：`aliases.go`、`errors.go` 已完成首批迁移，继续保持资源化命名不回退。

## 全局资产搜索

`GET /v1/assets:search` 是独立于 Target-scoped 列表的 query service。它只接受
`website` 或 `endpoint`，按类型调用对应的当前态 store；不得读取 Snapshot、漏洞
repository 或通过 `UNION` 合并两张资产表。默认 Website 由前端提供，application
不对缺失的 `assetType` 做静默降级。

查询先由专用 parser 完整消费，再构造 typed AST：普通文本只匹配完整原始 URL；结构化查询
只允许 `url`、`host`、`title`、`statusCode`、`tech` 和扁平 `&&`。`url` 的 `=` 与 `==`
都表示精确原始字符串匹配；只有 `host` 与 `title` 的 `=` 是不区分大小写的包含匹配。
`statusCode` 与 `tech` 两种运算符都表示精确匹配。输入上限、条件数、包含值字符下限、
page size 和 page token 都必须在调用 repository 前校验。

结果使用绑定查询形态的 URL-safe keyset token，排序为 `createdAt DESC, id DESC`，
不计算 total。repository 的超时错误映射为 `ErrGlobalAssetSearchTimeout`，不把
PostgreSQL 驱动错误或部分结果暴露给 handler。
