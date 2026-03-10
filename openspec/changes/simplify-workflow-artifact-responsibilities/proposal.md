# Change: 简化 workflow artifact 职责边界并收敛为单一事实源

## Why
当前 workflow 相关工件分为三类：
- `manifest`：目录/展示/编排说明
- `schema`：配置校验
- `profile`：默认/预设配置

这三类工件在概念上成立，但当前仓库里仍存在明显的职责重叠：
- `manifest` 同时携带大量参数约束与默认值，开始侵入 `schema/profile` 的职责；
- `schema` 和 `manifest` 可能同时描述参数类型/边界；
- `profile` 既承担默认值模板，又容易复制 schema/contract 已表达的事实；
- 当这些字段不是从单一来源生成时，会形成多处手工维护的漂移风险。

仓库当前已经具备 workflow contract generator 雏形，适合在继续扩大 workflow 数量前，明确工件边界并压缩重复维护面。

## What Changes
- 明确 workflow contract definition 是 workflow 结构、参数、默认值的唯一事实源。
- 保留 `manifest/schema/profile` 三类工件，但显著收窄各自职责：
  - `manifest` 仅保留目录/展示/编排元信息；
  - `schema` 仅保留配置校验规则；
  - `profile` 仅保留可运行默认值与场景差异值。
- `manifest` 不再承载详细参数约束和默认值，尤其是：
  - `defaultValue`
  - `minimum` / `maximum`
  - `minLength` / `maxLength`
  - `pattern`
  - `enum`
- `manifest` 中保留的参数信息降级为“目录/说明级”字段，仅在确有 UI/catalog 展示需求时保留：
  - `configKey`
  - `valueType`
  - `description`
  - `requiredWhenEnabled`（如仍被展示层需要）
- `default profile` 改为生成产物，而不是长期手工维护的完整 YAML。
- 非默认 profile（如 `fast`/`deep`/`safe`）改为 sparse overlay，只表达相对 default 的差异。
- server 侧 `manifest/schema/profile` 目录中的 canonical artifact 视为 generated output，不再鼓励手工编辑。
- 明确与现有变更关系：
  - 细化并收束 `refactor-workflow-schema-manifest-separation`
  - 细化并收束 `refactor-workflow-preset-generated-profiles`
  - 当前变更聚焦“字段归属与重复维护收敛”，不是重复定义 manifest/schema/profile 概念

## Impact
- Affected specs:
  - `workflow-artifact-responsibilities` (ADDED)
- Affected code:
  - `worker/internal/workflow/*/contract_definition.go`
  - `worker/cmd/workflow-contract-gen/*`
  - `worker/scripts/gen-workflow-contracts.sh`
  - `server/internal/workflow/manifest/*`
  - `server/internal/workflow/schema/*`
  - `server/internal/workflow/profile/*`
  - `server/internal/modules/catalog/*`
  - `server/internal/modules/scan/*`
- Behavioral impact:
  - server workflow catalog 继续读取 manifest，但 manifest 将更薄；
  - scan create 继续用 schema 校验，但 schema 成为唯一校验来源；
  - default profile 不再手工维护完整模板，而是从 contract 默认值生成；
  - scenario profiles 以 overlay 方式表达，不再复制默认值全集。
