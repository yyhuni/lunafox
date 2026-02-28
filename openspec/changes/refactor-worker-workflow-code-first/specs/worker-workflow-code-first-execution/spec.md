## ADDED Requirements

### Requirement: Worker workflow execution MUST be code-first and template-free
The system MUST define workflow stage orchestration, command definitions, and execution behavior in Go code, and MUST NOT depend on runtime command templates for execution.

#### Scenario: Code-defined workflow executes a configured scan
- **GIVEN** a worker receives a valid workflow name and YAML-derived configuration
- **WHEN** the workflow starts execution
- **THEN** all stage transitions and tool invocations are determined by Go code
- **AND** no runtime template loader is required to build executable commands

### Requirement: Pre-launch migration MUST cut over in one release without long-lived dual execution paths
For the pre-launch phase, the system MUST complete migration to code-first workflow execution in a single release cutover and MUST NOT keep long-lived template-driven runtime execution paths after cutover.

#### Scenario: Release readiness before first production launch
- **GIVEN** all existing workflows have code-first implementations and validation coverage
- **WHEN** release readiness checks are completed
- **THEN** runtime execution uses only code-first workflow paths
- **AND** template-driven execution code is removed or disabled from runtime

#### Scenario: Migration scope is limited to currently executable worker workflows
- **GIVEN** the worker binary has an explicit static workflow registration set
- **WHEN** pre-launch one-release cutover is executed
- **THEN** only currently executable workflows in that registration set are mandatory migration scope
- **AND** at present this scope includes `subdomain_discovery`

### Requirement: YAML MUST be treated as input parameters only
The system MUST treat YAML as user-provided workflow parameters and MUST NOT allow YAML to redefine executable command templates.

#### Scenario: YAML contains only configurable parameters
- **GIVEN** a user submits scan configuration YAML
- **WHEN** server and worker parse the configuration
- **THEN** YAML values are mapped into typed workflow configuration fields
- **AND** executable command definitions remain fixed in workflow code

### Requirement: Workflow configuration MUST use strong typing and strict validation
The system MUST decode workflow configuration into typed structs and MUST reject invalid types, missing required fields, out-of-range values, and unknown keys before execution.

#### Scenario: Unknown configuration key is provided
- **GIVEN** workflow configuration includes an unrecognized key
- **WHEN** validation runs before workflow execution
- **THEN** the request is rejected as invalid configuration
- **AND** no workflow command is executed

### Requirement: Schema and docs contracts MUST be auto-generated from code definitions
The system MUST auto-generate workflow schema and configuration docs from workflow code definitions, and MUST treat generated artifacts as build outputs rather than manually maintained sources.

#### Scenario: Contract generation runs in CI
- **GIVEN** workflow configuration fields are modified in code
- **WHEN** contract generation runs
- **THEN** schema artifacts are updated under `server/internal/engineschema`
- **AND** docs artifacts are updated under `docs/config-reference`
- **AND** CI fails if generated artifacts are out of date

#### Scenario: Contracts mirror output is disabled by default
- **GIVEN** normal development and CI generation flow
- **WHEN** contract generation runs without explicit distribution flag
- **THEN** no artifacts are written to `contracts/gen/engineschema`
- **AND** canonical schema/docs outputs remain in `server/internal/engineschema` and `docs/config-reference`

#### Scenario: Optional contracts mirror is enabled
- **GIVEN** repository enables external contract distribution
- **WHEN** contract generation runs with mirror output enabled
- **THEN** schema artifacts are mirrored to `contracts/gen/engineschema`
- **AND** server runtime validation still uses `server/internal/engineschema` as canonical source

### Requirement: Command execution MUST avoid shell string injection patterns
The system MUST execute tool commands using explicit binary and argument lists, and MUST NOT execute user-influenced shell-concatenated command strings.

#### Scenario: User parameter includes shell metacharacters
- **GIVEN** a user-provided parameter contains shell metacharacters
- **WHEN** the workflow builds the command invocation
- **THEN** the parameter is passed as an argument value in a safe execution API
- **AND** no additional unintended command is executed

#### Scenario: Runtime command execution does not use sh -c
- **GIVEN** a workflow command is ready for execution
- **WHEN** worker launches the external tool process
- **THEN** worker uses a binary-and-args execution interface
- **AND** worker does not invoke `sh -c` to interpret command strings

### Requirement: Worker MUST execute external tools through a standardized CmdRunner component
The system MUST provide a standardized command runner in `worker/internal/activity`, and workflow implementations MUST use this runner instead of directly invoking `os/exec`.

#### Scenario: Workflow executes an external command
- **GIVEN** a workflow stage needs to launch a scanning tool
- **WHEN** the stage prepares command invocation
- **THEN** execution is delegated to the standardized CmdRunner component
- **AND** workflow code does not directly call `os/exec` APIs

#### Scenario: Binary preflight validation fails
- **GIVEN** command invocation references a binary not available in worker runtime environment
- **WHEN** CmdRunner performs preflight validation
- **THEN** CmdRunner returns a clear executable-not-found error before process launch
- **AND** workflow receives a structured failure result without partial execution

#### Scenario: Cancellation and timeout propagate to child process tree
- **GIVEN** an external command is running under workflow execution
- **WHEN** context cancellation or timeout is triggered
- **THEN** CmdRunner terminates the command process tree
- **AND** CmdRunner returns a timeout/cancelled execution result

#### Scenario: Stdout/stderr are captured with safe truncation policy
- **GIVEN** a running command emits large output on stdout/stderr
- **WHEN** CmdRunner captures process output
- **THEN** output streams are captured through unified reader logic
- **AND** overlong lines are truncated according to policy and logged safely

### Requirement: Workflow extensibility MUST default to compile-time registration
The system MUST support adding new workflows by implementing Go workflow code and registering it at build time as the default extension mechanism.

#### Scenario: Team adds a new built-in workflow
- **GIVEN** developers implement a new workflow package in worker code
- **WHEN** the binary is built and started
- **THEN** the workflow is available through static registration
- **AND** no runtime plugin loading step is required

### Requirement: Dynamic workflow extension MUST be process-isolated if enabled
If dynamic third-party workflow extension is introduced, the system MUST use a process-isolated plugin protocol boundary and MUST NOT rely on in-process Go plugin loading in worker.

#### Scenario: Third-party workflow extension is enabled in a future phase
- **GIVEN** an operator enables external workflow extension capability
- **WHEN** worker executes third-party workflow logic
- **THEN** execution occurs through an isolated protocol boundary
- **AND** failures or incompatibilities in extension code do not require in-process Go plugin loading
