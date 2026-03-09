# Workflow Profile Config Clone Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 workflow profile 配置 clone 从浅拷贝升级为递归深拷贝，避免嵌套配置被外部修改时反向污染 loader 内部状态。

**Architecture:** 在 catalog wiring adapter 内新增一个递归辅助函数，对 `map[string]any` 与 `[]any` 进行深拷贝。`cloneWorkflowConfig` 保持原入口，只调整内部实现。通过单元测试直接验证嵌套 map 和 slice 在 clone 后不再共享引用。

**Tech Stack:** Go, Go test.

---

### Task 1: 为嵌套配置隔离补失败测试

**Files:**
- Create: `server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_profile_query_store_adapter_test.go`

**Step 1: Write the failing test**
- 验证 clone 后修改嵌套 map 与 slice，不会影响原始配置。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/bootstrap/wiring/catalog -count=1`
Expected: FAIL，浅拷贝下原始配置会被污染。

**Step 3: Write minimal implementation**
- 为 `cloneWorkflowConfig` 增加递归深拷贝逻辑。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/bootstrap/wiring/catalog -count=1`
Expected: PASS。
