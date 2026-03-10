# Change: Refactor worker log field semantics

## Why
The repository-wide interface naming cleanup already standardized API JSON, Gin context access, and most server-side structured log keys. The remaining drift is concentrated in worker runtime logs, where several fields still use `camelCase` keys such as `taskId`, `scanId`, `queueLength`, and `retryCount`.

Those keys are not provider-facing labels and are emitted in the application log body that Loki collects from container stdout. Keeping them in `camelCase` while the rest of the project uses semantic dotted fields makes observability inconsistent, complicates search patterns, and weakens the naming standard we already adopted.

## What Changes
- Refactor remaining worker structured log field keys from `camelCase` to semantic dotted names
- Extend interface naming CI enforcement so new worker/server `camelCase` zap keys are rejected while OpenTelemetry-style semantic fields remain allowed
- Add targeted regression tests for the affected worker/server log emitters and CI guardrails

## Impact
- Affected specs: `structured-log-naming`
- Affected code: worker runtime logging in `worker/cmd/worker/main.go`, `worker/internal/server/batch_sender.go`, `worker/internal/workflow/subdomain_discovery/*`, `worker/internal/activity/*`, and `scripts/ci/check-interface-naming.sh`
