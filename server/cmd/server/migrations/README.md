# Migration Baseline Policy

## Current Phase

The repository is in the `disposable-development` phase recorded by
`policy.json`. There is no supported persisted deployment, preserve-data
upgrade, or in-flight task recovery contract. The only migration files are the
squashed `000001_init_schema.up.sql` and `000001_init_schema.down.sql` pair.

Before the baseline freezes, maintainers may update and squash `000001` so a
new empty database directly receives the current schema. Historical cutover SQL
and compatibility migrations must not accumulate in this development baseline.

## Wordlist Resource Identity Development Cutover

The squashed baseline stores the immutable upload basename in
`wordlist.file_name`; the canonical API, scan, plan, Server, Agent, and Worker
identity is derived from the stable row ID as `wordlists/{id}`. A retained
development snapshot created before this hard cut may still have
`wordlist.name` and filename values in persisted Scan configuration or saved
execution plans. Do not start a mixed old/new stack against that snapshot.

This one approved development-data transition is intentionally an explicit
command rather than a second numbered SQL migration, so the fresh-install
baseline remains a single `000001` pair:

```bash
server wordlist-resource-migrate
```

Run it only after stopping Server, Agent, Worker, and Frontend processes, and
after taking a recoverable database plus wordlist-directory snapshot. The
command performs its full preflight inside one database transaction before it
changes a column or persisted reference. It verifies every Catalog row's ID,
unique `file_name`, physical file path, bytes, size, line count, and SHA-256
hash; then it checks `scan.configuration`, `scheduled_scan.configuration`, and
`scan_task.resolved_execution_plan`. Filename references must map uniquely to
an existing Catalog row. Missing files, duplicate or malformed names, changed
bytes, unresolved references, malformed plans, and descriptor metadata drift
all fail closed before mutation.

On success the command prints a JSON audit report, renames the legacy upload
column only when needed, recreates the `file_name` unique index, and replaces
known filename references with `wordlists/{id}` in the same transaction. It
never moves, renames, replaces, or deletes a physical wordlist file. Running it
again after success is a no-op except for a fresh audit report. A failed
transition is recovered by fixing the reported snapshot inconsistency and
rerunning, or by restoring the pre-command database/file snapshot; rollback
must never re-enable filename runtime resolution.

## Agent Operational Data Baseline

The current `000001` directly creates the authoritative Agent operational data
shape. `agent.registration_token_id` is a required foreign key and
`registration_token.ever_attributed_at` permanently records whether a token was
ever used. `agent_runtime_status.observed_source_ip` and
`observed_ip_generation` are Server-owned connection observations. The
`agent_location` and singleton `server_location_snapshot` tables store only
complete last-successful GeoIP snapshots; freshness and forced expiry do not
delete or rewrite successful provenance.

This shape has no legacy-column read, data backfill, or numbered forward
migration. Rehearse and release it Server-first: stop the old stack, create a
new empty PostgreSQL database from the updated `000001`, start the matching
Server, verify its Agent summary, map, registration-token, Agent detail, and
Scan detail boundaries, and only then switch the matching Frontend. The new
Frontend must not run against an old Server because it intentionally has no
bounded-list or mock-coordinate fallback.

Rollback restores the matching Frontend and Server source, then recreates an
empty database from that source revision's baseline. Applying `down` is only a
destructive test/development teardown and does not preserve or recover Agent,
token, location, Scan, or other business data.

## Nuclei POC Source Sync Development Cutover

The Nuclei POC tables in `000001` are a disposable-development hard cut. The
old repository configuration and local-preview boundary have no supported
runtime fallback, dual write, or data backfill. Stop the matching Server and
Frontend processes, discard any development database that still contains the
retired repository model, and create a fresh database from the updated
baseline before starting the new stack.

The new source, sync-task, candidate-import, catalog, and request-tombstone
tables persist only the public source identity, validated YAML metadata/content,
task progress, and the independent `isEnabled` overlay. If an environment must
retain old Nuclei data, release is blocked and a separate preserve-data change
must define its own migration and rollback semantics; do not add a compatibility
read or silently delete that data.

## Scan Workflow Aggregate

`000001` includes the `scan_workflow` runtime management aggregate. Scalar
metadata, ownership, request idempotency, row version, timestamps, and release
digest are relational columns; ordered Stage/Step pure orchestration is one
JSONB `stages` value. The table is the only runtime workflow source. Release
files are read only during bootstrap to synchronize built-ins, and created
Scans persist their own immutable plans rather than referencing a workflow
revision.

