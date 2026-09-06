# catalog/repository

catalog 模块 repository 规范：

- `target.go`：`TargetRepository` 结构体与构造函数。
- `target_query.go` / `target_command.go`：Target 查询/写操作。
- `target_mapper.go`：Target 的 `model <-> domain` 映射。
- `target_stats_query.go`：Target 统计查询（资产与漏洞计数）。
- `wordlist.go` + `wordlist_query.go` + `wordlist_command.go`：Wordlist 三分结构。
- `wordlist_mapper.go`：Wordlist 的 `model <-> domain` 映射。
- `subfinder_provider_settings.go` + `*_query.go` + `*_command.go`：Provider 配置三分结构。
- `subfinder_provider_settings_mapper.go`：Provider Settings 的 `model <-> domain` 映射。
- `scan_workflow.go` + `scan_workflow_query.go` + `scan_workflow_command.go`：持久化 scan workflow aggregate 的 Get/List/Create/CAS Update。
- `scan_workflow_mapper.go`：`scan_workflow` JSONB ordered topology 与 domain aggregate 的 fail-closed 映射。
- `scan_workflow_builtin_sync.go`：启动期内置 workflow 的 advisory-lock 事务同步；只能新增、恢复缺失行或同 ID 原位更新，绝不改变用户行归属。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- application command 链需要取消/期限传播时，使用显式 `...Context` query 方法并在 GORM query 上调用 `WithContext`；无 context 的历史入口不得被 result-ingest 路径调用。
- Wordlist repository 的 `file_name` 只用于上传唯一性、筛选/排序和 Catalog 展示属性；canonical resource name 不落库，必须通过 `id` 派生为 `wordlists/{id}`。execution 查询只按 canonical ID 读取，禁止恢复 filename resolver。
- workflow Stage/Step 是 aggregate 内 JSONB，不得拆成独立资源行；读取到无法解码或不符合纯编排约束的 JSONB 必须失败，不能返回部分 workflow。
