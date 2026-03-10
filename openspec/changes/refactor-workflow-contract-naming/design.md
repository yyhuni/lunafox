## Context
当前 workflow 契约体系已经具备 Go contract、generator、server loader 和 docs 产物，但命名还没有形成彻底一致的规则。问题不在于功能无法工作，而在于语义表达不够严格：同一个“名字”在不同层级可能表示 workflow ID、展示名称、配置键，或者仅仅是历史遗留字段。

在新的上位架构下，schema 已经不再承载 LunaFox 业务元数据，因此本变更需要从“schema 扩展字段命名统一”升级为“contract / manifest / API 命名统一”。同时，manifest 与 activity template metadata 也必须明确区分，否则命名统一会再次被混合职责破坏。

## Goals / Non-Goals
- Goals:
  - 建立 workflow contract 与 manifest 的统一命名法则。
  - 让 Go contract、generated manifest、server manifest loader 和 docs 产物完全同构。
  - 清除裸 `Name`、裸 `ID`、`Param` 缩写、`target_types` 一类旧话语体系。
  - 保证 manifest 与 activity template metadata 不复用同一套命名模型。
- Non-Goals:
  - 不改变 workflow 执行顺序。
  - 不保留旧命名兼容。
  - 不重新把 LunaFox 业务命名塞回 schema。

## Decisions
1. 采用单一命名法则。
- 机器标识使用 `...ID`
- 展示名称使用 `...DisplayName`
- 配置键使用 `...Key`
- 参数类型不再使用 `Param`，统一为 `Parameter`
- 在类型上下文已经明确时，不重复添加类型名前缀；优先使用“短但无歧义”的字段名。

2. 最终命名采用“显式语义 + 去前缀冗余”的版本。
- `WorkflowRegistration { WorkflowID, Factory, ConfigDecoder, WorkflowContract }`
- `WorkflowContract { WorkflowID, DisplayName, Description, SupportedTargetTypeIDs, DefaultProfile, Stages }`
- `WorkflowProfileContract { ProfileID, DisplayName, Description }`
- `WorkflowStageContract { StageID, DisplayName, Description, IsRequired, RunsInParallel, Notes, Tools }`
- `WorkflowToolContract { ToolID, Description, Parameters }`
- `WorkflowParameterContract { ConfigKey, ValueType, Description, RequiresValueWhenEnabled, DefaultValue, MinimumValue, MaximumValue, MinimumLength, MaximumLength, ValuePattern, AllowedValues }`

3. generated manifest 字段与 contract 保持同构。
- manifest JSON 字段使用 camelCase。
- 推荐命名集：
  - `manifestVersion`
  - `workflowId`
  - `displayName`
  - `description`
  - `configSchemaId`
  - `supportedTargetTypeIds`
  - `defaultProfileId`
  - `stages[].stageId`
  - `stages[].displayName`
  - `stages[].isRequired`
  - `stages[].runsInParallel`
  - `tools[].toolId`
  - `parameters[].configKey`
- `defaultProfileId` 来源于 `WorkflowContract.DefaultProfile.ProfileID`，但 manifest 不内嵌默认 profile 对象。

4. Go contract 是命名单一事实源。
- generated manifest、docs 和测试夹具都从 Go contract 命名映射而来。
- generator 只负责映射，不自行发明额外命名语义。

5. manifest 与 activity template metadata 使用不同模型。
- workflow manifest 有独立的结构、loader 与命名集合。
- activity template metadata 保持自己的语义边界，不参与 workflow manifest 命名收敛。
- 不允许继续复用 `worker/internal/workflow/workflow_metadata.go` 直接作为 manifest 模型。

6. schema 不再承载 LunaFox 业务命名。
- workflow schema 仅保留标准 JSON Schema 关键字与标准注解。
- `title` / `description` 是标准注解，不承担 workflow identity / catalog metadata 的单一事实源职责。

7. server 只消费新的 manifest 命名。
- server metadata loader 只接受新的 manifest 字段命名。
- 历史字段例如 `name`、`target_types`、`id`（当其语义本应为 `workflowId` / `stageId` / `toolId`）不再兼容。

8. 函数与方法命名同步切换。
- `GetContract` → `GetWorkflowContract`
- `ListContracts` → `ListWorkflowContracts`
- `validateContract` → `validateWorkflowContract`
- `GetContractDefinition` → `GetWorkflowContract`
- `loadDefinition` → `loadWorkflowContract`

9. 一次性切换。
- 不保留 alias
- 不保留 fallback
- 不做双读双写

## Risks / Trade-offs
- 改动范围大，但收益确定：语义统一且无歧义。
- 名称会更长，但可读性和长期维护性更高。
- 需要同步改写 generated manifest、docs 与测试夹具，否则容易出现“Go 命名已改、产物命名未改”的半迁移状态。

## Migration Plan
1. 先改 contract 类型与字段。
2. 再改 registration 层与 contract 相关函数 / 方法名。
3. 再改 generator 输出的 manifest 命名与 profile 引用字段。
4. 然后更新 server manifest loader、README 与 `docs/workflow/contract-generation.md`。
5. 拆分 workflow manifest 模型与 activity template metadata 模型。
6. 同步改写 `add-workflow-config-defaulting` 的命名基线文档。
7. 最后重生成 manifest / schema / profile / docs 产物并完成相关测试。