The baseline checks the disjoint namespaces: users use canonical
`wf-<lowercase-UUID>` IDs, built-ins use the reserved non-`wf-` namespace, and
only users can carry a create request ID. In this disposable phase the table is
part of the squashed fresh-install schema; a future preserve-data phase must
define its own forward migration and upgrade semantics before changing it.

## Scheduled Scan Execution Baseline

`000001` gives every `scheduled_scan` a strict five-field `cron_expression` and one authoritative UTC
`next_run_time`. A database check requires enabled rows to have a cursor and
disabled rows to have none. `run_count` is non-negative and means committed
occurrence attempt starts; `last_run_time` is the UTC time of the most recent
such start. These aggregates do not represent created, successful, completed,
or failed Scans.

The private `scheduled_scan_occurrence` table owns the one-shot dispatch ledger.
Schedule plus UTC `scheduled_for` is unique. `attempted_at` is written before
normal Scan Create and makes the occurrence non-retryable. Complete handoff may
set `dispatched_at`; an observed non-complete result uses bounded
`failure_kind` and `failure_message`; an attempted row with no outcome fields is
an intentional interrupted/unknown result. The table has due/candidate and
`(attempted_at, id)` retention indexes, and Schedule Delete hard-cascades its
rows.

Update, disable, Delete, materialization, and attempt start use Schedule-row
transactions in the repository. Disable clears the cursor and hard-deletes
unattempted occurrences; an attempt that committed first continues from frozen
current inputs. Delete removes only Schedule-owned rows and never removes an
ordinary Scan. The runtime retains attempted occurrences for at least seven
days by UTC `attempted_at`, then an independent hourly job deletes bounded
1,000-row batches, at most 100 batches per run.

This schema assumes the supported stop-before-start deployment with one main
Server and one in-process scheduler controller. It deliberately contains no
claim owner, lease, leader-election, Redis-lock, or `SKIP LOCKED` work-queue
state. It also adds no Schedule/occurrence provenance or uniqueness to `scan`,
`scan_task`, or saved plans; no user identity; and no occurrence HTTP/history,
Run Now, scheduler setting, metric, alert, dashboard, or health schema.

Because the development baseline is disposable, this is a hard fresh-install
cutover. Existing Scheduled Scan data is not backfilled with a guessed time
zone and no forward compatibility migration is added. Rebuild the database from
empty state and recreate required schedules under the UTC-only Cron contract.

## Scan Execution Input Source Hard Cut

The mutable `000001` baseline requires `scan.input_source` and
`scheduled_scan.input_source` with no database default. Both are non-null and
accept only `scan_snapshot` or `target_inventory`. Matching Server and Frontend
code must supply and strictly parse the value on every public and internal
write path; historical rows are not backfilled, converted, or compatibility
read.

Deploy this change only by stopping incompatible components and recreating an
empty development database from the matching revision's `000001`. Rollback
means restoring the matching source revision and recreating another empty
database from that revision's baseline. Do not add a forward migration or use
the down migration to preserve Scan or Scheduled Scan data across this schema
boundary.

## Notification Hard Cut Runbook

The notification module is also a disposable-development hard cut. Its outbox,
inbox, destination, delivery, and attempt tables replace the old
singleton notification and global read-state shapes. No notification history,
read state, category-specific inbox filter, destination credential, or enabled
setting is migrated, dual-written, copied, or read through a compatibility
path. A fresh baseline gives every user one enabled inbox-delivery setting;
notification kinds and categories remain fixed taxonomy rather than per-user
subscriptions.

Use this exact rollout order:

1. Stop the old Server, worker, and Frontend stack together.
2. Create an empty PostgreSQL database and apply the matching revision's
   `000001` baseline. Do not point the new Server at the old database.
3. Start the matching Server and its supervised notification outbox, delivery,
   and retention workers. Verify migrations, authenticated inbox routes,
   destination validation, worker recovery, provider acceptance, retry caps,
   redacted diagnostics, and retention before exposing the Frontend.
4. Deploy the matching Frontend only after the Server verification succeeds.
5. Re-enter Discord, WeCom, and Feishu credentials, select each destination's
   exact kind subscriptions, and explicitly enable each destination. Fresh
   destinations are intentionally disabled with an empty allowlist.

