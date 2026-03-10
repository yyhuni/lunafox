## ADDED Requirements

### Requirement: Workflow 契约命名必须显式区分机器标识、展示名称与配置键
The system MUST use explicit naming conventions across workflow contracts so that machine identifiers, display names, and config keys cannot be confused.

#### Scenario: 契约字段表达机器标识
- **GIVEN** a workflow contract defines an identifier field
- **WHEN** that field represents a machine-stable identity
- **THEN** the field name MUST end with `ID`

#### Scenario: 契约字段表达展示名称
- **GIVEN** a workflow contract defines a human-facing display field
- **WHEN** that field is intended for UI or docs display
- **THEN** the field name MUST contain `DisplayName`

#### Scenario: 契约字段表达配置键
- **GIVEN** a workflow contract defines a parameter key field
- **WHEN** that field maps to a config entry name
- **THEN** the field name MUST contain `Key`

### Requirement: Generated manifest 字段必须与 contract 命名保持同构
The system MUST generate workflow manifest fields that match workflow contract naming semantics.

#### Scenario: 生成 workflow manifest 元数据字段
- **GIVEN** a workflow contract has been defined with the new naming rules
- **WHEN** the generator emits workflow manifest metadata
- **THEN** it MUST emit `workflowId`
- **AND** it MUST emit `displayName`
- **AND** it MUST emit `defaultProfileId`
- **AND** stage metadata MUST use `stageId`, `displayName`, `isRequired`, and `runsInParallel`
- **AND** parameter metadata MUST use `configKey`

### Requirement: Server manifest 解析器必须只接受新命名字段
The system MUST parse only the new workflow manifest field names.

#### Scenario: manifest 使用旧字段命名
- **GIVEN** a workflow manifest uses legacy fields such as `name`, `target_types`, or `id` where `workflowId`, `displayName`, `stageId`, or `toolId` is required
- **WHEN** the server loads workflow metadata
- **THEN** the server MUST reject the manifest as invalid

### Requirement: Workflow manifest 与 activity template metadata 必须使用不同模型
The system MUST keep workflow manifest metadata and activity template metadata in separate models.

#### Scenario: worker 读取 activity template metadata
- **GIVEN** the worker loads activity template metadata for command templates
- **WHEN** workflow manifest naming evolves
- **THEN** activity template metadata MUST remain isolated from workflow manifest model changes
- **AND** the codebase MUST NOT reuse the same struct as both workflow manifest and activity template metadata

### Requirement: Workflow schema 不得再承载 LunaFox 业务命名约定
The system MUST keep LunaFox workflow business naming semantics out of workflow configuration schemas.

#### Scenario: 生成 workflow 配置 schema
- **GIVEN** the generator emits workflow configuration schema
- **WHEN** schema artifacts are produced
- **THEN** the schema MUST contain only standard JSON Schema keywords and standard annotations
- **AND** workflow identity and display naming semantics MUST be expressed in contract and manifest, not in schema-specific business extension fields

### Requirement: 契约相关程序化 API 必须遵守同一命名法则
The system MUST rename workflow contract programmatic APIs so they follow the same ID / DisplayName / Key naming semantics as the contract fields.

#### Scenario: 读取 workflow 契约 API
- **GIVEN** the codebase exposes programmatic helpers for reading workflow contracts
- **WHEN** those helpers are named
- **THEN** they MUST use `WorkflowContract` terminology rather than generic `Contract` terminology
