# Change: Runtime Compatibility Gate 按 Worker 能力判定（与 Agent 版本解耦）

## Why
当前调度兼容性 gate 通过 `agent.version` 推断 worker 能力。该模型在 `AGENT_VERSION` 与 `WORKER_IMAGE_REF` 非严格同源时会产生误判（可分配但实际不兼容，或可执行却被误拒绝），并且阻碍后续批量迁移 workflow 的能力管理。

## What Changes
- 将调度兼容性判定从“按 agent 版本映射”改为“按 worker 能力快照判定”。
- 在 agent heartbeat/runtime 上报链路中新增 worker 能力字段（至少含 `workerVersion`，可扩展 `supportedWorkflows`）。
- server 持久化 worker 能力快照，并提供 `TaskRuntime` 读取端口。
- `WorkflowCompatibilityGate` 改为基于 worker 能力判定，不再依赖 `agent.version` 静态映射表。
- 保持错误语义不变：不兼容仍返回 `WORKER_VERSION_INCOMPATIBLE`（stage 为 `scheduler_compatibility_gate`）。
- 增加“老 agent 无 worker 能力上报”时的显式失败策略（fail-closed），避免隐式放行。
- 同步纳入“防遗漏解耦清单”并分阶段执行：
  - P0（本次）: 兼容性 gate、heartbeat 能力字段、调度错误契约稳定。
  - P1（紧随）: 更新链路解耦（update_required 下发 worker 目标）、存储语义对齐（`scan_task.version` 漂移清理、agent/worker version 字段拆分）。
  - P2（后续）: 历史协议模型清理（`agentproto/protocol` 与 gRPC runtime 单源化）、跨端版本规则单源化、workflow catalog 解耦。

## Impact
- Affected specs:
  - `runtime-worker-compatibility` (ADDED)
- Affected code:
  - `contracts/gen/lunafox/runtime/v1/*`（由 proto 变更生成）
  - `agent/internal/runtime/heartbeat.go`
  - `server/internal/grpc/runtime/service/runtime_mappers.go`
  - `server/internal/modules/agent/application/*`（heartbeat 处理与持久化）
  - `server/internal/modules/scan/application/task_runtime_compatibility.go`
  - `server/internal/modules/scan/application/task_runtime_service.go`
  - `server/internal/bootstrap/wiring/scan/*`
- Data impact:
  - agent runtime 投影增加 worker 能力字段（数据库迁移）
- Operational impact:
  - 调度前兼容性判定更准确；能力缺失时更早失败，减少运行时隐式报错。
