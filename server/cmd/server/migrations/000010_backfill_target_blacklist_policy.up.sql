-- Backfill Target-local singleton policies for Targets created before the
-- blacklist policy lifecycle initialized rows during Target creation.
INSERT INTO blacklist_policy (scope, target_id, patterns)
SELECT 'target', target.id, '[]'::jsonb
FROM target
WHERE NOT EXISTS (
    SELECT 1
    FROM blacklist_policy
    WHERE blacklist_policy.scope = 'target'
      AND blacklist_policy.target_id = target.id
);
