# targetcleanup/application

`targetcleanup` owns the internal, durable reconciliation of a tombstoned
Target. It has no HTTP handler, router, API resource, UI, manual retry, pause,
skip, force-complete, or hot-reload configuration surface.

The synchronous Target DELETE transaction remains in `catalog`: it atomically
writes `target.deleted_at` and one unique `target_cleanup_job`. This module
starts only after that commit and may never make the Target available again.

## Reconciliation Contract

Every run begins by locking and confirming the Target tombstone, then repeats
the same database-truth sequence:

1. delete direct Target-scoped Scheduled Scans through their existing
   serialization/cascade path;
2. delete `organization_target` rows and only the Target-scoped blacklist
   policy;
3. cancel one active Scan at a time, then best-effort notify its assigned
   Agent after the cancellation transaction commits;
4. delete current assets in this fixed order: `subdomain`,
   `host_port_mapping`, `website`, `endpoint`, `directory`, `screenshot`,
   `vulnerability`;
5. re-lock the tombstone and mark the Job completed only when all completion
   conditions are absent.

There is deliberately no persisted phase, asset cursor, lease, `SKIP LOCKED`,
parallel Worker, or leader-election protocol. A restart starts from step one;
already committed work is an idempotent empty operation, and rolled-back work
is naturally rediscovered.

The single in-process Runner polls durable due Jobs ordered by
`next_retry_at,id`. Its in-memory wake signal is lossy optimization only.
`TARGET_CLEANUP_BATCH_SIZE`, `TARGET_CLEANUP_MAX_BATCHES_PER_RUN`, and
`TARGET_CLEANUP_MAX_RUN_DURATION` bound current-asset work and are startup-only
configuration.

## Ownership Boundary

This module may remove only Target control-plane rows and the seven current
asset tables. It never deletes `scan`, `scan_task`, `task_progress_log`, or any
Snapshot history. Cancelled and terminal Scan history remains readable until
the independent Scan history retention owner reclaims it.

Agent `task_cancel` is process-local best effort. Delivery failure is counted
and logged without rolling back database cancellation, creating an outbox, or
blocking Job completion.
