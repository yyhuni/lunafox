# Workflow Artifact Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 清理 workflow artifact 简化过程中遗留的编译残留与多余中间层，恢复生成链路与 catalog 构建通过。

**Architecture:** 先修复已确认的残留耦合：server 侧 `executor.APIVersion` 尾巴、worker 侧 profile overlay 旧模型；再把 catalog 从 `Manifest` 直接映射，去掉仅做字段搬运的 `WorkflowMetadata` 中间层。整个过程保持 manifest/profile/executor 现有语义不变，不额外收缩运行时必需字段。

**Tech Stack:** Go, embedded JSON/YAML artifacts, workflow-contract-gen, Go tests

---

### Task 1: 修复 server DTO 的 `APIVersion` 残留

**Files:**
- Modify: `server/internal/modules/catalog/dto/workflow_dto.go`
- Test: `server/internal/modules/catalog/handler/workflow_handler_test.go`

**Step 1: 红灯验证**
- Run: `cd server && go test ./internal/modules/catalog/... -count=1`
- Expected: 因 `APIVersion` 残留导致构建失败

**Step 2: 最小实现**
- 删除 DTO 构造中的 `APIVersion` 赋值，保持返回结构仅包含 `type/ref`

**Step 3: 绿灯验证**
- Run: `cd server && go test ./internal/modules/catalog/... -count=1`
- Expected: catalog 相关测试通过

### Task 2: 删除 worker generator 的 profile overlay 旧模型残留

**Files:**
- Modify: `worker/cmd/workflow-contract-gen/main.go`
- Modify: `worker/cmd/workflow-contract-gen/main_test.go`

**Step 1: 红灯验证**
- Run: `cd worker && go test ./cmd/workflow-contract-gen -count=1`
- Expected: 因 `ContractProfileOverlayDefinition`/`ProfileOverlays` 已删除而构建失败

**Step 2: 最小实现**
- `buildProfileArtifacts` 只生成默认 profile
- 删除 overlay 读取/合并/命名相关函数与测试

**Step 3: 绿灯验证**
- Run: `cd worker && go test ./cmd/workflow-contract-gen -count=1`
- Expected: 生成器测试通过

### Task 3: 去掉 manifest -> metadata 的中间搬运层

**Files:**
- Modify: `server/internal/workflow/manifest/types.go`
- Modify: `server/internal/workflow/manifest/loader.go`
- Modify: `server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_query_store_adapter.go`
- Modify: `server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_query_store_adapter_test.go`
- Modify: `server/internal/workflow/manifest/manifests_test.go`

**Step 1: 先写/改测试**
- 让 catalog adapter 直接依赖 `[]workflowmanifest.Manifest`
- 移除仅验证 `ListWorkflowMetadata` 的测试，改为验证 `ListManifests`/`GetManifest` 语义

**Step 2: 最小实现**
- 删除 `WorkflowMetadata` 类型与 `ListWorkflowMetadata`
- 让 adapter 直接从 `ListManifests()` 映射到应用层 DTO

**Step 3: 绿灯验证**
- Run: `cd server && go test ./internal/workflow/manifest ./internal/modules/catalog/... -count=1`
- Expected: 测试通过

### Task 5: 收缩 manifest 顶层字段

**Files:**
- Modify: `server/internal/workflow/manifest/types.go`
- Modify: `server/internal/workflow/manifest/validate.go`
- Modify: `server/internal/workflow/defaulting/defaulting.go`
- Modify: `worker/cmd/workflow-contract-gen/manifest.go`
- Test: `server/internal/workflow/manifest/manifests_test.go`
- Test: `worker/cmd/workflow-contract-gen/main_test.go`

**Step 1: 红灯验证**
- 先让测试断言 manifest 不再包含 `configSchemaId` / `supportedTargetTypeIds` / `defaultProfileId`
- 运行 `go test` 验证旧字段仍存在时测试失败

**Step 2: 最小实现**
- 删除 manifest 三个顶层字段
- defaulting 直接按 `workflowId` 读取默认 profile
- 重生成 server manifest/profile/docs

**Step 3: 绿灯验证**
- 运行 worker/server 相关回归，确认生成链路与默认值归一化通过

### Task 4: 全量回归

**Files:**
- Modify: `openspec/changes/introduce-workflow-executor-binding/*`（仅当文档仍提及 `apiVersion` 时）

**Step 1: 回归验证**
- Run: `cd worker && go test ./cmd/workflow-contract-gen ./cmd/worker ./internal/workflow/... -count=1`
- Run: `cd server && go test ./internal/workflow/manifest ./internal/modules/catalog/... -count=1`

**Step 2: 规格校验（如有需要）**
- Run: `openspec validate introduce-workflow-executor-binding --strict --no-interactive`

**Step 3: 收尾**
- 汇总已删除项、保留项与下一轮可继续收缩的字段（如 `configSchemaId` / `supportedTargetTypeIds` / `defaultProfileId`）
