# snapshot/repository

snapshot 模块 repository 规范：

- 每个快照资源统一拆分为三类文件：
  - `<resource>_snapshot.go`：仓储结构体、构造函数、筛选映射/公共类型。
  - `<resource>_snapshot_query.go`：查询方法（`Find/Get/Count/Stream/Scan`）。
  - `<resource>_snapshot_command.go`：写操作方法（`BulkCreate/BulkUpsert`）。
  - `<resource>_snapshot_mapper.go`：`domain <-> persistence model` 映射（按资源拆分）。
- 当前资源包括：`website / subdomain / endpoint / directory / host_port / screenshot / vulnerability`。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- 禁止使用聚合式泛名 mapper 文件（如 `snapshot_mapper.go`）。
