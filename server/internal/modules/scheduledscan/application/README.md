# scheduledscan/application

Scheduled scans persist the canonical `scanWorkflow` resource reference, the
schedule's complete configuration, a required `inputSource` (`scanSnapshot` or
`targetInventory`), an optional canonical `agent: agents/{id}` selection, and
one strict five-field Cron expression interpreted in UTC. They
do not persist a workflow revision, release digest, topology,
Profile, Package, execution plan, creator identity, or Server-local time-zone
fallback.

`inputSource` is independent of `scanMode: target|organization`, Workflow, and
Agent assignment. Create requires it explicitly. Update validates and writes a
replacement only when its update mask names `inputSource`; an update that omits
the field retains the persisted value. No service, repository, or scheduler
path infers `scanSnapshot` for missing data.

## Time Rule And Cursor Contract

`ScheduleCalculator` is the only application boundary that interprets Cron in
`time.UTC`. It accepts exactly minute, hour, day-of-month, month, and
day-of-week fields. Seconds, descriptors such as `@daily` or `@every`, and
embedded `TZ`/`CRON_TZ` are rejected.

`nextRunTime` is a persisted UTC cursor, not a read-time calculation. Enabled
Create and enabled `cronExpression` Update calculate the first match
strictly after one transaction reference time. Non-time Updates leave the
cursor unchanged. Disabled schedules always persist a null cursor; re-enable
starts from a new strictly future match and never catches up the disabled
interval.

The configuration uses the same strict Workflow Step envelope as normal Scan
creation: every selected Step must declare boolean `enabled`; enabled Steps
carry complete `engineConfig`, while disabled Steps carry only `enabled: false`.
Create and Update reject a schedule with no enabled Step. A disabled Step draft
may remain in the frontend session for read-only preview, but is never sent or
persisted; this development baseline does not accept the retired configuration
shape.

For an enabled Step, a current Engine Definition may declare one individual
`requiredEnabled` config section. That section must remain enabled with its
complete parameters; the outer Step remains independently controllable. The
retired Subdomain Discovery `dns` section is unknown to the current definition:
the application must not translate it into either resolver selection or apply a
fallback. A saved development schedule using that retired shape fails at normal
Scan creation before a Task or Scan History record exists. The operator-facing
diagnostic remains the Server Error log message `Scheduled scan handoff did not
complete`; this module adds no retry, notification, automatic disablement, or
new product failure surface.

Create and Update validate the selected workflow through the workflow
repository. At occurrence attempt start, the scheduler copies the latest
committed Schedule inputs, including the persisted input source, while holding
the Schedule serialization lock. The one attempt then delegates that frozen
input to normal Scan creation. Normal
Scan creation resolves the then-current Workflow, Engine, Target or
Organization, configuration resource, and selected Agent, and freezes every
successfully created Scan's topology, exact Engine Packages, Tasks, and plans.
Workflow updates do not inspect or block scheduled-scan references.

Create always validates a complete canonical configuration. Update validates a
replacement configuration only when its `updateMask` includes `configuration`;
an `isEnabled`-only update changes management metadata and does not revalidate
the stored Workflow, current Engine Definition, or configuration resources.
Re-enable still validates the stored time rule and calculates a fresh cursor.

After Workflow resolution and configuration canonicalization, Create and every
configuration-bearing Update call the Scan application-owned Workflow
config-resource validator before `store.Create` or `store.Update`. Validation
visits only enabled Steps and enabled configSections and requires each selected
wordlist to match the current Catalog record and complete regular readable file.
Confirmed unavailability, validation-dependency failure, and missing internal
assembly remain distinct typed causes and prevent persistence. An
`isEnabled`-only Update, including disabled-to-enabled, performs no Workflow,
Engine Definition, or config-resource lookup.

Successful save-time validation is not a reservation. Every occurrence attempt
revalidates current resources through normal Scan creation. A failure must not
change `isEnabled`, configuration, references, or other Schedule management
state, choose a replacement, or enqueue a repair. A later independent
occurrence reads current resources again and may succeed after repair.

Target-scoped Schedule persistence locks and revalidates an active Target in
its final write transaction. After a Target tombstone commits, ordinary
Schedule List/Get and runtime due/candidate queries hide the retained direct
Schedule immediately. The Target cleanup owner later hard-deletes that Schedule
through the same Schedule serialization path and the owned occurrence cascade.
Organization-scoped Schedules remain attached only to their Organization: an
attempt freezes that moment's active Target IDs in memory, and each ordinary
Scan create still performs its own final active-Target fence.

## Management Overview

