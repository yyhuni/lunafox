# Workflow Contract Naming Design

## Objective

建立 workflow 体系的统一命名语言，让以下几层完全同构：

- Go contract
- registration
- generated manifest
- server manifest loader
- docs / generated artifacts

并明确：workflow schema 不再承载 LunaFox 业务命名语义；activity template metadata 也不再冒充 workflow manifest 模型。

## Naming Law

本次重命名遵守以下绝对规则：

1. 机器标识必须显式命名为 `...ID`。
2. 展示名称必须显式命名为 `...DisplayName`。
3. 配置项键必须显式命名为 `...Key`。
4. 不允许裸 `Name` 作为通用字段存在。
5. 不允许裸 `ID` 用于非上下文唯一字段。
6. 契约层不再使用 `Param` 缩写，统一使用 `Parameter`。
7. generated manifest 与 Go contract 的语义必须一一对应。
8. 在类型上下文已经明确时，字段名不重复添加类型前缀。
9. workflow schema 只保留标准 JSON Schema 关键字与注解，不再表达 LunaFox 业务命名。
10. workflow manifest 与 activity template metadata 必须使用不同模型。

## Final Naming Set

### 类型名
- `ContractDefinition` → `WorkflowContract`
- `ContractProfileDefinition` → `WorkflowProfileContract`
- `ContractStageDefinition` → `WorkflowStageContract`
- `ContractToolDefinition` → `WorkflowToolContract`
- `ContractParamDefinition` → `WorkflowParameterContract`
- `Registration` → `WorkflowRegistration`
- `Factory` → `WorkflowFactory`
- `ConfigDecoder` → `WorkflowConfigDecoder`

### Registration 字段
- `WorkflowRegistration.Name` → `WorkflowID`
- `WorkflowRegistration.Contract` → `WorkflowContract`

### 顶层 contract 字段
- `WorkflowID` → 保留
- `DisplayName` → 保留
- `TargetTypes` → `SupportedTargetTypeIDs`
- `DefaultProfile` → 保留
- `Stages` → 保留

### Profile contract 字段
- `ID` → `ProfileID`
- `Name` → `DisplayName`

### Stage contract 字段
- `ID` → `StageID`
- `Name` → `DisplayName`
- `Required` → `IsRequired`
- `Parallel` → `RunsInParallel`
- `Tools` → 保留

### Tool contract 字段
- `ID` → `ToolID`
- `Params` → `Parameters`

### Parameter contract 字段
- `Key` → `ConfigKey`
- `Type` → `ValueType`
- `RequiredWhenEnabled` → `RequiresValueWhenEnabled`
- `Default` → `DefaultValue`
- `Minimum` → `MinimumValue`
- `Maximum` → `MaximumValue`
- `MinLength` → `MinimumLength`
- `MaxLength` → `MaximumLength`
- `Pattern` → `ValuePattern`
- `Enum` → `AllowedValues`

### 程序化 API
- `GetContract` → `GetWorkflowContract`
- `ListContracts` → `ListWorkflowContracts`
- `validateContract` → `validateWorkflowContract`
- `GetContractDefinition` → `GetWorkflowContract`
- `loadDefinition` → `loadWorkflowContract`

### Generated manifest 字段
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

其中：
- `defaultProfileId` 指向独立 profile artifact；
- manifest 不内嵌 profile 配置对象。

## Options Considered

### 方案 A：只统一 Go 字段名
- 优点：改动较小。
- 缺点：manifest、loader、docs 仍会保留历史命名杂音，做不到真正满分。

### 方案 B：字段名、类型名、manifest 字段全链路统一（推荐）
- 优点：机器标识、展示名称、配置键三类语义完全显式，Go 层与 manifest 层完全同构。
- 缺点：波及 registration、generator、server loader、docs 和测试，需要一次性切换。

### 方案 C：继续让 activity template metadata 兼任 manifest
- 优点：短期少改几个结构体。
- 缺点：职责混杂，后续命名会再次被两套语义拉扯。

## Decision
采用方案 B：一次性统一 workflow contract、registration、generated manifest、server loader 和生成产物命名，不保留兼容桥接，并要求 manifest 与 activity template metadata 分模。

## Why Schema Naming Is No Longer The End-State

即使把 schema 扩展字段重命名得再漂亮，例如：

- `x-workflow-id`
- `x-stage-id`
- `x-metadata.displayName`

它仍然是在把业务元数据塞进 schema 中。根据新的上位架构：

- schema 负责校验
- manifest 负责业务描述

因此，schema 扩展字段命名统一最多只是中间站，不是终局。真正应该做到 100 分的是 contract / manifest 命名系统。

## Detailed Decisions

### 1. 一次性切换，不做兼容
- 不保留旧字段。
- 不做 alias。
- 不做 fallback 解析。
- 不做双写或双读。

### 2. Go contract 是命名单一事实源
- 所有 generated manifest 与 docs 产物命名都从 Go contract 推导。
- generator 只负责映射，不自行发明额外命名语义。

### 3. Server 只消费新命名的 manifest 字段
- server manifest loader 只解析新字段：
  - `workflowId`
  - `displayName`
  - `supportedTargetTypeIds`
  - `defaultProfileId`
- 历史字段视为无效输入。

### 4. Registration 入口不再混用 ID 与 Name
- `WorkflowRegistration.WorkflowID` 成为注册入口的唯一机器标识字段。
- `WorkflowContract.WorkflowID` 必须与注册字段完全一致。

### 5. Manifest 与 activity template metadata 分模
- workflow manifest 独立建模。
- activity template metadata 保持独立用途。
- 不允许一个结构同时承担这两种职责。

### 6. docs 与 profile / manifest 产物同步更新
- docs 中的展示字段、stage 元数据字段与 generated manifest 命名保持一致。
- profile 文件名仍可保留 `<workflowID>.yaml`，但由 manifest 通过 `defaultProfileId` 引用。

## Risks / Trade-offs
- 风险：重命名范围广，测试和生成产物会出现级联变化。
  - 缓解：先写命名 contract 回归测试，再重生成全部产物。
- 风险：一次性切换后，任何遗漏都会导致 generator / server loader 链路断裂。
  - 缓解：按“contract → registration → generator → server loader → 产物 → 测试”顺序推进。
- 风险：命名更长。
  - 缓解：接受更长但无歧义的名称，优先保证语义清晰。

## Migration Plan
1. 先重命名 contract 类型与字段。
2. 再重命名 registration 与校验逻辑。
3. 然后修改 generator 输出字段。
4. 再修改 server manifest loader 与测试。
5. 再拆分 manifest / activity template metadata 模型。
6. 最后重生成 manifest / schema / profile / docs 并运行全套相关测试。
