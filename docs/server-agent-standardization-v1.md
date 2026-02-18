# Server Agent Standardization v1

## 1. 目标

统一 `agent` 模块的目录职责、命名规范、接口分组与守卫规则，确保后续迭代不回退到泛名与跨层耦合。

## 2. 目录模板

```text
server/internal/modules/agent/
├── domain/
│   ├── entity.go
│   ├── rules.go
│   ├── errors.go
│   ├── repository.go
│   └── events.go
├── application/
│   ├── facade_agent.go
│   ├── agent_query_service.go
│   ├── agent_command_service.go
│   ├── agent_registration_service.go
│   ├── agent_runtime_service.go
│   ├── agent_task_service.go
│   ├── agent_query_ports.go
│   ├── agent_command_ports.go
│   ├── agent_store_ports.go
│   ├── registration_token_store_ports.go
│   ├── agent_message_ports.go
│   ├── task_runtime_ports.go
│   ├── clock_ports.go
│   ├── token_generator_ports.go
│   ├── heartbeat_cache_ports.go
│   └── agent_errors.go
├── repository/
│   ├── agent.go
│   ├── agent_query.go
│   ├── agent_command.go
│   ├── agent_mapper.go
│   ├── registration_token.go
│   ├── registration_token_query.go
│   ├── registration_token_command.go
│   └── persistence/agent.go
├── handler/
│   ├── agent_request_handler.go
│   ├── agent_handler.go
│   ├── agent_registration_handler.go
│   ├── agent_query_handler.go
│   ├── agent_command_handler.go
│   ├── agent_ws_handler.go
│   ├── agent_task_handler.go
│   └── agent_http_mapper.go
└── router/
    └── routes.go
```

## 3. 命名矩阵

- `*_service.go`：application 用例逻辑
- `*_ports.go`：application 端口接口
- `*_errors.go`：application 错误别名与错误语义
- `*_handler.go`：HTTP/WS 适配
- `*_mapper.go`：DTO、payload、model 映射
- `*_query.go`：查询读口实现
- `*_command.go`：写口实现

## 4. 反例

- `handler/types.go`
- `handler/helpers.go`
- `handler/ws_types.go`
- 在 `application` 使用泛名 `contracts.go` 或 `ports.go`
- 在 `handler` 中直接实现业务编排（不经 application）

## 5. API 分组规范

- Runtime API：`/api/agent/*`
- Admin API：`/api/admin/agents/*`

## 6. 守卫映射

- `server/scripts/check-naming-conventions.sh`
  - 禁止 `handler/types.go|helpers.go|ws_types.go`
- `server/scripts/check-handler-boundaries.sh`
  - `handler` 文件名必须 `*_handler.go|*_mapper.go`
  - 顶层导出函数仅允许 `New*Handler`
