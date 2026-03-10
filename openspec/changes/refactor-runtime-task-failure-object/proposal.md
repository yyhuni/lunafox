# Change: 运行时任务与扫描失败信息收敛为结构化 failure 对象

## Why
当前 runtime/task 状态更新链路已经引入 `failureKind`，但失败信息仍以平铺字段形式在多个边界传递：`status`、`message`、`failureKind`。同时，task 与 scan 的失败建模并不对称：
- task 已经具备 `failure_kind + error_message` 语义；
- scan 仅保留 `errorMessage`，没有结构化失败分类；
- HTTP scan 查询响应只暴露人类可读 message，前端无法稳定消费机器可读失败原因。

这种设计会带来几个问题：
- 失败相关信息缺少单一语义边界，`errorMessage` 与 `failureKind` 的约束只能靠 helper 和调用约定维持。
- gRPC 入站、application service、repository port 都在重复携带失败平铺参数，接口扩展成本高。
- scan 虽然会被聚合推导为 `failed`，但 scan 自身没有与 task 对齐的结构化 failure 语义，导致模型不对称。
- 前端 scan 页面只能展示字符串 `errorMessage`，无法基于失败类型进行稳定展示、筛选或统计。

项目当前仍处于开发阶段，可以在不考虑兼容窗口的前提下，将 runtime/task/scan 失败信息一次性收敛为结构化对象，统一上下游建模。

## What Changes
- 在 runtime 协议层引入结构化 `TaskFailure` 对象，收敛任务失败信息。
- `TaskStatus` 从平铺 `message` + `failure_kind` 重构为 `failure` 对象。
- agent runtime client、executor、server gRPC runtime service、scan application service 统一改为传递 `failure` 对象，而非平铺失败参数。
- scan 聚合层引入与 task 对齐的结构化 `failure` 投影：
  - `scan.failure.kind`
  - `scan.failure.message`
- 定义 scan failure 的聚合语义：scan 的 failure 表示“导致 scan 最终进入 failed 的主失败摘要”，而不是所有 task failure 的全集。
- repository / SQL 持久化层继续采用列式存储：
  - `scan_task.error_message` / `scan_task.failure_kind`
  - `scan.error_message` / `scan.failure_kind`
- scan HTTP 查询响应扩展 `failure` 对象，让前端可以稳定获取 scan 级结构化失败信息。
- 本次不将数据库失败信息改为 JSON 字段。
- 本次将 `exit_code` 明确为非目标：若实现链路仍无消费价值，则在重构中移除该死字段；否则延期到独立变更处理。

## Impact
- Affected specs:
  - `runtime-communication-grpc` (MODIFIED)
  - `scan-task-runtime-failure-model` (ADDED)
- Affected code:
  - `proto/lunafox/runtime/v1/runtime.proto`
  - `contracts/gen/lunafox/runtime/v1/*`
  - `agent/internal/runtime/client.go`
  - `agent/internal/task/executor.go`
  - `server/internal/grpc/runtime/service/*`
  - `server/internal/modules/agent/application/*`
  - `server/internal/modules/scan/application/*`
  - `server/internal/modules/scan/repository/*`
  - `server/internal/modules/scan/dto/*`
  - `server/internal/modules/scan/handler/*`
  - `frontend/types/scan.types.ts`
- Schema / migration impact:
  - 需要为 `scan` 表新增 `failure_kind` 列，并与 `error_message` 一起承载 scan 级 failure 摘要。
- Testing impact:
  - 需要新增/修改 agent runtime、server grpc runtime、scan application、repository、scan query HTTP 的 TDD 测试。
- Deployment impact:
  - 本次按开发期一次性切换处理，不保留兼容层。
