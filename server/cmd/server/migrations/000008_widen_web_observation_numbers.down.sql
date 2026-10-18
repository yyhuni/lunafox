-- DESTRUCTIVE TEST TEARDOWN ONLY. Narrowing can fail when BIGINT observations exceed INTEGER.
ALTER TABLE website
    ALTER COLUMN status_code TYPE INTEGER USING status_code::INTEGER,
    ALTER COLUMN content_length TYPE INTEGER USING content_length::INTEGER;

ALTER TABLE endpoint
    ALTER COLUMN status_code TYPE INTEGER USING status_code::INTEGER,
    ALTER COLUMN content_length TYPE INTEGER USING content_length::INTEGER;

ALTER TABLE website_snapshot
    ALTER COLUMN status_code TYPE INTEGER USING status_code::INTEGER,
    ALTER COLUMN content_length TYPE INTEGER USING content_length::INTEGER;

ALTER TABLE website
    DROP COLUMN IF EXISTS response_headers_truncated;
ALTER TABLE website_snapshot
    DROP COLUMN IF EXISTS response_body_truncated,
    DROP COLUMN IF EXISTS response_headers_truncated;

ALTER TABLE endpoint_snapshot
    ALTER COLUMN status_code TYPE INTEGER USING status_code::INTEGER,
    ALTER COLUMN content_length TYPE INTEGER USING content_length::INTEGER;
