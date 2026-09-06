# catalog/application

通用规则请遵循：`server/docs/server-application-naming-template-v1.md`

catalog 模块补充规则：

- **入口聚合**：facade 按 target、wordlist、scan workflow 管理业务视角拆分；scan workflow 由 `workflow_management.go` 处理 Create/List/Get/Update，`workflow_profile.go` 处理 parent-scoped Profile singleton。
- **Facade 边界**：facade 只保留入口委派和边界错误语义映射调用；重复的 target/wordlist/file not-found 映射统一收口到 `catalog_facade_helpers.go`。
- **服务编排**：target command 按场景拆分为 `target_command_crud.go` 与 `target_command_batch.go`；其余资源遵循 query/command 分层。
- **adapter 例外**：当前无 transport-facing 或 cross-module bridge 型 adapter service 例外；`local_wordlist_file_store.go` 属于局部默认实现，不属于模块入口适配层。
- **端口拆分**：端口按资源职责拆分，wordlist 文件能力采用 port + default implementation（`wordlist_file_ports.go` + `local_wordlist_file_store.go`）。
- **实现唯一性**：`WordlistFileStore` 的默认实现仅保留在 `application/local_wordlist_file_store.go`，避免在 `infrastructure` 层出现同名重复实现造成歧义。
- **模型命名**：新增输入/输出/中间模型优先资源化命名（如 `*_query_inputs.go`、`*_item_models.go`）。
- **历史迁移**：`aliases.go`、`errors.go` 已完成首批迁移，继续保持资源化命名不回退。
- **Subfinder provider registry**：Subfinder API key settings 以 Engine Runtime Image 内的 Subfinder `v2.12.0` 为版本锚点，registry 是 provider key、字段、校验与 runtime YAML 生成的唯一行为源；历史八字段 provider 只允许出现在持久化迁移/status 处理和对应测试中。
- **Execution artifact reads**：`ExecutionWordlistSource` 与 `ExecutionProviderConfigSource` 只接受调用方 `context.Context`，并将取消/期限传到 repository；HTTP 管理 facade 保留面向产品管理面的既有 API，但不得用于 execution artifact 路径。
- **Wordlist identity**：Wordlist domain/persistence 使用 `FileName`/`file_name` 表示不可变的上传 basename；HTTP、Scan、Plan 和 execution source 的 `name` 一律由稳定 ID 派生为 `wordlists/{id}`。按 `fileName` 的直接资源读取不是 runtime 入口，候选匹配必须先经过完整 Catalog 列表并唯一解析。
- **Target 写入边界**：单条创建、批量创建/ensure 与 rename 必须统一通过 catalog domain canonical builder；DOMAIN 统一为 canonical ASCII DNS name，IPv4 统一为 canonical dotted decimal，IPv4 CIDR 持久化 masked network address；IPv6 literal、IPv6 CIDR 与 IPv4-mapped IPv6 在写入时直接拒绝，不得延迟到 scan planning 或 execution input producer。批量创建的 context-aware 路径必须在同一数据库事务内完成 active organization 业务分组校验、缺失 target 与 blacklist policy 写入、canonical target 查询和 organization-target 关联；REST 与 MCP 均复用该命令，提交后丢失响应不回滚已提交状态。
- **Scan workflow 管理**：数据库 workflow 是运行期真源。List/Get 动态计算 `isExecutable`，历史用户 workflow 缺少 Engine 时仍可读取；Create/Update 只接受纯编排并校验所有 `engineId`。内置 workflow 只能由启动同步原位更新，管理 API 必须拒绝修改。Profile 不是集合或持久化资源，而是按 workflow 从当前 Engine Definition 物化出的完整默认配置，并为每个 Step 显式输出 `enabled: true`；它是 Server 生成的初始编辑草稿，不是 Scan 请求解析 fallback。Step 开关属于 Workflow，`configSection` 开关仍属于 Engine 配置。`requiredEnabled:true` 仅投影为单个内层 section 必须启用的元数据：Profile 必须输出其完整启用默认值，catalog 不得将它扩展为分组、条件或运行时编排规则。
