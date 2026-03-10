## ADDED Requirements

### Requirement: Workflow configuration schema MUST remain a pure validation artifact
The system MUST keep workflow configuration schemas limited to standard JSON Schema validation and annotation semantics.

#### Scenario: 生成 workflow 配置 schema
- **GIVEN** a workflow contract is used to generate artifacts
- **WHEN** the generator emits `*.schema.json`
- **THEN** the schema MUST contain only standard JSON Schema keywords and standard annotations
- **AND** it MUST NOT contain LunaFox workflow business metadata extension fields such as `x-workflow`, `x-workflow-id`, `x-stage`, `x-stage-id`, or `x-metadata`

### Requirement: Workflow business metadata MUST be stored in a separate manifest artifact
The system MUST store workflow identity, display metadata, stage/tool metadata, and execution-oriented descriptive metadata in a separate workflow manifest artifact.

#### Scenario: 生成 workflow manifest
- **GIVEN** a workflow contract is used to generate artifacts
- **WHEN** the generator emits workflow business metadata
- **THEN** it MUST emit a separate `*.manifest.json` artifact
- **AND** that manifest MUST include `workflowId`, `displayName`, `configSchemaId`, `defaultProfileId`, and `stages`

### Requirement: Default profile references MUST remain external to workflow manifest
The system MUST keep default workflow profile configuration as a separate artifact referenced by manifest instead of embedding full profile payload into manifest.

#### Scenario: manifest 表达默认 profile
- **GIVEN** a workflow has a generated default profile artifact
- **WHEN** the generator emits workflow manifest
- **THEN** the manifest MUST reference that profile using `defaultProfileId`
- **AND** it MUST NOT inline the full profile configuration payload into manifest

### Requirement: Server catalog metadata MUST be loaded from workflow manifest
The system MUST load workflow catalog metadata from workflow manifest artifacts rather than from workflow configuration schemas.

#### Scenario: server 加载 workflow metadata
- **GIVEN** workflow schema and workflow manifest artifacts are both available
- **WHEN** the server builds workflow catalog metadata
- **THEN** it MUST read identity and display metadata from manifest
- **AND** it MUST NOT depend on workflow schema extension fields for catalog metadata

### Requirement: Workflow manifest MUST be strictly validated during loading
The system MUST strictly validate workflow manifest structure and semantics during loading.

#### Scenario: manifest 包含未知字段或非法引用
- **GIVEN** a workflow manifest contains unknown fields, invalid semantic IDs, or a `defaultProfileId` that does not resolve
- **WHEN** the server loads workflow manifest artifacts
- **THEN** the loader MUST reject the manifest as invalid

### Requirement: Executable defaulting metadata MUST not depend on schema extension fields
The system MUST derive runtime defaulting and canonical workflow configuration normalization from workflow contract or workflow manifest semantics, not from workflow schema extension metadata.

#### Scenario: server / worker 处理短配置
- **GIVEN** a workflow tool parameter has a declared default value in the workflow contract
- **WHEN** server or worker normalizes a short workflow configuration
- **THEN** both sides MUST use contract or manifest semantics as the source of truth
- **AND** they MUST NOT require workflow schema extension fields to determine executable defaults
