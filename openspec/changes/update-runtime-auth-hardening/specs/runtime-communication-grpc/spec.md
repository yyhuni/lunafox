## ADDED Requirements

### Requirement: Worker runtime token MUST be task-scoped and bound to task identity
The system MUST validate worker runtime requests on agent UDS using both token authenticity and task scope binding, so that a token issued for one task cannot be reused for another task.

#### Scenario: Token does not match requested task
- **GIVEN** a worker runtime request carries a non-empty `x-task-token`
- **WHEN** the token does not belong to the `task_id` in request payload
- **THEN** agent rejects the request with `PermissionDenied`
- **AND** the request is not forwarded to upstream runtime data calls

### Requirement: Agent data proxy RPCs MUST require authenticated agent identity
The system MUST require valid `x-agent-key` metadata for AgentDataProxy gRPC methods on server, and MUST reject anonymous calls.

#### Scenario: Data proxy call without agent key
- **GIVEN** a caller sends gRPC request to AgentDataProxy without `x-agent-key`
- **WHEN** server runtime gRPC receives the call
- **THEN** server rejects the request with `Unauthenticated`
- **AND** no data proxy handler logic is executed
