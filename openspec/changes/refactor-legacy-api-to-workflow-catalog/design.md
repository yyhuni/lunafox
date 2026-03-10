## Context
系统已完成运行时与扫描侧的能力内存化，能力来源是 `workflowschema` 与 `preset`，不是 DB 可写资源。  
但 API 与入参仍存在旧命名残留，导致语义不一致与认知负担。

本变更目标是在未发布阶段完成一次性语义收敛：旧术语统一为 `workflow`，并移除全部兼容层。

## Goals / Non-Goals
- Goals:
  - 统一外部 API 与内部领域术语为 `workflow`。
  - 建立“能力目录（schema）+ 模板目录（preset）”双源只读模型。
  - 清理所有旧目录路由与残留实现。
  - 扫描创建合同改为 `workflowNames`，并保持现有 schema 校验能力。
- Non-Goals:
  - 不引入 workflow 可写管理（无 CRUD）。
  - 不在本次引入动态远程插件注册。
  - 不做双路径兼容（不保留旧目录路径）。

## Decisions
- Decision: `workflow` 为唯一对外术语。
  - 路由、DTO、错误文案、日志、文档全部收敛到 `workflow`。

- Decision: Workflow 目录为只读能力接口。
  - `GET /api/workflows`：列出当前支持工作流。
  - `GET /api/workflows/:name`：返回单个工作流元信息（名称、可用版本等）。
  - 数据源来自 `workflowschema`，不落库。

- Decision: Preset 目录归属 Workflow 命名空间。
  - `GET /api/workflows/presets`
  - `GET /api/workflows/presets/:id`
  - 数据源来自 `preset.Loader`。

- Decision: 扫描创建合同改为 `workflowNames`。
  - 保持“配置必须包含对应顶层 workflow key”的规则。
  - 保持 schema 版本校验逻辑。

- Decision: 不提供兼容别名。
  - `/api/engines*` 直接移除，不做 301/302/透传映射。

## Risks / Trade-offs
- 风险: 前端与脚本调用如果仍使用旧目录路径将立即失败。  
  - Mitigation: 同步更新前端服务层与集成测试；发布前执行 API smoke 检查。

- 风险: 扫描创建请求字段变更可能影响外部自动化脚本。  
  - Mitigation: 在变更说明中给出迁移示例；在未发布阶段一次性切换。

## Migration Plan
1. 新增 workflow 目录与 workflow preset 路由及 handler。
2. 扫描创建 DTO/handler/application 从旧字段改为 `workflowNames`。
3. 删除旧目录路由注册与残留实现。
4. 更新回归测试与 OpenAPI/文档引用。
5. 执行后端全量测试与路由快照验证。

## Open Questions
- `GET /api/workflows/:name` 的返回结构是否需要包含 schema 版本矩阵（`apiVersion/schemaVersion` 列表）作为首版能力。
