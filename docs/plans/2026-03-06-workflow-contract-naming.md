# Workflow Contract Naming Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 workflow contract、registration、generated manifest、程序化 API 与 docs 产物的命名统一到一套满分命名规则下。

**Architecture:** 以 Go contract 作为命名单一事实源，先统一类型名、字段名与函数 / 方法名，再让 generated manifest、server manifest loader 与 docs 产物全部映射到新命名。workflow schema 已回归纯 JSON Schema，因此不再承载 LunaFox 业务命名，也不再是本次命名改造的终局对象。generated manifest 通过 `defaultProfileId` 引用独立 profile artifact；activity template metadata 与 workflow manifest 使用不同模型。整个改造不保留兼容桥接，并作为后续 defaulting 变更的命名基线。

**Tech Stack:** Go 1.26, JSON Schema Draft-07, YAML v3, OpenSpec

---

### Task 1: 锁定 contract / manifest 命名目标行为

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry_test.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition_constraints_test.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_generation_consistency_test.go`
- Check: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`

**Step 1: Write the failing test**

新增或调整测试，明确要求：
- contract 类型名和字段名只接受新命名；
- registration 层使用 `WorkflowID`，不再接受 `Name` 语义；
- manifest 字段只接受 `workflowId`、`displayName`、`supportedTargetTypeIds`、`defaultProfileId` 等新命名；
- contract 函数与方法名只接受 `WorkflowContract` 话语体系；
- activity template metadata 不得被当作 manifest 结构复用。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1
```

Expected: 测试或编译因旧命名仍存在而失败。

**Step 3: Write minimal implementation**

先不改 docs，先调整测试与辅助断言，让它们明确锁定新命名契约。

**Step 4: Run test to verify it still fails for the right reason**

Expected: 失败聚焦到生产代码仍使用旧命名。

---

### Task 2: 重命名 workflow contract 类型与字段

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry_test.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition_constraints_test.go`

**Step 1: Write the failing test**

新增一个聚焦测试，断言：
- registration 入口字段为 `WorkflowID`；
- stage / tool / parameter 使用 `StageID` / `ToolID` / `ConfigKey`；
- 展示名字段在类型上下文明确时使用 `DisplayName`；
- `DefaultProfile` 在 contract 层保留，但 manifest 层输出 `defaultProfileId`。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
```

Expected: 因 contract 仍保留旧字段而失败。

**Step 3: Write minimal implementation**

仅重命名 contract 与 registration，不马上动 generator 输出。

**Step 4: Run test to verify it passes**

Expected: contract / registration 相关测试通过，生成产物测试可能仍失败。

---

### Task 3: 统一 generated manifest 命名

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/*`

**Step 1: Write the failing test**

新增测试要求：
- generated manifest 使用 `workflowId`、`displayName`、`supportedTargetTypeIds`、`defaultProfileId`；
- stage / tool / parameter 元数据使用 `stageId`、`toolId`、`configKey`；
- 不再输出旧的 `name`、`target_types` 一类字段。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen ./internal/workflow/... -count=1
```

Expected: 当前产物仍沿用旧命名，测试失败。

**Step 3: Write minimal implementation**

修改 generator 的 manifest 输出映射，并同步更新 fixture。

**Step 4: Run test to verify it passes**

Expected: manifest 命名相关测试通过。

---

### Task 4: 拆分 manifest 与 activity template metadata 模型

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/activity/template_loader.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/activity/*`

**Step 1: Write the failing test**

新增测试明确：
- activity template metadata 不再承担 manifest 语义；
- manifest 字段命名变化不会影响 template loader；
- 两套模型的命名可以独立演进。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/activity/... ./internal/workflow/... -count=1
```

Expected: 当前结构仍存在复用风险，测试失败。

**Step 3: Write minimal implementation**

引入独立 manifest 模型，并把 activity template metadata 收口回自身职责。

**Step 4: Run test to verify it passes**

Expected: 边界测试通过。

---

### Task 5: 切换 server manifest loader 与 docs

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/*`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/plans/2026-03-06-workflow-config-defaulting*.md`

**Step 1: Write the failing test**

新增测试明确：
- server 只消费新的 manifest 字段命名；
- docs 与 catalog metadata 说明只出现新命名；
- defaulting 文档基线同步切换到新的 contract / manifest 命名。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... ./internal/modules/catalog/... -count=1
```

Expected: loader 或文档基线尚未切换时失败。

**Step 3: Write minimal implementation**

调整 manifest loader 字段映射与 docs 文案，移除对旧字段名的依赖。

**Step 4: Run test to verify it passes**

Expected: server metadata 读取与相关 docs 测试通过。
