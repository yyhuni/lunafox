## ADDED Requirements

### Requirement: Worker and server domain logs MUST use semantic dotted field names
The system MUST emit worker runtime fields and server domain observability fields using semantic dotted log keys instead of ad-hoc `camelCase` names in structured log bodies.

#### Scenario: worker startup log uses semantic identity keys
- **GIVEN** a worker process logs its startup context
- **WHEN** the structured entry is emitted
- **THEN** it uses fields such as `task.id`, `scan.id`, `target.id`, `target.name`, and `target.type`
- **AND** it does not use `taskId`, `scanId`, `targetId`, `targetName`, or `targetType`

#### Scenario: queue overflow log uses semantic batch keys
- **GIVEN** the worker batch sender drops data because the in-memory queue is full
- **WHEN** it logs the overflow event
- **THEN** it uses semantic fields such as `queue.length`, `queue.max_items`, `drop.total`, and `retry.count` where applicable
- **AND** it does not use legacy `camelCase` queue field names

### Requirement: HTTP semantic convention fields MUST remain allowed
The repository MUST keep OpenTelemetry-style HTTP semantic convention field names available in structured logs, even when those semantic field names include underscores.

#### Scenario: HTTP middleware log remains valid
- **GIVEN** HTTP middleware emits request observability fields
- **WHEN** interface naming enforcement runs
- **THEN** fields such as `http.response.status_code`, `http.server.request.duration_ms`, and `user_agent.original` remain allowed
- **AND** the enforcement only rejects non-OTel ad-hoc `camelCase` zap keys

### Requirement: CI MUST reject new worker/server camelCase zap keys
The repository MUST provide automated enforcement that rejects newly introduced worker/server structured log fields using ad-hoc `camelCase` names.

#### Scenario: CI rejects a new camelCase log key
- **GIVEN** a Go source file introduces a structured log field such as `queueLength` or `taskId`
- **WHEN** the interface naming check runs
- **THEN** the check fails with the matching rule name and file location
- **AND** documented OTel semantic convention fields remain allowed
