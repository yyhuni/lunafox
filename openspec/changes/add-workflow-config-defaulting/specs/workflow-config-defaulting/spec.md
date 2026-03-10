## ADDED Requirements

### Requirement: 扫描创建链路必须应用 contract 或 manifest 中声明的参数默认值
系统 MUST 在扫描创建阶段、schema 校验与持久化之前，应用 workflow contract 或 workflow manifest 中声明的参数默认值。

#### Scenario: 启用的工具省略了可补齐默认值的参数
- **GIVEN** 工作流配置中存在一个 `enabled` 为 `true` 的工具对象
- **AND** 该工具省略了一个在 workflow contract 或 workflow manifest 中声明了默认值的参数
- **WHEN** 创建扫描
- **THEN** 系统 MUST 先补齐声明的默认值
- **AND** MUST 使用补齐后的规范化配置继续后续校验与持久化

### Requirement: 缺失显式行为开关仍然必须被拒绝
系统 MUST 保持显式行为开关原则，不得因为存在默认值而自动推断 `stage.enabled` 或 `tool.enabled`。

#### Scenario: 缺失 enable 开关
- **GIVEN** 工作流配置缺少 `stage.enabled` 或 `tool.enabled`
- **WHEN** 创建扫描
- **THEN** 系统 MUST 拒绝该配置
- **AND** MUST NOT 因默认值存在而自动开启 stage 或 tool

### Requirement: 扫描创建链路必须持久化 canonical workflow YAML
系统 MUST 将 workflow 配置规范化为 canonical workflow YAML，并将其作为扫描创建链路的持久化结果。

#### Scenario: 持久化归一化后的 workflow YAML
- **GIVEN** 用户提交了一个省略了部分可默认参数的短 workflow 配置
- **WHEN** 扫描创建成功
- **THEN** 持久化到数据库的 workflow YAML MUST 是补齐默认值后的 canonical 表示
- **AND** task 侧使用的 workflow 配置 MUST 来自同一个 canonical 根配置

### Requirement: Worker 对短配置的归一化语义必须与 server 一致
系统 MUST 保证 worker 在强类型解码前应用与 server 相同的 workflow 配置归一化语义。

#### Scenario: Server 与 worker 处理同一短配置
- **GIVEN** server 与 worker 面对同一份省略默认参数的短 workflow 配置
- **WHEN** 两端分别执行归一化
- **THEN** 两端 MUST 得到语义一致的 canonical 配置结果
- **AND** worker MUST NOT 因可由默认值补齐的参数缺失而单独失败

### Requirement: 可执行默认值语义不得依赖 schema 扩展字段
系统 MUST 从 workflow contract 或 workflow manifest 推导可执行默认值语义，而不能依赖 workflow schema 扩展字段。

#### Scenario: Schema 不包含业务扩展字段
- **GIVEN** workflow schema 仅包含标准 JSON Schema 关键字与标准注解
- **WHEN** 系统执行 workflow 默认值补齐
- **THEN** 系统 MUST 仍可完成默认值归一化
- **AND** MUST NOT 依赖 `x-workflow*`、`x-stage*` 或 `x-metadata` 一类 schema 扩展字段
