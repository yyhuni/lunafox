# Scan Task Failure Kind Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 scan task 引入 `failureKind` 失败分类字段，并将其从 agent/worker 失败上报贯穿到 server 持久化，同时保持主状态机不扩容。

**Architecture:** 保持 `pending/running/completed/failed/cancelled` 主状态不变，仅新增 `failureKind` 作为机器可读失败分类。通过扩展 runtime proto `TaskStatus`、agent 上报接口、server runtime service、scan task repository 和数据库列，使 assignment/agent/worker/runtime 失败路径统一落为 `failed + failureKind`。`FailTaskClaim` 复用现有 `reason` 语义并持久化到 `failure_kind`。

**Tech Stack:** Go, gRPC/protobuf, GORM, PostgreSQL migrations, OpenSpec, Go test。

---

### Task 1: 为持久化模型增加 `failureKind`

**Files:**
- Create: `server/cmd/server/migrations/000002_add_scan_task_failure_kind.up.sql`
- Create: `server/cmd/server/migrations/000002_add_scan_task_failure_kind.down.sql`
- Modify: `server/internal/modules/scan/repository/persistence/task.go`
- Test: `server/internal/modules/scan/repository/persistence/task_model_contract_test.go`

**Step 1: Write the failing test**
- 为 `ScanTask` 模型增加 `FailureKind` 字段契约测试，验证列名为 `failure_kind`。

**Step 2: Run test to verify it fails**
Run: `go test ./server/internal/modules/scan/repository/persistence -run FailureKind -count=1`
Expected: FAIL，提示模型缺少 `FailureKind`。

**Step 3: Write minimal implementation**
- 为 `ScanTask` 模型添加 `FailureKind string` 字段。
- 新增数据库迁移添加 `failure_kind` 列。

**Step 4: Run test to verify it passes**
Run: `go test ./server/internal/modules/scan/repository/persistence -count=1`
Expected: PASS。

### Task 2: 为 repository 持久化/读取 `failureKind`

**Files:**
- Modify: `server/internal/modules/scan/domain/runtime_projection.go`
- Modify: `server/internal/modules/scan/repository/scan_mapper.go`
- Modify: `server/internal/modules/scan/repository/scan_task_command.go`
- Test: `server/internal/modules/scan/repository/scan_mapper_task_observability_test.go`

**Step 1: Write the failing test**
- 扩展 mapper 测试，验证 `failure_kind` 能从 model 映射到 runtime record。

**Step 2: Run test to verify it fails**
Run: `go test ./server/internal/modules/scan/repository -run FailureKind -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 为 runtime `TaskRecord` 增加 `FailureKind`。
- 更新 mapper 读写 `FailureKind`。
- 更新 `UpdateStatus` / `FailTaskClaim` 将 `failureKind` 写入持久化层。

**Step 4: Run test to verify it passes**
Run: `go test ./server/internal/modules/scan/repository -count=1`
Expected: PASS。

### Task 3: 扩展 runtime proto 和 agent 上报接口

**Files:**
- Modify: `proto/lunafox/runtime/v1/runtime.proto`
- Modify: `contracts/gen/lunafox/runtime/v1/runtime.pb.go`
- Modify: `contracts/gen/lunafox/runtime/v1/runtime_grpc.pb.go`
- Modify: `agent/internal/runtime/client.go`
- Modify: `agent/internal/task/executor.go`
- Test: `agent/internal/task/executor_test.go`
- Test: `server/internal/grpc/runtime/service/agent_runtime_connect_test.go`

**Step 1: Write the failing test**
- 为 executor 增加失败分类测试，验证缺 socket、docker 不可用、启动 worker 失败会携带不同 `failureKind`。
- 为 runtime service 连接测试增加断言，验证 gRPC `TaskStatus.failure_kind` 能透传到 task runtime。

**Step 2: Run test to verify it fails**
Run: `go test ./agent/internal/task ./server/internal/grpc/runtime/service -run FailureKind -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 在 proto `TaskStatus` 增加 `failure_kind`。
- 重新生成 Go proto。
- 扩展 agent runtime client `UpdateStatus` 签名。
- executor 在不同失败路径传入具体 `failureKind`。

**Step 4: Run test to verify it passes**
Run: `go test ./agent/internal/task ./server/internal/grpc/runtime/service -count=1`
Expected: PASS。

### Task 4: 扩展 application/runtime service 写入 `failureKind`

**Files:**
- Modify: `server/internal/modules/scan/application/facade_task_status.go`
- Modify: `server/internal/modules/scan/application/task_runtime_service.go`
- Modify: `server/internal/modules/scan/application/task_command_ports.go`
- Modify: `server/internal/modules/scan/application/task_runtime_command_ports.go`
- Modify: `server/internal/bootstrap/wiring/scan/wiring_scan_task_store_adapter.go`
- Modify: `server/internal/bootstrap/wiring/scan/wiring_scan_task_runtime_store_adapter.go`
- Test: `server/internal/modules/scan/application/task_runtime_service_test.go`

**Step 1: Write the failing test**
- 为 `TaskRuntimeService.UpdateStatus` 增加失败分类透传测试。
- 为 `PullTask` claim 失败路径增加断言，验证 `FailTaskClaim` 会保留 `reason` 到 `failureKind`。

**Step 2: Run test to verify it fails**
Run: `go test ./server/internal/modules/scan/application -run FailureKind -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 扩展相关接口签名以接收 `failureKind`。
- `TaskRuntimeService.UpdateStatus` 透传 `failureKind`。
- `failClaimedTask` 将当前 `reason` 持久化为 `failureKind`。

**Step 4: Run test to verify it passes**
Run: `go test ./server/internal/modules/scan/application -count=1`
Expected: PASS。

### Task 5: 更新文档与变更任务状态

**Files:**
- Modify: `docs/plans/2026-03-08-scan-task-lease-state-machine-design.md`
- Modify: `openspec/changes/refactor-scan-task-lease-state-machine/tasks.md`

**Step 1: Update docs**
- 将文档中的建议字段与实现范围对齐。
- 勾选已完成的任务项。

### Task 6: 最终验证

**Files:**
- No code changes expected

**Step 1: Run targeted tests**
Run: `go test ./server/internal/modules/scan/... ./server/internal/grpc/runtime/service ./agent/internal/runtime ./agent/internal/task -count=1`
Expected: PASS。

**Step 2: Run proto and OpenSpec verification**
Run: `make proto-check && openspec validate refactor-scan-task-lease-state-machine --strict --no-interactive`
Expected: PASS。
