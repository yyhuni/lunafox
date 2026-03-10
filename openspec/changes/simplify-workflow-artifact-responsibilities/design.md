## Context
当前仓库中，workflow artifact 已经初步走向生成链路：worker 侧 contract definition 可以生成 manifest/schema 等产物，server 侧分别消费 manifest、schema、profile。但字段边界仍然偏宽，尤其 manifest 中混入了大量 schema/profile 才应持有的事实。

如果继续保持这种“主题相近但边界模糊”的状态，后续增加 workflow、preset、catalog UI 时会遇到：
- 参数默认值和约束多处维护；
- 生成产物与手工文件并存，事实来源不稳定；
- profile 既像模板又像默认值镜像，难以区分 default 与 scenario preset。

## Goals / Non-Goals
- Goals:
  - 明确 `contract -> manifest/schema/default profile` 的单向生成关系；
  - 让 manifest 回归目录/展示/编排说明职责；
  - 让 schema 回归纯校验职责；
  - 让 profile 回归“可运行值配置”，并将 scenario profile 收敛为 overlay。
- Non-Goals:
  - 本次不引入 workflow 动态 CRUD；
  - 本次不改变 workflow 执行逻辑本身；
  - 本次不移除 manifest/schema/profile 三类工件中的任意一类。

## Decisions
1. Single source of truth
- workflow contract definition 是唯一事实源。
- 任何可从 contract 推导出的结构、默认值、约束，都不应在 server 侧工件里长期手工重复维护。

2. Manifest ownership
- `manifest` 保留：
  - workflow identity
  - display metadata
  - supported target types
  - default profile reference
  - stage/tool structure
  - notes / descriptive fields
- `manifest` 移除或降级以下字段：
  - `defaultValue`
  - `minimum` / `maximum`
  - `minLength` / `maxLength`
  - `pattern`
  - `enum`
- `valueType`、`configKey`、`description` 可以保留，但仅作为目录/展示投影。

3. Schema ownership
- `schema` 是唯一校验来源。
- 所有参数边界约束必须以 schema/contract 为准，manifest 不能再定义另一套约束真相。

4. Profile ownership
- `default profile` 从 contract 默认值生成。
- scenario profile 采用 sparse overlay：
  - 只记录与 default 不同的字段
  - 生成阶段合并成最终可加载 profile artifact
- profile 不定义参数结构，只定义值。

5. Generated artifact policy
- server 侧 `manifest/*.json`、`schema/*.json`、默认 `profiles/*.yaml` 视为 generated artifacts。
- 手工变更应回到 contract 或 overlay source，而不是直接编辑 generated file。

6. Migration strategy
- 先保持 runtime 消费侧接口不变：loader 仍读取 manifest/schema/profile 三类工件。
- 先改变“工件生成与字段归属”，再视需要收紧 loader 对 generated-only 的假设。

## Ownership Matrix
| Concern | Contract | Manifest | Schema | Profile |
|--------|----------|----------|--------|---------|
| workflowId / displayName / description | Source | Projected | No | Optional reference only |
| supportedTargetTypeIds | Source | Projected | No | No |
| stage/tool structure | Source | Projected | No | No |
| value type | Source | Optional display projection | Projected | No |
| min/max/pattern/enum | Source | No | Projected | No |
| default values | Source | No | Optional generated annotation | Default profile generated |
| scenario overrides | Source or overlay source | No | No | Overlay |

## Recommended Simplification Order
1. Thin manifest field set
2. Generated default profile
3. Sparse overlay profiles
4. CI drift check for generated artifacts

## Risks
- 若 manifest 字段瘦身过快，而 UI/catalog 仍依赖其中部分参数元信息，可能造成临时消费缺口。
- 若 profile overlay 设计过于激进，但当前 loader 只支持完整配置，需补一层生成/合并步骤。

## Validation Strategy
- 生成后的 manifest 不能丢失 catalog 所需字段。
- 生成后的 schema 必须继续通过 scan create 配置校验。
- default profile 生成后必须通过 schema 校验。
- scenario overlay 合并后的最终 profile 也必须通过 schema 校验。
