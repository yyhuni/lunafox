## ADDED Requirements
### Requirement: Workflow 目录接口
系统 MUST 提供只读 Workflow 目录接口，且能力来源于内存 schema，而非数据库可写资源。

#### Scenario: 列出所有可用 Workflow
- **WHEN** 客户端请求 `GET /api/workflows`
- **THEN** 服务端返回当前 schema 中可识别的 workflow 列表
- **AND** 返回结果不包含数据库生成的可写资源标识

#### Scenario: 查询单个 Workflow
- **WHEN** 客户端请求 `GET /api/workflows/:name`
- **THEN** 当 workflow 存在时返回其元信息
- **AND** 当 workflow 不存在时返回 404

### Requirement: Workflow Preset 模板接口
系统 MUST 在 Workflow 命名空间下提供 preset 模板查询接口。

#### Scenario: 查询 Preset 列表
- **WHEN** 客户端请求 `GET /api/workflows/presets`
- **THEN** 返回 preset 列表
- **AND** 每个条目包含可直接用于扫描创建的配置模板字符串

#### Scenario: 查询单个 Preset
- **WHEN** 客户端请求 `GET /api/workflows/presets/:id`
- **THEN** 当 preset 存在时返回模板详情
- **AND** 当 preset 不存在时返回 404

### Requirement: 扫描创建合同使用 Workflow 命名
系统 MUST 使用 `workflowNames` 作为扫描创建合同字段，且配置校验基于 workflow 名称执行。

#### Scenario: 合法 Workflow 创建扫描
- **WHEN** 客户端提交 `workflowNames` 且配置中存在同名顶层 key
- **THEN** 扫描创建成功
- **AND** 服务端对每个 workflow 执行 schema 校验

#### Scenario: 非法 Workflow 名称
- **WHEN** 客户端提交不存在于 schema 目录中的 workflow 名称
- **THEN** 服务端返回 schema invalid 类错误

### Requirement: Workflow 管理接口完全下线
系统 MUST 不再注册任何旧目录路由。

#### Scenario: 访问旧目录接口
- **WHEN** 客户端请求任意旧目录路径
- **THEN** 路由不存在（404 或等价“未注册”行为）
