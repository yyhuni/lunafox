# Worker B-Class Chain Audit

## Audit Scope
本次只审查原先在前端命名治理中被标记为 **B 类（先记录，不强改）** 的 worker 链路，核心目标是回答两个问题：
- 这条链路是否仍然对接当前 Go 后端主线？
- 是否应该从 B 类升级为 A 类并纳入命名治理？

审计对象：
- `frontend/services/worker.service.ts`
- `frontend/hooks/use-workers.ts`
- `frontend/types/worker.types.ts`

## Parallel Audit Tracks
本次按三条独立链路并行审计：

### Track A: Frontend worker HTTP chain
审计文件：
- `frontend/services/worker.service.ts`
- `frontend/hooks/use-workers.ts`
- `frontend/types/worker.types.ts`

结论：
- `worker.service.ts` 仍然硬编码请求 `/workers/`，并继续手写 `page_size`、`ip_address`、`ssh_port` 这类 snake_case 字段
- `use-workers.ts` 只依赖 `worker.service.ts`
- 这条 service/hook 链在当前前端运行链中**没有真实页面调用**，只剩自测文件引用

直接证据：
- `frontend/services/worker.service.ts:20` 使用 ```${BASE_URL}/?page=${page}&page_size=${pageSize}```
- `frontend/services/worker.service.ts:49` / `frontend/services/worker.service.ts:63` 继续构造 `ip_address`、`ssh_port`
- `frontend/hooks/use-workers.ts:46` 导出 `useWorkers`
- 全仓库对 `useWorkers` / `workerService` 的真实引用只剩：`frontend/hooks/use-workers.ts` 自身与 `frontend/hooks/__tests__/use-workers.mutation.test.ts`

### Track B: Current app entry and replacement chain
审计对象：
- `frontend/app/[locale]/settings/workers/page.tsx`
- `frontend/components/settings/workers/worker-list.tsx`
- `frontend/hooks/use-agents.ts`
- `frontend/services/agent.service.ts`
- `server/internal/modules/agent/router/routes.go`

结论：
- 用户可访问的 `/settings/workers/` 页面已经不走 worker HTTP 链，而是走 **agent** 主线
- 当前真实主线是：`WorkersPage -> AgentList -> useAgents -> agentService -> /api/admin/agents`
- 后端存在完整的 `admin/agents` API，但**不存在**对应的 `/workers` HTTP API

直接证据：
- `frontend/app/[locale]/settings/workers/page.tsx:6` 动态加载的是 `AgentList`
- `frontend/components/settings/workers/worker-list.tsx:23` 引入的是 `useAgents`
- `frontend/components/settings/workers/worker-list.tsx:61` 实际调用 `useAgents(page, pageSize)`
- `frontend/services/agent.service.ts:16` 定义 `BASE_URL = '/admin/agents'`
- `server/internal/modules/agent/router/routes.go:21` 注册 `/admin/agents`
- `server/internal/modules/agent/router/routes.go:23` ~ `server/internal/modules/agent/router/routes.go:28` 提供 list/detail/delete/config/log/token 等真实接口

### Track C: Orphan UI / deploy terminal chain
审计对象：
- `frontend/components/settings/workers/deploy-terminal-dialog-state.ts`
- `frontend/components/settings/workers/deploy-terminal-dialog.tsx`
- `frontend/components/settings/workers/deploy-terminal-dialog-sections.tsx`
- `frontend/types/worker.types.ts`

结论：
- `worker.types.ts` 仍被 deploy-terminal 相关组件引用，但这条链不是当前 app 的已接通 HTTP 主线
- deploy terminal 组件内部硬编码 WebSocket 地址 `/ws/workers/:id/deploy/`
- 当前仓库中**找不到任何服务端 `/ws/workers/:id/deploy/` 路由实现**
- `DeployTerminalDialog` 本身也未在真实页面链中被挂载，当前只存在于自身文件和 demo registry 索引里

直接证据：
- `frontend/components/settings/workers/deploy-terminal-dialog-state.ts:67` 连接 `/ws/workers/${worker.id}/deploy/`
- 全仓库搜索只在 `frontend/components/settings/workers/deploy-terminal-dialog-state.ts:67` 找到 `/ws/workers`
- `frontend/components/settings/workers/deploy-terminal-dialog.tsx:20` 定义组件
- 全仓库对 `DeployTerminalDialog` 的引用只见其自身文件与 `frontend/components/demo/business-demo-registry.generated.ts`

## Runtime Classification Result
结合三条并行审计，当前 worker B 类链路应拆成两部分看：

### B1. Legacy HTTP boundary chain
包含：
- `frontend/services/worker.service.ts`
- `frontend/hooks/use-workers.ts`

判定：
- **不是当前 Go 后端主线**
- **不应升级为 A 类**
- 更准确地说，它已经接近“孤儿/遗留链路”

原因：
- 没有真实页面调用
- 没有对应后端 `/workers` HTTP 路由证据
- 命名与主线标准显著分叉，且依赖明显过时的 snake_case 假设

### B2. Placeholder UI chain
包含：
- `frontend/types/worker.types.ts`
- `frontend/components/settings/workers/deploy-terminal-dialog*.tsx`

判定：
- **不是当前可执行后端边界主线**
- 也**不应直接升级为 A 类**
- 更像“未接通/未完成的占位 UI 设计残留”

原因：
- 依赖不存在的 `/ws/workers/:id/deploy/` 路由
- 未挂接到真实 page flow
- 只保留了局部类型和展示/终端交互壳层

## Answer to the Original B-Class Question
结论非常明确：
- **B 类 worker 链路当前不应该纳入前端边界命名治理 A 类范围**
- 继续保持“先记录，不强改”是安全选择
- 如果追求更准确的分类，`worker.service.ts` / `use-workers.ts` 已经可以从“B 类待确认”进一步下沉为“遗留孤儿链路”

## Recommended Next Steps
按标准做法，后续应二选一，而不是继续半兼容：

### Option 1: Remove legacy worker frontend chain
适用条件：
- 产品主线已经确认统一走 agent 模型
- `/settings/workers/` 页面确定不会再回到 `/workers` API

动作：
- 删除 `frontend/services/worker.service.ts`
- 删除 `frontend/hooks/use-workers.ts`
- 删除只服务于该链路的测试
- 将 deploy-terminal 相关 UI 单独评估：要么删除，要么迁移到 agent contract

### Option 2: Rebuild worker chain as a real supported capability
适用条件：
- 产品确实还需要独立 worker 概念，而不只是 agent

动作：
- 先补服务端真实 `/workers` HTTP / WS 规范与实现
- 再把前端 worker service/hook/types 全量迁移到当前标准契约
- 之后才有资格把它升级为 A 类并纳入 CI 命名治理

## Recommended Decision
基于当前仓库事实，我建议：
- **短期决策**：维持 B 类不动，不纳入本轮 A 类命名治理
- **更精确的后续治理**：发起一个单独 change，专门处理“worker 前端遗留链路清理或重建”
- **优先级建议**：优先做“删除/归档遗留 worker 链路”的方案，而不是继续在旧链路上修 snake_case
