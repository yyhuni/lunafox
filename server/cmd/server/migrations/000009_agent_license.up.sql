CREATE TABLE agent_license (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton = TRUE),
    signed_key TEXT NOT NULL CHECK (octet_length(signed_key) BETWEEN 1 AND 4096)
);

COMMENT ON TABLE agent_license IS 'Current offline Agent entitlement; original signed Key is the only authorization source.';
