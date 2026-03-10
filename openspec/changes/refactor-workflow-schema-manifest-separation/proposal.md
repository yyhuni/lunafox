# Change: Workflow Schema / Manifest 分离

## Why
当前 workflow `*.schema.json` 同时承担了两类职责：

- 配置校验：JSON Schema 结构、类型、约束；
- 业务描述：workflow ID、展示名、stage 元数据、target types、默认 profile 信息等。

这种做法在项目内部可工作，但从长期扩展性看并不是终局最佳实践：

- 业务元数据被塞进 `x-*` 扩展字段后，schema 不再是“纯 JSON Schema 工件”；
- catalog、defaulting、docs、runtime 契约都被迫依赖 schema 扩展字段，职责耦合；
- 即使把 `x-workflow` 重命名成 `x-workflow-id`，本质上仍是“用校验工件承载业务描述”；
- 后续若继续扩展 UI、文档、默认值、执行提示或版本元数据，会持续把更多业务语义塞进 schema。

在不考虑兼容性的前提下，更标准、也更利于未来扩展的方案是：

- `schema` 只负责配置校验；
- `manifest` 负责 workflow 业务描述与可执行元数据；
- `profile` 继续作为独立 artifact 存在，由 manifest 通过 `defaultProfileId` 引用；
- server / worker / generator 全链路围绕这几个工件明确分层。

## What Changes
- 引入独立的 workflow manifest 工件，作为 workflow 业务描述的单一事实来源。
- workflow `*.schema.json` 回归纯 JSON Schema：只保留标准关键字与标准注解，不再承载 LunaFox 业务元数据扩展字段。
- 将以下信息从 schema 扩展字段迁移到 manifest：
  - `workflowId`
  - `displayName`
  - `description`
  - `supportedTargetTypeIds`
  - `defaultProfileId`
  - `stages` / `tools` / 参数元数据
- `defaultProfile` 不再以内嵌对象放入 manifest；manifest 只通过 `defaultProfileId` 引用独立生成的 profile artifact。
- server catalog 与 metadata loader 改为读取 manifest，而不是解析 schema `x-*` 字段。
- workflow 默认值补齐与最终执行配置归一化改为以 manifest / contract 为准；schema 中的 `default` 如保留，仅作为注解镜像，不再作为执行语义事实源。
- generator 从单一 workflow contract 产出三类工件：
  - `*.schema.json`
  - `*.manifest.json`
  - `profiles/<workflowId>.yaml`
- 明确 manifest 第一阶段校验策略：采用 Go 强类型解码、拒绝 unknown fields、并执行显式语义校验；后续如有需要再补独立 manifest schema。
- 明确 manifest 模型与 activity template metadata 模型分离：现有 activity/template metadata 结构不是 workflow manifest，不得继续混用。
- 明确本变更与现有提案的关系：
  - `refactor-workflow-contract-naming` 中关于 schema 扩展字段命名的部分，降级为中间方案，不再作为终局目标；
  - `refactor-workflow-id-semantics` 保留 ID-first 原则，但 workflow ID 的事实来源从 schema 扩展字段迁移为 manifest；
  - `add-workflow-config-defaulting` 必须改为基于 manifest / contract 读取默认值与参数元数据。

## Impact
- Affected specs:
  - `workflow-schema-manifest-separation` (ADDED)
- Affected code:
  - `/Users/yangyang/Desktop/lunafox/worker/cmd/workflow-contract-gen/main.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/contract_types.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/workflow/workflow_metadata.go`
  - `/Users/yangyang/Desktop/lunafox/worker/internal/activity/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/schema/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/manifest/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/workflow/profile/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/catalog/*`
  - `/Users/yangyang/Desktop/lunafox/server/internal/modules/scan/*`
  - `/Users/yangyang/Desktop/lunafox/docs/workflow/contract-generation.md`
  - generated schema / manifest / profile / docs 产物
- Behavioral impact:
  - workflow catalog 元数据不再从 schema 扩展字段读取。
  - workflow 配置 schema 将不再暴露 `x-workflow*` / `x-stage*` / `x-metadata`。
  - manifest 将通过 `defaultProfileId` 引用独立 profile artifact，而不是内嵌 profile 对象。
  - server / worker 的默认值归一化语义会从“schema 附带注解”切换到“manifest / contract 驱动”。
  - activity template metadata 与 workflow manifest 将成为两套独立模型。
