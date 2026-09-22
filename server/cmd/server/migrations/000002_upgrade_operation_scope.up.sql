-- Upgrade Operation scope facts are additive. Empty values preserve historical
-- rows as explicit legacy/unknown evidence; a selective plan has stricter
-- cross-column constraints because it is allowed to bypass work cancellation.
ALTER TABLE upgrade_operation
    ADD COLUMN IF NOT EXISTS execution_mode VARCHAR(32) NOT NULL DEFAULT '' CHECK (
        execution_mode IN ('', 'full', 'frontend_only')
    ),
    ADD COLUMN IF NOT EXISTS work_disposition VARCHAR(32) NOT NULL DEFAULT '' CHECK (
        work_disposition IN ('', 'not_required', 'cancelled', 'cancellation_failed', 'legacy_unknown')
    ),
    ADD COLUMN IF NOT EXISTS plan_summary JSONB NOT NULL DEFAULT '{"touchedServices":[]}'::jsonb CHECK (
        jsonb_typeof(plan_summary) = 'object'
    ),
    ADD COLUMN IF NOT EXISTS plan_digest VARCHAR(71) NOT NULL DEFAULT '' CHECK (
        plan_digest = '' OR plan_digest ~ '^sha256:[a-f0-9]{64}$'
    ),
    ADD COLUMN IF NOT EXISTS baseline_deployment_digest VARCHAR(71) NOT NULL DEFAULT '' CHECK (
        baseline_deployment_digest = '' OR baseline_deployment_digest ~ '^sha256:[a-f0-9]{64}$'
    ),
    ADD COLUMN IF NOT EXISTS confirmed_deployment_version VARCHAR(64) NOT NULL DEFAULT '' CHECK (
        confirmed_deployment_version = '' OR (
            confirmed_deployment_version = btrim(confirmed_deployment_version)
            AND length(confirmed_deployment_version) <= 64
        )
    );

ALTER TABLE upgrade_operation
    ADD CONSTRAINT upgrade_operation_not_required_scope CHECK (
        work_disposition <> 'not_required' OR execution_mode = 'frontend_only'
    ),
    ADD CONSTRAINT upgrade_operation_frontend_only_scope CHECK (
        execution_mode <> 'frontend_only' OR (
            work_disposition = 'not_required'
            AND plan_digest ~ '^sha256:[a-f0-9]{64}$'
            AND baseline_deployment_digest ~ '^sha256:[a-f0-9]{64}$'
            AND confirmed_deployment_version <> ''
            AND plan_summary = '{"touchedServices":["frontend"]}'::jsonb
        )
    ),
    ADD CONSTRAINT upgrade_operation_full_scope_evidence CHECK (
        execution_mode <> 'full' OR (
            plan_digest ~ '^sha256:[a-f0-9]{64}$'
            AND baseline_deployment_digest = ''
            AND confirmed_deployment_version = ''
        )
    );
