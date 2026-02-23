# 实施计划：数据库健康标准化接入

**分支**：`002-database-health-standardization`（实际 Git 分支：`codex/002-database-health-standardization`）  
**日期**：2026-02-23  
**规格**：`/Users/yangyang/Desktop/lunafox/specs/features/002-database-health-standardization/spec.md`

## 目标摘要

为数据库健康页面提供生产可用的后端接口与统一健康语义，核心目标如下：
- 新增受保护接口 `GET /api/system/database-health`。
- 健康状态由后端统一计算，前端不再本地推导顶层状态。
- 核心指标与可选指标分层，避免误报 `offline`。
- 标准化时间字段为 ISO 8601，并补齐前端错误态与陈旧态可见性。

## 技术上下文

- **语言/版本**：
  - Frontend: TypeScript 5, React 19, Next.js 15
  - Backend: Go 1.26, Gin, GORM
- **核心依赖**：
  - Frontend: TanStack Query, next-intl, axios
  - Backend: gorm.io/gorm, github.com/gin-gonic/gin
- **存储/依赖**：PostgreSQL（必需）、Redis（非本功能关键路径）
- **测试**：
  - Frontend: `pnpm test`, `pnpm typecheck`
  - Backend: `go test ./...`
- **目标平台**：Linux Server + Web 前端
- **项目类型**：前后端分离 Web 应用
- **性能目标**：
  - 页面首个有效快照 p95 <= 3 秒（与规格 SC-001 对齐）
  - 轮询周期 10 秒，后端单次采集在超时预算内完成
- **关键约束**：
  - 后端保持现有模块分层（router -> handler -> application -> repository）
  - 接口挂在受保护 `/api` 路由下
  - mock 与真实结构必须契约兼容

## 合规检查（Constitution Check）

由于项目 `constitution` 模板未定义具体条款，本次按现有仓库约束执行：
- 必须遵守后端分层和边界检查脚本约束。
- 必须保留前端 `service + hook + view` 分层。
- 必须提供可回归的测试路径（至少覆盖正常/降级/失败三类场景）。

当前评估：通过，无阻塞项。

## 项目结构

### 文档工件（本 feature）

```text
specs/features/002-database-health-standardization/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── database-health.openapi.yaml
└── tasks.md
```

### 代码落点（预期）

```text
server/
├── internal/bootstrap/routes.go
├── internal/bootstrap/wiring.go
├── internal/modules/asset/router/
│   └── system_database_health.go (new)
├── internal/modules/asset/handler/
│   └── database_health.go (new)
└── internal/modules/asset/application/
    ├── database_health.go (new)
    └── database_health_ports.go (new)

frontend/
├── services/database-health.service.ts
├── hooks/use-database-health.ts
├── types/database-health.types.ts
├── mock/data/database-health.ts
├── components/settings/database-health/database-health-view.tsx
└── messages/{zh,en}.json
```

**结构决策**：
- 后端先采用“在现有 asset 模块内新增 system/database-health 路由 + handler/application”的增量方案，避免一次性新建完整模块带来过高迁移成本。
- 路径语义保持 `/api/system/database-health`，对前端与运维保持稳定。

## Phase 0：研究与定稿

本阶段产出：
- `research.md`

需要收敛的问题：
- 状态机语义（`online/degraded/offline/maintenance`）精确定义
- 核心指标最小集及可选指标策略
- 时间字段与告警字段规范
- 失败降级与超时预算

## Phase 1：设计与契约

本阶段产出：
- `data-model.md`
- `contracts/database-health.openapi.yaml`
- `quickstart.md`

要点：
- 定义 API 响应结构（核心信号、可选信号、不可用信号、告警、时间）
- 明确前端展示/轮询/错误态行为
- 明确后端采集与状态判定边界

## 实施顺序（供 Tasks 阶段继承）

1. 后端接口与状态判定
2. 前端类型与渲染调整
3. mock 对齐与测试补齐
4. 回归验证与稳定性确认
