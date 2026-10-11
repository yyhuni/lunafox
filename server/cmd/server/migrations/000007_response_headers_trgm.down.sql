-- DESTRUCTIVE TEST TEARDOWN ONLY.
-- Production recovery uses an approved forward fix, not index removal.
DROP INDEX IF EXISTS idx_endpoint_response_headers_trgm;
DROP INDEX IF EXISTS idx_website_response_headers_trgm;
