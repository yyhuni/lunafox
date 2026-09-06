# scan/dto

scan 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 scan/task-progress-log/task 相关 DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- scan 业务 DTO 独立维护，不复用共享层业务 DTO（DTO 文件可为 `dto.go` 或 `*_dto.go`）。
- 仅保留对共享 HTTP DTO 的依赖。
- 已分配 Scan 的 detail 必须显式序列化 `agentDeleted`；Agent 存在时返回 `agentName`、`agentStatus`，有 runtime row 时再返回 `agentHealthState`。未分配 Scan 省略整组 Agent 字段，已删除 Agent 只保留 canonical `agent`、`assignmentMode` 与 `agentDeleted=true`。

## Workflow Configuration

`configuration` remains a raw object at the transport boundary because Step IDs
are dynamic. The application must pass it to
`contracts/scanworkflow/configuration`; DTO binding must not add a default,
turn a missing value into `{}`, or infer `enabled` from a Go zero value.

## Batch 操作限制

一般批量操作的数组字段上限为 **5,000**（`max=5000`）；同步
`scans:batchStop` 是锁事务边界，单独限制为 **100** 个 canonical Scan
resource names。具体上限以各 DTO/接口契约为准，禁止在 handler 中静默截断。
