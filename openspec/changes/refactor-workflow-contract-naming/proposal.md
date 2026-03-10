# Change: Workflow 契约命名全链路统一

## Why
当前 workflow 契约层虽然已经可用，但命名系统还没有做到真正统一：机器标识、展示名称、配置键在 contract、registration、manifest metadata、生成产物与程序化 API 中仍有混用现象。尤其是裸 `Name`、裸 `ID`、`Key`、`Param`、`target_types` 这类遗留写法，会持续制造认知噪音。

在 `refactor-workflow-schema-manifest-separation` 确立后，workflow 的终局命名面已经发生变化：

- schema 不再承载 LunaFox 业务元数据；
- 命名统一的主战场转为 contract、manifest、registration、docs 与程序化 API；
- manifest 通过 `defaultProfileId` 引用独立 profile artifact，而不是内嵌 profile 对象；
- activity template metadata 与 workflow manifest 必须使用不同模型；
- 因此本变更不再以“优化 schema 扩展字段命名”为终局，而是以“统一 contract / manifest 话语体系”为终局。

项目已经明确不需要兼容历史命名，因此现在最合适的做法是一次性把 workflow 契约命名系统统一到单一法则，不保留旧名或兼容桥接。

## What Changes
- 统一 workflow 契约层类型名与字段名：
  - 机器标识统一为 `...ID`
  - 展示名称统一为 `...DisplayName`
  - 配置键统一为 `...Key`
- 在类型上下文已经明确的前提下，避免字段名重复前缀，优先采用“语义显式但不赘述”的命名。
- 统一 registration 层命名，移除 `Registration.Name` 这类历史语义，并将注册入口显式收敛为 `WorkflowRegistration.WorkflowID` 与 `WorkflowRegistration.WorkflowContract`。
- 统一函数与方法命名，消除 `GetContract`、`ListContracts`、`validateContract`、`GetContractDefinition` 一类旧话语体系。
- 统一 generated manifest 的 JSON 字段命名，明确 workflow 业务描述一律采用 camelCase，并与 Go contract 语义同构。
- generated manifest 使用 `defaultProfileId` 引用默认 profile，不使用 `defaultProfile` 内嵌对象。
- 不再把 activity template metadata 结构视为 manifest 契约；必要时拆分为独立模型与命名空间。
- 明确 schema 的职责边界：schema 保留标准 JSON Schema 命名与注解，不再承载 LunaFox 业务字段，也不再是本变更的命名对象。
- 更新 generator、manifest loader、docs 和生成产物，确保全链路使用同一套命名。
- 不保留旧命名兼容。
- 本变更保留 `refactor-workflow-id-semantics` 的 ID-first 决策，并作为 `add-workflow-config-defaulting` 在 contract / manifest 维度的命名基线。

## Impact
- Affected specs:
  - `workflow-contract-naming` (ADDED)
- Affected code:
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/registry.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/activity/*`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/subdomain_discovery/contract_definition.go`
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/*`
  - `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`
  - `/Users/yangyang/Desktop/lunafox/openspec/changes/add-workflow-config-defaulting/*`
  - `/Users/yangyang/Desktop/lunafox/docs/plans/2026-03-06-workflow-config-defaulting.md`
  - `/Users/yangyang/Desktop/lunafox/docs/plans/2026-03-06-workflow-config-defaulting-design.md`
  - related tests and generated artifacts
- Behavioral impact:
  - contract 与 manifest 的字段名会整体收敛到统一法则。
  - manifest 会以 `defaultProfileId` 引用 profile artifact。
  - activity template metadata 与 manifest 会拆成独立模型。
  - 历史命名不再被解析或接受。
