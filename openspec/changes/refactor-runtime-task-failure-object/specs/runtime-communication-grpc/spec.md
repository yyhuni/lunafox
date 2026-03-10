## MODIFIED Requirements

### Requirement: Agent runtime task status MUST carry structured failure details
The system MUST represent task failure details in the runtime gRPC status contract as a structured failure object instead of parallel message/kind scalar fields.

#### Scenario: Failed task reports failure kind and message
- **GIVEN** an agent reports a task status update with `status = "failed"`
- **WHEN** the runtime gRPC request reaches server
- **THEN** the request payload includes a structured `failure` object
- **AND** `failure.message` contains the human-readable failure description
- **AND** `failure.kind` contains the machine-readable failure classification

#### Scenario: Non-failed task omits failure object
- **GIVEN** an agent reports a task status update with `status = "completed"` or `status = "cancelled"`
- **WHEN** the runtime gRPC request is sent
- **THEN** the request does not carry a failure object
- **AND** server MUST NOT infer stale failure data from previous updates

### Requirement: Scan query responses MUST expose structured scan failure summary
The system MUST expose scan-level failure summary in HTTP query responses as a structured failure object so that frontend consumers can rely on stable machine-readable failure categories.

#### Scenario: Failed scan response includes failure object
- **GIVEN** a scan has been aggregated to `status = "failed"`
- **WHEN** the client queries scan list or scan detail HTTP API
- **THEN** the response includes `failure.kind`
- **AND** the response includes `failure.message`

#### Scenario: Non-failed scan response omits failure object
- **GIVEN** a scan is `pending`, `running`, `completed`, or `cancelled`
- **WHEN** the client queries scan list or scan detail HTTP API
- **THEN** the response does not include a failure object
