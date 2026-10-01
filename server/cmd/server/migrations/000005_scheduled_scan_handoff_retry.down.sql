-- DESTRUCTIVE TEST TEARDOWN ONLY.
-- Production recovery uses an approved forward fix. Dropping these columns
-- discards retry state and the last-handoff-failure summary and is not a
-- rollback.
ALTER TABLE scheduled_scan_occurrence
    DROP CONSTRAINT IF EXISTS scheduled_scan_occurrence_retry_requires_attempt;

ALTER TABLE scheduled_scan_occurrence
    DROP COLUMN IF EXISTS last_failure_cause,
    DROP COLUMN IF EXISTS next_retry_at,
    DROP COLUMN IF EXISTS retry_count;

ALTER TABLE scheduled_scan
    DROP COLUMN IF EXISTS last_handoff_failure_time,
    DROP COLUMN IF EXISTS last_handoff_failure_cause;
