## Context
当前 workflow 体系已经在 contract 中声明默认值，但这些默认值还没有成为真正的双端执行语义。server 侧主要依赖 schema 校验；worker 侧主要依赖强类型解码与执行前校验；scan create 持久化的还是用户原始 YAML。这使得：

- `default` 看起来像能力，实际上只是注解；
- 短配置没有统一的 canonical 归一化结果；
- server / worker 很容易在未来演进中出现默认值语义漂移。

在新的上位架构下，workflow schema 只负责配置校验；workflow manifest / contract 负责业务语义与默认值语义。因此，本变更必须显式改为基于 manifest / contract 实现。

## Goals / Non-Goals
- Goals:
  - 将 contract / manifest 声明的默认值升级为真实执行语义。
  - 在 scan create 时持久化 canonical workflow YAML。
  - 保证 worker 对短配置的处理与 server 一致。
  - 保留 worker 作为最终执行边界的防御性校验职责。
- Non-Goals:
  - 不回填历史 scan。
  - 不自动补 `enabled` 或缺失的 stage / tool 对象。
  - 不移除 worker 的执行前防御性校验。
  - 不把 schema `default` 重新定义为运行时唯一事实源。

## Decisions
1. 默认值事实源切换到 workflow contract / manifest。
- server 和 worker 的默认值补齐，都必须以 workflow contract / generated manifest 为事实源。
- workflow schema 中的标准 `default` 可以保留，但只作为注解镜像，不能作为唯一执行语义来源。

2. server 先规范化，再校验，再持久化。
- scan create 流程改为：解析用户输入 → 基于 contract / manifest 规范化 → 用 workflow schema 校验规范化结果 → 持久化 canonical workflow YAML。
- `scan.yaml_configuration` 与 task 侧派生出来的 workflow 配置，都来自同一个 canonical 根配置。

3. worker 在强类型解码前执行同语义规范化。
- worker 在 typed decode 前应用与 server 一致的规范化逻辑。
- worker 继续保留 unknown field 拒绝、结构存在性校验和业务校验，但不再因为“可由默认值补齐的参数未显式书写”而失败。

4. 默认值补齐边界严格受限。
- 只补已存在且 `tool.enabled == true` 的 tool 参数。
- 不补行为开关。
- 不自动创建缺失 stage / tool 对象。
- 不因为存在默认值而绕过必需结构校验。

5. generator 的职责是镜像，而不是定义执行语义。
- generator 继续从 contract 生成 schema 与 manifest。
- 若 schema 中输出标准 `default` 注解，它只是对 contract / manifest 语义的镜像表达。
- canonical config 所需的“是否在 enabled 时必须显式值”“参数默认值”等执行信息，事实源仍然是 contract / manifest。

## Risks / Trade-offs
- 文本级 YAML 断言测试可能需要调整为结构级比较。
- server / worker 若规范化实现不一致会导致执行漂移，因此必须补充共享 fixture 与一致性测试。
- 当 schema 与 manifest 同时存在默认值相关信息时，必须防止未来维护者误把 schema 当成主事实源。

## Migration Plan
1. 先在 contract / manifest 层明确默认值语义的唯一来源。
2. 为 server 补充 canonical config 归一化测试与持久化测试。
3. 为 worker 补充短配置 typed decode 归一化测试。
4. 保证两端对同一输入生成相同 canonical config。
5. 最后重生成 schema / manifest / profile / docs 产物，并更新下游文档。
