# Change: 收敛扫描任务状态机并补充失败分类字段

## Why
当前扫描任务调度链路里，真正的问题不是状态太少，而是失败语义不够清晰：

- 任务在 agent 领取后、真正执行前，也可能失败。
- 如果这些失败都直接落入现有 `failed`，系统本身可以工作，但排障时很难区分失败发生在哪个阶段。
- 之前的扩展设计通过引入 `claimed` 等中间状态来显式建模这些阶段，语义更精细，但也会明显增加状态迁移复杂度。

结合项目当前仍在开发期、且希望优先控制状态机复杂度的诉求，本次变更选择更务实的方向：

- 保持主状态机不扩容
- 继续允许领取后失败直接进入 `failed`
- 用机器可读失败分类字段补足语义

## What Changes
- 保持 scan task 主状态为 `pending`、`running`、`completed`、`failed`、`cancelled`。
- 不在本次引入 `claimed`、`deadLettered`、`retryableFailed` 等新状态。
- 明确 `running` 表示任务已被 agent 接手并进入处理链路，而不是严格表示 worker 已开始执行。
- 将 claim 兜底失败、agent 断线、worker 启动失败、worker 解码配置失败、workflow 执行失败统一收口到 `failed`。
- 为失败任务增加或规范机器可读失败分类字段（例如 `failureKind`），以区分 claim 兜底失败、agent 断线、worker 启动失败、worker 解码失败和 runtime 执行失败。
- 保留创建阶段严格配置校验为主防线，worker/agent 侧校验继续作为兜底防线。

## Impact
- Affected specs:
  - `scan-task-runtime-leasing`（delta 内容将收敛为少状态失败语义）
- Affected code:
  - `server/internal/modules/scan/application/*`
  - `server/internal/modules/scan/repository/*`
  - `server/internal/grpc/runtime/service/*`
  - `server/internal/job/*`
  - `worker/cmd/worker/*`
  - `agent/*`
- Operational impact:
  - 主状态机数量保持不变，但 `running` 与 `failed` 的语义会被重新文档化。
  - 需要新增或规范失败分类字段，并更新日志、监控和测试口径。
