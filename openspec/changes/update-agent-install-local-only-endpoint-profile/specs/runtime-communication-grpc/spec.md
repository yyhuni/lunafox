## ADDED Requirements

### Requirement: Install script profile MUST be selected via explicit endpoint
The system MUST expose explicit install-script endpoints for local and remote deployment profiles, and MUST NOT rely on `mode` query fallback behavior.

#### Scenario: Local install script is requested
- **GIVEN** client requests `/api/agent/install-script/local?token=...`
- **WHEN** server validates token and renders script
- **THEN** server responds with HTTP 200 and returns install script content

#### Scenario: Remote install script is requested
- **GIVEN** client requests `/api/agent/install-script/remote?token=...`
- **WHEN** server validates token and renders script
- **THEN** server responds with HTTP 200 and returns install script content

### Requirement: Install script endpoint ownership MUST be explicit by caller type
The system MUST enforce caller ownership in implementation and tests: installer flow uses local endpoint, frontend flow uses remote endpoint.

#### Scenario: Installer flow requests local endpoint
- **GIVEN** `tools/installer` downloads install script for project-install local agent bootstrap
- **WHEN** installer builds install script request URL
- **THEN** request path is `/api/agent/install-script/local`
- **AND** request query MUST NOT include `mode`

#### Scenario: Frontend flow requests remote endpoint
- **GIVEN** frontend agent install dialog builds one-click install command
- **WHEN** frontend renders install command
- **THEN** command URL path is `/api/agent/install-script/remote`
- **AND** command URL query MUST NOT include `mode`

### Requirement: Install script endpoint MUST reject mode query parameter
The system MUST reject install-script requests carrying `mode` query parameter and MUST return a clear migration error without script content.

#### Scenario: Local endpoint receives mode
- **GIVEN** install script request is `/api/agent/install-script/local?...&mode=local`
- **WHEN** server handles the request
- **THEN** server responds with HTTP 400
- **AND** response states `mode` is no longer supported

#### Scenario: Remote endpoint receives mode
- **GIVEN** install script request is `/api/agent/install-script/remote?...&mode=remote`
- **WHEN** server handles the request
- **THEN** server responds with HTTP 400
- **AND** no install script content is returned

### Requirement: Install script endpoints MUST render endpoint matrix without manual user input
The system MUST pre-render `REGISTER_URL` and `RUNTIME_GRPC_URL` according to selected profile, and users MUST NOT be required to manually provide these values during normal installation flow.

#### Scenario: Local profile endpoint matrix
- **GIVEN** install script is requested via local endpoint
- **WHEN** server renders endpoint variables
- **THEN** `REGISTER_URL` is rendered from `PUBLIC_URL`
- **AND** `RUNTIME_GRPC_URL` is rendered as `http://server:<SERVER_GRPC_PORT>`

#### Scenario: Remote profile endpoint matrix
- **GIVEN** install script is requested via remote endpoint
- **WHEN** server renders endpoint variables
- **THEN** `REGISTER_URL` is rendered from `PUBLIC_URL`
- **AND** `RUNTIME_GRPC_URL` is rendered from `PUBLIC_URL`

### Requirement: Local profile script MUST fail fast on missing Docker network
The local profile install script MUST validate required Docker network existence before starting registration or agent container, and MUST NOT silently downgrade to default bridge.

#### Scenario: Local profile network is missing
- **GIVEN** local profile script starts and configured network does not exist
- **WHEN** script performs preflight checks
- **THEN** script exits with explicit actionable error
- **AND** script does not run registration or `docker run` for agent container

#### Scenario: Local profile network exists
- **GIVEN** local profile script starts and configured network exists
- **WHEN** script proceeds to registration and agent startup
- **THEN** registration and agent container both use the same configured network
