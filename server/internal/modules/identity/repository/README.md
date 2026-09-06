# identity/repository

identity 模块 repository 规范：

- `user.go` + `user_query.go` + `user_command.go`：用户仓储三分结构。
- `organization.go`：组织仓储公共定义（错误、映射、结构体）。
- `organization_query.go`：组织查询方法。
- `organization_command.go`：组织写操作方法。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
