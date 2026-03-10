## ADDED Requirements

### Requirement: Scan task runtime MUST validate task failure object consistency
The scan task runtime MUST validate task status updates against a structured failure object contract so that invalid combinations are rejected before task persistence.

#### Scenario: Failed task update without failure message is rejected
- **GIVEN** a task status update requests `status = "failed"`
- **AND** the update carries no failure object or an empty `failure.message`
- **WHEN** application runtime service validates the update
- **THEN** the update is rejected as invalid
- **AND** no task status persistence occurs

#### Scenario: Non-failed task update with failure payload is rejected
- **GIVEN** a task status update requests `status = "completed"` or `status = "cancelled"`
- **AND** the update carries a non-empty failure object
- **WHEN** application runtime service validates the update
- **THEN** the update is rejected as invalid
- **AND** no task failure fields are persisted

### Requirement: Task repository persistence MUST flatten failure object into task columns
The task repository persistence layer MUST map a validated task failure object into `scan_task.error_message` and `scan_task.failure_kind` columns while keeping relational queryability.

#### Scenario: Failed task persists flattened failure object
- **GIVEN** application runtime passes a validated task failure object with `kind` and `message`
- **WHEN** repository updates the scan task row
- **THEN** `scan_task.error_message` stores the failure message
- **AND** `scan_task.failure_kind` stores the failure kind

#### Scenario: Failed task with empty kind is normalized to unknown
- **GIVEN** application runtime passes a failed status with non-empty failure message and empty failure kind
- **WHEN** task repository persistence is requested after application normalization
- **THEN** `scan_task.failure_kind` stores `unknown`
- **AND** `scan_task.error_message` stores the original failure message

### Requirement: Scan aggregate MUST project structured failure summary
The scan aggregate MUST expose a structured failure summary derived from task failures when the scan enters failed status.

#### Scenario: Failed scan stores projected failure summary from canonical failed task
- **GIVEN** multiple failed tasks exist under the same scan
- **AND** each failed task carries a validated failure object
- **WHEN** scan status is recalculated into `failed`
- **THEN** the system selects one canonical failed task using deterministic ordering
- **AND** `scan.error_message` stores the canonical task failure message
- **AND** `scan.failure_kind` stores the canonical task failure kind

#### Scenario: Canonical failed task prefers root-cause failure kind
- **GIVEN** one failed task has `failure.kind = "schema_invalid"`
- **AND** another failed task has `failure.kind = "task_timeout"`
- **WHEN** scan failure summary is projected
- **THEN** the `schema_invalid` task is selected as canonical
- **AND** scan failure summary uses `schema_invalid`

#### Scenario: Canonical failed task ordering is stable for same priority
- **GIVEN** multiple failed tasks share the same failure priority tier
- **WHEN** scan failure summary is projected
- **THEN** the system resolves ties by smaller stage, then earlier completion time, then smaller task ID
- **AND** repeated projections produce the same scan failure summary

#### Scenario: Non-failed scan clears failure summary
- **GIVEN** a scan aggregate is `pending`, `running`, `completed`, or `cancelled`
- **WHEN** scan status is persisted
- **THEN** the scan does not expose a failure summary object
- **AND** failure summary fields are cleared or omitted from query projection
