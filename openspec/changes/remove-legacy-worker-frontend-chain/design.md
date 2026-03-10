# Design: Remove legacy worker frontend chain

## Context
The worker B-class audit confirms that the currently accessible workers page is backed by the agent management chain:

`WorkersPage -> AgentList -> useAgents -> agentService -> /api/admin/agents`

At the same time, the repository still contains an older worker frontend chain that:
- calls `/workers`
- uses hard-coded snake_case request fields such as `page_size`, `ip_address`, and `ssh_port`
- has no matching Go backend HTTP route
- is only referenced by itself and its tests

A separate placeholder UI chain also remains for deploy-terminal behavior, but it depends on `/ws/workers/:id/deploy/`, which is also not implemented on the backend.

## Decision
Adopt a cleanup-first strategy:
1. Treat the old worker HTTP service/hook chain as unsupported legacy code and remove it
2. Treat deploy-terminal worker-only artifacts as unsupported placeholders and remove them unless they are first migrated onto a real agent-backed contract
3. Keep the supported workers settings page on the agent management path only

## Scope
### In scope
- Remove legacy worker frontend service/hook/test files
- Remove unsupported worker-only placeholder UI and types when they are not part of the agent-backed runtime surface
- Update references, exports, mocks, and demos that still point to the removed chain
- Add targeted regression checks proving `/settings/workers/` remains agent-backed

### Out of scope
- Adding a new `/workers` backend HTTP API
- Adding a new `/ws/workers/:id/deploy/` backend WebSocket API
- Reintroducing a separate worker domain beside the current agent domain

## Verification strategy
- Search the repository to confirm there are no remaining runtime imports of the removed worker chain
- Run targeted frontend tests for the workers/agents settings surface
- Validate the OpenSpec change strictly before implementation handoff
