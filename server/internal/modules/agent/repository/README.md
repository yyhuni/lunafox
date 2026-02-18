# agent/repository

agent 模块 repository 规范：

- `agent.go`：`AgentRepository` 接口、仓储结构体与构造函数。
- `agent_query.go`：Agent 查询方法（`Find* / List`）。
- `agent_command.go`：Agent 写操作方法（`Create / Update / Delete / Update*`）。
- `registration_token.go`：注册令牌仓储接口与构造函数。
- `registration_token_query.go`：注册令牌查询方法。
- `registration_token_command.go`：注册令牌写操作方法。

约束：

- 禁止使用 `*_mutation.go` 命名。
- 禁止使用泛名 `types.go`。
- `*_query.go` 不得出现写操作方法；`*_command.go` 不得出现查询方法。
