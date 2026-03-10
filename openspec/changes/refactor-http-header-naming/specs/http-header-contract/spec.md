## ADDED Requirements

### Requirement: Application request correlation MUST use Request-Id
The system MUST read and emit the application request correlation header as `Request-Id`, and MUST stop using `X-Request-ID` as the canonical application contract.

#### Scenario: Middleware generates request ID when header is absent
- **GIVEN** an HTTP request does not include `Request-Id`
- **WHEN** the request logger middleware handles the request
- **THEN** it generates a request ID and stores it in request context
- **AND** it returns the generated value in the `Request-Id` response header

#### Scenario: Middleware preserves caller-provided request ID
- **GIVEN** an HTTP request includes `Request-Id`
- **WHEN** the request logger middleware handles the request
- **THEN** it reuses the provided request ID for logging and context
- **AND** it does not require `X-Request-ID` to be present

### Requirement: Application header modernization MUST NOT rewrite proxy compatibility headers
The system MUST keep reverse-proxy compatibility headers outside the scope of application header modernization, so that infrastructure behavior remains unchanged.

#### Scenario: Reverse proxy forwards compatibility headers
- **GIVEN** the deployment proxy forwards request metadata to the backend
- **WHEN** the proxy configuration is reviewed under this change
- **THEN** headers such as `X-Forwarded-For`, `X-Forwarded-Proto`, and `X-Real-IP` remain valid compatibility headers
- **AND** they are not renamed solely to remove the `X-` prefix
