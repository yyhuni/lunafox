## 1. Application task failure object contract (TDD)
- [ ] 1.1 在 `server/internal/modules/scan/application/task_runtime_service_test.go` 增加 failed 状态缺失 `failure.message` 的失败测试（RED）
- [ ] 1.2 在 `server/internal/modules/scan/application/task_runtime_service_test.go` 增加 non-failed 状态携带 `failure` 被拒绝的失败测试（RED）
- [ ] 1.3 运行 `go test ./internal/modules/scan/application -run Failure -count=1`，确认测试先失败
- [ ] 1.4 最小实现 `FailureDetail` 应用层模型与 task `UpdateStatus` 对象校验（GREEN）
- [ ] 1.5 运行 `go test ./internal/modules/scan/application -count=1`，确认相关测试通过

## 2. Scan failure projection and persistence (TDD)
- [ ] 2.1 在 `server/internal/modules/scan/application/task_runtime_service_test.go` 增加 scan 被回算为 failed 时同步投影 `failure.kind/message` 的失败测试（RED）
- [ ] 2.2 在 `server/internal/modules/scan/application/task_runtime_service_test.go` 增加 canonical failed task 优先级选择测试（RED）
- [ ] 2.3 在 `server/internal/modules/scan/application/task_runtime_service_test.go` 增加同优先级下按 stage/完成时间/task ID 稳定打平测试（RED）
- [ ] 2.4 在 `server/internal/modules/scan/repository` 增加 `scan.failure_kind` 持久化契约测试（RED）
- [ ] 2.5 为 `scan` 表新增 `failure_kind` 迁移并更新 `server/internal/modules/scan/repository/persistence/scan.go`（GREEN）
- [ ] 2.6 最小实现 canonical failed task 选取逻辑，并让 scan status 更新链路持久化 scan failure 摘要（GREEN）
- [ ] 2.7 运行 `go test ./internal/modules/scan/application ./internal/modules/scan/repository -count=1`，确认相关测试通过

## 3. Runtime proto and agent transport (TDD)
- [ ] 3.1 在 `agent/internal/task/executor_test.go` 增加失败对象透传测试（RED）
- [ ] 3.2 在 `server/internal/grpc/runtime/service/agent_runtime_connect_test.go` 增加 gRPC `TaskStatus.failure` 透传测试（RED）
- [ ] 3.3 运行 `go test ./internal/task -run Failure -count=1`（agent）与 `go test ./internal/grpc/runtime/service -run Failure -count=1`（server），确认测试先失败
- [ ] 3.4 修改 `proto/lunafox/runtime/v1/runtime.proto`，引入 `FailureDetail` 子消息并移除平铺失败字段（GREEN）
- [ ] 3.5 重新生成 `contracts/gen/lunafox/runtime/v1/*`
- [ ] 3.6 最小实现 `agent/internal/runtime/client.go` 与 `agent/internal/task/executor.go` 的 failure 对象上报（GREEN）
- [ ] 3.7 最小实现 `server/internal/grpc/runtime/service/agent_runtime_service.go` 的 failure 对象接收与传递（GREEN）
- [ ] 3.8 运行 `go test ./internal/runtime ./internal/task -count=1`（agent）与 `go test ./internal/grpc/runtime/service -count=1`（server），确认相关测试通过

## 4. Repository object flattening (TDD)
- [ ] 4.1 在 `server/internal/modules/scan/repository/scan_task_command_test.go` 增加 task failure 对象拍平到 `error_message` / `failure_kind` 的测试（RED）
- [ ] 4.2 运行 `go test ./internal/modules/scan/repository -run Failure -count=1`，确认测试先失败
- [ ] 4.3 修改 `server/internal/modules/scan/repository/scan_task.go` 与 `scan_task_command.go`，使 task repository 接收 `FailureDetail` 或等价输入对象（GREEN）
- [ ] 4.4 保持 SQL 层继续写入 task `error_message` 和 `failure_kind` 两列（GREEN）
- [ ] 4.5 运行 `go test ./internal/modules/scan/repository -count=1`，确认相关测试通过

## 5. Scan HTTP response projection (TDD)
- [ ] 5.1 在 `server/internal/modules/scan/handler` 增加 scan list/detail HTTP 响应返回 `failure.kind/message` 的失败测试（RED）
- [ ] 5.2 运行 `go test ./internal/modules/scan/handler -run Failure -count=1`，确认测试先失败
- [ ] 5.3 修改 `server/internal/modules/scan/domain/query_projection.go`、`dto/scan_dto.go`、`handler/scan_http_mapper.go`，把 scan failure 对象暴露到 HTTP 响应（GREEN）
- [ ] 5.4 视需要更新 `frontend/types/scan.types.ts`，对齐新的 `failure` 对象结构（GREEN）
- [ ] 5.5 运行 `go test ./internal/modules/scan/handler -count=1`，确认相关测试通过

## 6. Cross-layer refactor cleanup
- [ ] 6.1 清理 `normalizeFailureKind`、平铺 `errorMessage/failureKind` 参数和冗余 helper
- [ ] 6.2 统一 application / agent / grpc / repository / HTTP query 命名为 `failure`
- [ ] 6.3 确认 `exit_code` 是移除还是保留，并更新对应测试/生成代码

## 7. Final verification
- [ ] 7.1 运行 `go test ./internal/runtime ./internal/task`（agent）
- [ ] 7.2 运行 `go test ./internal/grpc/runtime/service ./internal/modules/scan/application ./internal/modules/scan/repository ./internal/modules/scan/handler`（server）
- [ ] 7.3 运行 `openspec validate refactor-runtime-task-failure-object --strict --no-interactive`
