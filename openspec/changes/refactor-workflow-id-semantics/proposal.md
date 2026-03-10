# Change: Workflow ID-first 语义全链路收敛

## Why
当前 workflow 体系虽然已经朝 ID-first 方向推进，但“ID 的事实来源”和“展示名称的事实来源”仍然需要在新的上位架构下重新钉死。

在 `refactor-workflow-schema-manifest-separation` 确立后：

- workflow ID 不应再来源于 schema 扩展字段；
- catalog、scan、runtime、seed、worker env 语义都应围绕 manifest `workflowId` 收敛；
- `displayName` 必须与 `workflowId` 彻底分离，避免再次混用。

项目当前仍处于可一次性切换的阶段，因此最合理的做法是把 ID-first 语义完整收敛到 `workflowId` 上，不保留历史 `name` 语义兼容层。

## What Changes
- 将 workflow 唯一稳定机器标识的事实来源从 schema 扩展字段迁移为 manifest `workflowId`。
- 保持并强化展示字段分离原则：`displayName` 只承载展示语义，不参与机器标识主键约束。
- catalog DTO、scan DTO、runtime proto、worker env、repository / domain / service 方法命名统一切换为 `workflowId` / `workflowIds` 语义。
- server 在 manifest 加载阶段执行 `workflowId` 的缺失、格式、重复校验。
- 列表排序与查询语义统一以 `workflowId` 为准，保证稳定输出与可预测性。
- 明确数据库迁移纳入本变更：将持久化层中的 `workflow_name` / `workflow_names` 收敛为 `workflow_id` / `workflow_ids`，并同步调整 repository 与 DTO 映射。
- 不保留旧命名兼容。

## Impact
- Affected specs:
  - `workflow-id-semantics` (ADDED)
- Affected code:
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/grpc/runtime/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/database/migrations/*`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/*`
  - `/Users/yangyang/Desktop/lunafox/tools/seed-api/*`
  - related tests and fixtures
- Contract change matrix:
  - Manifest identity field: `name` / schema `x-workflow` -> `workflowId`
  - Manifest display field: `name` / schema `title` -> `displayName`
  - HTTP route param: `:name` -> `:workflowId`
  - Catalog DTO identity field: `name` -> `workflowId`
  - Scan DTO field: `workflowName(s)` -> `workflowId(s)`
  - Scan persistence column: `workflow_name(s)` -> `workflow_id(s)`
  - Workflow profile DTO field: `workflowNames` -> `workflowIds`
  - Internal method naming: `*ByWorkflowName` -> `*ByWorkflowID`
  - Runtime workflow field key: `workflow_name` -> `workflow_id`
  - Worker env contract: `WORKFLOW_NAME` -> `WORKFLOW_ID`
- Breaking changes:
  - **BREAKING (identity source)**: workflow 唯一标识来源从 schema metadata 切换到 manifest `workflowId`。
  - **BREAKING (API/contract semantics)**: workflow 查询与传递语义统一切换为 ID-first。
  - **BREAKING (persistence naming)**: scan 与相关持久化语义从 `workflow_name(s)` 切换为 `workflow_id(s)`。
  - **BREAKING (internal naming)**: 内部接口与方法由 name 语义重命名为 ID 语义。
