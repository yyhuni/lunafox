# Workflow Final Implementation Priority Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 按最小返工路径完成 workflow 体系的终局重构：`schema / manifest / profile` 分层、contract/manifest 命名统一、ID-first 语义收敛，以及 canonical workflow YAML 默认值归一化。

**Architecture:** 先完成上位架构落地，再做命名统一，再做 ID-first 持久化与运行链路收敛，最后做 defaulting 与 canonical YAML 持久化。核心原则是：先定工件边界，再定命名，再切 identity/persistence，最后再切执行语义，避免同一批文件被反复返工。

**Tech Stack:** Go 1.26, JSON Schema Draft-07, YAML v3, OpenSpec, Go test

---

### Task 1: 锁定终局工件边界与分模规则

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/activity/template_loader.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_generation_consistency_test.go`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/manifests_test.go`
- Check: `/Users/yangyang/Desktop/lunafox/openspec/changes/refactor-workflow-schema-manifest-separation/design.md`

**Step 1: Write the failing test**

新增测试锁定以下行为：
- workflow manifest 是独立模型，不复用 activity template metadata；
- generated manifest 至少包含 `workflowId`、`displayName`、`configSchemaId`、`defaultProfileId`、`stages`；
- activity template metadata 仍然只服务 worker 模板系统。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... ./internal/activity/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1
```

Expected: 由于 manifest 尚未独立建模、loader 尚未存在，测试失败。

**Step 3: Write minimal implementation**

先引入独立 manifest 结构体与测试夹具，不立刻删除旧 schema metadata 逻辑。

**Step 4: Run test to verify it passes**

Run 同 Step 2。

Expected: 分模相关测试通过；其他链路测试仍可能失败。

---

### Task 2: 先落地 `schema / manifest / profile` 三类工件

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition.go`
- Add: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/subdomain_discovery.manifest.json`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/subdomain_discovery.schema.json`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/subdomain_discovery.manifest.json`

**Step 1: Write the failing test**

新增测试要求：
- generator 同时输出纯 schema、manifest、profile；
- manifest 用 `defaultProfileId` 引用默认 profile；
- schema 不再包含 `x-workflow*` / `x-stage*` / `x-metadata`。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/schema -count=1
```

Expected: 当前产物仍未达到终局结构，测试失败。

**Step 3: Write minimal implementation**

扩展 generator 同时产出 manifest 与 profile 引用，并移除 schema 中的业务扩展字段。

**Step 4: Run test to verify it passes**

Expected: 产物结构类测试通过。

---

### Task 3: 切换 server 到 strict manifest loader

**Files:**
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/manifests.go`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/manifests_test.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_query_store_adapter.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/loader.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/handler/workflow_handler_test.go`

**Step 1: Write the failing test**

新增测试要求 loader：
- strict decode
- reject unknown fields
- 校验 `workflowId` / `stageId` / `toolId` / `configKey`
- 校验 `defaultProfileId` 能解析到现有 profile
- catalog metadata 只从 manifest 获取

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... ./internal/modules/catalog/... -count=1
```

Expected: 当前 wiring 仍依赖 schema metadata，测试失败。

**Step 3: Write minimal implementation**

新增 manifest loader，并切 catalog / metadata 查询到 manifest。

**Step 4: Run test to verify it passes**

Expected: catalog / manifest 读取链路测试通过。

---

### Task 4: 在新架构上做 contract / manifest 命名统一

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry_test.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/manifests.go`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`

**Step 1: Write the failing test**

新增测试要求：
- contract 字段统一为 `WorkflowID` / `DisplayName` / `ConfigKey` 语义；
- manifest 字段统一为 `workflowId` / `displayName` / `supportedTargetTypeIds` / `defaultProfileId`；
- 旧字段如 `name`、`target_types` 不再被解析。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... ./cmd/workflow-contract-gen -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... -count=1
```

Expected: 当前 Go 层与产物层命名仍不完全统一，测试失败。

**Step 3: Write minimal implementation**

按新命名法则依次改 contract、registration、generator、manifest loader，不保留兼容层。

