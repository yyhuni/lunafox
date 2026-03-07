# Frontend Boundary Naming Audit

## Audit Result
本轮审计结论：
- **A 类（立即收敛）**：只包含已确认对接当前 Go 后端主线、且存在真实命名问题或误导性契约说明的前端边界文件
- **B 类（先记录，不强改）**：当前无法确认是否仍接当前 Go 后端主线的 worker 相关前端链路
- **C 类（不纳入）**：注释中的 route 示例、localStorage key、mock 值、测试计划 ID、`proto` / generated 等非 HTTP JSON 契约边界

## Evidence Baseline
确认当前后端主线使用 `camelCase` 的直接证据：
- `server/internal/dto/pagination.go`：分页 query / response 使用 `pageSize`、`totalPages`
- `server/internal/modules/identity/dto/organization_dto.go`：组织关联请求体使用 `targetIds`
- 当前仓库未发现 `frontend/lib/api-client.ts` 中真实启用的全局 snake/camel 自动转换实现
- 当前仓库未发现 Go 后端 `/workers` 主线路由实现

## A Class
### A1. 请求构造与边界基础设施
以下文件已确认属于当前 Go 后端 HTTP 边界，且需要立即收敛：
- `frontend/lib/api-client.ts`
  - 问题：文档注释仍宣称会自动把 `camelCase` 请求转换为 `snake_case`
  - 动作：删除错误说明，改为当前真实契约说明
- `frontend/services/scheduled-scan.service.ts`
  - 问题：真实请求仍手写 `target_id`、`organization_id`
  - 动作：改为 `targetId`、`organizationId`，删除“需要手转 snake_case”的说明
- `frontend/services/organization.service.ts`
  - 问题：`targetId` / `targetIds` 请求体本身已是对的，但注释仍声称会被 interceptor 转成 `target_id` / `target_ids`
  - 动作：保留请求体字段，删除错误注释
- `frontend/services/endpoint.service.ts`
  - 问题：存在“api-client.ts 会自动把 camelCase 转成 snake_case”的误导注释
  - 动作：删除错误注释；不把 route 示例注释纳入 CI

### A2. DTO 与兼容字段
以下 DTO 文件仍保留旧响应兼容字段或错误契约说明：
- `frontend/types/notification.types.ts`
  - 问题：保留 `created_at`、`read_at`、`is_read`、`page_size`、`total_pages`
  - 动作：仅保留 `createdAt`、`readAt`、`isRead`、`pageSize`、`totalPages`
- `frontend/types/command.types.ts`
  - 问题：保留 `page_size`、`total_count`、`total_pages`
  - 动作：只保留当前主线响应字段；是否保留 `total_count` 需在实现前再次核对真实调用点
- `frontend/types/endpoint.types.ts`
  - 问题：保留 `page_size`、`total_pages`，并带有错误的自动转换说明
  - 动作：收敛到 `pageSize`、`totalPages`，删除误导注释
- `frontend/types/tool.types.ts`
  - 问题：保留 `page_size`、`total_pages`，并带有错误的自动转换说明
  - 动作：收敛到 `pageSize`、`totalPages`，删除误导注释
- `frontend/types/subdomain.types.ts`
  - 问题：仅有“response interceptor 会自动转 camelCase”的误导注释
  - 动作：保留类型定义，删除错误说明
- `frontend/types/common.types.ts`
  - 问题：`sortBy` 注释仍举 `created_at`、`updated_at` 作为请求字段示例
  - 动作：改为当前真实字段或删除该误导示例
- `frontend/types/organization.types.ts`
  - 问题：仍保留 DRF 风格兼容字段 `count`、`next`、`previous`，并带有 `created_at` 历史注释
  - 动作：根据 A 类实现范围收敛到当前 Go 后端实际返回字段
- `frontend/types/target.types.ts`
  - 问题：仍保留 `count`、`next`、`previous` 兼容字段，并带有历史注释
  - 动作：根据当前使用点收敛到主线字段

### A3. 响应解析与分页兼容链
以下文件直接承接 API 响应，仍保留旧兼容 fallback：
- `frontend/lib/response-parser.ts`
  - 问题：`getPaginationMeta()` 仍兼容 `page_size` / `total_pages`
  - 动作：对 A 类响应只保留 `pageSize` / `totalPages`
- `frontend/hooks/_shared/pagination.ts`
  - 问题：`PaginationResponse` 与 `normalizePagination()` 仍兼容 `page_size` / `total_pages`
  - 动作：收敛到 `camelCase`
- `frontend/hooks/_shared/use-stable-pagination-info.ts`
  - 问题：仍兼容 `response.total_pages`
  - 动作：只保留 `totalPages`
