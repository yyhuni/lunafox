## Context
当前 workflow 生成链路把“配置校验”和“业务描述”混在同一个 `*.schema.json` 内：server 既用 schema 做配置校验，也从 schema 中解析 catalog 元数据；generator 既输出 JSON Schema，也顺带输出业务元信息；defaulting 讨论也围绕 schema `default` 和 `x-*` 展开。

这会带来三个长期问题：

1. `schema` 不是纯校验工件，导致其职责越来越膨胀；
2. 业务元数据和执行语义被绑定到 JSON Schema 扩展字段，不利于后续 catalog / docs / UI / runtime 扩展；
3. 任何命名优化都只能缓解“字段名不好”，但不能解决“工件职责混杂”。

## Goals / Non-Goals
- Goals:
  - 让 workflow schema 回归纯 JSON Schema 职责。
  - 为 workflow 引入独立、稳定、可扩展的 manifest 描述层。
  - 让 catalog、defaulting、docs、runtime 描述不再依赖 schema 扩展字段。
  - 保留 code-first 生成链路，但把产物拆分为不同职责的工件。
- Non-Goals:
  - 不在本次引入 workflow 动态 CRUD。
  - 不在本次引入远程 manifest 拉取或插件市场机制。
  - 不在本次改变 workflow 业务执行顺序本身。

## Approaches

### A. 继续使用 schema 扩展字段，只做 namespacing
- 做法：保留 schema 承载业务元数据，但把 `x-workflow-id`、`x-stage-id`、`x-metadata` 收口为 `x-lunafox` 等 namespaced 扩展对象。
- 优点：实现成本较低，比散落的 `x-*` 更整齐。
- 缺点：依然没有解决“schema 同时承担校验与业务描述”的根问题。

### B. Schema / Manifest 分离（推荐）
- 做法：schema 只做配置校验；manifest 承载 workflow 身份、展示、stage/tool 元数据、参数描述、默认值语义与 docs/canonical 配置所需信息；独立 profile artifact 继续保存默认 profile 配置内容，由 manifest 通过 `defaultProfileId` 引用。
- 优点：职责边界最清晰，最符合长期扩展方向，server / worker / docs 可以围绕 manifest 独立演进。
- 缺点：需要调整 generator、server loader 与现有 OpenSpec 依赖关系。

### C. 直接把 schema 废掉，只保留 manifest
- 做法：所有参数约束、UI 元数据、默认值与执行语义都放进 manifest，再由运行时手写校验。
- 优点：模型单一。
- 缺点：失去 JSON Schema 生态价值，也与当前已有的校验能力不一致。

## Decision
采用 **B. Schema / Manifest 分离**。

## Decisions

### 1. 引入独立 workflow manifest 工件
- 每个 workflow 生成一份独立 manifest，例如：`subdomain_discovery.manifest.json`。
- manifest 是 workflow 业务描述、catalog 元数据和执行语义的单一事实来源。
- schema 与 manifest 从同一个 Go workflow contract 生成，但职责不同。

### 2. schema 只保留标准 JSON Schema 职责
- schema 允许保留标准 JSON Schema 关键字与标准注解，例如：`$schema`、`$id`、`title`、`description`、`type`、`properties`、`required`、`default`、`minimum`、`enum`、`$defs`、`$ref`。
- schema 不再包含任何 workflow 业务扩展字段，例如：
  - `x-workflow`
  - `x-workflow-id`
  - `x-stage`
  - `x-stage-id`
  - `x-metadata`
- `title` / `description` 可保留为标准注解，但不再作为 catalog 的事实来源。

### 3. manifest 承载业务身份与元数据
- manifest 至少包含以下顶层语义：
  - `manifestVersion`
  - `workflowId`
  - `displayName`
  - `description`
  - `configSchemaId`
  - `supportedTargetTypeIds`
  - `defaultProfileId`
  - `stages`