**Step 4: Run test to verify it passes**

Expected: 命名一致性测试通过。

---

### Task 5: 再做 ID-first 语义与持久化迁移

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/grpc/runtime/*`
- Add: `/Users/yangyang/Desktop/lunafox/server/internal/database/migrations/*workflow_id*.sql`
- Modify: `/Users/yangyang/Desktop/lunafox/tools/seed-api/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/handler/workflow_handler_test.go`

**Step 1: Write the failing test**

新增测试要求：
- catalog / scan / runtime / worker env 统一使用 `workflowId` / `workflowIds`；
- DB 持久化字段与迁移脚本切到 `workflow_id` / `workflow_ids`；
- 列表返回稳定按 `workflowId ASC` 排序。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/catalog/... ./internal/modules/scan/... ./internal/grpc/runtime/... -count=1
```

Expected: 因 API / persistence 仍有 `workflow_name` 语义残留而失败。

**Step 3: Write minimal implementation**

先完成 migration 与 repository / DTO 收敛，再切 handler / service / runtime 传递语义。

**Step 4: Run test to verify it passes**

Expected: ID-first 跨层测试通过。

---

### Task 6: 最后落地 defaulting 与 canonical workflow YAML

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/config_typed.go`
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_typed_consistency_test.go`

**Step 1: Write the failing test**

新增测试要求：
- server 基于 contract / manifest 归一化短配置；
- worker 在 typed decode 前执行同语义归一化；
- `scan.yaml_configuration` 持久化 canonical workflow YAML；
- schema `default` 只是注解镜像，不是执行事实源。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/... ./internal/workflow/... -count=1
cd /Users/yangyang/Desktop/lunafox/worker && go test ./internal/workflow/... -count=1
```

Expected: 当前 defaulting 语义尚未完全切换，测试失败。

**Step 3: Write minimal implementation**

抽出 shared normalization 语义入口，再分别接入 server create-scan 与 worker typed decode。

**Step 4: Run test to verify it passes**

Expected: 双端 canonical config 一致性测试通过。

---

### Task 7: 收尾重生成与全链路验证

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/generated/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/*`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`
- Modify: `/Users/yangyang/Desktop/lunafox/openspec/changes/*workflow*`

**Step 1: Write the failing test**

新增最终一致性测试，确保：
- generated schema / manifest / profile 彼此引用一致；
- server 内嵌产物与 worker 生成产物一致；
- `workflowId`、`defaultProfileId`、stage/tool/parameter 命名在各层完全一致。

**Step 2: Run full verification**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen ./internal/workflow/... ./internal/activity/... -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/... ./internal/modules/catalog/... ./internal/modules/scan/... ./internal/grpc/runtime/... -count=1
cd /Users/yangyang/Desktop/lunafox && openspec validate refactor-workflow-schema-manifest-separation --strict --no-interactive && openspec validate refactor-workflow-contract-naming --strict --no-interactive && openspec validate refactor-workflow-id-semantics --strict --no-interactive && openspec validate add-workflow-config-defaulting --strict --no-interactive
```

Expected: 全部通过。

**Step 3: Update task docs**

同步更新相关 `docs/plans` 与 OpenSpec `tasks.md`，确保实现状态可追踪。

**Step 4: Prepare execution handoff**

将本计划作为总控顺序，后续具体实施按 `executing-plans` 或 `subagent-driven-development` 分批推进。

---

## Priority Summary

1. **先做上位架构**：`schema / manifest / profile` 分层 + 分模
2. **再做命名统一**：contract / manifest / API 一次性收敛
3. **再做 ID-first**：identity source + DB / runtime / DTO 迁移
4. **最后做 defaulting**：因为它依赖前面三层稳定后才能不返工

## Why This Order

- 先做 defaulting 会被 manifest 结构与命名重构反复打回。
- 先做 ID migration 但不先切 manifest，会让 identity source 仍然不稳定。
- 先做命名但不先拆模型，会让 manifest 与 activity template metadata 再次耦合。
- 因此最佳顺序必须是：**边界 → 命名 → 身份/持久化 → 执行语义**。
