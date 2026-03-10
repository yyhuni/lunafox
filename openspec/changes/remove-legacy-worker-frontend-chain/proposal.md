# Change: Remove legacy worker frontend chain

## Why
The current `/settings/workers/` runtime surface already uses the agent management contract, while the old frontend worker service/hook chain still targets a non-existent `/workers` HTTP API and keeps unsupported snake_case assumptions. Keeping both chains increases confusion, dead code, and naming drift.

## What Changes
- Remove the legacy frontend worker HTTP service/hook chain that targets `/workers`
- Remove or isolate unsupported worker-only UI artifacts that rely on non-existent `/ws/workers/:id/deploy/`
- Keep the supported `/settings/workers/` runtime surface aligned with the agent management contract
- Document the cleanup scope and migration rationale in project plans

## Impact
- Affected specs: `worker-admin-surface`
- Affected code: `frontend/services/worker.service.ts`, `frontend/hooks/use-workers.ts`, `frontend/types/worker.types.ts`, related tests, and unsupported worker-only UI artifacts
