-- Scheduled scan handoff gains bounded retry state on the occurrence ledger
-- and a Schedule-level last-handoff-failure summary written by settlement.
ALTER TABLE scheduled_scan_occurrence
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_failure_cause VARCHAR(32);

ALTER TABLE scheduled_scan_occurrence
    ADD CONSTRAINT scheduled_scan_occurrence_retry_requires_attempt CHECK (
        next_retry_at IS NULL OR attempted_at IS NOT NULL
    );

ALTER TABLE scheduled_scan
    ADD COLUMN IF NOT EXISTS last_handoff_failure_cause VARCHAR(32),
    ADD COLUMN IF NOT EXISTS last_handoff_failure_time TIMESTAMPTZ;
