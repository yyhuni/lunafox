# security/repository

security 模块 repository 规范：

- `vulnerability.go`：仓储结构体、构造函数、筛选映射与公共常量。
- `vulnerability_query.go`：漏洞查询与统计查询方法。
- `vulnerability_command.go`：漏洞写操作方法（批量创建、删除、审核状态变更）。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