Rollback means restoring the matching old Server, worker, and Frontend source
revision and creating a new empty database from that old revision's baseline.
It does not mean applying this revision's down migration to preserve data or
recover notification history, read state, credentials, or in-flight delivery.

## Development Rollback

Development rollback means restoring code through Git or branch history and
running another empty-state fresh install. It does not mean applying the down
migration to preserve or recover business data. The current down migration is a
destructive development/test teardown, and it provides no database rollback or
in-flight task recovery promise.

`MigrateToVersion` with a lower target also invokes a destructive down migration
and is limited to development/test use; it is not a production rollback API.

The default installer preclean removes LunaFox data volumes and configuration.
`uninstall.sh --keep-data` only preserves files during uninstall; it does not
turn a later default install into a supported upgrade or rollback path.

## Explicit Workflow Step Enablement Cutover

The Workflow Step enablement contract is a hard cut in this disposable phase.
Existing `scan.configuration` and Scheduled Scan rows that lack explicit Step
`enabled` values are not backfilled, quarantined, converted at read time, or
interpreted from current Workflow/Engine state. Before using the new code,
rebuild the development database from an empty baseline, discard historical
Scan and Scheduled Scan rows, and recreate required schedules from a fresh
all-disabled Profile draft through the strict Create request after explicitly
enabling at least one Step.

## Scan History Partition Baseline

The disposable baseline partitions only scan history: the seven snapshot tables
and `task_progress_log` use the same `RANGE(scan_id)` 10,000-ID half-open
intervals and deterministic child names. The initial baseline creates `[0,
10000)` and `[10000, 20000)`; server lifecycle provisioning creates the
current and next ranges under a PostgreSQL advisory lock. There is deliberately
no `DEFAULT` partition, so a missing range rejects history writes rather than
silently storing data outside the retention layout.

`scan`, `scan_task`, and current asset tables remain ordinary tables. `scan_task`
retains primary key `id` and has `UNIQUE (scan_id, id)` solely to support the
`task_progress_log (scan_id, task_id)` foreign key. The backend's fixed
retention lifecycle keeps terminal histories for at least 30 days based on
`scan.stopped_at`; whole eligible ranges are soft-hidden, then their eight
history child tables are detached/dropped before bounded task deletion and the
final scan deletion. This behavior has no frontend retention setting.

## Target Cleanup Baseline

`000001` includes the internal `target_cleanup_job` queue and the indexes used
to reconcile a tombstoned Target without putting relationship, Schedule, Scan,
or current-asset work in the DELETE transaction. The queue has one row per
Target and records only durable retry diagnostics; it intentionally has no
phase, asset cursor, lease, multi-worker claim, or management API state.

Target identity uniqueness applies only to active `(type, name)` rows. A
tombstoned Target may therefore be recreated immediately with a new ID while
the old cleanup job remains strictly scoped to its original `target_id`.
Current-asset cleanup uses bounded `(target_id, id)` candidate queries. It
does not delete Scan, Task, task-progress, or Snapshot history. The existing
`SCAN_HISTORY_RETENTION_MODE=enforce` default remains unchanged and continues
to be the sole physical history reclaimer.

This is another empty-database cutover. Stop the development stack, discard
the old development database, and let the updated `000001` create a new empty
database before starting the matching Server. Do not apply an invented
incremental migration or backfill. If any deployed data must be retained,
stop and create a separate approved baseline-freeze and forward-migration
change before proceeding.

## Freeze Trigger

The migration baseline must freeze before whichever happens first:

- the first public stable release;
- the first non-disposable deployment that must retain data.

The current release-policy gate intentionally rejects both trigger conditions.
Reaching either trigger requires a separate approved change that updates the
policy and its gate together, freezes all published migration bytes, and moves
later schema changes to new numbered forward migrations. That change must also
define and verify upgrade compatibility, backup, recovery, and rollback before
the stable release or data-retaining deployment proceeds.

Do not flip `stableReleaseAllowed` or `dataRetainingDeploymentAllowed` alone.
The current checker accepts only the complete disposable-development policy, so
any phase transition must deliberately replace its assumptions and tests.

## Verification

```bash
(cd server && go test ./cmd/server)
node scripts/ci/check-migration-baseline-policy.mjs
node scripts/ci/check-migration-baseline-policy-selftest.mjs
make verify-release-contract
```
