# asset/repository

asset 模块 repository 规范：

- `<resource>.go`：资源仓储结构体与构造函数（如 `website.go`、`host_port.go`）。
- `<resource>_query.go`：查询职责（只读）。
- `<resource>_command.go`：写入职责（增删改/批量写）。
- `<resource>_mapper.go`：`domain <-> persistence model` 映射（按资源拆分）。

当前资源：

- `website`
- `endpoint`
- `directory`
- `subdomain`
- `host_port`
- `screenshot`

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
- 禁止使用聚合式泛名 mapper 文件（如 `asset_mapper.go`）。
