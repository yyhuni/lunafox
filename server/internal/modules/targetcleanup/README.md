# targetcleanup

`targetcleanup` owns durable, asynchronous reconciliation after a Target has
been tombstoned. It is an internal Server module: it exposes no HTTP resource,
UI, manual retry, pause, skip, force-complete, or runtime configuration API.

The synchronous Target DELETE command remains in `catalog`. Its sole durable
work is one transaction that writes `target.deleted_at` and creates or confirms
one `target_cleanup_job`. It must not scan, count, cascade, cancel, or delete
Target-owned rows whose cost grows with data volume.

## Reconciliation

One in-process Runner polls due pending Jobs in `next_retry_at,id` order. Every
run starts from the current database truth, not a phase or cursor:

1. confirm the Target tombstone;
2. delete only Target-scoped Schedules, Organization relationships, and
   Target-scoped blacklist policy;
3. cancel one active Scan at a time, then send `task_cancel` after commit on a
   best-effort basis;
4. delete current assets in the fixed order `subdomain`, `host_port_mapping`,
   `website`, `endpoint`, `directory`, `screenshot`, `vulnerability`;
5. re-check every completion condition before marking the Job completed.

There is intentionally no persisted phase, asset cursor, owner, lease,
heartbeat, `SKIP LOCKED`, parallel Worker, or leader-election protocol. A
restart repeats step one; committed work becomes an idempotent no-op and rolled
back work is discovered again.

## Ownership And Operations

This module may not delete `scan`, `scan_task`, `task_progress_log`, or any
Snapshot history. Cancelled and terminal Scan history stays readable until the
independent Scan retention owner reclaims it. Agent delivery failure is an
observable best-effort result and never rolls back cancellation or blocks Job
completion.

`TARGET_CLEANUP_BATCH_SIZE`, `TARGET_CLEANUP_MAX_BATCHES_PER_RUN`, and
`TARGET_CLEANUP_MAX_RUN_DURATION` are startup-only limits. Cleanup transactions
set local PostgreSQL lock and statement timeouts; they do not change session or
database defaults.

Run the heavy release evidence separately from ordinary Go tests:

```bash
make verify-target-cleanup-postgres
make verify-target-cleanup-scale
```

The commands provision isolated PostgreSQL 18 instances and preserve logs in
`dist/target-cleanup-postgres/` and `dist/target-cleanup-scale/`. Release CI
uploads those logs as artifacts.
