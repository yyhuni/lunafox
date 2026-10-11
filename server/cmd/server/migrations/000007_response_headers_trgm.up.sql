-- Response-header contains search matches the stored raw header blob.
-- gin_trgm_ops lets a selective ILIKE use an index instead of scanning the
-- asset table. The parser rejects two-character contains patterns because
-- they extract no trigram.
CREATE INDEX IF NOT EXISTS idx_website_response_headers_trgm ON website USING GIN (response_headers gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_endpoint_response_headers_trgm ON endpoint USING GIN (response_headers gin_trgm_ops);
