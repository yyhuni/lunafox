# Upgrade Release Policy

This module owns Server-side validation shared by update checks and Upgrade
Operation creation. Public Compose deployments use
`infrastructure.ChannelManifestSource`, configured once with a fixed
`stable`/`canary` channel, metadata base URL, and image Registry. It accepts
only the four-field schema-v3 channel record, same-origin HTTPS redirects, a
bounded `manifests/vX.Y.Z*.yaml` path, and matching raw SHA-256 bytes. A failed
refresh never falls back to cached channel data. Validated bytes are persisted
by digest under the shared upgrade state so an Operation retry reloads its
original target after the channel advances.

Development and legacy wiring without the three public release settings keeps
using `application.ManifestLoader` for one Server-configured local path. Both
sources validate the exact YAML digest and `lunafox-<releaseVersion>` identity;
neither accepts image references, commands, URLs, channels, or paths from a
client. The application compares strict semantic versions and offers or creates
an Operation only when the candidate is newer than the running release. It also
parses `upgrade.compatibilityRange` and blocks automatic Operations when the
running version falls outside that range.

`application.ParseMigrationPolicy` strictly parses the Server-owned
`cmd/server/migrations/policy.json`. `domain.EvaluateMigration` is the
eligibility gate: a release without a database migration is allowed in the
current `disposable-development` phase; data-retaining or destructive changes
are rejected with stable error codes and structured diagnostics until an
explicit phase-transition policy is approved. `domain.EnsureSameTarget` binds
every retry to the original manifest identity and digest.

`CreateOperation` performs one final target check and returns the stable
`upgrade_no_update_available` precondition when the validated release version
is already the running version; it never creates or dispatches a no-op
Operation.

The application layer now exposes the protected update-check and Upgrade
Operation use cases. It persists only bounded, non-secret Operation evidence
through the repository and hands the fixed operation identity/digest to an
independent Compose-managed upgrader. HTTP adapters live in `handler/` and `router/`; they
must continue to use the authenticated Server user ID and the canonical
`/v1/system:checkForUpdates`, `/v1/upgradeOperations`, and operation resource
boundaries.

## Execution boundary

The supported upgrade target is a single-node Compose deployment. The MVP does
not promise HA, rolling or zero-downtime upgrades, automatic backups, data
retention rollback, or cross-node recovery. The current migration policy is
still `disposable-development`: `migrate down` is a development/test teardown
only and is never an automatic upgrade recovery action.

The Server hands the Compose-managed upgrader only the operation ID, a fixed action, and
the already validated manifest digest over the private Unix socket under
`.lunafox/upgrade/`. It never accepts or forwards image references, Compose
paths, shell commands, migration versions, or `down` arguments, and the
production Server container does not mount the Docker socket. The upgrader has
no network port, uses the shared state volume and a fixed `/deployment` layout,
and is the only upgrade control-plane process that owns the privileged Compose
argv. Existing Agent execution and log collection containers retain their
separate Docker Socket access.

The upgrader recreates only the fixed resident `agent` service with `--no-deps`.
Remote Agents remain outside the Compose project and continue the existing
`update_required` self-update lifecycle. Server waits for every in-scope Agent
to reconnect, match the target version/digest, report healthy, and become
claim-ready before the Agent portion can succeed.

After service, migration, resident Agent, health, and digest verification pass,
the upgrader atomically installs the target as `/deployment/compose.override.yaml`.
Ordinary Compose lifecycle commands then keep using the confirmed images and
runtime versions. The override contains no database, public address, or secret
configuration. Releases that require a changed Compose structure or new host
resources require a newly downloaded deployment package.

The completion receipt is only a host deployment proof. It is reconciled with
the database Upgrade Operation and host journal by `operationId` and manifest
digest, but it cannot mark an Operation `succeeded` while migration, service,
API, or Agent verification remains incomplete. Migration failure or uncertain
outcome is `needs_recovery`; an Agent validation timeout is `needs_attention`;
only pre-migration failures are retryable without a recovery decision.

## Recovery and operator controls

The protected HTTP surface exposes the durable recovery view alongside the
existing operation resource:

- `GET /v1/upgradeOperations:active` returns the one non-terminal operation;
  an empty active set is a `404` (`upgrade_not_found`) rather than a synthetic
  operation. The frontend treats only that response as an empty view and
  fails closed on other lookup errors.
- `POST /v1/upgradeOperations/{operation}:stop` accepts only
  `{ "confirmed": true }`. It sends a bounded `stop` action to the host daemon
  and remains locked to the upgrade route until a host checkpoint or watchdog
  records a terminal state. Stop is cancellation, never migration rollback;
  migration or post-migration evidence is classified as `needs_recovery`.

`RecoveryJob` reads the current host journal and compares the current stage's
last checkpoint (`stageTimes`) with a bounded watchdog derived from the
manifest maintenance window. A missing checkpoint is allowed to settle during
the handoff grace period; a corrupt checkpoint is classified immediately. A
poll/reconciliation error is emitted through the structured server logger and
does not terminate the background loop.

Operation responses include the running Server's `currentVersion` and a
maximum-32-entry `logs` projection. Entries are fixed lifecycle milestones,
safe diagnostics, bounded cancellation counts, and host-generated progress
events. Progress events are single-line catalog messages with bounded metadata;
they are an observation stream for the current stage, not a lifecycle source
of truth. Raw Compose stdout/stderr, paths, commands, image references,
environment values, tokens, and credentials are never persisted or returned.
The frontend uses the active view as its route-lock authority, renders the
bounded timeline and shared `RawLogViewer` progress stream, and offers stop,
retry, recheck, and terminal exit actions according to the durable state.

The progress stream is deliberately separate from Server/Loki system logs.
During service update or restart the Server and its log API may be unavailable,
so the host journal remains the durable handoff source. Event append and
delivery are best effort and must never block Compose execution or change the
upgrade terminal result. Host command output remains available through the
operator's host-side `logs.sh` workflow when deeper diagnosis is required.
