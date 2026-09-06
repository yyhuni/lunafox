# snapshot/repository

snapshot 模块 repository 规范：

- 每个快照资源统一拆分为三类文件：
  - `<resource>_snapshot.go`：仓储结构体、构造函数、筛选映射/公共类型。
  - `<resource>_snapshot_query.go`：查询方法（`List/Get/Count/ForEach`）；大结果导出可在 repository 内部使用游标，但对外只暴露 domain callback，不暴露 public `Stream/ScanRow` cursor API。
  - 完整行导出流使用 `ForEachByScanID` / 明确过滤变体，保持快照行语义，不在导出阶段做会改变行数或行身份的 `DISTINCT` 聚合。
  - Website、Endpoint、Directory、Screenshot 和 Vulnerability URL 是原始观测值；自然键、普通 URL filter 和 lookup 必须使用完整字符串 `=`，不得通过 `ILIKE`、大小写折叠、解析重建或百分号转换扩大匹配。
  - host-port snapshot 的 `ListByScan` 是已迁移的 IP 聚合业务列表：查询必须先限定 `scan_id`，再应用 `ip/host/port` 筛选，再按 `ip` 聚合，最后按白名单 `ip` 或聚合 `MIN(created_at)` 排序分页；`totalSize` 统计筛选后的不同 IP 数量。公开端口筛选、IP 排序/搜索、host 搜索和默认 `createdAt desc` 必须分别有 `scan_id, port, ip`、`scan_id, ip`、`scan_id, host, ip` 加 `host gin_trgm_ops`、`scan_id, created_at, ip` 组合索引或等价查询计划依据，不能只依赖单列索引。`ListPortOptionsByScanID` 的 `count` 是同一 scan 全集下拥有该端口的不同 IP 数，不随当前列表搜索、筛选、排序或分页条件联动。
  - website snapshot 的 `ListByScan` 是已迁移的 scan-scoped 后端分页业务列表：查询必须先限定 `scan_id`，再应用 `url/statusCode/tech/webserver/contentType/vhost` 白名单筛选，再按 `statusCode/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 scan 下筛选后的站点数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `scan_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `scan_id, created_at, id`，状态码和内容长度筛选/排序使用 `scan_id, status_code, id`、`scan_id, content_length, id`，Web 服务器、内容类型和虚拟主机等值筛选使用对应 `scan_id, <field>, id` 组合索引，技术栈数组重叠使用 `tech` GIN 索引。不要把单列索引当作父资源范围全量筛选/排序的完成依据。`ListFilterOptionsByScanID` 的 `count` 是同一 scan 全集下该字段值出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
  - endpoint URL snapshot 的 `ListByScan` 是已迁移的 scan-scoped 后端分页业务列表：查询必须先限定 `scan_id`，再应用 `url/statusCode/tech/webserver/contentType/vhost` 白名单筛选，再按 `statusCode/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 scan 下筛选后的 URL 数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `scan_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `scan_id, created_at, id`，状态码和内容长度筛选/排序使用 `scan_id, status_code, id`、`scan_id, content_length, id`，Web 服务器、内容类型和虚拟主机等值筛选使用对应 `scan_id, <field>, id` 组合索引，技术栈数组重叠使用 `tech` GIN 索引。`ListFilterOptionsByScanID` 的 `count` 是同一 scan 全集下该字段值出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
  - directory snapshot 的 `ListByScan` 是已迁移的 scan-scoped 后端分页业务列表：查询必须先限定 `scan_id`，再应用 `url/status/contentType` 白名单筛选，再按 `status/contentLength/createdAt` 白名单排序，最后分页；`totalSize` 统计 scan 下筛选后的目录数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `scan_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `scan_id, created_at, id`，状态筛选/排序使用 `scan_id, status, id`，内容长度排序使用 `scan_id, content_length, id`，内容类型等值筛选使用 `scan_id, content_type, id`；不要把单列索引当作父资源范围全量筛选/排序的完成依据。
  - screenshot snapshot 的 `ListByScan` 是已迁移的 scan-scoped 后端分页图库列表：查询必须先限定 `scan_id`，再应用 `url/statusCode` 白名单筛选，再按 `statusCode/createdAt` 白名单排序，最后分页；`totalSize` 统计 scan 下筛选后的截图数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `scan_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `scan_id, created_at, id`，状态码筛选/排序使用 `scan_id, status_code, id`。列表查询必须显式投影元数据字段，不得选择 `image` blob；`ListFilterOptionsByScanID` 的 `count` 是同一 scan 全集下该状态码出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
  - vulnerability snapshot 的 `ListByScan` 是已迁移的 scan-scoped 后端分页业务列表：查询必须先限定 `scan_id`，再应用 `url/severity/source/vulnType` 白名单筛选，再按 `createdAt` 白名单排序，最后分页；`totalSize` 统计 scan 下筛选后的漏洞数量。普通 URL 筛选使用完整原始值的 `=` 与既有 `scan_id + url` 自然键索引，不执行 `%term%` 搜索或 URL 排序。默认排序使用 `scan_id, created_at, id`，严重程度、来源和漏洞类型筛选使用 `scan_id, severity/source/vuln_type, id` 组合索引。scan snapshot 没有 `reviewed` 字段，不得开放审查 tab 或 `isReviewed` filter。不得开放 URL、severity、source、vulnType、cvssScore 排序；不要因为存在单列索引就把父资源范围全量筛选/排序视为已完成能力。`ListFilterOptionsByScanID` 的 `count` 是同一 scan 全集下该字段值出现次数，不随当前列表搜索、筛选、排序或分页条件联动。
  - 扫描输入、host/ip:port 等集合型业务物化需要独立的投影流方法，命名应显式表达 scope、字段与语义；非 URL 事实可按其契约在 DB 层做规范化、去重和排序。Website、Endpoint、Directory、Screenshot 和 Vulnerability 的 URL 流必须逐行读取已存储值，不能以 `DISTINCT`、规范化、重排或 Target 过滤改变内容或身份。
