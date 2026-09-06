# agent/dto

agent 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 agent 管理与注册相关 DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- agent 业务 DTO 不放回共享层。
- 与 scan/catalog 等模块交互通过 service 层，不直接引用其 DTO。
- Agent 注册只接受 Agent 运行时身份；已删除的 `workerVersion` 即使显式为 `null` 也必须拒绝，不能作为未知字段或兼容输入读取。
- `AgentListQuery` 使用 lowerCamelCase 边界字段：`pageSize`、`pageToken`、`filter`、`orderBy`。DTO 层不声明 `page`、独立 `status`、`include` 或 `sort*` 兼容字段；旧参数拒绝在 handler 层显式完成。
- Agent 管理资源以 `connectionIp` 作为唯一当前连接地址；detail 中的 `observedSourceIp`/`observedIpGeneration` 仅表示 GeoIP 公网观测，且与最后成功 `location.sourceObservedIp` provenance 分离。任何一个 GeoIP 字段都不能作为管理地址或空值回退；`locationState` 固定为 `current`、`expired` 或 `unknown`，unknown 时 `location` 为 `null`。
- 注册令牌创建 DTO 只在创建响应返回一次 bearer `token`，并同时返回 `agentRegistrationTokens/{registration_token}` canonical `name`；后续资源 DTO 仅返回 `name`、`expiresAt`、`state=active|expired` 与完整归因 `agents`，不得再次携带 secret。