- `frontend/hooks/use-notification-sse.ts`
  - 问题：仍从 `created_at` / `is_read` fallback 到 `createdAt` / `isRead`
  - 动作：对 A 类通知负载只保留 `camelCase`
- `frontend/hooks/use-targets/queries.ts`
  - 问题：局部响应类型仍含 `page_size` / `total_pages`
  - 动作：收敛为 `camelCase`
- `frontend/hooks/use-endpoints.ts`
  - 问题：局部响应类型仍含 `page_size` / `total_pages`
  - 动作：收敛为 `camelCase`
- `frontend/hooks/use-vulnerabilities/queries.ts`
  - 问题：局部响应类型仍含 `page_size` / `total_pages`
  - 动作：收敛为 `camelCase`
- `frontend/hooks/use-organizations.ts`
  - 问题：仍使用 `response.total || response.count` 兼容 DRF 风格返回
  - 动作：仅保留当前 Go 后端主线的 `total`
- `frontend/hooks/_shared/targets-helpers.ts`
  - 问题：`TargetSelectResponse` 仍显式生成兼容字段 `count`
  - 动作：需要在实现阶段确认是否仍保留为前端内部派生字段；若仅是前端内部派生，可保留但需与 API DTO 解耦

## B Class
以下文件当前不建议进入首轮标准化或 CI 强校验：
- `frontend/services/worker.service.ts`
  - 原因：当前未找到 Go 后端 `/workers` 主线路由证据
  - 风险：贸然收敛可能改坏 legacy / 占位链路
  - 退出条件：确认后端真实路由或确认前端该模块已废弃
- `frontend/hooks/use-workers.ts`
  - 原因：直接依赖 `worker.service.ts`
  - 退出条件：与 `worker.service.ts` 一致
- `frontend/types/worker.types.ts`
  - 原因：直接依赖 worker 链路契约
  - 退出条件：与 `worker.service.ts` 一致

## C Class
以下内容不纳入本轮命名治理，也不应被 CI 规则误伤：
- service 文件中的 route 示例注释，例如 `{target_id}`、`{scan_id}`
  - 代表位置：`frontend/services/directory.service.ts`、`frontend/services/website.service.ts`、`frontend/services/screenshot.service.ts`、`frontend/services/subdomain.service.ts`
  - 说明：这些是注释文本，不是当前可执行请求字段
- localStorage key、内部持久化 key、测试计划 ID、mock 业务值
- `proto` 文件与 generated contracts
- 非 API 边界的组件内部 state 与 UI 逻辑

## Exact Implementation Scope
首轮实施只修改以下 A 类文件：
- `frontend/lib/api-client.ts`
- `frontend/lib/response-parser.ts`
- `frontend/hooks/_shared/pagination.ts`
- `frontend/hooks/_shared/use-stable-pagination-info.ts`
- `frontend/hooks/use-notification-sse.ts`
- `frontend/hooks/use-targets/queries.ts`
- `frontend/hooks/use-endpoints.ts`
- `frontend/hooks/use-vulnerabilities/queries.ts`
- `frontend/hooks/use-organizations.ts`
- `frontend/hooks/_shared/targets-helpers.ts`
- `frontend/services/scheduled-scan.service.ts`
- `frontend/services/organization.service.ts`
- `frontend/services/endpoint.service.ts`
- `frontend/types/notification.types.ts`
- `frontend/types/command.types.ts`
- `frontend/types/endpoint.types.ts`
- `frontend/types/tool.types.ts`
- `frontend/types/subdomain.types.ts`
- `frontend/types/common.types.ts`
- `frontend/types/organization.types.ts`
- `frontend/types/target.types.ts`

## Exact Test Scope
### Existing tests to modify
- `frontend/lib/__tests__/response-parser.test.ts`
- `frontend/hooks/_shared/__tests__/pagination.test.ts`

### New tests recommended for A 类 implementation
- `frontend/services/__tests__/scheduled-scan.service.contract.test.ts`
  - 目标：锁定 query params 使用 `camelCase`
- `frontend/hooks/__tests__/use-notification-sse.transform.test.ts`
  - 目标：锁定通知负载只接受 `createdAt` / `isRead`
- `frontend/hooks/__tests__/use-organizations.query-contract.test.ts`
  - 目标：锁定组织分页只基于 `total` / `pageSize` / `totalPages`

## Guardrails for CI Expansion
- 仅检查 `Exact Implementation Scope` 中的 A 类文件
- 显式排除 B 类文件
- 不对注释、route 示例文本、mock 值、`proto`、generated 文件做自由文本命名检查
- 若某个 A 类文件既包含可执行边界逻辑又包含 route 示例注释，CI 只检查代码模式，不检查注释文本
