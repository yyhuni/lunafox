# Frontend Boundary Naming Audit-First Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 先完成 LunaFox 前端边界命名的 A / B / C 审计，再据此执行精确的 A 类收敛，避免误伤 legacy、proto 与非边界代码。

**Architecture:** 审计阶段已经产出精确的 A 类实施清单、B 类排除清单和 CI 守门边界。后续实现必须只修改 A 类文件，并用精确测试文件与 CI allowlist 驱动，不再使用 `frontend/services/*` 这类通配范围。

**Tech Stack:** OpenSpec, TypeScript, Vitest, Bash, ripgrep, GitHub Actions

---

### Task 1: 完成审计阶段

**Files:**
- Read: `openspec/project.md`
- Read: `docs/plans/2026-03-06-frontend-boundary-naming-audit.md`
- Read: `openspec/changes/refactor-frontend-boundary-naming/proposal.md`
- Read: `openspec/changes/refactor-frontend-boundary-naming/design.md`

**Step 1: 核对 A / B / C 清单**

确认 A 类只包含当前 Go 后端主线边界文件，B 类已记录 worker 链路，C 类已排除注释文本与非边界代码。

**Step 2: 校验 OpenSpec**

Run: `openspec validate refactor-frontend-boundary-naming --strict --no-interactive`
Expected: PASS

### Task 2: 先补失败测试

**Files:**
- Modify: `frontend/lib/__tests__/response-parser.test.ts`
- Modify: `frontend/hooks/_shared/__tests__/pagination.test.ts`
- Create: `frontend/services/__tests__/scheduled-scan.service.contract.test.ts`
- Create: `frontend/hooks/__tests__/use-notification-sse.transform.test.ts`
- Create: `frontend/hooks/__tests__/use-organizations.query-contract.test.ts`

**Step 1: 删除 snake_case 正向预期**

把分页、通知、scheduled-scan query params、organization 分页兼容等测试改为只表达 `camelCase` 契约。

**Step 2: 运行定点测试并确认失败**

Run: `cd frontend && pnpm test -- --run lib/__tests__/response-parser.test.ts hooks/_shared/__tests__/pagination.test.ts services/__tests__/scheduled-scan.service.contract.test.ts hooks/__tests__/use-notification-sse.transform.test.ts hooks/__tests__/use-organizations.query-contract.test.ts`
Expected: FAIL

### Task 3: 收敛 A 类前端边界

**Files:**
- Modify: `frontend/lib/api-client.ts`
- Modify: `frontend/lib/response-parser.ts`
- Modify: `frontend/hooks/_shared/pagination.ts`
- Modify: `frontend/hooks/_shared/use-stable-pagination-info.ts`
- Modify: `frontend/hooks/use-notification-sse.ts`
- Modify: `frontend/hooks/use-targets/queries.ts`
- Modify: `frontend/hooks/use-endpoints.ts`
- Modify: `frontend/hooks/use-vulnerabilities/queries.ts`
- Modify: `frontend/hooks/use-organizations.ts`
- Modify: `frontend/hooks/_shared/targets-helpers.ts`
- Modify: `frontend/services/scheduled-scan.service.ts`
- Modify: `frontend/services/organization.service.ts`
- Modify: `frontend/services/endpoint.service.ts`
- Modify: `frontend/types/notification.types.ts`
- Modify: `frontend/types/command.types.ts`
- Modify: `frontend/types/endpoint.types.ts`
- Modify: `frontend/types/tool.types.ts`
- Modify: `frontend/types/subdomain.types.ts`
- Modify: `frontend/types/common.types.ts`
- Modify: `frontend/types/organization.types.ts`
- Modify: `frontend/types/target.types.ts`

**Step 1: 清理真实 snake_case 请求字段**

把 `frontend/services/scheduled-scan.service.ts` 的 query params 改为 `camelCase`。

**Step 2: 清理 A 类 DTO / 解析兼容字段**

删除经审计确认不再需要的 `page_size`、`total_pages`、`created_at`、`is_read`、`count` / `next` / `previous` 等旧兼容字段或 fallback。

**Step 3: 清理误导注释**

删除“interceptor / api-client 会自动转 snake_case”的错误注释，更新为当前真实契约说明。

**Step 4: 运行定点测试直到通过**

Run: `cd frontend && pnpm test -- --run lib/__tests__/response-parser.test.ts hooks/_shared/__tests__/pagination.test.ts services/__tests__/scheduled-scan.service.contract.test.ts hooks/__tests__/use-notification-sse.transform.test.ts hooks/__tests__/use-organizations.query-contract.test.ts`
Expected: PASS

### Task 4: 扩展命名守门脚本

**Files:**
- Modify: `scripts/ci/check-interface-naming.sh`
- Modify: `scripts/ci/check-interface-naming-test.sh`
- Modify: `.github/workflows/ci.yml`

**Step 1: 只对 A 类前端边界增加规则**

脚本只检查 `docs/plans/2026-03-06-frontend-boundary-naming-audit.md` 中的 A 类文件，不扫描整个 `frontend/`。

**Step 2: 显式排除 B 类与注释文本**

把 `frontend/services/worker.service.ts`、`frontend/hooks/use-workers.ts`、`frontend/types/worker.types.ts` 排除在规则之外；脚本只检查代码模式，不检查 route 示例注释。

**Step 3: 补脚本夹具**

增加前端 good / bad fixture，覆盖 scheduled-scan query params、DTO 字段和自动转换误导注释。

**Step 4: 运行脚本并确认通过**

Run: `bash scripts/ci/check-interface-naming-test.sh`
Run: `bash scripts/ci/check-interface-naming.sh`
Expected: PASS

### Task 5: 收尾验证

**Files:**
- Modify: `openspec/changes/refactor-frontend-boundary-naming/tasks.md`
- Modify: `openspec/project.md`

**Step 1: 更新项目规范总表**

把前端 / HTTP / proto / DB / logs / context 的命名边界写入项目规范。

**Step 2: 标记任务完成**

把已完成任务更新为 `- [x]`。

**Step 3: 运行最终校验**

Run: `cd frontend && pnpm test -- --run lib/__tests__/response-parser.test.ts hooks/_shared/__tests__/pagination.test.ts services/__tests__/scheduled-scan.service.contract.test.ts hooks/__tests__/use-notification-sse.transform.test.ts hooks/__tests__/use-organizations.query-contract.test.ts`
Run: `bash scripts/ci/check-interface-naming.sh`
Run: `openspec validate refactor-frontend-boundary-naming --strict --no-interactive`
Expected: 全部通过
