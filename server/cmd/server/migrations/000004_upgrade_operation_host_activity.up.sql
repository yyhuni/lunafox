-- Host activity is optional current observation data. NULL preserves every
-- pre-observability Operation as an explicit legacy/no-current-activity row.
ALTER TABLE upgrade_operation
    ADD COLUMN IF NOT EXISTS host_activity JSONB CHECK (
        host_activity IS NULL OR jsonb_typeof(host_activity) = 'object'
    );
