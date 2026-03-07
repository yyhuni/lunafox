# Workflow Config Object Canonical Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 workflow 配置从字符串 canonical 改为对象 canonical，去掉 profile 的 YAML-in-YAML，并让 catalog / scan / persistence / frontend 统一围绕对象模型工作。

**Architecture:** server、frontend 和 profile generator 统一以结构化对象作为 canonical workflow config。YAML 仅保留为 profile 文件语法、编辑器文本视图与必要协议边界投影；scan 与 scan_task 的事实源改为 JSONB object，runtime 若仍要求 YAML，则只在 outbound mapper 上临时序列化。

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL JSONB, YAML v3, Next.js 15, TypeScript, Vitest, OpenSpec

---

### Task 1: 锁定 profile 对象化契约

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/types.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/loader.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/validator.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/loader_test.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/validator_test.go`

**Step 1: Write the failing test**

新增测试明确：
- `Profile.Configuration` 是对象而不是字符串；
- profile 文件中的 `configuration` 必须是 YAML mapping；
- `ExtractWorkflowIDs` 直接从对象顶层键提取，而不是从字符串二次 parse。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/profile -count=1
```

Expected: 当前实现仍要求 `Configuration string`，测试失败。

**Step 3: Write minimal implementation**

先只改 `Profile` 模型、loader 和 validator，让 profile 加载链能够稳定返回对象配置，不要同时修改 catalog / scan。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/profile -count=1
```

Expected: profile 对象化测试通过。

---

### Task 2: 锁定 catalog profile API 为对象契约

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/domain/workflow_catalog.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/dto/profile_dto.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/bootstrap/wiring/catalog/wiring_catalog_workflow_profile_query_store_adapter.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/handler/workflow_profile_handler_test.go`

**Step 1: Write the failing test**

新增或更新测试明确：
- `/api/workflows/profiles` 返回 `configuration` JSON object；
- `workflowIds` 由结构化配置提取；
- adapter 不再依赖字符串式 `ExtractWorkflowIDs`。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/catalog/... -count=1
```

Expected: handler / DTO 仍按字符串契约返回，测试失败。

**Step 3: Write minimal implementation**

把 catalog domain、adapter 和 DTO 改成对象配置返回，保持接口最小变化，只动 `configuration` 字段语义。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/catalog/... -count=1
```

Expected: catalog profile 契约测试通过。

---

### Task 3: 锁定 scan create 与 task slice 的对象 canonical

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/dto/scan_dto.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_inputs.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_yaml.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_plan_test.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/scan_create_execute_test.go`

**Step 1: Write the failing test**

新增测试明确：
- scan create 输入的 `configuration` 是对象；
- `buildScanTasks` 从对象根切 workflow slice；
- task canonical slice 在 server 内仍保持对象，而不是一创建就先 marshal 成 YAML。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/application/... -count=1
```

Expected: 现有 scan create 仍按 YAML 字符串链路运行，测试失败。

**Step 3: Write minimal implementation**

先把 scan create 输入、解析和 task slice 派生都改成对象语义；保留单独的 YAML 序列化 helper 仅供必要边界使用。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/application/... -count=1
```

Expected: scan create 对象化测试通过。

---

### Task 4: 把持久化事实源切到 JSONB

**Files:**
- Create: `/Users/yangyang/Desktop/lunafox/server/cmd/server/migrations/000002_workflow_config_object_canonical.up.sql`
- Create: `/Users/yangyang/Desktop/lunafox/server/cmd/server/migrations/000002_workflow_config_object_canonical.down.sql`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/persistence/scan.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/persistence/task.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/scan_mapper.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/persistence/scan_model_contract_test.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/persistence/task_model_contract_test.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/repository/persistence/task_json_contract_test.go`

**Step 1: Write the failing test**

新增测试明确：
- `scan` 使用 `configuration JSONB` 作为事实源；
- `scan_task` 使用 `workflow_config JSONB` 作为事实源；
- 旧文本列不再承载 authoritative 配置。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/repository/... -count=1
```

Expected: 当前 persistence model 仍映射文本列，测试失败。

**Step 3: Write minimal implementation**

先新增 JSONB 列与模型字段，再更新 mapper；不要在这一步顺手改 runtime DTO。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/repository/... -count=1
```

Expected: JSONB 持久化契约测试通过。

---

### Task 5: 把 runtime YAML 收敛成 outbound projection

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/domain/create_projection.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/domain/runtime_projection.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/task_runtime_outputs.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/task_runtime_service.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/grpc/runtime/service/runtime_mappers.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/application/task_runtime_service_test.go`
- Test: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/dto/task_dto_test.go`

**Step 1: Write the failing test**

