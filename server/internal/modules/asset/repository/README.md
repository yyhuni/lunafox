# asset/repository

asset 模块 repository 规范：

- `<resource>.go`：资源仓储结构体与构造函数（如 `website.go`、`host_port.go`）。
- `<resource>_query.go`：查询职责（只读）；大结果导出可在 repository 内部使用游标，但对外只暴露 domain callback，不暴露 public `Stream/ScanRow` cursor API。
- `asset_statistics.go`：Overview 全局当前态聚合。它只返回绑定的 domain 投影，不暴露其他模块 persistence model；历史按现存行的 `created_at` 重建且受 30 天窗口限制。
- Website、Endpoint、Directory 和 Screenshot URL 是原始观测值；自然键、Target-scoped URL filter 和 lookup 必须使用完整字符串 `=`，不得通过 `ILIKE`、大小写折叠、解析重建或百分号转换扩大匹配。全局资产搜索的 URL contains 只属于独立的只读搜索谓词，不改变 URL 身份。Website Scope 只在详情关联读取时临时派生，绝不写回或参与自然键。
- 完整行导出流使用 `ForEachByTargetID` / 明确过滤变体，保持资产行语义，不在导出阶段做会改变行数或行身份的 `DISTINCT` 聚合。
- host-port IP 列表例外是已迁移的聚合业务列表：查询必须先限定 `target_id`，再应用 Website read scope 的精确归一化 `host==`（如存在），再应用普通 `ip/host/port` 筛选，再按 `ip` 聚合，最后按白名单 `ip` 或聚合 `MIN(created_at)` 排序分页；`totalSize` 统计筛选后的不同 IP 数量。精确 `host==` 不是归属关系，不改变 HostPort identity 或写入；普通 `host=` 继续走包含搜索。公开端口筛选、IP 排序/搜索、host 搜索和默认 `createdAt desc` 必须分别有 `target_id, port, ip`、`target_id, ip`、`target_id, host, ip` 加 `host gin_trgm_ops`、`target_id, created_at, ip` 组合索引或等价查询计划依据，不能只依赖单列索引。`ListPortOptionsByTargetID` 的 `count` 是同一 target 全集下拥有该端口的不同 IP 数，不随当前列表搜索、筛选、排序或分页条件联动。
- website 列表是已迁移的 target-scoped 后端分页业务列表：查询必须先限定 `target_id`，再应用 `url/statusCode/tech/webserver/contentType/vhost` 白名单筛选，再按 `statusCode/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 target 下筛选后的站点数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `target_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `target_id, created_at, id`，状态码和内容长度筛选/排序使用 `target_id, status_code, id`、`target_id, content_length, id`，Web 服务器、内容类型和虚拟主机等值筛选使用对应 `target_id, <field>, id` 组合索引，技术栈数组重叠使用 `tech` GIN 索引。不要把单列索引当作父资源范围全量筛选/排序的完成依据。`ListFilterOptionsByTargetID` 的 `count` 是同一 target 全集下该字段值出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
- endpoint URL 列表是已迁移的 target-scoped 后端分页业务列表：查询必须先限定 `target_id`，再应用 parsed `websiteUrl==` Scope（如存在），再应用 `url/statusCode/tech/webserver/contentType/vhost` 白名单筛选，再按 `statusCode/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 target 下筛选后的 URL 数量。Scope 仅派生同源、有效端口和路径层级的读时边界，保留完整持久化 URL 并不得退回客户端筛选。普通 URL 筛选使用完整原始值的 `=` 与既有 `target_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `target_id, created_at, id`，状态码和内容长度筛选/排序使用 `target_id, status_code, id`、`target_id, content_length, id`，Web 服务器、内容类型和虚拟主机等值筛选使用对应 `target_id, <field>, id` 组合索引，技术栈数组重叠使用 `tech` GIN 索引。`ListFilterOptionsByTargetID` 的 `count` 是同一 target 全集下该字段值出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
- directory 列表是已迁移的 target-scoped 后端分页业务列表：查询必须先限定 `target_id`，再应用解析后的 `websiteUrl==` Scope（如存在），再应用 `url/status/contentType` 白名单筛选，再按 `status/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 target 下筛选后的目录数量。Scope 仅派生同源、有效端口和路径层级的读时边界，保留完整持久化 URL 并不得退回客户端筛选。普通 URL 筛选使用完整原始值的 `=` 与既有 `target_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `target_id, created_at, id`，状态筛选/排序使用 `target_id, status, id`，内容长度排序使用 `target_id, content_length, id`，内容类型等值筛选使用 `target_id, content_type, id`；不要把单列索引当作父资源范围全量筛选/排序的完成依据。
- screenshot 列表是已迁移的 target-scoped 后端分页图库列表：查询必须先限定 `target_id`，再应用 `url/statusCode` 白名单筛选，再按 `statusCode/createdAt` 白名单排序，最后分页；`totalSize` 统计 target 下筛选后的截图数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `target_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `target_id, created_at, id`，状态码筛选/排序使用 `target_id, status_code, id`。列表查询必须显式投影元数据字段，不得选择 `image` blob；`ListFilterOptionsByTargetID` 的 `count` 是同一 target 全集下该状态码出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
- 扫描输入、host/ip:port 等集合型业务物化需要独立的投影流方法，命名应显式表达 scope、字段与语义；非 URL 事实可按其契约在 DB 层做规范化、去重和排序。Website、Endpoint、Directory、Screenshot 和 Vulnerability 的 URL 流必须逐行读取已存储值，不能以 `DISTINCT`、规范化、重排或 Target 过滤改变内容或身份。
- `<resource>_command.go`：写入职责（增删改/批量写）。
- 对 Website、Endpoint、Directory 和有效 Screenshot 的自然键冲突，`*_command.go` 必须从 `EXCLUDED` 完整替换所有可变观测字段；不得使用 `COALESCE`、`NULLIF` 或技术栈 union 保留旧值。ID、target 归属、自然键和 `created_at` 不得更新；Screenshot 仅在有效 winner 成功写入时刷新 `updated_at`。
- Directory Asset 以精确 `(target_id, url)` 为自然键，`content_length` 与纳秒 `duration` 使用 PostgreSQL `BIGINT`。Target Directory 列表、计数、筛选和导出不得通过 originating Scan/Task 终态隐藏已确认结果。
- `<resource>_mapper.go`：`domain <-> persistence model` 映射（按资源拆分）。

当前资源：

- `website`
- `endpoint`
- `directory`
- `subdomain`
- `host_port`
- `screenshot`

ResultIngest 事务内的 Asset 投影必须以 caller context 通过
`dbtx.Resolve(ctx, root).WithContext(ctx)` 复用 ambient transaction。不得在该路径使用
root DB、`context.Background()` 或嵌套事务；协调器已经锁定 Scan/Target，跨连接写入会形成
进程内锁等待并破坏 Snapshot、Asset、summary 的原子提交。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- 禁止使用聚合式泛名 mapper 文件（如 `asset_mapper.go`）。

## 全局资产搜索

`SearchGlobalWebsites` 和 `SearchGlobalEndpoints` 是全局搜索专用的窄 query port，
分别固定读取 `website` 或 `endpoint` 当前态表，并通过 active Target join 排除
墓碑 Target。它们不得复用 Target-scoped 的 offset/count 查询，不得读取 Snapshot、
漏洞数据或引入跨表 `UNION`。

查询只接收 application 已校验的 typed AST，所有值使用绑定参数。全局搜索的 URL、
host/title 单等号使用带字面量转义的 PostgreSQL `ILIKE`，包含值至少 2 个 Unicode 字符；双等号使用完整字符串 `=`；
普通文本 URL 也走 URL contains 谓词。`statusCode` 使用整数 `=`，`tech` 使用数组
`@>` 完整元素谓词。固定排序为 `(created_at DESC, id DESC)`，通过 tuple keyset 和
`LIMIT pageSize+1` 分页。

每次搜索在短事务中执行 `SET LOCAL statement_timeout = '5s'`，确保连接池复用不把
超时设置带到其他请求。URL/host/title 的包含查询依赖对应 trigram GIN，精确查询继续
依赖原有 B-tree。若 Website detail 的虚拟 Scope 使用既有 URL trigram 缩小候选集，它
只是内部预筛选，后续 Scope predicate 才是权威判断，不构成公开 URL contains 查询。

### 全局搜索全量导出

导出 repository port 必须独立于分页搜索 port，不得复用 `pageSize + 1`、capped total 或
客户端结果数组。grant 存储只保留不可逆哈希及用户 ID、token version、assetType、规范化
typed AST 和过期时间；消费必须原子地保证一次性使用。导出查询在一个
`REPEATABLE READ READ ONLY` 快照内复用全局搜索谓词和 active Target join，按
`created_at DESC, id DESC` 顺序通过 domain callback 逐行读取，不向 handler 暴露 `sql.Rows`。

CSV writer 的数据映射必须保持以下固定后端字段契约：
`id,target_id,url,host,location,title,status_code,content_length,content_type,webserver,tech,response_body,response_headers,vhost,created_at`。
编码由最终 CSV 字节流决定，包含 BOM、header、转义和换行；同一快照需要先精确预检行数与
字节数，再执行流写，允许恰好 100000 行/256 MiB，超过则不提交响应。读取或提交失败时
必须释放 rows、事务和连接，并将结构化失败交给上层中断传输。

每个导出会在整个请求期间占用一个数据库连接，预算为 120 秒并支持 context cancellation。
当前实现不增加导出专用并发信号量或排队器；多个导出直接竞争现有连接池，连接池耗尽时
普通请求按既有池行为等待。该连接池竞争风险必须在运行监控和容量评估中保留，不能通过
静默截断结果来规避。
