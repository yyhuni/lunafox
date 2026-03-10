## ADDED Requirements

### Requirement: Agent runtime control plane MUST use gRPC bidirectional streaming
The system MUST establish `agent <-> server` runtime control communication through a gRPC bidirectional stream instead of WebSocket and task HTTP endpoints.

#### Scenario: Agent establishes runtime stream successfully
- **GIVEN** an agent has valid credentials
- **WHEN** the agent starts and opens the runtime channel
- **THEN** the server authenticates the stream and keeps it open as the canonical control channel
- **AND** the agent can send heartbeat and receive control events over the same stream

#### Scenario: Runtime stream is interrupted
- **GIVEN** an established runtime stream between agent and server
- **WHEN** network interruption or server restart occurs
- **THEN** the agent retries connection with backoff
- **AND** control communication resumes on the recovered stream without fallback to WebSocket

### Requirement: Task lifecycle signaling MUST be unified in the runtime stream
The system MUST deliver task assignment, cancellation, and status updates through typed gRPC runtime messages.

#### Scenario: Agent requests and receives a task assignment
- **GIVEN** an agent runtime stream is connected and the agent has available capacity
- **WHEN** the agent sends a `request_task` message on the stream
- **THEN** the server responds with a `task_assign` runtime event containing the next eligible task
- **AND** the agent reports `task_status(running)` as implicit acknowledgement before subsequent status transitions

#### Scenario: Server cancels a running task
- **GIVEN** a task is running on an agent
- **WHEN** cancellation is requested
- **THEN** the server sends a `task_cancel` runtime event on the stream
- **AND** the agent reports terminal cancellation status through the same stream

### Requirement: Worker MUST communicate only with local agent over UDS gRPC
The system MUST route worker runtime calls through a local Unix Domain Socket gRPC service exposed by agent, and worker MUST NOT call server runtime APIs directly.

#### Scenario: Worker performs runtime data request
- **GIVEN** a worker container is started by agent
- **WHEN** the worker needs provider configuration, wordlist content, or result persistence
- **THEN** the worker sends the request to the local agent UDS gRPC endpoint
- **AND** no direct network call from worker to server runtime API is performed

#### Scenario: Worker request is missing task-scoped credentials
- **GIVEN** a worker request reaches the local UDS endpoint without valid task-scoped token
- **WHEN** agent validates the request metadata
- **THEN** agent rejects the request with an authentication error
- **AND** the request is not forwarded to server

### Requirement: Agent MUST proxy worker data-plane requests to server via gRPC
The system MUST make agent responsible for forwarding worker-originated runtime data requests to server through typed gRPC APIs.

#### Scenario: Worker submits batch scan results
- **GIVEN** worker sends a batch-upsert request to local agent
- **WHEN** the request passes local validation
- **THEN** agent forwards it to server via gRPC data proxy service
- **AND** agent returns structured success or error response back to worker

#### Scenario: Proxy call to server fails transiently
- **GIVEN** agent is forwarding a worker request to server
- **WHEN** a retryable transport or server error occurs
- **THEN** agent applies configured retry/backoff policy
- **AND** returns a typed error when retries are exhausted

### Requirement: Legacy runtime HTTP and WebSocket endpoints MUST be removed after gRPC cutover
The system MUST remove runtime endpoints previously used by agent and worker (`/api/agent/ws`, `/api/agent/tasks/*`, `/api/worker/*`) once gRPC runtime paths are active.

#### Scenario: Legacy runtime endpoint is called after cutover
- **GIVEN** gRPC runtime communication has been enabled and old handlers removed
- **WHEN** an internal component attempts to call a legacy runtime endpoint
- **THEN** the request is rejected as unsupported
- **AND** system logs indicate the endpoint has been retired

### Requirement: Runtime protobuf contracts MUST be versioned and code-generated for all components
The system MUST define versioned protobuf contracts for runtime communication and MUST use generated types/stubs in server, agent, and worker implementations.

#### Scenario: Runtime contract is updated
- **GIVEN** a protobuf runtime message or service definition changes
- **WHEN** the build pipeline runs
- **THEN** generated code for server, agent, and worker is refreshed consistently
- **AND** CI fails if generated artifacts and proto sources are out of sync

### Requirement: Server runtime gRPC listener MUST use a dedicated internal port
The system MUST expose runtime gRPC services on a dedicated internal listener port, while preserving unified external ingress through existing `443` reverse-proxy routing.

#### Scenario: External clients use unified ingress
- **GIVEN** external ingress is terminated at Nginx on `443`
- **WHEN** runtime gRPC traffic is received
- **THEN** Nginx forwards gRPC requests to the dedicated internal gRPC listener
- **AND** HTTP management API requests continue to route to the existing HTTP listener

### Requirement: Phase-1 gRPC authentication MUST use token metadata without mTLS/SPIFFE
The system MUST authenticate runtime gRPC requests using existing token semantics carried in gRPC metadata, and phase-1 implementation MUST NOT require mTLS/SPIFFE infrastructure.

#### Scenario: Agent connects to server runtime stream
- **GIVEN** agent initiates gRPC runtime connection
- **WHEN** metadata contains valid `x-agent-key`
- **THEN** server authenticates the stream successfully
- **AND** connection does not require SPIFFE certificate identity in phase-1

#### Scenario: Worker-originated request is proxied by agent
- **GIVEN** worker sends request to agent UDS runtime endpoint
- **WHEN** request carries valid task-scoped token and agent forwards to server with valid service token metadata
- **THEN** server accepts the proxied request
- **AND** invalid or missing tokens are rejected with authentication errors
