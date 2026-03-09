# 扫描任务少状态失败语义设计（草案）

## 背景
LunaFox 当前已经具备多 agent 并发领取任务的数据库基础能力，但任务生命周期里同时存在两类失败：

- 真正执行中的失败
- 领取、下发、启动、解码等执行前失败

之前的扩展设计倾向于通过增加 `claimed` 等中间状态来把这些阶段全部显式化。但结合当前项目仍在开发期、且希望保持状态机简洁的诉求，更合适的方案是：

- 保持主状态机少而稳定
- 不额外引入 `claimed`
- 用失败分类字段补足语义

## 目标
- 保持任务主状态机简洁，避免一次性引入过多状态和流转规则。
- 允许领取后但执行前的失败直接进入 `failed`。
- 用字段区分失败发生在分配、启动、解码还是执行阶段。
- 保持创建阶段校验为主防线，worker/agent 校验作为兜底防线。
- 为将来若确实需要 `claimed` 预留升级空间，但不在本次引入。

## 推荐方案
采用“**少状态 + 失败分类字段**”方案：

- 主状态机保持为：`pending -> running -> completed/failed/cancelled`
- 不新增 `claimed`
- `running` 语义定义为：任务已被某个 agent 接手并进入处理链路
- 以下场景都允许直接进入 `failed`：
  - server 侧 claim 后的静态兜底失败
  - agent 收到任务后立即断线
  - agent 本地拉起 worker 失败
  - worker `decodeWorkflowConfig` 失败
  - workflow 实际执行失败
- 通过字段区分失败来源，而不是再拆多个失败状态

## 为什么当前阶段更适合这个方案
- 复杂度最低：不需要引入新状态，也不需要扩展过多状态迁移测试矩阵。
- 迁移成本最低：现有 repository、application、runtime、agent 行为都能在当前结构上增量演进。
- 工程收益足够：虽然语义不如完整租约模型精细，但对当前问题已经足够可观测、可排障、可维护。
- 升级路径保留：如果未来确实需要精细租约恢复，再从当前模型升级为显式 `claimed` 也不晚。

## 命名收敛约束
- 本次将 `failureKind` 的规范值统一为 `lower_snake_case`。
- 由于项目仍处于开发期，不保留旧 camelCase 失败分类值的兼容重写。
- 仅保留 workflow 领域错误码到 `failureKind` 的显式翻译，例如 `SCHEMA_INVALID -> schema_invalid`。

## 状态模型
### 主状态
- `pending`：等待分配。
- `running`：已被 agent 接手，处于下发、启动或执行处理链路中。
- `completed`：成功完成。
- `failed`：本次任务以失败结束，无论失败发生在执行前还是执行中。
- `cancelled`：被 scan 停止、删除或系统操作取消。

### 补充字段
建议补充或规范以下字段：
- `failureKind`：机器可读失败分类。
- `errorMessage`：人类可读错误说明。
- `startedAt`：如实现成本可接受，后续可逐步收敛为“真实 worker 开始执行时间”；本次不把它作为状态机的强依赖条件。
- `agentId`：记录当前接手任务的 agent。

### 建议的 `failureKind`
- `agent_disconnected`
- `worker_start_failed`
- `decode_config_failed`
- `runtime_error`
- `unknown`

## 数据流
### 1. 创建阶段
- 创建 scan 时继续在 server 侧做 workflow manifest/defaulting/schema 的严格校验。
- 非法配置应尽量在创建阶段被拦住，而不是进入可调度队列。

### 2. 领取阶段
- 调度器继续通过原子 SQL 将任务从 `pending` 改为 `running`。
- 这一步表示该任务已经被某个 agent 接手，而不再强求它已真正进入 worker 执行。

### 3. 失败阶段
以下情况都可以直接进入 `failed`，并写入对应 `failureKind`：
- server 侧 claim 后的静态兜底失败：沿用现有 `reason`/workflow error code 持久化
- agent 断线：`agent_disconnected`
- worker 启动失败：`worker_start_failed`
- worker 配置解码失败：`decode_config_failed`
- workflow 执行失败：`runtime_error`

### 4. 成功阶段
- workflow 正常执行并保存结果成功后，任务进入 `completed`。

## 为什么这里可以直接用 `failed`
在当前方案里，`failed` 被定义为“**任务最终未成功完成**”，而不是狭义上的“worker 已经开始执行后失败”。

也就是说，系统承认 `failed` 可以同时覆盖：
- 执行前失败
- 执行中失败

这种做法的代价是语义没有显式 `claimed` 精细，但优点是：
- 状态机更简单
- 系统行为更直接
- 对当前阶段的工程复杂度更友好

只要 `failureKind` 记录充分，就仍然能区分失败发生在什么阶段。

## Scan 聚合语义
- 只要存在 `pending` 或 `running` 任务，scan 就仍处于活跃状态。
- 当没有 `pending` / `running` 任务时：
  - 若存在 `failed`，scan 进入 `failed`
  - 若全部为 `completed`，scan 进入 `completed`
  - 若为取消场景，scan 进入 `cancelled`

该聚合规则与当前模型保持接近，不需要为了新增中间状态调整太多统计口径。

## Migration Plan
1. 明确 `running` 的新语义：表示任务已被 agent 接手并进入处理链路。
2. 为任务失败补充 `failureKind` 字段或等价持久化能力。
3. 将领取后、执行前、执行中的各类失败路径统一收口到 `failed + failureKind`。
4. 更新日志、监控和测试，使其不再默认把所有 `failed` 都解释为“执行阶段失败”。
5. 保留未来升级到显式 `claimed` 的空间，但本次不引入额外状态。

## Out of Scope
- 本次不引入 gRPC 状态上报重试、ACK 确认或 stale running 自动补偿逻辑。
- 本次不引入 `claimed` 等新增状态。

## Risks / Trade-offs
- 风险：`failed` 会同时承载多种失败场景。
  - 缓解：通过 `failureKind` 和 `errorMessage` 保持可观测性。
- 风险：`running` 语义不再等同于“worker 真正已开始执行”。
  - 缓解：在文档、日志和监控口径中明确其含义为“agent 已接手”。
- 风险：后续若要做精细租约恢复，需要再次演进状态机。
  - 缓解：本次先把失败分类收敛好，未来升级路径仍然清晰。

## Testing Strategy
- repository：验证领取后失败路径能正确写入 `failed` 和 `failureKind`。
- application：验证 claim 兜底失败、agent 断线、worker 启动失败、decode 失败都统一收口到 `failed`。
- runtime service：验证错误分类和日志字段正确。
- verification：实现前后运行 OpenSpec 校验与相关 Go tests。
