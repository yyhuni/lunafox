# Agent Control Ingress

`agentcontrol` owns Server-side Agent connection-source observation. The
observation is display-only and non-trusted: Agent registration, heartbeat,
and control payloads cannot supply or overwrite it.

## Source Observation

- Built-in Nginx overwrites Agent gRPC `x-forwarded-for` with its direct
  `$remote_addr`; it never appends an Agent-controlled forwarded chain.
- Server first requires one usable gRPC direct peer. When present, one valid
  and normalizable `x-forwarded-for` value takes precedence. Missing, repeated,
  chained, malformed, or special-use forwarded metadata falls back to the
  direct peer. No proxy CIDR, transport identity, or chain peeling is used.
- The normalized candidate becomes `connectionIp` when it is a usable public,
  private, loopback, link-local, or CGNAT IPv4/IPv6 address. Missing,
  malformed, unspecified, multicast, documentation, benchmarking, reserved,
  and other unsupported special-use candidates leave it empty. `connectionIp`
  is the current transport observation, not a host address or a reachability
  promise.
- `observedSourceIp` is the strict-public subset of the same candidate. Only it
  owns GeoIP, location provenance, and its monotonic generation. A private
  `connectionIp` is still shown to operators but never triggers GeoIP or
  changes public-location provenance. A location is an egress/proxy best-effort
  result, not an Agent-location guarantee.
- Runtime status records both values before `SessionReady` and clears only
  `connectionIp` when the control connection disconnects or times out. Agent
  registration, heartbeat, and control payloads cannot supply or overwrite
  either value. Neither address may enter authentication, authorization,
  audit, alerting, risk, policy, scheduling, routing, rate limiting, or any
  other business decision.

The default topology has one internal Server gRPC listener on `SERVER_GRPC_PORT`
(`9090`) and requires the existing Agent authentication token metadata
(`lunafox-agent-authentication-token`) for every unary and streaming RPC. It
does not add a proxy listener, mutual TLS, a proxy principal, or an internal
certificate authority.

## Agent Runtime Transport Topology

Server keeps the existing single gRPC server and listener. The control-plane,
data-plane, and Execution Artifact service implementations are each registered
exactly once on that server; Agent isolation is provided by two client-side
`grpc.ClientConn` transports, not by duplicate Server listeners or service
implementations. Authentication metadata is required independently on every
RPC and stream on both connections.

The control connection carries only the long-lived `Connect` stream and control
messages. The data connection carries progress/reporting/result unary calls and
Artifact streams. A data connection outage therefore returns through the
existing application recovery boundary and cannot be rerouted over the control
connection. Keepalive and stream limits remain configured at the shared Server
transport; the two client transports still have independent HTTP/2 flow-control
and failure state.
