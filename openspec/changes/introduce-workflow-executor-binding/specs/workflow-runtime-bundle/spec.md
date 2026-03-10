## ADDED Requirements

### Requirement: Workflow runtime bundle MUST include executor binding metadata
The system MUST include explicit executor binding metadata in workflow runtime bundle artifacts so that runtime does not rely on implicit `workflowId -> builtin implementation` assumptions.

#### Scenario: Builtin workflow manifest includes executor binding
- **GIVEN** a builtin workflow contract is defined
- **WHEN** workflow artifacts are generated
- **THEN** manifest includes `executor.type = "builtin"`
- **AND** manifest includes a non-empty builtin `executor.ref`

### Requirement: Worker MUST resolve builtin executor through binding
The worker runtime MUST resolve builtin workflow execution through explicit executor binding metadata.

#### Scenario: Builtin binding resolves registry implementation
- **GIVEN** a workflow manifest declares `executor.type = "builtin"` and `executor.ref = "subdomain_discovery"`
- **WHEN** worker prepares to execute the workflow
- **THEN** worker resolves the builtin executor by `ref`
- **AND** worker executes the resolved builtin workflow implementation

#### Scenario: Unsupported executor type is rejected explicitly
- **GIVEN** runtime bundle declares an executor type not supported by current worker runtime
- **WHEN** worker prepares execution
- **THEN** worker fails fast with explicit unsupported executor error
- **AND** runtime does not silently fall back to implicit builtin lookup
