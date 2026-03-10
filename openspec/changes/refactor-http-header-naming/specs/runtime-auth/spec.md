## ADDED Requirements

### Requirement: Agent HTTP authentication MUST use Authorization Bearer
The system MUST authenticate agent-facing HTTP requests with `Authorization: Bearer <agent-token>`, and MUST stop using `X-Agent-Key` as the primary application authentication contract.

#### Scenario: Agent request provides bearer token
- **GIVEN** an agent HTTP request sends `Authorization: Bearer <agent-token>`
- **WHEN** agent authentication middleware validates the request
- **THEN** the request is authenticated using the bearer token value
- **AND** downstream handlers can access the authenticated agent through middleware helpers

#### Scenario: Agent request omits bearer token
- **GIVEN** an agent HTTP request does not send a valid `Authorization` bearer header
- **WHEN** agent authentication middleware validates the request
- **THEN** the request is rejected as unauthorized
- **AND** the middleware does not require `X-Agent-Key`

### Requirement: New runtime HTTP authentication MUST NOT introduce private X-prefixed headers
The system MUST NOT introduce new application-layer runtime authentication headers with an `X-` prefix for agent or worker HTTP authentication.

#### Scenario: Future runtime auth extension is proposed
- **GIVEN** a new runtime HTTP endpoint needs token authentication
- **WHEN** the authentication contract is defined
- **THEN** it reuses `Authorization` or another standardized mechanism
- **AND** it does not define a new header such as `X-Agent-Key` or `X-Worker-Token`

### Requirement: Worker HTTP authentication MUST be treated as legacy until an HTTP route is restored
The system MUST treat `X-Worker-Token` based worker HTTP authentication as legacy behavior, and MUST NOT extend it to new runtime routes while worker runtime writes remain on gRPC.

#### Scenario: Worker runtime writes stay on gRPC data path
- **GIVEN** worker runtime data writes are handled by the gRPC runtime data proxy
- **WHEN** maintainers review HTTP worker authentication code
- **THEN** `X-Worker-Token` middleware is handled as legacy or cleanup scope
- **AND** no new HTTP worker route is introduced that depends on it
