# agent/dto

agent 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 agent 管理与注册相关 DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- agent 业务 DTO 不放回共享层。
- 与 scan/catalog 等模块交互通过 service 层，不直接引用其 DTO。
