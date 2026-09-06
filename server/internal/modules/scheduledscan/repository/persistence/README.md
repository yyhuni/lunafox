`scheduledscan` keeps GORM persistence models and all SQL/GORM operations behind
the repository package. Application callers receive copied records and frozen
dispatch inputs; they never retain or mutate persistence models.

The repository persists one strict five-field `cron_expression` interpreted in
UTC and the authoritative UTC `next_run_time` cursor.
Enabled rows require a non-null cursor and disabled rows require a null cursor.
API reads project this stored value directly. Time-rule validation and cursor
calculation use the application-owned `ScheduleCalculator`; persistence must not
parse Cron independently or fall back to the database or Server time zone.

`scheduled_scan_occurrence` is a private ledger. `(scheduled_scan_id,
scheduled_for)` is unique, `attempted_at` is the immutable one-shot boundary,
`dispatched_at` means complete normal Scan handoff, and bounded
`failure_kind`/`failure_message` record only an observed non-complete result. An
attempted row with no outcome fields is the valid interrupted/unknown state.
There is no creator identity, Scan foreign key, provenance field, retry state,
claim owner, lease, or user-facing history model.

`scheduled_scan` also retains durable non-negative `run_count`,
`successful_handoff_count`, and `failed_handoff_count` aggregates. The latter
two describe only the Schedule-to-normal-Scan creation handoff, never the
subsequent Scan or Scan Task lifecycle.

Purpose-specific transactions own the concurrency contract:

- Create and enabled time-rule Update calculate and persist the next strictly
  future cursor from one transaction reference time.
- Materialization locks and rechecks one Schedule, inserts only the latest due
  occurrence, and advances the cursor atomically.
- `StartAttempt` locks and rechecks the Schedule and occurrence, copies current
  committed inputs, writes `attempted_at`, increments `run_count` once, and sets
  `last_run_time` to that same UTC timestamp in one transaction.
- `RecordOutcome` conditionally settles one attempted occurrence and increments
  exactly one owning Schedule handoff aggregate in the same transaction. A
  repeated or missing occurrence leaves both aggregates unchanged; failure to
  update the owner rolls back the occurrence settlement.
- Update-first is visible to a later attempt; attempt-first continues with its
  frozen copy.
- Disable clears the cursor and hard-deletes all unattempted occurrences in the
  same transaction. Delete hard-deletes the Schedule and cascades every owned
  occurrence. Neither operation joins to or mutates ordinary Scan tables.
- Outcome writeback after concurrent Schedule Delete is an expected zero-row
  update and must not recreate the owner.

Repository row locks are consistency locks for the supported single-Server,
single-controller topology, not a multi-instance claim protocol. Do not add
`SKIP LOCKED`, Redis locks, leader election, or transferable leases without a
separate approved topology change.

The Scheduled Scan management overview runs its visible-set aggregate and
five-row upcoming projection in one transaction. PostgreSQL uses a read-only
repeatable-read snapshot; test SQLite uses the same transaction boundary
without PostgreSQL-specific SQL. Both reads reuse the ordinary active Target
visibility predicate, count only persisted `next_run_time` values, and never
read Cron or occurrence tables.

Retention selects only non-null `attempted_at` at or before the UTC seven-day
cutoff, orders by `(attempted_at, id)`, and hard-deletes at most 1,000 rows per
independent transaction. It never uses `scheduled_for`, `created_at`,
`dispatched_at`, or Scan lifecycle timestamps and never joins a Scan table.

This directory remains the stable landing point required by the repository
boundary checks if persistence models are split into dedicated files later.
