# Workflow Config Defaulting Design

## Objective

把 workflow 默认值从“看起来存在”变成“真正执行”，并确保：

- server 在 scan create 时产出 canonical workflow YAML；
- worker 在执行前对短配置的处理与 server 完全一致；
- 默认值语义来自 `workflow contract / manifest`，而不是 schema 扩展字段或 schema 解析副作用。

## Why The Old Framing Is Not Enough

旧 framing 更接近：

- schema 有 `default`
- server 做校验
- worker 做 typed decode

但在新的终局架构下，这样的 framing 不够严谨，因为：

1. schema 只应负责校验与标准注解；
2. 可执行默认值属于 workflow 业务语义，不应绑定到 schema metadata；
3. 如果 server / worker 分别解释 schema 或各自硬编码默认值，就一定会发生漂移。

因此，本变更必须建立在 `schema / manifest 分离` 之上。

## Source Of Truth

默认值相关事实源按优先级固定为：

1. Go workflow contract
2. generated workflow manifest
3. generated workflow schema 中的标准 `default` 注解（仅镜像，不是执行事实源）

也就是说：

- contract / manifest 定义“默认值是什么、什么时候应补齐”；
- schema 只定义“补齐后的配置是否合法”。

## Normalization Boundary

允许系统自动补齐的边界必须非常严格：

- 只补已存在且 `tool.enabled == true` 的 tool 参数；
- 不自动创建 stage / tool 对象；
- 不自动推断 `stage.enabled` / `tool.enabled`；
- 不因为存在默认值而跳过结构性校验；
- worker 的执行前防御性校验仍然保留。

## Server Pipeline

推荐 server 侧顺序：

1. 解析用户提交的 workflow YAML
2. 依据 contract / manifest 做 canonical config 归一化
3. 用 workflow schema 校验归一化后的结果
4. 将 canonical YAML 持久化到 scan
5. 用同一 canonical 根配置派生 task workflow config

这样做的好处是：

- 数据库存的是最终执行配置；
- 之后任何重试、排障、审计都面对同一 canonical 版本；
- server 不会把“原始用户输入”和“最终执行输入”混在一起。

## Worker Pipeline

worker 侧推荐顺序：

1. 接收 workflow config
2. 在 typed decode 前基于 contract / manifest 做同语义归一化
3. 再执行 strict typed decode 与业务校验
4. 继续实际执行

这样可以保证：

- worker 仍是最终执行边界；
- 但 worker 不会因为“本可以自动补齐的默认参数没显式写”而额外报错；
- 双端面对同一输入时语义一致。

## Generator Implication

generator 需要继续产出：

- `workflow.schema.json`
- `workflow.manifest.json`

其中：

- schema 里的 `default` 是标准注解，可保留；
- manifest / contract 提供 defaulting 所需的业务语义；
- defaulting 逻辑不能通过反向解析 schema 自定义字段来实现。

## Testing Strategy

最关键的不是“某一个字段对不对”，而是“双端结果是否一致”。

因此测试要分三层：

1. server canonical config 测试
2. worker typed decode 前归一化测试
3. server / worker 同输入一致性测试

并尽量使用结构比较，而不是纯文本 YAML 比较，避免排序与缩进造成噪音。

## Final Recommendation

`workflow-config-defaulting` 的最终落地方式应当是：

- **schema 负责校验**
- **contract / manifest 负责默认值语义**
- **server 持久化 canonical YAML**
- **worker 与 server 对短配置的归一化结果一致**

这才是与上位 `schema / manifest 分离` 架构兼容、且可长期扩展的实现方式。
