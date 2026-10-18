-- Widen Website and Endpoint observation numbers so unusual HTTPX values are
-- retained without overflow or semantic-range rejection.
ALTER TABLE website
    ALTER COLUMN status_code TYPE BIGINT,
    ALTER COLUMN content_length TYPE BIGINT;
ALTER TABLE website
    ADD COLUMN IF NOT EXISTS response_headers_truncated BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE endpoint
    ALTER COLUMN status_code TYPE BIGINT,
    ALTER COLUMN content_length TYPE BIGINT;

ALTER TABLE website_snapshot
    ALTER COLUMN status_code TYPE BIGINT,
    ALTER COLUMN content_length TYPE BIGINT;
ALTER TABLE website_snapshot
    ADD COLUMN IF NOT EXISTS response_body_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS response_headers_truncated BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE endpoint_snapshot
    ALTER COLUMN status_code TYPE BIGINT,
    ALTER COLUMN content_length TYPE BIGINT;
