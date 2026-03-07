# Workflow Config Defaulting Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 workflow 参数默认值从“文档注解”升级为真实执行语义，并在 scan create 持久化 canonical workflow YAML，同时保证 worker 与 server 对短配置的处理完全一致。

**Architecture:** 以 `workflow contract + generated manifest` 作为默认值与参数执行语义的单一事实源。server 在创建扫描时先做 canonical config 归一化，再用 schema 校验归一化结果并持久化；worker 在 typed decode 前执行相同语义的归一化。workflow schema 中的标准 `default` 只作为注解镜像，不承担执行事实源职责。

**Tech Stack:** Go 1.26, JSON Schema Draft-07, YAML v3, OpenSpec

---

### Task 1: 锁定 canonical config 目标行为

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`

**Step 1: Write the failing test**

新增或调整测试，明确要求：
- server 在 scan create 时会先基于 contract / manifest 归一化 workflow 配置；
- 持久化结果是 canonical workflow YAML，而不是原始短配置；
- 缺失 `stage.enabled` / `tool.enabled` 仍然失败。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/... ./internal/workflow/... -count=1
```

Expected: 测试或实现因当前仍未形成 canonical config 归一化链路而失败。

**Step 3: Write minimal implementation**

优先新增一个显式归一化入口，不急于同时调整所有生成产物。

**Step 4: Run test to verify it still fails for the right reason**

Expected: 失败聚焦到生产代码尚未接入完整归一化流程。

---

### Task 2: 在 server 落地 canonical config 归一化

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`

**Step 1: Write the failing test**

新增测试要求：
- 已启用 tool 的默认参数在 server 侧被补齐；
- 归一化后的配置再进入 schema 校验；
- `yaml_configuration` 保存的是 canonical YAML。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/... ./internal/workflow/... -count=1
```

Expected: 因 scan create 仍未持久化 canonical YAML 而失败。

**Step 3: Write minimal implementation**

将归一化入口接入 scan create 主链路，并明确校验顺序为“先归一化，后校验，最后持久化”。

**Step 4: Run test to verify it passes**

Expected: server 侧 canonical config 相关测试通过。

---

### Task 3: 在 worker 落地同语义默认值补齐

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/config_typed.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`

**Step 1: Write the failing test**

新增测试明确：
- worker 在 typed decode 前补齐默认值；
- 可由默认值补齐的参数缺失时，worker 不单独失败；
- 缺失 enable 开关仍然失败。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
```

Expected: 当前 worker 仍要求显式参数，测试失败。

**Step 3: Write minimal implementation**

优先在 typed decode 入口前挂上归一化逻辑，避免散落在多个 decode 分支中。

**Step 4: Run test to verify it passes**

Expected: worker 侧短配置与 enable 约束测试通过。

---

### Task 4: 锁定 server / worker 一致性

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/*`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`

**Step 1: Write the failing test**

新增共享 fixture 或一致性测试，断言：
- server 与 worker 对同一短配置会生成相同 canonical 结果；
- schema 标准 `default` 不是双端执行语义的主来源。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
```

Expected: 任一侧尚未对齐时测试失败。

**Step 3: Write minimal implementation**

抽出共享默认值语义入口，避免 server / worker 分别维护两套事实源解释。

**Step 4: Run test to verify it passes**

Expected: 双端 canonical config 一致性测试通过。

---

### Task 5: 重生成产物并更新文档

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`

**Step 1: Write the failing test**

新增一致性测试，确保：
- generated schema 中的标准 `default` 只是 contract 语义镜像；
- generated manifest 与 canonical config defaulting 所需语义保持一致。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen ./internal/workflow/... -count=1
```

Expected: 生成产物或文档尚未对齐时失败。

**Step 3: Write minimal implementation**

重生成 schema / manifest / profile / docs，并更新文档中的默认值事实源说明。

**Step 4: Run test to verify it passes**

Expected: 相关一致性测试通过。