The protected Scheduled Scan management overview reads only the complete
ordinary List-visible set of current Schedule rows. It accepts no display-zone
input, reads one UTC `asOfTime`, and returns enabled and paused task totals,
task counts whose persisted `nextRunTime` falls within the UTC calendar day or
the rolling UTC 24-hour window, and at most five upcoming tasks. It neither
parses Cron nor reads occurrence or Scan lifecycle data.

The application derives the calendar day from consecutive UTC midnights and
passes UTC half-open ranges to the repository, which must provide one
consistent visible-set snapshot. A retired `timeZone` query parameter is
rejected at the HTTP boundary; there is no Server, database, or browser-zone
fallback.

## Management Batch Status Contract

`POST /v1/scheduledScans:batchUpdate` atomically assigns the enabled state of
between one and one hundred Scheduled Scans. Its body uses one canonical
resource name and an exact `isEnabled` update mask per item:

```json
{
  "requests": [
    {
      "name": "scheduledScans/12",
      "isEnabled": false,
      "updateMask": "isEnabled"
    }
  ]
}
```

Names must be unique canonical `scheduledScans/{id}` values. The Server
validates the complete request before persistence, locks target and Schedule
rows in deterministic order, and returns `{ "updatedCount": N }` only after
the transaction commits. A malformed, missing, unavailable, calculation,
occurrence-cleanup, or persistence failure rolls back every requested row.
Disabling clears `nextRunTime` and deletes only unattempted occurrences;
enabling calculates a fresh cursor strictly after the batch reference time.

## One-Shot Runtime Ownership

The one supported main Server owns one always-on scheduler controller with one
goroutine and at most one occurrence-to-Scan-Create handoff in flight. The
supported deployment stops the old Server before starting the new Server. This
module has no leader election, Redis lock, transferable lease, or multi-Server
claim protocol; any overlapping or multi-replica topology requires a separate
OpenSpec change first.

The controller materializes at most the latest unmaterialized match missed
during downtime. An occurrence is uniquely identified by Schedule and UTC
`scheduledFor`. `StartAttempt` commits `attemptedAt`, increments `runCount`,
sets `lastRunTime` to the same UTC instant, and freezes the latest committed
inputs before normal Scan creation starts. One Organization occurrence still
increments the aggregate once, regardless of how many child Scans are created.

Each occurrence receives one attempt with a fixed five-minute handoff deadline.
There is no automatic retry after ordinary failure, partial success, timeout,
cancellation, process interruption, or restart, and no compensation for
missing Organization batch items. Already committed ordinary Scans remain
valid. Their queued, running, completed, failed, and canceled states are owned
only by the Scan module and never update scheduling state.

Each Schedule retains three durable, non-negative aggregates: `runCount` counts
committed attempt starts, `successfulHandoffCount` counts completed normal Scan
creation handoffs, and `failedHandoffCount` counts observed non-complete
handoffs. A process interruption after `attemptedAt` and before outcome
writeback remains in `runCount` only. Neither outcome aggregate represents the
later lifecycle of an ordinary Scan or Scan Task.

## Lifecycle Serialization

Update, disable, Delete, materialization, and attempt start serialize on the
owning Schedule row:

- Update-first means the later attempt freezes the updated values;
  attempt-first keeps using its already frozen values.
- Disable-first clears `nextRunTime` and hard-deletes every unattempted
  occurrence in the same transaction. Attempt-first continues and is not
  canceled by disable.
- Delete hard-deletes the Schedule and cascades all private occurrences. An
  already-started attempt continues from memory, treats outcome writeback to a
  missing row as expected, and never recreates scheduling data.
- Disable and Delete never cancel, delete, or mutate an ordinary Scan.

## Diagnostics, Retention, And Exclusions

Complete handoff records `dispatchedAt`. Observed non-complete handoff records
one closed `failureKind` and a bounded safe UTF-8 `failureMessage`; raw errors,
configuration, paths, secrets, stacks, and complete batch payloads are never
persisted. Outcome settlement and the exactly-one successful or failed Schedule
aggregate increment commit in one repository transaction. A crash after
`attemptedAt` but before outcome writeback intentionally remains attempted with
an unknown result and is not replayed.

A separate managed retention job starts asynchronously with the Server and then
runs hourly. It retains attempted occurrences for at least seven days based
only on UTC `attemptedAt`, deletes at most 1,000 rows per transaction and
100,000 rows per five-minute run, and performs no immediate or backoff retry.
Schedule Delete bypasses retention through its ownership cascade.

The occurrence ledger is internal. This phase adds no occurrence/history API or
UI, Scheduled Scan Run Now, Scan provenance, scheduler setting, runtime tuning,
metric, built-in alert, dashboard, health-probe dependency, retry queue,
Undelete, or Scan lifecycle tracking.
