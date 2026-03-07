# Worker Log Field Semantics Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把剩余 worker 应用日志字段统一为语义化点分命名，并增加 CI 守门防止 `camelCase` zap key 回流。

**Architecture:** 这次是 focused refactor，只触碰剩余日志发射点和命名检查脚本。实现顺序是先写失败测试锁定旧 key，再做最小改名，最后扩 CI fixture 与真实仓库扫描，确保 OTel 语义字段不被误伤。

**Tech Stack:** Go, Zap, Bash, ripgrep, OpenSpec

---

### Task 1: 锁定 worker 旧日志 key

**Files:**
- Modify: `worker/cmd/worker/main.go`
- Modify: `worker/internal/server/batch_sender.go`
- Modify: `worker/internal/workflow/subdomain_discovery/stage_merge.go`
- Modify: `worker/internal/workflow/subdomain_discovery/workflow.go`
- Modify: `worker/internal/activity/runner_output.go`
- Modify: `worker/internal/activity/runner_execution.go`
- Test: existing adjacent `*_test.go` files, or add focused log field tests beside touched packages

**Step 1: Write the failing tests**
- Add assertions that old keys like `taskId`, `queueLength`, `retryCount`, `sampleCount`, `maxBytes`, and `exitCode` no longer appear.
- Assert replacement keys like `task.id`, `queue.length`, `retry.count`, `sample.count`, `buffer.max_bytes`, and `process.exit_code` appear.

**Step 2: Run targeted tests to verify they fail**
- Run package-level worker tests for the touched packages.
- Expected: failures reference old log key names still emitted by current code.

**Step 3: Write minimal implementation**
- Replace only the remaining legacy key strings; do not refactor unrelated logging structure.
- Keep OTel-aligned HTTP fields untouched.

**Step 4: Run targeted tests to verify they pass**
- Re-run the touched Go package tests until the new semantic keys are locked.

### Task 2: 扩展 CI 命名守门

**Files:**
- Modify: `scripts/ci/check-interface-naming.sh`
- Modify: `scripts/ci/check-interface-naming-test.sh`

**Step 1: Write the failing fixture test**
- Add a bad fixture that emits `zap.Int("taskId", 1)` or `zap.Int("queueLength", 2)` in worker/server code.
- Add a good fixture that keeps OTel semantic fields like `http.response.status_code`.

**Step 2: Run fixture test to verify it fails**
- Run `scripts/ci/check-interface-naming-test.sh`.
- Expected: the new fixture passes through current script and exposes the enforcement gap.

**Step 3: Write minimal implementation**
- Extend the script with a new rule that rejects ad-hoc `camelCase` zap keys in Go source.
- Exclude documented OTel semantic convention keys and generated/test files.

**Step 4: Run fixture test to verify it passes**
- Re-run `scripts/ci/check-interface-naming-test.sh` and confirm all fixtures behave as expected.

### Task 3: 全量验证

**Files:**
- Verify only; no new files required

**Step 1: Run targeted Go tests**
- Run worker package tests for all touched logging packages.

**Step 2: Run naming guardrails**
- Run `./scripts/ci/check-interface-naming-test.sh`
- Run `./scripts/ci/check-interface-naming.sh`

**Step 3: Validate OpenSpec**
- Run `openspec validate refactor-worker-log-field-semantics --strict --no-interactive`

**Step 4: Mark checklist complete**
- Update `openspec/changes/refactor-worker-log-field-semantics/tasks.md` to checked state only after commands pass.
