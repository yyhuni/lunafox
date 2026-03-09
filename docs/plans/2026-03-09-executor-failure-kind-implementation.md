# Executor Failure Kind Refinement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 executor 的 `failureKind` 上报优先基于明确失败路径分类，仅在容器退出日志场景使用文本分类兜底。

**Architecture:** 在 `execute`/`handleTimeout` 的具体失败分支直接传入结构化 `failureKind`，减少对 `classifyFailureKind` 的依赖。`classifyFailureKind` 仅保留对容器非零退出日志的细分，其中配置解码失败返回 `decode_config_failed`，其余返回 `container_exit_failed`。

**Tech Stack:** Go, Go test.

---

### Task 1: 为新的失败分类补测试

**Files:**
- Modify: `agent/internal/task/executor_test.go`

**Step 1: Write the failing test**
- 增加 `Wait` 普通错误时应上报 `container_wait_failed`。
- 增加 timeout 时应上报 `task_timeout`。
- 调整容器非零退出默认分类为 `container_exit_failed`。

**Step 2: Run test to verify it fails**
Run: `go test ./agent/internal/task -count=1`
Expected: FAIL，旧实现仍返回 `runtime_error`。

**Step 3: Write minimal implementation**
- 更新 executor 失败分类常量与分支。

**Step 4: Run test to verify it passes**
Run: `go test ./agent/internal/task -count=1`
Expected: PASS。
