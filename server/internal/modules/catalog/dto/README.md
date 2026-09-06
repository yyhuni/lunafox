# catalog/dto

catalog 模块 DTO 规范：

- `*_dto.go`：按资源拆分放置 catalog 业务 DTO（target/wordlist/engine/workflow）。Profile 是 workflow 的 parent-scoped singleton，不是 collection DTO。
- `httpdto_adapter.go`：仅作为薄适配层，重导出 `server/internal/modules/httpdto` 的共享 HTTP DTO 能力（绑定、分页、统一错误响应）。

约束：

- 不得在 DTO 文件（`dto.go` 或 `*_dto.go`）中复用 `server/internal/dto` 的业务 DTO 别名。
- 不得 import 其他业务模块的 `dto`。
- 与其他模块的字段映射通过 service/application 显式转换。

## Scan Workflow Responses

- `GET /v1/scanWorkflows` returns scan workflow summaries and omits `configuration`; list consumers should not depend on default engine config being present in collection items.
- `GET /v1/scanWorkflows/:scanWorkflow` returns the same pure orchestration resource shape and omits `configuration`.
- `GET /v1/scanWorkflows/:scanWorkflow/profile` returns the only Profile shape: parent identity plus every Step as `{enabled: true, engineConfig: <complete current defaults>}`. The Profile is a fully enabled editing draft; callers can disable Steps before submit. The Step switch is Workflow-owned; Engine `configSection` switches remain inside `engineConfig`.

## Batch 操作限制

| 资源 | 操作 | 单条字段约束 | 每批上限 |
|---|---|---|---|
| Target | `BatchCreateTargetRequest` | `name` max 300 字符 | **5,000** |
| Target | `BatchDeleteTargetRequest` | — | **5,000** |

- 限制通过 struct tag `binding:"required,min=1,max=5000,dive"` 在 DTO 绑定时执行。
- 前端对应常量见 `frontend/lib/target-validator.ts` 的 `MAX_TARGET_BATCH_LINES`。
