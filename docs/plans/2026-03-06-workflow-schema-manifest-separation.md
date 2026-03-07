# Workflow Schema / Manifest Separation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 workflow 工件拆成职责清晰的三层：`schema` 只负责配置校验，`manifest` 负责业务描述、catalog 元数据与默认值归一化所需语义，`profile` 保持独立 artifact 并由 manifest 通过 `defaultProfileId` 引用。

**Architecture:** 保持 Go workflow contract 为单一生成源，由 generator 同时产出 `*.schema.json`、`*.manifest.json` 与 profile artifact。server 侧拆分 schema validation 与 manifest metadata loading；catalog、defaulting、canonical YAML 持久化统一改为基于 manifest / contract，而不再依赖 schema 扩展字段。activity template metadata 与 workflow manifest 使用不同模型，不再混用。

**Tech Stack:** Go 1.26, JSON Schema Draft-07, YAML v3, OpenSpec

---

### Task 1: 锁定终局工件边界

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/schemas.go`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/*`
- Check: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`

**Step 1: Write the failing test**

新增测试锁定以下终局行为：
- generated schema 不再包含 `x-workflow*` / `x-stage*` / `x-metadata`；
- generated manifest 独立产出，并包含 `workflowId`、`displayName`、`configSchemaId`、`defaultProfileId`、`stages`；
- generated profile artifact 继续独立存在；
- activity template metadata 与 workflow manifest 使用不同模型。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1 ./internal/modules/catalog/... -count=1
```

Expected: 测试因生成器与 server loader 仍依赖 schema metadata，且模型边界未拆开而失败。

**Step 3: Write minimal implementation**

先只新增 manifest 结构、profile 引用规则与测试夹具，不急于删除旧 schema metadata 读取逻辑。

**Step 4: Run test to verify it still fails for the right reason**

Expected: 失败聚焦到 loader / generator 尚未切换完成。

---

### Task 2: 引入 manifest 生成产物与 profile 引用

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition.go`
- Add: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/subdomain_discovery.manifest.json`

**Step 1: Write the failing test**

新增测试要求 generator 同时输出 schema、manifest 与 profile 引用，并断言 manifest 与 contract 的 `WorkflowID`、`DisplayName`、`defaultProfileId`、stage/tool/parameter 元数据一致。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen ./internal/workflow/... -count=1
```

Expected: 因 manifest 尚未输出或未正确引用 profile 而失败。

**Step 3: Write minimal implementation**

让 generator 在保留现有 schema 与 profile 输出的同时新增 manifest 输出，并把 manifest 结构控制在最小必要范围。

**Step 4: Run test to verify it passes**

Expected: manifest / profile 引用相关测试通过，schema 相关旧测试可能仍失败。

---

### Task 3: 拆分 server 侧 schema / manifest loader

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/schemas.go`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/manifests.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_query_store_adapter.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/handler/workflow_handler_test.go`

**Step 1: Write the failing test**

新增测试明确：
- schema loader 只做配置校验；
- manifest loader 负责 metadata；
- manifest loader 必须 strict decode、拒绝 unknown fields，并校验 `defaultProfileId` 引用；
- catalog 查询只从 manifest 取 `workflowId` / `displayName` / `description` / `target types`。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... ./internal/modules/catalog/... -count=1
```

Expected: 当前 wiring 仍依赖 schema metadata，测试失败。

**Step 3: Write minimal implementation**

新增 manifest loader，并把 catalog wiring 切换过去；保留 schema validation API 不动。

**Step 4: Run test to verify it passes**

Expected: catalog / manifest 相关测试通过。

---

### Task 4: 拆分 manifest 与 activity template metadata 模型

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/activity/template_loader.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/activity/*`

**Step 1: Write the failing test**

新增测试明确：
- workflow manifest 不再复用 activity template metadata 结构；
- activity template 校验仍然使用自己的元数据约束；
- 两套模型可以独立演进而不互相污染。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/activity/... ./internal/workflow/... -count=1
```

Expected: 当前结构仍有混用风险，测试失败。

**Step 3: Write minimal implementation**

优先引入独立 manifest 模型，再逐步把 activity template metadata 限定回自身职责。

**Step 4: Run test to verify it passes**

Expected: 两套模型的边界测试通过。

---

### Task 5: 把 defaulting 重定基到 manifest / contract

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/config_typed.go`

**Step 1: Write the failing test**

新增测试，断言 server / worker 在面对短配置时：
- 使用 manifest / contract 默认值补齐；
- 输出一致的 canonical workflow YAML；
- 不依赖 schema 扩展字段判断执行语义。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/... ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
```

Expected: 当前默认值来源仍未完全切换，测试失败。

**Step 3: Write minimal implementation**

优先抽出共享的 defaulting 语义入口，再分别接到 server 与 worker 解码链路。

**Step 4: Run test to verify it passes**

Expected: server / worker 对短配置的归一化结果一致。

---

### Task 6: 清理旧 schema metadata 并重生成产物

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/subdomain_discovery.schema.json`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/subdomain_discovery.schema.json`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/subdomain_discovery.manifest.json`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/README.md`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`

**Step 1: Write the failing test**

新增一致性测试，确保：
- schema 不再出现 LunaFox 业务扩展字段；
- manifest 与 schema 的 `$id` / `configSchemaId` 对得上；
- manifest 的 `defaultProfileId` 指向存在的 profile；
- 生成产物与 server 内嵌产物一致。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1
```

Expected: 因旧 schema 仍含 metadata 而失败。

**Step 3: Write minimal implementation**

删除 schema 内的 workflow metadata 扩展字段，新增 manifest 生成与嵌入，更新文档。

**Step 4: Run test to verify it passes**

Expected: 相关一致性测试通过。
