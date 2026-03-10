## ADDED Requirements

### Requirement: HTTP JSON contracts MUST use camelCase field names
The system MUST expose HTTP JSON request and response fields in `camelCase`, and MUST remove legacy `snake_case` JSON names when no compatibility bridge is required.

#### Scenario: recovery error response includes requestId
- **GIVEN** an HTTP request triggers panic recovery
- **WHEN** the server returns the JSON error payload
- **THEN** the payload includes `requestId`
- **AND** the payload does not include `request_id`

### Requirement: Gin context data MUST be accessed through middleware accessors
The system MUST encapsulate request, user, and agent context data behind middleware accessor helpers, so that business code does not rely on raw context key strings.

#### Scenario: agent middleware stores authenticated identity
- **GIVEN** agent authentication succeeds
- **WHEN** downstream code reads agent identity
- **THEN** it uses middleware accessors instead of hard-coded context key strings
- **AND** the agent ID is available as a typed integer value

### Requirement: Structured logs MUST use semantic field names
The system MUST emit structured log fields using semantic names instead of legacy `snake_case` field aliases.

#### Scenario: HTTP request log emits semantic fields
- **GIVEN** an HTTP request completes
- **WHEN** the request middleware writes a log entry
- **THEN** the log contains semantic fields such as `request.id`, `http.request.method`, and `http.response.status_code`
- **AND** the log does not contain legacy fields such as `request_id` or `user_agent`

### Requirement: External label contracts MUST preserve provider-compatible names
The system MUST preserve provider-compatible label names for Loki / Prometheus style external labels, even when internal observability fields use semantic names.

#### Scenario: Loki query uses provider-compatible labels
- **GIVEN** the server builds a LogQL selector for agent logs
- **WHEN** the query string is generated
- **THEN** it uses labels such as `agent_id` and `container_name`
- **AND** it does not use dotted field names

### Requirement: Public argument names MUST use camelCase when explicitly named
The system MUST use `camelCase` field names in externally visible error messages and explicitly named route parameters.

#### Scenario: batch upsert validation reports missing IDs
- **GIVEN** a runtime batch upsert request misses required identifiers
- **WHEN** the server returns an invalid argument error
- **THEN** the message refers to `scanId` and `targetId`
- **AND** it does not refer to `scan_id` or `target_id`

### Requirement: Repository MUST enforce boundary naming rules automatically
The repository MUST provide an automated interface naming check that blocks newly introduced boundary naming regressions while allowing documented exception areas.

#### Scenario: CI rejects a new snake_case boundary field
- **GIVEN** a change introduces a new `snake_case` JSON field, legacy log key, raw Gin context key, or explicit snake_case path parameter in source code
- **WHEN** the interface naming check runs in CI
- **THEN** the check fails with the matching rule name and file location
- **AND** documented exception areas such as generated protobuf files, GORM column names, SQL, Loki labels, and provider placeholders remain allowed