新增测试明确：
- task runtime 输出优先来自 task 对象配置；
- `workflowConfigYAML` 只在 outbound mapper 上派生；
- 缺失 canonical object 时失败，而不是依赖持久化的旧 YAML 文本。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/application/... ./internal/grpc/runtime/... -count=1
```

Expected: 当前 runtime 仍依赖 `WorkflowConfigYAML` 文本字段，测试失败。

**Step 3: Write minimal implementation**

保留对外 `workflowConfigYAML` 字段，内部改成从 canonical object 即时序列化，不再把文本当事实源。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/modules/scan/application/... ./internal/grpc/runtime/... -count=1
```

Expected: runtime outbound projection 测试通过。

---

### Task 6: 前端切到对象主状态，YAML 仅作视图

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/types/workflow.types.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/types/scan.types.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/services/workflow.service.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/services/scan.service.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/lib/workflow-config.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/components/scan/workflow-profile-selector.tsx`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/components/scan/scan-config-editor-state.ts`
- Modify: `/Users/yangyang/Desktop/lunafox/frontend/components/scan/initiate-scan-dialog-state.ts`
- Test: `/Users/yangyang/Desktop/lunafox/frontend/hooks/__tests__/use-workflows.query-contract.test.ts`
- Create: `/Users/yangyang/Desktop/lunafox/frontend/lib/__tests__/workflow-config.test.ts`

**Step 1: Write the failing test**

新增测试明确：
- workflow profile hook 收到 `configuration` object；
- capability 提取、preset merge 都围绕对象工作；
- YAML 编辑器只接受对象序列化后的文本视图并回写对象。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/frontend && pnpm vitest run hooks/__tests__/use-workflows.query-contract.test.ts lib/__tests__/workflow-config.test.ts
```

Expected: 当前前端仍依赖字符串配置，测试失败。

**Step 3: Write minimal implementation**

先改类型、service 和 helper，再改 dialog / selector state；避免直接在组件里散落 YAML parse 逻辑。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/frontend && pnpm vitest run hooks/__tests__/use-workflows.query-contract.test.ts lib/__tests__/workflow-config.test.ts
```

Expected: 前端对象 canonical 测试通过。

---

### Task 7: 更新生成器、fixtures 与文档

**Files:**
- Modify: `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
- Modify: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/profiles/subdomain_discovery.yaml`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`
- Modify: `/Users/yangyang/Desktop/lunafox/docs/config-reference/subdomain_discovery.md`
- Modify: `/Users/yangyang/Desktop/lunafox/openspec/changes/refactor-workflow-config-object-canonical/*`

**Step 1: Write the failing test**

新增测试明确：
- 生成器输出的 profile `configuration` 是 mapping；
- fixture 与文档不再展示 YAML block-scalar 内嵌配置；
- loader 仍可加载生成产物并通过 schema 校验。

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/profile -count=1
```

Expected: 当前生成器仍写出字符串式 `configuration`，测试失败。

**Step 3: Write minimal implementation**

更新 profile generator、重生成 profile fixture，并同步回写文档示例。

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen -count=1
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/profile -count=1
```

Expected: 生成器与 profile 工件测试通过。

---

### Task 8: 整体验证

**Files:**
- Check: `/Users/yangyang/Desktop/lunafox/openspec/changes/refactor-workflow-config-object-canonical/`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/`
- Check: `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/`
- Check: `/Users/yangyang/Desktop/lunafox/frontend/`

**Step 1: Run OpenSpec validation**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox && openspec validate refactor-workflow-config-object-canonical --strict --no-interactive
```

Expected: change 工件格式校验通过。

**Step 2: Run targeted Go tests**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/server && go test ./internal/workflow/profile ./internal/modules/catalog/... ./internal/modules/scan/... ./internal/grpc/runtime/... -count=1
cd /Users/yangyang/Desktop/lunafox/worker && go test ./cmd/workflow-contract-gen -count=1
```

Expected: 相关后端与生成器测试通过。

**Step 3: Run targeted frontend tests**

Run:

```bash
cd /Users/yangyang/Desktop/lunafox/frontend && pnpm vitest run hooks/__tests__/use-workflows.query-contract.test.ts lib/__tests__/workflow-config.test.ts
```

Expected: 前端对象 canonical 相关测试通过。

**Step 4: Final review**

确认以下几点：
- 没有任何业务主链路再以 YAML 文本作为 canonical 配置状态；
- `workflowConfigYAML` 若仍存在，仅作为 outbound projection；
- profile 文件与文档示例已经全部切换到 mapping 形态。

Plan complete and saved to `docs/plans/2026-03-07-workflow-config-object-canonical.md`. Two execution options:

**1. Subagent-Driven (this session)** - 我在当前会话按任务逐步执行与回看。

**2. Parallel Session (separate)** - 你开新会话，按 `executing-plans` 技能批量执行。
