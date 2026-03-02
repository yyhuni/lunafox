## ADDED Requirements

### Requirement: Scheduler compatibility gate MUST evaluate worker capability snapshot instead of agent version
The scheduler compatibility gate MUST determine workflow tuple compatibility from worker capability snapshot bound to the agent runtime record, and MUST NOT use `agent.version` as the primary capability source.

#### Scenario: Agent version appears compatible but worker version is incompatible
- **GIVEN** an agent record reports `agent.version = 1.0.0`
- **AND** the same runtime snapshot reports incompatible `workerVersion`
- **WHEN** scheduler evaluates `workflow + apiVersion + schemaVersion`
- **THEN** task assignment is rejected before dispatch
- **AND** the rejection is surfaced as `WORKER_VERSION_INCOMPATIBLE`

#### Scenario: Worker capability snapshot is compatible
- **GIVEN** worker capability snapshot supports requested workflow tuple
- **WHEN** scheduler evaluates compatibility during task pull
- **THEN** task assignment is allowed

### Requirement: Missing worker capability snapshot MUST fail closed
If worker capability snapshot is missing or empty, scheduler compatibility gate MUST reject task assignment (fail-closed) and MUST provide explicit diagnostic message.

#### Scenario: Legacy heartbeat without worker capability fields
- **GIVEN** runtime heartbeat does not provide worker capability fields
- **WHEN** scheduler evaluates compatibility
- **THEN** task assignment is rejected
- **AND** error message indicates missing worker capability snapshot

### Requirement: Runtime heartbeat pipeline MUST persist worker capability fields
The runtime heartbeat handling pipeline MUST map and persist worker capability fields so that scheduler compatibility can consume the latest snapshot.

#### Scenario: Heartbeat carries workerVersion
- **GIVEN** agent sends heartbeat containing `workerVersion`
- **WHEN** runtime service handles the heartbeat
- **THEN** worker version is persisted in runtime storage
- **AND** subsequent scheduler compatibility checks read the persisted value

### Requirement: Incompatibility rejection contract MUST remain stable
When compatibility check fails, the system MUST keep existing workflow error contract and transport mapping stable.

#### Scenario: Compatibility gate rejects tuple in task pull
- **GIVEN** scheduler compatibility gate returns incompatible
- **WHEN** `PullTask` handles the rejected assignment
- **THEN** task claim is released
- **AND** error code remains `WORKER_VERSION_INCOMPATIBLE`
- **AND** gRPC mapping remains `FailedPrecondition`