- `<resource>_snapshot_command.go`：写操作方法（`BatchCreate/BatchUpsert`）。
- Website、Endpoint、Directory 和有效 Screenshot 的冲突写入必须完整替换可变观测字段，保留行 ID、scan 归属、自然键和 `created_at`。应用层负责在调用仓储前按最后有效项去重，仓储不得用 sparse merge 掩盖合法空值。
- Directory Snapshot 以精确 `(scan_id, url)` 为自然键，`content_length` 与纳秒 `duration` 使用 PostgreSQL `BIGINT`。Directory 列表、计数、筛选和导出只按请求的 scan scope 查询，不得因 Scan/Task 后续为 `failed` 或 `cancelled` 而过滤已提交行。
 - `<resource>_snapshot_mapper.go`：`domain <-> persistence model` 映射（按资源拆分）。
- 当前资源包括：`website / subdomain / endpoint / directory / host_port / screenshot / vulnerability`。
- ResultIngest 事务内的 Snapshot 写入必须以 caller context 通过 `dbtx.Resolve(ctx, root).WithContext(ctx)` 复用 ambient transaction。不得在该路径使用 root DB、`context.Background()` 或嵌套事务；协调器已经锁定 Scan/Target，跨连接写入会形成进程内锁等待并破坏 Snapshot、Asset、summary 的原子提交。

## 扫描历史分区与保留

- 七张 snapshot 表与 `task_progress_log` 是唯一的扫描历史分区集合：统一按 `scan_id` 的半开区间 `RANGE` 分区，跨度固定为 10,000；子表名为 `<parent>_p%08d`。`scan`、`scan_task` 和当前资产投影表都不分区。
- 写入必须命中已预建的当前或下一范围。没有 `DEFAULT` 分区；范围缺失时数据库必须拒绝写入，不能在结果接收热路径临时执行 DDL 或写入无分区回退表。
- 分区生命周期仅由后端作业负责。固定策略至少保留 30 天，以终态 `succeeded`、`failed`、`cancelled` 和 `scan.stopped_at` 判断；同一范围有任一扫描未结束或未到期时，整个范围延后回收。
- 当前可重建开发基线默认以 `enforce` 运行；`report` 只记录候选范围，`disabled` 暂停清理。`enforce` 先软删 scan 使历史不可见，再 detach/drop 八张历史子表，随后限速分批删除该范围的 `scan_task`，最后硬删 scan。当前资产投影绝不能被该流程删除。
- 清理 run 在固定 PostgreSQL 连接上持有 session advisory lock，但准备、每个 task 批次和最终 scan 删除各自独立提交。默认每轮最多处理 1 个范围、每范围最多删除 100 个 1,000 行 task 批次、总时长最多 5 分钟；触及预算时记录 `task_budget_exhausted` 并在下一轮从已软删 scan 和剩余 task 继续。最终删除前必须确认八张历史子表均已消失且范围内没有 task，避免外键级联绕过批次预算。
- 10,000 的跨度预算由 `snapshot_input_scale_postgres_test.go` 在隔离 PostgreSQL 中记录每个范围的行数/字节、分区裁剪、detach/drop、分批 task 删除与 WAL 证据。运行 `make verify-snapshot-input-scale` 后，依据测量结果评估范围数量、最大范围大小、清理时长和被边界范围延后的保留年龄；不要在没有可比数据时承诺固定性能倍数。

当前夹具的验收基线是每个受测父表至少两个已覆盖的 ID 范围，并在一个范围内验证 100k 和 1m 行样本。2026-07-27 的隔离 PostgreSQL 运行中，最大的受测单表范围为 host-port 的 263,176,192 bytes，detach/drop 为 37.9 ms；查询/游标预算仍是 3 分钟，task 删除批次固定为 1,000 行。这些数字是容量评审阈值和回归信号，不是生产 SLA。运行态必须对缺失范围立即报错，并记录 `oldest_blocked_boundary_age`；该年龄超过 30 天仅表示完整范围仍被边界扫描阻塞，需结合扫描状态处理，不能改为逐行提前删除。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- 禁止使用聚合式泛名 mapper 文件（如 `snapshot_mapper.go`）。
