## ADDED Requirements

### Requirement: Task lifecycle SHALL remain compact while preserving failure semantics
The system SHALL keep the primary scan task lifecycle limited to `pending`, `running`, `completed`, `failed`, and `cancelled`, and SHALL use supporting fields to explain failure stage and cause.

#### Scenario: Scheduler assigns a pending task
- **WHEN** the scheduler assigns a pending task to an agent
- **THEN** the task status SHALL transition from `pending` to `running`
- **AND** the system SHALL NOT require a separate `claimed` status in this change

### Requirement: Pre-execution failures SHALL be recorded as failed with classification
The system SHALL allow claim-stage fallback failures, agent bootstrap failures, worker bootstrap failures, and worker config decode failures to terminate a task as `failed`, provided that a machine-readable failure classification is recorded.

#### Scenario: Claim fallback fails before workflow execution begins
- **WHEN** claim 后的静态兜底失败、agent 断线、worker 启动失败或 worker 配置解码失败发生
- **THEN** the task SHALL transition to `failed`
- **AND** the system SHALL record a `failureKind` that identifies the failure stage

#### Scenario: Worker fails to decode workflow config
- **WHEN** the worker cannot decode the assigned workflow config
- **THEN** the task SHALL transition to `failed`
- **AND** the system SHALL record a failure classification for config decode failure

### Requirement: Create-time validation SHALL remain the primary correctness gate
The system SHALL continue to validate workflow identity and schema correctness during scan creation, while worker-side decoding remains a defense-in-depth fallback.

#### Scenario: Invalid workflow config is rejected during scan creation
- **WHEN** scan creation receives invalid workflow config
- **THEN** the server SHALL reject the scan request before creating runnable tasks

#### Scenario: Corrupted task data reaches the worker path
- **WHEN** corrupted or incompatible workflow config bypasses create-time validation and reaches the worker
- **THEN** the worker path SHALL fail the task
- **AND** the task SHALL still record a machine-readable failure classification
