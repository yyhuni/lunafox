# Scan Task Compact Failure Semantics Design

## Context
当前任务调度已经使用数据库原子更新保障多 agent 下的并发领取安全。系统真正面临的问题，不是“状态一定太少”，而是“失败语义还不够清晰”。

具体来说，下面这些情况都可能出现在任务被分配之后：
- server 侧 claim 后静态兜底失败
- agent 收到任务后立刻断线
- agent 本地拉起 worker 失败
- worker `decodeWorkflowConfig` 失败
- workflow 实际执行失败

这些情况未必都需要扩展为新的显式状态。对于当前阶段，更务实的做法是保持主状态机紧凑，并通过字段补足失败分类语义。

## Goals / Non-Goals

### Goals
- 保持任务主状态机简洁。
- 允许领取后、执行前失败直接进入 `failed`。
- 用机器可读字段区分失败发生阶段。
- 保持 server 创建期校验为主防线，worker/agent 端校验为兜底。
- 保留未来升级为显式 `claimed` 的空间。

### Non-Goals
- 不在本次引入 `claimed`、`retryableFailed`、`invalid`、`deadLettered` 等新状态。
- 不在本次引入完整租约续期、DLQ、attempt 表等复杂调度模型。
- 不要求本次同时把 `startedAt` 完全重定义为执行起点门槛。

## Approaches

### A. 维持现状，不补失败分类
- 做法：状态机和字段都不变。
- 优点：无迁移成本。
- 缺点：排障时无法稳定区分 claim、agent、worker、runtime 四类失败。

### B. 增加 `claimed` 显式状态
- 做法：引入 `pending -> claimed -> running`。
- 优点：语义最清晰。
- 缺点：状态迁移、协议、测试和监控都更复杂。

### C. 保持少状态，补 `failureKind` 字段（推荐）
- 做法：继续使用 `pending -> running -> completed/failed/cancelled`，把领取后但未成功完成的失败统一记为 `failed`，并记录失败分类。
- 优点：实现成本最低，状态机保持紧凑，足以解决当前主要排障问题。
- 缺点：`failed` 承载的语义更宽，需要依赖字段解释具体失败阶段。

## Final Recommendation
推荐采用 **方案 C：少状态 + 失败分类字段**。

这套方案承认一个工程现实：
- 对当前阶段来说，“少状态、强分类”比“多状态、全显式”更符合项目复杂度预算。
- 只要失败原因可被可靠记录，系统并不一定需要为了这次问题引入额外状态。
- 由于项目仍处于开发期，`failureKind` 规范值会直接收敛为统一命名，不保留 legacy camelCase 兼容重写；仅保留 workflow 领域错误码到失败分类码的翻译。

## State Model

### Task Statuses
- `pending`：等待调度。
- `running`：任务已被 agent 接手并进入处理链路。
- `completed`：执行成功。
- `failed`：任务最终失败，失败可以发生在执行前或执行中。
- `cancelled`：任务被外部取消。

### Supporting Fields
建议新增或统一以下字段：
- `failureKind`
- `errorMessage`
- `agentId`
- `startedAt`（可选增强，不作为本次状态机切换前提）

### Suggested Failure Kinds
- `agent_disconnected`
- `worker_start_failed`
- `decode_config_failed`
- `runtime_error`
- `unknown`

## Data Flow

### 1. Create Phase
- scan 创建阶段继续执行 manifest/defaulting/schema 严格校验。
- 非法 workflow 配置应尽量在 server 创建阶段就被拦截。

### 2. Claim / Assignment Phase
- 调度器继续使用现有原子 SQL 将任务从 `pending` 改为 `running`。
- 该状态表示“任务已进入 agent 处理链路”，不再额外拆出 `claimed`。

### 3. Failure Handling
下列情况统一进入 `failed`，同时记录 `failureKind`：
- server 侧 claim 后静态兜底失败
- agent 在领取后断线
- agent 无法启动 worker
- worker 无法解码 workflow 配置
- workflow 执行过程中报错

### 4. Success Handling
- workflow 正常执行并保存结果后，任务进入 `completed`。

## Scan Aggregation Semantics
- 只要仍存在 `pending` 或 `running` 任务，scan 继续保持活跃。
- 当不存在活跃任务时：
  - 若存在 `failed`，scan 进入 `failed`
  - 若全部为 `completed`，scan 进入 `completed`
  - 若属于停止/删除场景，scan 进入 `cancelled`

## Migration Plan
1. 明确 `running` 与 `failed` 的新文档语义。
2. 为任务失败增加或规范 `failureKind` 字段持久化。
3. 将各类领取后失败路径统一写入 `failed + failureKind`。
4. 更新日志、测试与观测口径。
5. 仅当未来出现明确收益时，再升级为显式 `claimed`。

## Out of Scope
- This change SHALL NOT introduce gRPC status retry, ACK confirmation, or stale-running compensation logic.
- This change SHALL NOT add a separate `claimed` status.

## Risks / Trade-offs
- 风险：`failed` 语义范围更宽。
  - 缓解：依赖 `failureKind` 和错误消息做分类。
- 风险：`running` 不再严格等于“worker 已执行”。
  - 缓解：将其文档化为“agent 已接手并处于处理链路中”。
- 风险：未来若做租约恢复，可能需要再次演进。
  - 缓解：本次先解决当前主要排障痛点，未来升级路径清晰。

## Testing Strategy
- repository：验证失败路径统一进入 `failed` 并带正确 `failureKind`。
- application：验证 claim 兜底、agent、worker 启动、decode、runtime 执行几类错误分类。
- runtime service：验证日志与错误映射保持一致。
- verification：运行 OpenSpec 校验与相关 Go tests。
