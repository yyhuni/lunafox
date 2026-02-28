## 1. Agent task scope hardening (TDD)
- [x] 1.1 在 `worker_runtime_server_test.go` 增加 task token 与 task_id 不匹配时拒绝的失败测试（RED）
- [x] 1.2 运行 `go test ./internal/runtime -run TestWorkerRuntimeRejectsTaskScopeMismatch`，确认测试先失败
- [x] 1.3 最小实现 task session 校验接口与校验逻辑（GREEN）
- [x] 1.4 运行 `go test ./internal/runtime`，确认相关测试通过

## 2. Server data proxy auth hardening (TDD)
- [x] 2.1 在 `server/internal/grpc/runtime/server/server_test.go` 增加 DataProxy 缺失 agent key 被拒绝的失败测试（RED）
- [x] 2.2 运行 `go test ./internal/grpc/runtime/server -run TestServerDataProxyRejectsMissingAgentKey`，确认测试先失败
- [x] 2.3 最小实现 runtime gRPC server 的 DataProxy 鉴权拦截（GREEN）
- [x] 2.4 运行 `go test ./internal/grpc/runtime/server ./internal/grpc/runtime/service`，确认测试通过

## 3. Verification
- [x] 3.1 运行 `go test ./internal/runtime ./internal/task`（agent）
- [x] 3.2 运行 `go test ./internal/grpc/runtime/...`（server）
- [x] 3.3 运行 `openspec validate update-runtime-auth-hardening --strict --no-interactive`
