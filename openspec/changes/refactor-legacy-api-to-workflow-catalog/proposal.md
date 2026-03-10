# Change: 下线旧术语并标准化 Workflow 目录接口

## Why
当前系统已经完成“工作流能力内存化”，但外部语义仍混用旧术语与 `workflow`，会造成以下问题：
- API 名称与真实能力来源不一致（schema 内存目录 vs 可管理资源）。
- 使用方容易误以为仍支持旧目录 CRUD。
- 扫描入参与路由命名语义不统一，增加长期维护成本。

项目仍处于未发布阶段，适合一次性完成术语与接口收敛，不保留兼容层。

## What Changes
- 移除全部旧目录路由（含 presets 子路径），不提供兼容别名。
- 新增只读 Workflow 目录接口：
  - `GET /api/workflows`
  - `GET /api/workflows/:name`
- 将 preset 模板接口迁移到 Workflow 语义：
  - `GET /api/workflows/presets`
  - `GET /api/workflows/presets/:id`
- 扫描创建合同从旧字段统一改为 `workflowNames`，并同步更新 DTO、handler、application、测试与错误文案。
- 明确能力来源与职责边界：
  - “支持哪些工作流”来自 `workflowschema`（内存 schema 目录）
  - “默认模板配置”来自 `preset`（模板目录）
- 删除剩余的 workflow 管理遗留实现与文档引用（domain/application/repository/wiring/handler/dto）。

## Impact
- Affected specs:
  - `workflow-catalog-api`（新增）
- Affected code:
  - `server/internal/bootstrap/routes.go`
  - `server/internal/modules/catalog/router/*`
  - `server/internal/modules/catalog/handler/*`
  - `server/internal/modules/catalog/dto/*`
  - `server/internal/modules/scan/dto/*`
  - `server/internal/modules/scan/handler/*`
  - `server/internal/modules/scan/application/*`
  - `server/internal/workflowschema/*`
  - `server/internal/preset/*`
- Breaking changes:
  - **BREAKING**: 旧目录路由全量下线，客户端必须迁移到 `/api/workflows*`
  - **BREAKING**: 扫描创建请求字段从旧字段改为 `workflowNames`
