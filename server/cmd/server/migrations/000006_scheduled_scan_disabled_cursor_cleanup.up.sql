-- Retained databases written before disable transitions cleared the cursor
-- (pre 2026-08-04 repository behavior, on baselines without the
-- cursor-presence check) can still hold a stale next_run_time on disabled
-- schedules; the management UI projects it as a past "next execution" time.
-- Normalize retained rows to the documented invariant: a disabled schedule
-- has no cursor. The repository owns this invariant on every disable
-- transition since the fix; this migration reconciles only pre-existing rows.
UPDATE scheduled_scan
SET next_run_time = NULL
WHERE is_enabled = FALSE AND next_run_time IS NOT NULL;
