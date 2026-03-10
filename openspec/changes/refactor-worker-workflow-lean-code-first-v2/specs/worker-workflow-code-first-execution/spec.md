## ADDED Requirements

### Requirement: Worker configuration decoding MUST be strict-typed without runtime schema compilation
The worker MUST decode workflow configuration via strict typed decoding and MUST NOT depend on per-workflow runtime schema compilation for execution-time validation.

#### Scenario: Worker decodes valid typed config
- **GIVEN** a workflow config that already passed server schema gate
- **WHEN** worker decodes config into typed struct
- **THEN** decode succeeds without compiling schema at runtime
- **AND** worker continues with business validation and execution

#### Scenario: Worker schema runtime path is removed from execution critical path
- **GIVEN** workflow execution starts in worker
- **WHEN** configuration is decoded and validated
- **THEN** worker does not compile embedded JSON schema during execution
- **AND** validation behavior remains fail-fast for invalid semantic config

### Requirement: Worker strict decode MUST reject unknown fields
The worker MUST reject unknown configuration fields during typed decoding.

#### Scenario: Unknown key is provided in workflow config
- **GIVEN** a config contains an unrecognized key under workflow scope
- **WHEN** worker performs strict typed decoding
- **THEN** worker rejects the config before stage execution
- **AND** no command is launched

#### Scenario: Required field absence is rejected during worker decode
- **GIVEN** a config misses a required workflow field that server schema gate could normally reject
- **WHEN** worker performs strict typed decode and required-presence checks
- **THEN** worker rejects the config before execution
- **AND** failure is not silently converted to zero-value defaults

### Requirement: Validation boundary MUST remain two-layer and non-overlapping
The system MUST keep server schema gate as structural guard and worker validation as business guard, while avoiding duplicated runtime schema validation in worker.

#### Scenario: Structural error rejected at server
- **GIVEN** config has type mismatch or missing required field
- **WHEN** server schema gate validates request
- **THEN** request is rejected before dispatch

#### Scenario: Business semantic error rejected at worker
- **GIVEN** config passes server schema gate
- **AND** config violates worker business constraints
- **WHEN** worker runs typed `Validate()`
- **THEN** worker rejects config with fail-fast semantic error

### Requirement: Contract generation MUST support global workflow generation entry
The toolchain MUST provide a global generation entry to produce schema/docs/typed artifacts for workflows, reducing per-workflow manual generator boilerplate.

#### Scenario: Global generation command runs for registered workflows
- **GIVEN** one or more workflows are registered in worker
- **WHEN** developer runs global contract generation command
- **THEN** schema/docs/typed artifacts are generated with deterministic naming
- **AND** consistency tests can validate outputs without per-workflow manual steps

#### Scenario: Global generation entry is the authoritative path
- **GIVEN** contracts are generated in CI or release workflow
- **WHEN** global generation command runs
- **THEN** global generation command is treated as the only authoritative path
- **AND** per-workflow `contract_assets.go` boilerplate is not required

### Requirement: Worker validation flow MUST avoid duplicated semantic validation
The worker MUST avoid duplicated semantic validation calls in the same execution path after successful typed decode.

#### Scenario: Semantic validation runs once in normal decode path
- **GIVEN** workflow config is decoded through standard worker decode pipeline
- **WHEN** workflow execution initializes runtime context
- **THEN** semantic `Validate()` is executed once for that config path
- **AND** failure timing and error semantics remain unchanged
