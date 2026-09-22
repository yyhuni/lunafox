-- Scheduled Scan rules retain their existing UTC cursor while gaining an
-- explicit IANA wall-clock interpretation for all future calculations.
ALTER TABLE scheduled_scan
    ADD COLUMN IF NOT EXISTS time_zone VARCHAR(100);

UPDATE scheduled_scan
SET time_zone = 'UTC'
WHERE time_zone IS NULL;

ALTER TABLE scheduled_scan
    ALTER COLUMN time_zone SET NOT NULL;
