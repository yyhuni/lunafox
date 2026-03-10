## ADDED Requirements

### Requirement: 系统必须将 manifest `workflowId` 视为 workflow 唯一稳定标识
The system MUST treat `workflowId` in workflow manifest as the unique stable identity of a workflow and validate it during loading.

#### Scenario: manifest 缺失 workflowId
- **GIVEN** a workflow manifest is loaded without `workflowId`
- **WHEN** the system validates workflow metadata
- **THEN** the system MUST reject the manifest as invalid

#### Scenario: manifest 存在重复 workflowId
- **GIVEN** two workflow manifests declare the same `workflowId`
- **WHEN** the system validates workflow metadata during startup
- **THEN** the system MUST fail loading and report duplicate workflow identity

### Requirement: 系统必须将 workflow displayName 与 workflowId 语义分离
The system MUST keep `displayName` separate from `workflowId` so that display text is never used as machine identity.

#### Scenario: 返回 workflow 元数据列表
- **GIVEN** the catalog returns workflow metadata
- **WHEN** the response is serialized
- **THEN** each item MUST expose both `workflowId` and `displayName`
- **AND** `displayName` MUST NOT be used as the workflow identity key

### Requirement: Workflow 查询与传递链路必须采用 ID-first 语义
The system MUST use workflow ID semantics across query, persistence, and runtime boundaries.

#### Scenario: 创建扫描时引用 workflow
- **GIVEN** a client creates a scan for one or more workflows
- **WHEN** the request crosses handler, service, repository, and persistence boundaries
- **THEN** the workflow reference fields MUST use `workflowId` or `workflowIds`
- **AND** they MUST NOT use ambiguous `name` semantics

#### Scenario: runtime 下发 workflow 任务
- **GIVEN** the runtime assigns a workflow task to worker
- **WHEN** workflow identity is transmitted through proto or environment variables
- **THEN** the transmitted field MUST use workflow ID semantics
- **AND** worker-side contracts MUST interpret it as machine-stable workflow identity

### Requirement: Workflow metadata listing MUST remain stable by workflowId ordering
The system MUST return workflow metadata in stable `workflowId` ascending order.

#### Scenario: 枚举 workflow metadata
- **GIVEN** multiple workflow manifests are loaded successfully
- **WHEN** the system returns workflow metadata list
- **THEN** the list MUST be sorted by `workflowId` ascending
