-- DESTRUCTIVE TEST TEARDOWN ONLY.
-- Production recovery uses an approved forward fix. Dropping this column
-- discards current host-observation evidence and is never a rollback.
ALTER TABLE upgrade_operation
    DROP COLUMN IF EXISTS host_activity;
