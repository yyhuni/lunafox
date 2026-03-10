# Change: 数据库健康页后端接入与健康语义标准化

## Why
当前数据库健康页前端已具备展示骨架，但核心问题仍未解决：
- 后端缺少 `/api/system/database-health/` 对应接口，页面在非 mock 模式不可用。
- 健康状态阈值在前端硬编码，`offline/degraded` 语义不统一，容易与运维实际认知偏离。
- 时间和告警字段以展示文案为主，跨时区与自动化分析能力不足。

为了让数据库健康页可用于生产运维决策，需要将健康判定逻辑收敛到后端，并约束最小核心指标集合与状态语义。

## What Changes
- 新增受保护 API：`GET /api/system/database-health/`，提供数据库健康快照。
- 将健康状态判定（online/degraded/offline/maintenance）迁移到后端统一计算。
- 定义核心指标与可选指标分层：核心指标必须稳定可得，可选指标不作为 `offline` 判定依据。
- 标准化时间字段为机器可解析时间（ISO 8601），前端负责本地化展示。
- 前端数据库健康页改为以服务端状态为准，补充错误态/陈旧数据态展示。
- 为后端接口与前端 hook 增加必要测试，覆盖正常、部分失败、不可用三个关键路径。

## Impact
- Affected specs: `database-health-monitoring` (new capability delta)
- Affected code:
  - `server/internal/bootstrap/routes.go`
  - `server/internal/bootstrap/wiring.go`
  - `server/internal/modules/*` (新增 system/database health 对应 router/handler/application/repository 组合)
  - `frontend/services/database-health.service.ts`
  - `frontend/hooks/use-database-health.ts`
  - `frontend/components/settings/database-health/database-health-view.tsx`
  - `frontend/types/database-health.types.ts`
