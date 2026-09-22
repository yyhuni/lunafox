-- DESTRUCTIVE TEST TEARDOWN ONLY.
-- Production recovery uses an approved forward fix. Dropping this column
-- removes each Schedule's wall-clock interpretation and is not a rollback.
ALTER TABLE scheduled_scan
    DROP COLUMN IF EXISTS time_zone;
