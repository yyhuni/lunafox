## Context
当前 runtime 任务状态更新链路已经将失败分类 `failureKind` 引入 agent/server/repository，但仍与 `errorMessage` 以平铺字段形式跨层传递。与此同时，task 与 scan 的失败模型不对称：task 已经有结构化失败分类，scan 作为聚合对象仅保留 `errorMessage`，没有 `failure.kind` 投影，也没有对外 HTTP 失败对象。

## Goals / Non-Goals
- Goals:
  - 在 runtime/application 边界将 task 失败信息收敛为单一 `failure` 对象。
  - 为 scan 聚合模型引入与 task 对齐的 `failure.kind` / `failure.message` 摘要。
  - 扩展 scan HTTP 查询响应，使前端能稳定消费 scan 级 failure 对象。
  - 保持 repository/SQL 查询能力，不牺牲 `failure_kind` / `error_message` 的列式可查询性。
  - 让 TDD 直接验证 failure 对象约束与 scan failure 聚合规则。
- Non-Goals:
  - 本次不把 task 明细列表 HTTP 接口扩展成完整 failure 浏览界面。
  - 本次不将数据库失败信息改为 JSON 字段。
  - 本次不处理历史兼容、双写、双读。
  - 本次不引入 `failure_details JSONB` 扩展字段。

## Decisions
1. Failure object naming
- 使用 `FailureDetail` / `failure`，不使用 `error`。
- 原因：Go 语言内建 `error` 概念过重，业务对象与 `err error` 同名会降低可读性。
- task 与 scan 统一使用相同的 failure 值对象形状：
  - `kind`
  - `message`

2. Runtime protocol shape
- `TaskStatus` 保留 `task_id` 与 `status`。
- 将 `message` 与 `failure_kind` 收敛为 `failure` 子消息：
  - `kind`
  - `message`
- `exit_code` 本次视为死字段候选：如果没有明确业务消费路径，直接从 proto 移除。

3. Task application contract
- `TaskRuntimeService.UpdateStatus` 改为接收 `failure *FailureDetail`。
- 约束集中在 application 层校验：
  - non-failed 状态：`failure == nil`
  - failed 状态：`failure != nil && strings.TrimSpace(failure.Message) != ""`
  - `failure.Kind == ""` 时由 application 统一归一化为 `unknown`
- repository task port 不再暴露 `errorMessage` / `failureKind` 两个平铺参数。

4. Scan failure projection
- scan 也拥有结构化 failure：
  - `scan.failure.kind`
  - `scan.failure.message`
- scan failure 是聚合投影，不是 task failure 的全集。
- scan failure 必须来自一个被选中的 canonical failed task，而不是任意 failed task。
- canonical failed task 选取顺序固定为：
  - 先按 `failure.kind` 优先级排序，优先暴露更接近根因的失败类型；
  - 同优先级时，优先选择更早 stage 的 task；
  - 再同级时，优先选择更早完成失败的 task；
  - 最后以更小的 task ID 打平，确保结果稳定。
- 推荐的 `failure.kind` 优先级分层：
  - P1 配置/契约类：`schema_invalid`、`workflow_config_invalid`、`workflow_prereq_missing`、`decode_config_failed`
  - P2 调度/准入类：`scheduler_rejected` 及后续兼容性 gate 失败
  - P3 启动/控制面类：`worker_start_failed`、`agent_disconnected`
  - P4 运行期基础设施类：`task_timeout`、`container_wait_failed`、`container_exit_failed`
  - P5 兜底类：`runtime_error`、`unknown`
- `scan.failure.kind/message` 直接投影自 canonical failed task 的 `failure.kind/message`。

5. Persistence mapping
- `scan_task.error_message` 存储 `task.failure.message`
- `scan_task.failure_kind` 存储 `task.failure.kind`
- `scan.error_message` 存储 `scan.failure.message`
- `scan.failure_kind` 存储 `scan.failure.kind`
- repository 层负责对象拍平和失败值归一化后的持久化。

6. HTTP response shape
- scan HTTP 查询响应统一扩展 `failure` 对象，而不是继续只暴露 `errorMessage`：
```json
{
  "status": "failed",
  "failure": {
    "kind": "task_timeout",
    "message": "task timed out"
  }
}
```
- 为降低前端切换成本，可在实现期评估是否暂时保留 `errorMessage` 作为过渡字段；但从本次设计目标看，最终应以 `failure.message` 为 canonical response。

7. Domain responsibility
- `ScanTask` 与 `Scan` 领域主状态机仍主要关注 `pending/running/completed/failed/cancelled`。
- `FailureDetail` 可作为 application/query contract 值对象存在，不必强行把所有 failure 规则塞进现有 domain entity。
- query projection、DTO、frontend type 以统一 failure 对象暴露聚合结果。

8. TDD strategy
- 先从 application 层写 task failure contract 测试（RED），确认平铺时代的实现无法满足对象约束。
- 再写 scan failure projection 测试（RED），确认 scan failed 时 failure.kind/message 被同步投影。
- 再写 gRPC/proto 与 agent executor 的传输测试（RED），确认 runtime 协议改造必要性。
- 最后写 repository 与 HTTP response 测试（RED），确认对象输入与对象输出都稳定。

## Architecture Sketch
```text
agent executor
  -> reportStatus(status, failure)
  -> runtime client UpdateStatus(taskID, status, failure)
  -> grpc TaskStatus { task_id, status, failure }
  -> server AgentRuntimeService
  -> scan TaskRuntimeService.UpdateStatus(agentID, taskID, status, failure)
  -> repository UpdateTaskStatus(id, status, failure)
  -> SQL task columns: error_message = failure.message, failure_kind = failure.kind
  -> recalculateScanStatus(scanID, failure)
  -> repository UpdateScanStatus(id, status, failure)
  -> SQL scan columns: error_message = failure.message, failure_kind = failure.kind
  -> HTTP scan response: failure { kind, message }
```

## Failure Model Draft
```go
type FailureDetail struct {
    Kind    string
    Message string
}
```

Proto draft:
```proto
message FailureDetail {
  string kind = 1;
  string message = 2;
}

message TaskStatus {
  int32 task_id = 1;
  string status = 2;
  FailureDetail failure = 3;
}
```

HTTP scan response draft:
```json
{
  "id": 101,
  "status": "failed",
  "failure": {
    "kind": "decode_config_failed",
    "message": "decode workflow config subdomain_discovery: invalid config"
  }
}
```

## Validation Rules
- failed + nil failure => reject
- failed + empty trimmed failure.message => reject
- non-failed + non-nil failure => reject（开发期 fail-fast）
- failure.kind empty => normalized to `unknown` only after failed validation passes
- scan.status == failed => scan.failure must be non-nil
- scan.status != failed => scan.failure must be nil

## Open Questions
- 当前 scan HTTP 是否在一轮重构内保留 `errorMessage` 过渡字段，还是直接切到 `failure.message`。
