-- DESTRUCTIVE TEST TEARDOWN ONLY.
-- Production recovery uses a verified backup restore or an approved forward
-- fix. Removing these columns discards audit evidence and is never rollback.
ALTER TABLE upgrade_operation
    DROP CONSTRAINT IF EXISTS upgrade_operation_full_scope_evidence,
    DROP CONSTRAINT IF EXISTS upgrade_operation_frontend_only_scope,
    DROP CONSTRAINT IF EXISTS upgrade_operation_not_required_scope,
    DROP COLUMN IF EXISTS confirmed_deployment_version,
    DROP COLUMN IF EXISTS baseline_deployment_digest,
    DROP COLUMN IF EXISTS plan_digest,
    DROP COLUMN IF EXISTS plan_summary,
    DROP COLUMN IF EXISTS work_disposition,
    DROP COLUMN IF EXISTS execution_mode;
