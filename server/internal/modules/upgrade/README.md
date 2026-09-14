# Upgrade Release Policy

This module owns Server-side validation shared by update checks and Upgrade
Operation creation. `application.ManifestLoader` reads one Server-configured
manifest path and revalidates the exact YAML digest and `lunafox-<releaseVersion>`
identity. It never accepts image references, commands, or paths from a client,
and it has no fallback manifest.

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
independent host upgrader. HTTP adapters live in `handler/` and `router/`; they
must continue to use the authenticated Server user ID and the canonical
`/v1/system:checkForUpdates`, `/v1/upgradeOperations`, and operation resource
boundaries.

## Execution boundary

The supported upgrade target is a single-node Compose deployment. The MVP does
not promise HA, rolling or zero-downtime upgrades, automatic backups, data
retention rollback, or cross-node recovery. The current migration policy is
still `disposable-development`: `migrate down` is a development/test teardown
only and is never an automatic upgrade recovery action.

The Server hands the host upgrader only the operation ID, a fixed action, and
the already validated manifest digest over the private Unix socket under
`.lunafox/upgrade/`. It never accepts or forwards image references, Compose
paths, shell commands, migration versions, or `down` arguments, and the
production Server container does not mount the Docker socket. The host process
re-reads the fixed deployment manifest and owns the privileged Compose argv and
journal lifecycle.

The host upgrader does not replace Agent containers. After host services return,
the Server continues the existing `update_required` self-update lifecycle and
waits for each in-scope Agent to reconnect, match the target version/digest,
report healthy, and become claim-ready before the Agent portion can succeed.

The completion receipt is only a host deployment proof. It is reconciled with
the database Upgrade Operation and host journal by `operationId` and manifest
digest, but it cannot mark an Operation `succeeded` while migration, service,
API, or Agent verification remains incomplete. Migration failure or uncertain
outcome is `needs_recovery`; an Agent validation timeout is `needs_attention`;
only pre-migration failures are retryable without a recovery decision.