- `stages` 继续描述 stage / tool / parameter 层级元数据，供 catalog、docs、defaulting 和 UI 使用。
- manifest JSON 字段统一使用 camelCase。

### 4. default profile 通过独立 artifact 引用
- 默认 profile 配置继续作为独立 profile artifact 存在，而不是内嵌进 manifest。
- manifest 使用 `defaultProfileId` 指向对应 profile。
- 这样可以避免 manifest 里重复承载完整配置文本，并复用现有 profile loader / catalog 能力。

### 5. executable defaults 不以 schema 为事实源
- workflow 参数默认值、是否在 enabled 时要求显式值、canonical config 归一化所需信息，都以 workflow contract / generated manifest 为事实源。
- schema `default` 可以保留为标准注解镜像，便于通用工具与文档消费，但运行时不得把 schema 解析结果当作唯一执行事实源。

### 6. server 模块边界同步调整
- `server/internal/workflow/schema` 只负责：
  - schema 发现
  - schema 校验
  - YAML / map 配置验证
- workflow metadata / catalog 读取迁移到独立 manifest loader（例如 `server/internal/workflow/manifest`）。
- `ListWorkflowMetadata` 的事实来源改为 manifest，而不是 schema。

### 7. manifest 第一阶段采用严格 Go 校验
- 第一阶段不再把 manifest 校验策略留作开放问题。
- loader 必须执行：
  - 强类型解码
  - unknown field reject
  - 必填字段校验
  - `workflowId` / `stageId` / `toolId` / `configKey` 的语义校验
  - 引用关系校验（例如 `defaultProfileId` 指向存在的 profile）
- 后续可以再增加独立 manifest schema，但它是增强项，不是第一阶段前置条件。

### 8. manifest 模型与 activity template metadata 模型必须分离
- 现有 `worker/internal/workflow/workflow_metadata.go` 及 activity template 相关元数据结构，不视为 workflow manifest。
- activity template metadata 继续服务 worker template / command builder 语义。
- workflow manifest 必须拥有独立模型与 loader，避免一套结构承担两种不同职责。

### 9. 现有变更的重定位规则
- `refactor-workflow-id-semantics`
  - 保留 “workflowId 是唯一稳定机器标识” 这一原则；
  - 但将 “workflowId 来源于 schema `x-workflow`” 改为 “workflowId 来源于 manifest `workflowId`”。
- `refactor-workflow-contract-naming`
  - 保留 contract / manifest 层的 ID / DisplayName / Key 命名法则；
  - 删除其关于 schema 扩展字段命名的终局性定义，因为 schema 扩展字段将被移除。
- `add-workflow-config-defaulting`
  - 保留“server / worker 默认值补齐一致、持久化 canonical workflow YAML”目标；
  - 但默认值与参数语义读取来源切换到 manifest / contract。

## Risks / Trade-offs
- 风险：generator 与 server loader 的改造范围会比纯命名重构更大。
  - 缓解：按“生成器产物 → loader 拆分 → catalog 切换 → defaulting 切换”顺序推进。
- 风险：已有提案文档会出现阶段性冲突。
  - 缓解：将本变更定义为上位架构变更，后续下游提案统一 rebase 到本变更。
- 风险：同时维护 schema、manifest、profile 三类工件。
  - 缓解：三者都从同一个 contract 生成，不允许人工双维护。

## Migration Plan
1. 在 Go workflow contract 层明确区分“配置校验信息”“workflow manifest 信息”“profile 信息”。
2. 扩展 generator，同时产出 `*.schema.json`、`*.manifest.json` 与 profile artifact。
3. 让 server 新增 manifest loader，并把 catalog / metadata 查询切换到 manifest。
4. 将 defaulting / canonical config 归一化逻辑改为读取 manifest / contract。
5. 清理 schema 中所有 LunaFox 业务扩展字段与相应测试。
6. 将 activity template metadata 与 workflow manifest 模型彻底拆分。
7. 更新 docs、生成产物与相关 OpenSpec 变更文档基线。
