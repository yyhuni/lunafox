## ADDED Requirements

### Requirement: Authenticated database health snapshot API
The system MUST provide an authenticated endpoint `GET /api/system/database-health/` that returns a database health snapshot for the settings page.

#### Scenario: Authorized user requests database health snapshot
- **GIVEN** a user has a valid JWT token
- **WHEN** the user calls `GET /api/system/database-health/`
- **THEN** the system returns HTTP 200 with a structured health payload
- **AND** the payload includes status, core signals, timestamps, and alerts fields

#### Scenario: Unauthorized request is rejected
- **GIVEN** a request without valid authentication
- **WHEN** the request calls `GET /api/system/database-health/`
- **THEN** the system rejects the request using the existing protected-route authentication behavior

### Requirement: Server-owned health status semantics
The system MUST compute `online`, `degraded`, `offline`, and `maintenance` status in backend logic and MUST expose the computed status directly to clients.

#### Scenario: Core probe fails
- **GIVEN** database probe cannot connect or times out
- **WHEN** health snapshot is generated
- **THEN** status is `offline`
- **AND** the response contains failure context for diagnostics

#### Scenario: Core probe succeeds but thresholds are violated
- **GIVEN** database remains reachable
- **AND** one or more core metrics exceed degradation thresholds
- **WHEN** health snapshot is generated
- **THEN** status is `degraded`
- **AND** status is NOT set to `offline`

### Requirement: Core metrics are mandatory and stable
The system MUST provide a minimum core metrics set that is consistently available for operational health decisions.

#### Scenario: Snapshot includes core metrics
- **GIVEN** snapshot collection succeeds
- **WHEN** the API responds
- **THEN** the payload includes probe latency, connection usage, replication lag (when applicable), and backup freshness
- **AND** each core metric is represented in a machine-readable numeric format

### Requirement: Optional metrics must not determine offline state
The system SHOULD expose optional performance metrics when available, but MUST NOT mark the database `offline` only because optional metrics are missing.

#### Scenario: Optional metric collection fails
- **GIVEN** optional metrics cannot be collected due to permissions or transient query failure
- **WHEN** health snapshot is generated
- **THEN** the response still includes core metrics and computed status
- **AND** missing optional metrics are explicitly reported as unavailable signals

### Requirement: Timestamp standardization for frontend rendering
The system MUST return machine-readable timestamps (ISO 8601) for health observation and alert occurrence fields.

#### Scenario: Frontend receives health payload
- **GIVEN** the API response includes timestamp fields
- **WHEN** frontend renders database health
- **THEN** frontend can compute locale-specific display text without parsing human-readable backend strings
- **AND** backend does not depend on locale-specific relative-time text

### Requirement: Frontend must trust backend health status
The frontend MUST render top-level health state directly from backend status and MUST NOT apply independent status derivation for core health semantics.

#### Scenario: Backend reports degraded status
- **GIVEN** backend response status is `degraded`
- **WHEN** frontend refreshes the page data
- **THEN** the page shows degraded as the canonical state
- **AND** frontend does not override it to `online` or `offline` through local threshold logic
