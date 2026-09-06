-- Initial schema migration for Go backend
-- This migration creates all tables from scratch

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================
-- Core tables (no dependencies)
-- ============================================

-- auth_user (Django compatible)
CREATE TABLE IF NOT EXISTS auth_user (
    id SERIAL PRIMARY KEY,
    password VARCHAR(128) NOT NULL,
    token_version INTEGER NOT NULL DEFAULT 0,
    last_login TIMESTAMPTZ,
    is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
    username VARCHAR(150) NOT NULL,
    first_name VARCHAR(150) NOT NULL DEFAULT '',
    last_name VARCHAR(150) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    locale VARCHAR(2) NOT NULL DEFAULT 'en' CHECK (locale IN ('zh', 'en')),
    locale_initialized_at TIMESTAMPTZ,
    is_staff BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    date_joined TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_user_username ON auth_user(username);

-- login_visual_discovery is account-owned cosmetic state. It deliberately
-- grants no media-management authority; that remains an active-superuser check.
CREATE TABLE IF NOT EXISTS login_visual_discovery (
    user_id INTEGER PRIMARY KEY REFERENCES auth_user(id) ON DELETE CASCADE,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- mcp_key stores one active, user-bound MCP credential per user. The
-- plaintext key is never persisted; callers store only its SHA-256 digest.
CREATE TABLE IF NOT EXISTS mcp_key (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE REFERENCES auth_user(id) ON DELETE CASCADE,
    key_digest CHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_mcp_key_digest ON mcp_key(key_digest);

-- django_session (Django compatible)
CREATE TABLE IF NOT EXISTS django_session (
    session_key VARCHAR(40) PRIMARY KEY,
    session_data TEXT NOT NULL,
    expire_date TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_django_session_expire_date ON django_session(expire_date);

-- organization
CREATE TABLE IF NOT EXISTS organization (
    id SERIAL PRIMARY KEY,
    name VARCHAR(300) NOT NULL,
    description VARCHAR(1000) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_org_name ON organization(name);
CREATE INDEX IF NOT EXISTS idx_org_created_at ON organization(created_at);
CREATE INDEX IF NOT EXISTS idx_org_deleted_at ON organization(deleted_at);
CREATE INDEX IF NOT EXISTS idx_org_name_trgm_active ON organization USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_org_name_id_active ON organization(name, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_org_created_at_id_active ON organization(created_at, id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_org_active_name_unique ON organization(name) WHERE deleted_at IS NULL;

-- target
CREATE TABLE IF NOT EXISTS target (
    id SERIAL PRIMARY KEY,
    name VARCHAR(300) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'domain',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_scanned_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_target_name ON target(name);
CREATE INDEX IF NOT EXISTS idx_target_type ON target(type);
CREATE INDEX IF NOT EXISTS idx_target_created_at ON target(created_at);
CREATE INDEX IF NOT EXISTS idx_target_deleted_at ON target(deleted_at);
CREATE INDEX IF NOT EXISTS idx_target_name_trgm_active ON target USING GIN (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_name_id_active ON target(name, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_created_at_id_active ON target(created_at, id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_last_scanned_at_desc_id_active ON target(last_scanned_at DESC NULLS LAST, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_last_scanned_at_asc_id_active ON target(last_scanned_at ASC NULLS LAST, id ASC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_type_created_at_id_active ON target(type, created_at, id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_target_active_identity_unique
    ON target(type, name) WHERE deleted_at IS NULL;

-- target_cleanup_job is the durable, internal reconciliation queue for a
-- tombstoned Target. It deliberately has no phase, cursor, or worker lease:
-- every run re-checks database truth from the first cleanup step.
CREATE TABLE IF NOT EXISTS target_cleanup_job (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL UNIQUE REFERENCES target(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
    retry_count INTEGER NOT NULL DEFAULT 0 CHECK (retry_count >= 0),
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error VARCHAR(2000) NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_target_cleanup_job_due
    ON target_cleanup_job(status, next_retry_at, id) WHERE status = 'pending';

-- organization_target (many-to-many)
CREATE TABLE IF NOT EXISTS organization_target (
    organization_id INTEGER NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    PRIMARY KEY (organization_id, target_id)
);
CREATE INDEX IF NOT EXISTS idx_org_target_org ON organization_target(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_target_target ON organization_target(target_id);
CREATE INDEX IF NOT EXISTS idx_org_target_target_organization ON organization_target(target_id, organization_id);

-- wordlist
CREATE TABLE IF NOT EXISTS wordlist (
    id SERIAL PRIMARY KEY,
    file_name VARCHAR(200) NOT NULL,
    description VARCHAR(200) NOT NULL DEFAULT '',
    file_path VARCHAR(500) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    line_count INTEGER NOT NULL DEFAULT 0,
    file_hash VARCHAR(64) NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- engine: successfully installed current engine inventory. Cache locations are
-- derived from package_digest and deliberately never persisted here.
CREATE TABLE IF NOT EXISTS engine (
    id SERIAL PRIMARY KEY,
    engine_id VARCHAR(255) NOT NULL UNIQUE CHECK (engine_id <> ''),
    publisher VARCHAR(255) NOT NULL CHECK (publisher <> ''),
    package_version VARCHAR(255) NOT NULL CHECK (package_version <> ''),
    artifact_ref VARCHAR(1000) NOT NULL UNIQUE CHECK (artifact_ref ~ '^.+@sha256:[a-f0-9]{64}$'),
    package_digest VARCHAR(71) NOT NULL UNIQUE CHECK (package_digest ~ '^sha256:[a-f0-9]{64}$'),
    manifest JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- scan_workflow is the runtime management-plane source for ordered scan
-- orchestration. Engine configuration remains owned by the installed Engine.
CREATE TABLE IF NOT EXISTS scan_workflow (
    scan_workflow_id VARCHAR(100) PRIMARY KEY,
    display_name VARCHAR(300) NOT NULL CHECK (btrim(display_name) <> ''),
    description TEXT NOT NULL DEFAULT '',
    stages JSONB NOT NULL,
    is_builtin BOOLEAN NOT NULL,
    definition_digest VARCHAR(64),
    request_id UUID,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT scan_workflow_namespace CHECK (
        (is_builtin AND scan_workflow_id !~ '^wf-')
        OR (
            NOT is_builtin
            AND scan_workflow_id ~ '^wf-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
        )
    ),
    CONSTRAINT scan_workflow_builtin_digest_shape CHECK (
        (is_builtin AND definition_digest ~ '^[0-9a-f]{64}$')
        OR (NOT is_builtin AND definition_digest IS NULL)
    ),
    CONSTRAINT scan_workflow_request_ownership CHECK (
        (is_builtin AND request_id IS NULL) OR NOT is_builtin
    )
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scan_workflow_request_id
    ON scan_workflow(request_id) WHERE request_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scan_workflow_list_order
    ON scan_workflow(is_builtin DESC, display_name ASC, scan_workflow_id ASC);
CREATE UNIQUE INDEX IF NOT EXISTS unique_wordlist_file_name ON wordlist(file_name);
CREATE INDEX IF NOT EXISTS idx_wordlist_created_at ON wordlist(created_at);
CREATE INDEX IF NOT EXISTS idx_wordlist_updated_at_id ON wordlist(updated_at, id);
CREATE INDEX IF NOT EXISTS idx_wordlist_line_count_id ON wordlist(line_count, id);
CREATE INDEX IF NOT EXISTS idx_wordlist_file_size_id ON wordlist(file_size, id);
CREATE INDEX IF NOT EXISTS idx_wordlist_tags_gin ON wordlist USING GIN (tags);

-- Nuclei POC source/catalog state. This is a disposable-development hard cut;
-- a fresh baseline has no legacy repository table or runtime fallback. The
-- enablement default is intentionally changed by rebuilding this baseline;
-- existing development rows are not backfilled.
CREATE TABLE IF NOT EXISTS nuclei_poc_source (
    id UUID PRIMARY KEY,
    source_type VARCHAR(16) NOT NULL CHECK (source_type IN ('git', 'gitee', 'custom')),
    repo_url TEXT NOT NULL CHECK (btrim(repo_url) <> ''),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    commit_sha VARCHAR(64) NOT NULL DEFAULT '' CHECK (commit_sha = '' OR commit_sha ~ '^[a-fA-F0-9]{40}$'),
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_nuclei_poc_source_active_singleton
    ON nuclei_poc_source (is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_source_updated_at
    ON nuclei_poc_source (updated_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS nuclei_poc_sync_task (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL UNIQUE,
    request_fingerprint CHAR(64) NOT NULL CHECK (request_fingerprint ~ '^[a-f0-9]{64}$'),
    source_type VARCHAR(16) NOT NULL CHECK (source_type IN ('git', 'gitee', 'custom')),
    repo_url TEXT NOT NULL CHECK (btrim(repo_url) <> ''),
    source_id UUID NOT NULL REFERENCES nuclei_poc_source(id) ON DELETE RESTRICT,
    state VARCHAR(32) NOT NULL CHECK (state IN ('VALIDATING_SOURCE', 'CLONING', 'SCANNING_FILES', 'VALIDATING_TEMPLATES', 'COMMITTING', 'CLEANING', 'SUCCEEDED', 'FAILED')),
    phase VARCHAR(32) NOT NULL CHECK (phase IN ('VALIDATING_SOURCE', 'CLONING', 'SCANNING_FILES', 'VALIDATING_TEMPLATES', 'COMMITTING', 'CLEANING', 'SUCCEEDED', 'FAILED')),
    files_seen BIGINT CHECK (files_seen IS NULL OR files_seen >= 0),
    yaml_files_seen BIGINT CHECK (yaml_files_seen IS NULL OR yaml_files_seen >= 0),
    templates_validated BIGINT CHECK (templates_validated IS NULL OR templates_validated >= 0),
    bytes_read BIGINT CHECK (bytes_read IS NULL OR bytes_read >= 0),
    commit_sha VARCHAR(64) NOT NULL DEFAULT '' CHECK (commit_sha = '' OR commit_sha ~ '^[a-fA-F0-9]{40}$'),
    committed_poc_count BIGINT NOT NULL DEFAULT 0 CHECK (committed_poc_count >= 0),
    failure_code VARCHAR(64) NOT NULL DEFAULT '',
    failure_summary VARCHAR(500) NOT NULL DEFAULT '',
    diagnostics JSONB NOT NULL DEFAULT '{}'::jsonb,
    cleanup_status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (cleanup_status IN ('pending', 'clean', 'residual')),
    workspace_key VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_nuclei_poc_sync_task_request_fingerprint
    ON nuclei_poc_sync_task(request_id, request_fingerprint);
CREATE UNIQUE INDEX IF NOT EXISTS idx_nuclei_poc_sync_task_active_singleton
    ON nuclei_poc_sync_task((TRUE))
    WHERE state NOT IN ('SUCCEEDED', 'FAILED');
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_sync_task_retention
    ON nuclei_poc_sync_task(state, completed_at, id);

CREATE TABLE IF NOT EXISTS nuclei_poc_candidate_import (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL REFERENCES nuclei_poc_sync_task(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES nuclei_poc_source(id) ON DELETE CASCADE,
    template_id VARCHAR(255) NOT NULL CHECK (btrim(template_id) <> '' AND template_id !~ '[[:space:]]'),
    display_name TEXT NOT NULL DEFAULT '',
    severity VARCHAR(16) NOT NULL DEFAULT 'info',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    cve JSONB NOT NULL DEFAULT '[]'::jsonb,
    cwe JSONB NOT NULL DEFAULT '[]'::jsonb,
    "references" JSONB NOT NULL DEFAULT '[]'::jsonb,
    remediation TEXT NOT NULL DEFAULT '',
    relative_path TEXT NOT NULL,
    content_sha256 CHAR(64) NOT NULL CHECK (content_sha256 ~ '^[a-f0-9]{64}$'),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT nuclei_poc_candidate_template_unique UNIQUE (task_id, template_id)
);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_candidate_task ON nuclei_poc_candidate_import(task_id, id);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_candidate_source ON nuclei_poc_candidate_import(source_id, template_id);

CREATE TABLE IF NOT EXISTS nuclei_poc (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES nuclei_poc_source(id) ON DELETE RESTRICT,
    template_id VARCHAR(255) NOT NULL UNIQUE CHECK (btrim(template_id) <> '' AND template_id !~ '[[:space:]]'),
    display_name TEXT NOT NULL DEFAULT '',
    severity VARCHAR(16) NOT NULL DEFAULT 'info',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    cve JSONB NOT NULL DEFAULT '[]'::jsonb,
    cwe JSONB NOT NULL DEFAULT '[]'::jsonb,
    "references" JSONB NOT NULL DEFAULT '[]'::jsonb,
    remediation TEXT NOT NULL DEFAULT '',
    relative_path TEXT NOT NULL,
    content_sha256 CHAR(64) NOT NULL CHECK (content_sha256 ~ '^[a-f0-9]{64}$'),
    content TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_source ON nuclei_poc(source_id, template_id);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_severity ON nuclei_poc(severity, template_id);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_updated_at ON nuclei_poc(updated_at DESC, template_id);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_tags_gin ON nuclei_poc USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_template_id_trgm ON nuclei_poc USING GIN(template_id gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_display_name_trgm ON nuclei_poc USING GIN(display_name gin_trgm_ops);

CREATE TABLE IF NOT EXISTS nuclei_poc_sync_request_tombstone (
    request_id UUID PRIMARY KEY,
    request_fingerprint CHAR(64) NOT NULL CHECK (request_fingerprint ~ '^[a-f0-9]{64}$'),
    received_at TIMESTAMPTZ NOT NULL,
    terminal_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_nuclei_poc_tombstone_created_at
    ON nuclei_poc_sync_request_tombstone(created_at, request_id);

-- blacklist_policy stores exactly one global singleton and one local singleton
-- for each Target. Rule parsing and etags remain Server-derived concerns.
CREATE TABLE IF NOT EXISTS blacklist_policy (
    id SERIAL PRIMARY KEY,
    scope VARCHAR(20) NOT NULL,
    target_id INTEGER REFERENCES target(id) ON DELETE CASCADE,
    patterns JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT blacklist_policy_scope CHECK (scope IN ('global', 'target')),
    CONSTRAINT blacklist_policy_scope_target_shape CHECK (
        (scope = 'global' AND target_id IS NULL)
        OR (scope = 'target' AND target_id IS NOT NULL)
    ),
    CONSTRAINT blacklist_policy_patterns_string_array CHECK (
        jsonb_typeof(patterns) = 'array'
        AND NOT jsonb_path_exists(patterns, '$[*] ? (@.type() != "string")')
    )
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_policy_global_singleton
    ON blacklist_policy (scope) WHERE scope = 'global';
CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_policy_target_singleton
    ON blacklist_policy (target_id) WHERE scope = 'target';
CREATE INDEX IF NOT EXISTS idx_blacklist_policy_target_cleanup
    ON blacklist_policy(target_id, id) WHERE scope = 'target';
INSERT INTO blacklist_policy (scope, target_id, patterns)
VALUES ('global', NULL, '[]'::jsonb)
ON CONFLICT DO NOTHING;

-- login visual settings retain singleton draft/published pointers while media
-- bytes remain in the managed Server persistent directory.
CREATE TABLE IF NOT EXISTS login_visual_media (
    id UUID PRIMARY KEY,
    kind VARCHAR(16) NOT NULL CHECK (kind IN ('image', 'video')),
    content_type VARCHAR(64) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    storage_key VARCHAR(128) NOT NULL UNIQUE,
    poster_key VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS login_visual_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    draft_media_id UUID REFERENCES login_visual_media(id) ON DELETE SET NULL,
    published_media_id UUID REFERENCES login_visual_media(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO login_visual_settings (id) VALUES (1) ON CONFLICT DO NOTHING;

-- subfinder_provider_settings (singleton)
CREATE TABLE IF NOT EXISTS subfinder_provider_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    providers JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Scan related tables (depends on target)
-- ============================================

-- scan
CREATE TABLE IF NOT EXISTS scan (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    scan_workflow_id VARCHAR(100) NOT NULL CHECK (scan_workflow_id <> ''),
    configuration JSONB NOT NULL DEFAULT '{}',
    input_source VARCHAR(32) NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
    trigger_type VARCHAR(16) NOT NULL CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
    status VARCHAR(20) NOT NULL DEFAULT 'initiated',
    results_dir VARCHAR(100) NOT NULL DEFAULT '',
    container_ids VARCHAR(100)[] NOT NULL DEFAULT '{}',
    agent_id INTEGER,
    assignment_mode VARCHAR(20) NOT NULL DEFAULT 'automatic' CHECK (assignment_mode IN ('automatic', 'pinned')),
    error_message VARCHAR(2000) NOT NULL DEFAULT '',
    failure_kind VARCHAR(100) NOT NULL DEFAULT '',
    progress INTEGER NOT NULL DEFAULT 0,
    current_stage VARCHAR(50) NOT NULL DEFAULT '',
    stage_progress JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    stopped_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    -- Cached statistics
    cached_subdomains_count INTEGER NOT NULL DEFAULT 0,
    cached_websites_count INTEGER NOT NULL DEFAULT 0,
    cached_endpoints_count INTEGER NOT NULL DEFAULT 0,
    cached_ips_count INTEGER NOT NULL DEFAULT 0,
    cached_directories_count INTEGER NOT NULL DEFAULT 0,
    cached_screenshots_count INTEGER NOT NULL DEFAULT 0,
    cached_vulns_total INTEGER NOT NULL DEFAULT 0,
    cached_vulns_critical INTEGER NOT NULL DEFAULT 0,
    cached_vulns_high INTEGER NOT NULL DEFAULT 0,
    cached_vulns_medium INTEGER NOT NULL DEFAULT 0,
    cached_vulns_low INTEGER NOT NULL DEFAULT 0,
    stats_updated_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_scan_target ON scan(target_id);
CREATE INDEX IF NOT EXISTS idx_scan_status ON scan(status);
CREATE INDEX IF NOT EXISTS idx_scan_created_at ON scan(created_at);
CREATE INDEX IF NOT EXISTS idx_scan_deleted_at ON scan(deleted_at);
CREATE INDEX IF NOT EXISTS idx_scan_target_active_cleanup ON scan(target_id, status, id)
    WHERE status IN ('pending', 'running');

-- scan_operation is the durable MCP polling handle for a Scan. It stores no
-- independent lifecycle state: reads always project the linked Scan/Task facts.
CREATE TABLE IF NOT EXISTS scan_operation (
    id UUID PRIMARY KEY,
    scan_id INTEGER NOT NULL UNIQUE REFERENCES scan(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL,
    request_id VARCHAR(128),
    request_fingerprint VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT scan_operation_request_fingerprint_format CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);
CREATE INDEX IF NOT EXISTS idx_scan_operation_scan_id ON scan_operation(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_operation_target_id ON scan_operation(target_id, scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_operation_request_id
    ON scan_operation(request_id) WHERE request_id IS NOT NULL;

-- mcp_request_replay binds an optional business request_id to exactly one
-- action/fingerprint/result. It is shared by scan starts and review actions;
-- expiry bounds replay semantics without creating an operation-specific GC.
CREATE TABLE IF NOT EXISTS mcp_request_replay (
    request_id VARCHAR(128) PRIMARY KEY,
    action VARCHAR(80) NOT NULL,
    request_fingerprint VARCHAR(64) NOT NULL,
    response JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT mcp_request_replay_action_nonempty CHECK (action <> ''),
    CONSTRAINT mcp_request_replay_fingerprint_format CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);
CREATE INDEX IF NOT EXISTS idx_mcp_request_replay_expires_at ON mcp_request_replay(expires_at, request_id);

-- scan_blacklist_snapshot is scan-owned immutable input state. It is not part
-- of the Scan read model or any saved Engine plan.
CREATE TABLE IF NOT EXISTS scan_blacklist_snapshot (
    scan_id INTEGER PRIMARY KEY REFERENCES scan(id) ON DELETE CASCADE,
    patterns JSONB NOT NULL,
    CONSTRAINT scan_blacklist_snapshot_patterns_string_array CHECK (
        jsonb_typeof(patterns) = 'array'
        AND NOT jsonb_path_exists(patterns, '$[*] ? (@.type() != "string")')
    )
);

-- scan_task: executable workflow step engine record for one scan
CREATE TABLE IF NOT EXISTS scan_task (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    stage_order INTEGER NOT NULL DEFAULT 0,
    stage_id VARCHAR(100) NOT NULL CHECK (stage_id <> ''),
    step_order INTEGER NOT NULL DEFAULT 0,
    step_id VARCHAR(100) NOT NULL CHECK (step_id <> ''),
    engine_id VARCHAR(255) NOT NULL CHECK (engine_id <> ''),
    engine_config JSONB NOT NULL DEFAULT '{}',
    task_execution_config JSONB NOT NULL DEFAULT '{}',
    resolved_execution_plan BYTEA NOT NULL DEFAULT ''::bytea,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    assigned_agent_id INTEGER,
    assigned_session_id VARCHAR(64),
    assigned_session_epoch BIGINT,
    assigned_request_id VARCHAR(36),
    terminal_reconciliation_pending BOOLEAN NOT NULL DEFAULT FALSE,
    error_message VARCHAR(4096) NOT NULL DEFAULT '',
    failure_kind VARCHAR(100) NOT NULL DEFAULT '',
    failure_detail VARCHAR(500) NOT NULL DEFAULT '',
    engine_diagnostics JSONB,
    skip_reason VARCHAR(1000) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    CONSTRAINT scan_task_claim_tuple_shape CHECK (
        (assigned_agent_id IS NULL AND assigned_session_id IS NULL AND assigned_request_id IS NULL)
        OR
        (assigned_agent_id IS NOT NULL AND assigned_session_id IS NOT NULL AND assigned_session_epoch IS NOT NULL AND assigned_request_id IS NOT NULL)
    ),
    CONSTRAINT scan_task_running_plan_claim_shape CHECK (
        status <> 'running'
        OR (
            OCTET_LENGTH(resolved_execution_plan) > 0
            AND assigned_agent_id IS NOT NULL
            AND assigned_session_id IS NOT NULL
            AND assigned_session_epoch IS NOT NULL
            AND assigned_request_id IS NOT NULL
        )
    )
);
CREATE INDEX IF NOT EXISTS idx_scan_task_scan ON scan_task(scan_id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_task_scan_id_id ON scan_task(scan_id, id);
CREATE INDEX IF NOT EXISTS idx_scan_task_status ON scan_task(status);
CREATE INDEX IF NOT EXISTS idx_scan_task_pending_order ON scan_task(status, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_scan_task_created_at ON scan_task(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_task_claim_request ON scan_task(assigned_agent_id, assigned_session_epoch, assigned_request_id) WHERE assigned_request_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scan_task_terminal_reconciliation_pending ON scan_task(id) WHERE terminal_reconciliation_pending = TRUE;

-- scheduled_scan
CREATE TABLE IF NOT EXISTS scheduled_scan (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    scan_workflow_id VARCHAR(100) NOT NULL CHECK (scan_workflow_id <> ''),
    configuration JSONB NOT NULL DEFAULT '{}',
    input_source VARCHAR(32) NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
    organization_id INTEGER REFERENCES organization(id) ON DELETE SET NULL,
    target_id INTEGER REFERENCES target(id) ON DELETE SET NULL,
    agent_id INTEGER,
    cron_expression VARCHAR(100) NOT NULL DEFAULT '0 2 * * *',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    run_count INTEGER NOT NULL DEFAULT 0 CHECK (run_count >= 0),
    successful_handoff_count INTEGER NOT NULL DEFAULT 0 CHECK (successful_handoff_count >= 0),
    failed_handoff_count INTEGER NOT NULL DEFAULT 0 CHECK (failed_handoff_count >= 0),
    last_run_time TIMESTAMPTZ,
    next_run_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT scheduled_scan_enabled_cursor_consistency CHECK (
        (is_enabled = TRUE AND next_run_time IS NOT NULL)
        OR (is_enabled = FALSE AND next_run_time IS NULL)
    )
);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_name ON scheduled_scan(name);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_enabled ON scheduled_scan(is_enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_created_at ON scheduled_scan(created_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_due ON scheduled_scan(next_run_time, id)
    WHERE is_enabled = TRUE AND next_run_time IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_target_id ON scheduled_scan(target_id, id)
    WHERE target_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS scheduled_scan_occurrence (
    id BIGSERIAL PRIMARY KEY,
    scheduled_scan_id INTEGER NOT NULL REFERENCES scheduled_scan(id) ON DELETE CASCADE,
    scheduled_for TIMESTAMPTZ NOT NULL,
    attempted_at TIMESTAMPTZ,
    dispatched_at TIMESTAMPTZ,
    failure_kind VARCHAR(100),
    failure_message VARCHAR(2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT scheduled_scan_occurrence_identity UNIQUE (scheduled_scan_id, scheduled_for),
    CONSTRAINT scheduled_scan_occurrence_dispatch_requires_attempt CHECK (
        dispatched_at IS NULL OR attempted_at IS NOT NULL
    ),
    CONSTRAINT scheduled_scan_occurrence_outcome_consistency CHECK (
        (dispatched_at IS NOT NULL AND failure_kind IS NULL AND failure_message IS NULL)
        OR (
            dispatched_at IS NULL
            AND (
                (failure_kind IS NULL AND failure_message IS NULL)
                OR (attempted_at IS NOT NULL AND failure_kind IS NOT NULL AND failure_message IS NOT NULL)
            )
        )
    )
);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_occurrence_candidate
    ON scheduled_scan_occurrence(scheduled_scan_id, scheduled_for, id)
    WHERE attempted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_occurrence_retention
    ON scheduled_scan_occurrence(attempted_at, id)
    WHERE attempted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_occurrence_schedule_id_id
    ON scheduled_scan_occurrence(scheduled_scan_id, id);

-- ============================================
-- Asset tables (depends on target)
-- ============================================

-- subdomain
CREATE TABLE IF NOT EXISTS subdomain (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    dns_name VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subdomain_target ON subdomain(target_id);
CREATE INDEX IF NOT EXISTS idx_subdomain_target_id ON subdomain(target_id, id);
CREATE INDEX IF NOT EXISTS idx_subdomain_dns_name ON subdomain(dns_name);
CREATE INDEX IF NOT EXISTS idx_subdomain_dns_name_trgm ON subdomain USING GIN (dns_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_subdomain_created_at ON subdomain(created_at);
CREATE INDEX IF NOT EXISTS idx_subdomain_target_created_at_id ON subdomain(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_subdomain_target_dns_name_id ON subdomain(target_id, dns_name, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_subdomain_dns_name_target ON subdomain(dns_name, target_id);

-- host_port_mapping
CREATE TABLE IF NOT EXISTS host_port_mapping (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    host VARCHAR(1000) NOT NULL,
    ip INET NOT NULL,
    port INTEGER NOT NULL,
    CONSTRAINT host_port_mapping_host_nonblank CHECK (BTRIM(host) <> ''),
    CONSTRAINT host_port_mapping_ip_ipv4_host CHECK (family(ip) = 4 AND masklen(ip) = 32),
    CONSTRAINT host_port_mapping_port_range CHECK (port BETWEEN 1 AND 65535),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_hpm_target ON host_port_mapping(target_id);
CREATE INDEX IF NOT EXISTS idx_hpm_target_id ON host_port_mapping(target_id, id);
CREATE INDEX IF NOT EXISTS idx_hpm_host ON host_port_mapping(host);
CREATE INDEX IF NOT EXISTS idx_hpm_ip ON host_port_mapping(ip);
CREATE INDEX IF NOT EXISTS idx_hpm_port ON host_port_mapping(port);
CREATE INDEX IF NOT EXISTS idx_hpm_created_at ON host_port_mapping(created_at);
CREATE INDEX IF NOT EXISTS idx_hpm_target_port_ip ON host_port_mapping(target_id, port, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_target_ip ON host_port_mapping(target_id, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_target_host_ip ON host_port_mapping(target_id, host, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_host_trgm ON host_port_mapping USING GIN (host gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_hpm_target_created_at_ip ON host_port_mapping(target_id, created_at, ip);
CREATE UNIQUE INDEX IF NOT EXISTS unique_target_host_ip_port ON host_port_mapping(target_id, host, ip, port);

-- website
CREATE TABLE IF NOT EXISTS website (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
	response_body_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    content_type TEXT NOT NULL DEFAULT '',
    tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
    status_code INTEGER,
    content_length INTEGER,
    vhost BOOLEAN,
    response_headers TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_website_target ON website(target_id);
CREATE INDEX IF NOT EXISTS idx_website_target_id ON website(target_id, id);
CREATE INDEX IF NOT EXISTS idx_website_url ON website(url);
CREATE INDEX IF NOT EXISTS idx_website_host ON website(host);
CREATE INDEX IF NOT EXISTS idx_website_title ON website(title);
CREATE INDEX IF NOT EXISTS idx_website_status_code ON website(status_code);
CREATE INDEX IF NOT EXISTS idx_website_created_at ON website(created_at);
CREATE INDEX IF NOT EXISTS idx_website_target_created_at_id ON website(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_website_url_trgm ON website USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_website_host_trgm ON website USING GIN (host gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_website_title_trgm ON website USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_website_target_status_code_id ON website(target_id, status_code, id);
CREATE INDEX IF NOT EXISTS idx_website_target_content_length_id ON website(target_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_website_target_webserver_id ON website(target_id, webserver, id);
CREATE INDEX IF NOT EXISTS idx_website_target_content_type_id ON website(target_id, content_type, id);
CREATE INDEX IF NOT EXISTS idx_website_target_vhost_id ON website(target_id, vhost, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_website_url_target ON website(url, target_id);

-- endpoint
CREATE TABLE IF NOT EXISTS endpoint (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
	response_body_truncated BOOLEAN NOT NULL DEFAULT FALSE,
	content_type TEXT NOT NULL DEFAULT '',
	tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
	status_code INTEGER,
	content_length INTEGER,
	vhost BOOLEAN,
	response_headers TEXT NOT NULL DEFAULT '',
	response_headers_truncated BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_endpoint_target ON endpoint(target_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_id ON endpoint(target_id, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_url ON endpoint(url);
CREATE INDEX IF NOT EXISTS idx_endpoint_host ON endpoint(host);
CREATE INDEX IF NOT EXISTS idx_endpoint_title ON endpoint(title);
CREATE INDEX IF NOT EXISTS idx_endpoint_status_code ON endpoint(status_code);
CREATE INDEX IF NOT EXISTS idx_endpoint_created_at ON endpoint(created_at);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_created_at_id ON endpoint(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_url_trgm ON endpoint USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_endpoint_host_trgm ON endpoint USING GIN (host gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_endpoint_title_trgm ON endpoint USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_status_code_id ON endpoint(target_id, status_code, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_content_length_id ON endpoint(target_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_webserver_id ON endpoint(target_id, webserver, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_content_type_id ON endpoint(target_id, content_type, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_target_vhost_id ON endpoint(target_id, vhost, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_endpoint_url_target ON endpoint(url, target_id);


-- directory
CREATE TABLE IF NOT EXISTS directory (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status INTEGER,
    content_length BIGINT,
    content_type VARCHAR(1024) NOT NULL DEFAULT '',
    duration BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_directory_target ON directory(target_id);
CREATE INDEX IF NOT EXISTS idx_directory_target_id ON directory(target_id, id);
CREATE INDEX IF NOT EXISTS idx_directory_url ON directory(url);
CREATE INDEX IF NOT EXISTS idx_directory_status ON directory(status);
CREATE INDEX IF NOT EXISTS idx_directory_created_at ON directory(created_at);
CREATE INDEX IF NOT EXISTS idx_directory_target_created_at_id ON directory(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_directory_url_trgm ON directory USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_directory_target_status_id ON directory(target_id, status, id);
CREATE INDEX IF NOT EXISTS idx_directory_target_content_length_id ON directory(target_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_directory_target_content_type_id ON directory(target_id, content_type, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_directory_url_target ON directory(target_id, url);

-- screenshot
CREATE TABLE IF NOT EXISTS screenshot (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status_code SMALLINT,
    image BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_screenshot_target ON screenshot(target_id);
CREATE INDEX IF NOT EXISTS idx_screenshot_target_id ON screenshot(target_id, id);
CREATE INDEX IF NOT EXISTS idx_screenshot_created_at ON screenshot(created_at);
CREATE INDEX IF NOT EXISTS idx_screenshot_target_created_at_id ON screenshot(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_screenshot_url_trgm ON screenshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_screenshot_target_status_code_id ON screenshot(target_id, status_code, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_screenshot_per_target ON screenshot(target_id, url);

-- vulnerability
CREATE TABLE IF NOT EXISTS vulnerability (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL DEFAULT '',
    vuln_type VARCHAR(200) NOT NULL DEFAULT '',
    severity VARCHAR(20) NOT NULL DEFAULT 'unknown',
    source VARCHAR(100) NOT NULL DEFAULT '',
    cvss_score DECIMAL(3,1) NOT NULL DEFAULT 0.0,
    description TEXT NOT NULL DEFAULT '',
    raw_output JSONB NOT NULL DEFAULT '{}',
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vuln_target ON vulnerability(target_id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_id ON vulnerability(target_id, id);
CREATE INDEX IF NOT EXISTS idx_vuln_url ON vulnerability(url);
CREATE INDEX IF NOT EXISTS idx_vuln_url_trgm ON vulnerability USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_vuln_type ON vulnerability(vuln_type);
CREATE INDEX IF NOT EXISTS idx_vuln_severity ON vulnerability(severity);
CREATE INDEX IF NOT EXISTS idx_vuln_source ON vulnerability(source);
CREATE INDEX IF NOT EXISTS idx_vuln_created_at ON vulnerability(created_at);
CREATE INDEX IF NOT EXISTS idx_vuln_created_at_id ON vulnerability(created_at, id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_created_at_id ON vulnerability(target_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_severity_id ON vulnerability(target_id, severity, id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_source_id ON vulnerability(target_id, source, id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_type_id ON vulnerability(target_id, vuln_type, id);
CREATE INDEX IF NOT EXISTS idx_vuln_reviewed_created_at_id ON vulnerability(reviewed, created_at, id);
CREATE INDEX IF NOT EXISTS idx_vuln_target_reviewed ON vulnerability(target_id, reviewed);
CREATE INDEX IF NOT EXISTS idx_vuln_target_reviewed_created_at_id ON vulnerability(target_id, reviewed, created_at, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_vulnerability_observation ON vulnerability(target_id, url, vuln_type, severity);

-- ============================================
-- Snapshot tables (depends on scan)
-- ============================================

-- subdomain_snapshot
CREATE TABLE IF NOT EXISTS subdomain_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    dns_name VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan ON subdomain_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_dns_name ON subdomain_snapshot(dns_name);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_dns_name_trgm ON subdomain_snapshot USING GIN (dns_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_created_at ON subdomain_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan_created_at_id ON subdomain_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan_dns_name_id ON subdomain_snapshot(scan_id, dns_name, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_subdomain_dns_name_per_scan_snapshot ON subdomain_snapshot(scan_id, dns_name);

-- host_port_mapping_snapshot
CREATE TABLE IF NOT EXISTS host_port_mapping_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    host VARCHAR(1000) NOT NULL,
    ip INET NOT NULL,
    port INTEGER NOT NULL,
    CONSTRAINT host_port_mapping_snapshot_host_nonblank CHECK (BTRIM(host) <> ''),
    CONSTRAINT host_port_mapping_snapshot_ip_ipv4_host CHECK (family(ip) = 4 AND masklen(ip) = 32),
    CONSTRAINT host_port_mapping_snapshot_port_range CHECK (port BETWEEN 1 AND 65535),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan ON host_port_mapping_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_host ON host_port_mapping_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_ip ON host_port_mapping_snapshot(ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_port ON host_port_mapping_snapshot(port);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_created_at ON host_port_mapping_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_port_ip ON host_port_mapping_snapshot(scan_id, port, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_ip ON host_port_mapping_snapshot(scan_id, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_host_ip ON host_port_mapping_snapshot(scan_id, host, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_host_trgm ON host_port_mapping_snapshot USING GIN (host gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_created_at_ip ON host_port_mapping_snapshot(scan_id, created_at, ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan_effective_host_port ON host_port_mapping_snapshot(scan_id, (COALESCE(NULLIF(LOWER(BTRIM(host)), ''), HOST(ip))), port);
CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_host_ip_port_snapshot ON host_port_mapping_snapshot(scan_id, host, ip, port);

-- website_snapshot
CREATE TABLE IF NOT EXISTS website_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    status_code INTEGER,
    content_length INTEGER,
    location TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
    response_body TEXT NOT NULL DEFAULT '',
    vhost BOOLEAN,
    response_headers TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan ON website_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_website_snap_url ON website_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_website_snap_host ON website_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_website_snap_title ON website_snapshot(title);
CREATE INDEX IF NOT EXISTS idx_website_snap_created_at ON website_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_created_at_id ON website_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_website_snap_url_trgm ON website_snapshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_status_code_id ON website_snapshot(scan_id, status_code, id);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_content_length_id ON website_snapshot(scan_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_webserver_id ON website_snapshot(scan_id, webserver, id);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_content_type_id ON website_snapshot(scan_id, content_type, id);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan_vhost_id ON website_snapshot(scan_id, vhost, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_website_per_scan_snapshot ON website_snapshot(scan_id, url);


-- endpoint_snapshot
CREATE TABLE IF NOT EXISTS endpoint_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    status_code INTEGER,
    content_length INTEGER,
    location TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
	response_body TEXT NOT NULL DEFAULT '',
	response_body_truncated BOOLEAN NOT NULL DEFAULT FALSE,
	vhost BOOLEAN,
	response_headers TEXT NOT NULL DEFAULT '',
	response_headers_truncated BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan ON endpoint_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_url ON endpoint_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_host ON endpoint_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_title ON endpoint_snapshot(title);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_status_code ON endpoint_snapshot(status_code);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_webserver ON endpoint_snapshot(webserver);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_created_at ON endpoint_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_created_at_id ON endpoint_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_url_trgm ON endpoint_snapshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_status_code_id ON endpoint_snapshot(scan_id, status_code, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_content_length_id ON endpoint_snapshot(scan_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_webserver_id ON endpoint_snapshot(scan_id, webserver, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_content_type_id ON endpoint_snapshot(scan_id, content_type, id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan_vhost_id ON endpoint_snapshot(scan_id, vhost, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_endpoint_per_scan_snapshot ON endpoint_snapshot(scan_id, url);

-- directory_snapshot
CREATE TABLE IF NOT EXISTS directory_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status INTEGER,
    content_length BIGINT,
    content_type VARCHAR(1024) NOT NULL DEFAULT '',
    duration BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan ON directory_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_url ON directory_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_directory_snap_status ON directory_snapshot(status);
CREATE INDEX IF NOT EXISTS idx_directory_snap_content_type ON directory_snapshot(content_type);
CREATE INDEX IF NOT EXISTS idx_directory_snap_created_at ON directory_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_created_at_id ON directory_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_url_trgm ON directory_snapshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_status_id ON directory_snapshot(scan_id, status, id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_content_length_id ON directory_snapshot(scan_id, content_length, id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan_content_type_id ON directory_snapshot(scan_id, content_type, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_directory_per_scan_snapshot ON directory_snapshot(scan_id, url);

-- screenshot_snapshot
CREATE TABLE IF NOT EXISTS screenshot_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status_code SMALLINT,
    image BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan ON screenshot_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_created_at ON screenshot_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan_created_at_id ON screenshot_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_url_trgm ON screenshot_snapshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan_status_code_id ON screenshot_snapshot(scan_id, status_code, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_screenshot_per_scan_snapshot ON screenshot_snapshot(scan_id, url);

-- vulnerability_snapshot
CREATE TABLE IF NOT EXISTS vulnerability_snapshot (
    id SERIAL NOT NULL,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL DEFAULT '',
    vuln_type VARCHAR(200) NOT NULL DEFAULT '',
    severity VARCHAR(20) NOT NULL DEFAULT 'unknown',
    source VARCHAR(100) NOT NULL DEFAULT '',
    cvss_score DECIMAL(3,1) NOT NULL DEFAULT 0.0,
    description TEXT NOT NULL DEFAULT '',
    raw_output JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id)
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan ON vulnerability_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_url ON vulnerability_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_url_trgm ON vulnerability_snapshot USING GIN (url gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_type ON vulnerability_snapshot(vuln_type);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_severity ON vulnerability_snapshot(severity);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_source ON vulnerability_snapshot(source);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_created_at ON vulnerability_snapshot(created_at);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_created_at_id ON vulnerability_snapshot(scan_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_severity_id ON vulnerability_snapshot(scan_id, severity, id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_source_id ON vulnerability_snapshot(scan_id, source, id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan_type_id ON vulnerability_snapshot(scan_id, vuln_type, id);
CREATE UNIQUE INDEX IF NOT EXISTS unique_vulnerability_observation_per_scan_snapshot
    ON vulnerability_snapshot(scan_id, url, vuln_type, severity);

-- Initial and next aligned scan history ranges. The lifecycle service creates
-- further 10,000-ID ranges before the scan sequence reaches them.
CREATE TABLE IF NOT EXISTS subdomain_snapshot_p00000000 PARTITION OF subdomain_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS subdomain_snapshot_p00010000 PARTITION OF subdomain_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS website_snapshot_p00000000 PARTITION OF website_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS website_snapshot_p00010000 PARTITION OF website_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS endpoint_snapshot_p00000000 PARTITION OF endpoint_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS endpoint_snapshot_p00010000 PARTITION OF endpoint_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS directory_snapshot_p00000000 PARTITION OF directory_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS directory_snapshot_p00010000 PARTITION OF directory_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS host_port_mapping_snapshot_p00000000 PARTITION OF host_port_mapping_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS host_port_mapping_snapshot_p00010000 PARTITION OF host_port_mapping_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS screenshot_snapshot_p00000000 PARTITION OF screenshot_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS screenshot_snapshot_p00010000 PARTITION OF screenshot_snapshot FOR VALUES FROM (10000) TO (20000);
CREATE TABLE IF NOT EXISTS vulnerability_snapshot_p00000000 PARTITION OF vulnerability_snapshot FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS vulnerability_snapshot_p00010000 PARTITION OF vulnerability_snapshot FOR VALUES FROM (10000) TO (20000);

-- ============================================
-- Statistics tables
-- ============================================

-- asset_statistics (singleton)
CREATE TABLE IF NOT EXISTS asset_statistics (
    id SERIAL PRIMARY KEY,
    total_targets INTEGER NOT NULL DEFAULT 0,
    total_subdomains INTEGER NOT NULL DEFAULT 0,
    total_ips INTEGER NOT NULL DEFAULT 0,
    total_endpoints INTEGER NOT NULL DEFAULT 0,
    total_websites INTEGER NOT NULL DEFAULT 0,
    total_vulns INTEGER NOT NULL DEFAULT 0,
    total_assets INTEGER NOT NULL DEFAULT 0,
    prev_targets INTEGER NOT NULL DEFAULT 0,
    prev_subdomains INTEGER NOT NULL DEFAULT 0,
    prev_ips INTEGER NOT NULL DEFAULT 0,
    prev_endpoints INTEGER NOT NULL DEFAULT 0,
    prev_websites INTEGER NOT NULL DEFAULT 0,
    prev_vulns INTEGER NOT NULL DEFAULT 0,
    prev_assets INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- statistics_history
CREATE TABLE IF NOT EXISTS statistics_history (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    total_targets INTEGER NOT NULL DEFAULT 0,
    total_subdomains INTEGER NOT NULL DEFAULT 0,
    total_ips INTEGER NOT NULL DEFAULT 0,
    total_endpoints INTEGER NOT NULL DEFAULT 0,
    total_websites INTEGER NOT NULL DEFAULT 0,
    total_vulns INTEGER NOT NULL DEFAULT 0,
    total_assets INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_statistics_date ON statistics_history(date);
CREATE INDEX IF NOT EXISTS idx_statistics_date ON statistics_history(date);

-- Notification is a disposable-development hard cut. These tables replace the
-- old singleton settings and global read-state shape; no legacy row is copied,
-- dual-written, or compatibly read.
CREATE TABLE IF NOT EXISTS notification_outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL UNIQUE,
    kind VARCHAR(64) NOT NULL CHECK (kind IN ('scan-succeeded', 'scan-failed', 'vulnerability-observed', 'agent-offline', 'nuclei-poc-sync-succeeded', 'nuclei-poc-sync-failed')),
    payload_version INTEGER NOT NULL CHECK (payload_version > 0),
    subject_name VARCHAR(500) NOT NULL CHECK (btrim(subject_name) <> ''),
    occurred_at TIMESTAMPTZ NOT NULL,
    priority VARCHAR(16) NOT NULL CHECK (priority IN ('normal', 'high', 'critical')),
    payload JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'published', 'unsupported_terminal')),
    available_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_owner VARCHAR(128),
    lease_expires_at TIMESTAMPTZ,
    failure_code VARCHAR(128) NOT NULL DEFAULT '',
    terminal_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_claim
    ON notification_outbox(available_at, id)
    WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_notification_outbox_published_retention
    ON notification_outbox(published_at, id)
    WHERE status = 'published';

CREATE TABLE IF NOT EXISTS notification_fact (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL UNIQUE,
    kind VARCHAR(64) NOT NULL CHECK (kind IN ('scan-succeeded', 'scan-failed', 'vulnerability-observed', 'agent-offline', 'nuclei-poc-sync-succeeded', 'nuclei-poc-sync-failed')),
    payload_version INTEGER NOT NULL CHECK (payload_version > 0),
    subject_name VARCHAR(500) NOT NULL CHECK (btrim(subject_name) <> ''),
    category VARCHAR(32) NOT NULL CHECK (category IN ('scan', 'vulnerability', 'system')),
    priority VARCHAR(16) NOT NULL CHECK (priority IN ('normal', 'high', 'critical')),
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    audience_frozen_at TIMESTAMPTZ,
    destinations_frozen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_notification_fact_created_at ON notification_fact(created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS notification_recipient (
    id BIGSERIAL PRIMARY KEY,
    fact_id BIGINT NOT NULL REFERENCES notification_fact(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES auth_user(id),
    locale VARCHAR(2) NOT NULL CHECK (locale IN ('zh', 'en')),
    category VARCHAR(32) NOT NULL CHECK (category IN ('scan', 'vulnerability', 'system')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (fact_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_notification_recipient_user ON notification_recipient(user_id, id);

CREATE TABLE IF NOT EXISTS notification_inbox (
    id BIGSERIAL PRIMARY KEY,
    fact_id BIGINT NOT NULL REFERENCES notification_fact(id) ON DELETE CASCADE,
    recipient_id BIGINT NOT NULL REFERENCES notification_recipient(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES auth_user(id),
    name VARCHAR(500) NOT NULL UNIQUE,
    kind VARCHAR(64) NOT NULL CHECK (kind IN ('scan-succeeded', 'scan-failed', 'vulnerability-observed', 'agent-offline', 'nuclei-poc-sync-succeeded', 'nuclei-poc-sync-failed')),
    category VARCHAR(32) NOT NULL CHECK (category IN ('scan', 'vulnerability', 'system')),
    priority VARCHAR(16) NOT NULL CHECK (priority IN ('normal', 'high', 'critical')),
    subject_name VARCHAR(500) NOT NULL,
    locale VARCHAR(2) NOT NULL CHECK (locale IN ('zh', 'en')),
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMPTZ,
    UNIQUE (fact_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_notification_inbox_user_visibility
    ON notification_inbox(user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_notification_inbox_user_unread
    ON notification_inbox(user_id, created_at DESC, id DESC)
    WHERE read_at IS NULL;

CREATE TABLE IF NOT EXISTS notification_destination (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(16) NOT NULL UNIQUE CHECK (provider IN ('discord', 'wecom', 'feishu')),
    credential TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS notification_destination_subscription (
    destination_id BIGINT NOT NULL REFERENCES notification_destination(id) ON DELETE CASCADE,
    kind VARCHAR(64) NOT NULL CHECK (kind IN ('scan-succeeded', 'scan-failed', 'vulnerability-observed', 'agent-offline')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (destination_id, kind)
);
INSERT INTO notification_destination (provider, credential, enabled)
VALUES ('discord', '', FALSE), ('wecom', '', FALSE), ('feishu', '', FALSE)
ON CONFLICT (provider) DO NOTHING;

CREATE TABLE IF NOT EXISTS notification_delivery (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL,
    fact_id BIGINT NOT NULL REFERENCES notification_fact(id) ON DELETE CASCADE,
    destination_id BIGINT NOT NULL REFERENCES notification_destination(id) ON DELETE CASCADE,
    provider VARCHAR(16) NOT NULL CHECK (provider IN ('discord', 'wecom', 'feishu')),
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'retrying', 'delivered', 'failed_terminal')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    first_attempt_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    lease_owner VARCHAR(128),
    lease_expires_at TIMESTAMPTZ,
    locale VARCHAR(2) NOT NULL CHECK (locale IN ('zh', 'en')),
    template_version INTEGER NOT NULL DEFAULT 1,
    title VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    provider_snapshot JSONB NOT NULL CHECK (jsonb_typeof(provider_snapshot) = 'object'),
    terminal_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, destination_id)
);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_claim
    ON notification_delivery(next_attempt_at, id)
    WHERE status IN ('pending', 'processing', 'retrying');
CREATE INDEX IF NOT EXISTS idx_notification_delivery_terminal_retention
    ON notification_delivery(terminal_at, id)
    WHERE status IN ('delivered', 'failed_terminal');

CREATE TABLE IF NOT EXISTS notification_delivery_attempt (
    id BIGSERIAL PRIMARY KEY,
    delivery_id BIGINT NOT NULL REFERENCES notification_delivery(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL CHECK (attempt_number > 0),
    attempted_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    http_status INTEGER,
    error_class VARCHAR(128) NOT NULL DEFAULT '',
    provider_request_id VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (delivery_id, attempt_number)
);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_attempt_delivery
    ON notification_delivery_attempt(delivery_id, attempt_number);

-- ============================================
-- GIN Indexes for array fields
-- ============================================

-- GIN index for website.tech array
CREATE INDEX IF NOT EXISTS idx_website_tech_gin ON website USING GIN (tech);
CREATE INDEX IF NOT EXISTS idx_website_snap_tech_gin ON website_snapshot USING GIN (tech);

-- GIN index for endpoint.tech array
CREATE INDEX IF NOT EXISTS idx_endpoint_tech_gin ON endpoint USING GIN (tech);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_tech_gin ON endpoint_snapshot USING GIN (tech);

-- GIN index for scan.container_ids array
CREATE INDEX IF NOT EXISTS idx_scan_container_ids_gin ON scan USING GIN (container_ids);

-- ============================================
-- Seed data
-- ============================================

-- Default admin user (password: admin)
-- Password hash generated with bcrypt
INSERT INTO auth_user (username, password, is_superuser, is_staff, is_active, date_joined)
VALUES ('admin', '$2b$12$.4wL49eZfJuwVjP85Qxa7.xFb7HE3TDer4wcF9Z7c.oTOo7fExlgq', TRUE, TRUE, TRUE, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO NOTHING;

-- Insert default row with all providers disabled
INSERT INTO subfinder_provider_settings (id, providers) VALUES (1, '{
    "alienvault": {"enabled": false, "status": "unconfigured", "values": {}},
    "bevigil": {"enabled": false, "status": "unconfigured", "values": {}},
    "bufferover": {"enabled": false, "status": "unconfigured", "values": {}},
    "builtwith": {"enabled": false, "status": "unconfigured", "values": {}},
    "c99": {"enabled": false, "status": "unconfigured", "values": {}},
    "censys": {"enabled": false, "status": "unconfigured", "values": {}},
    "certspotter": {"enabled": false, "status": "unconfigured", "values": {}},
    "chaos": {"enabled": false, "status": "unconfigured", "values": {}},
    "chinaz": {"enabled": false, "status": "unconfigured", "values": {}},
    "digitalyama": {"enabled": false, "status": "unconfigured", "values": {}},
    "dnsdb": {"enabled": false, "status": "unconfigured", "values": {}},
    "dnsdumpster": {"enabled": false, "status": "unconfigured", "values": {}},
    "dnsrepo": {"enabled": false, "status": "unconfigured", "values": {}},
    "domainsproject": {"enabled": false, "status": "unconfigured", "values": {}},
    "driftnet": {"enabled": false, "status": "unconfigured", "values": {}},
    "facebook": {"enabled": false, "status": "unconfigured", "values": {}},
    "fofa": {"enabled": false, "status": "unconfigured", "values": {}},
    "fullhunt": {"enabled": false, "status": "unconfigured", "values": {}},
    "github": {"enabled": false, "status": "unconfigured", "values": {}},
    "intelx": {"enabled": false, "status": "unconfigured", "values": {}},
    "merklemap": {"enabled": false, "status": "unconfigured", "values": {}},
    "netlas": {"enabled": false, "status": "unconfigured", "values": {}},
    "onyphe": {"enabled": false, "status": "unconfigured", "values": {}},
    "profundis": {"enabled": false, "status": "unconfigured", "values": {}},
    "pugrecon": {"enabled": false, "status": "unconfigured", "values": {}},
    "quake": {"enabled": false, "status": "unconfigured", "values": {}},
    "redhuntlabs": {"enabled": false, "status": "unconfigured", "values": {}},
    "robtex": {"enabled": false, "status": "unconfigured", "values": {}},
    "rsecloud": {"enabled": false, "status": "unconfigured", "values": {}},
    "securitytrails": {"enabled": false, "status": "unconfigured", "values": {}},
    "shodan": {"enabled": false, "status": "unconfigured", "values": {}},
    "threatbook": {"enabled": false, "status": "unconfigured", "values": {}},
    "virustotal": {"enabled": false, "status": "unconfigured", "values": {}},
    "whoisxmlapi": {"enabled": false, "status": "unconfigured", "values": {}},
    "windvane": {"enabled": false, "status": "unconfigured", "values": {}},
    "zoomeyeapi": {"enabled": false, "status": "unconfigured", "values": {}},
    "hackertarget": {"enabled": false, "status": "unconfigured", "values": {}},
    "leakix": {"enabled": false, "status": "unconfigured", "values": {}},
    "reconeer": {"enabled": false, "status": "unconfigured", "values": {}}
}'::jsonb) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- Agent runtime system tables
-- ============================================

-- registration_token
CREATE TABLE IF NOT EXISTS registration_token (
    id                  SERIAL PRIMARY KEY,
    token               VARCHAR(8) NOT NULL UNIQUE,
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 hour'),
    ever_attributed_at  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

-- agent
CREATE TABLE IF NOT EXISTS agent (
    id              SERIAL PRIMARY KEY,
    instance_id     VARCHAR(64) NOT NULL UNIQUE,
    display_name    VARCHAR(100) NOT NULL,
    authentication_token         VARCHAR(8) NOT NULL UNIQUE,
    status          VARCHAR(20) DEFAULT 'offline',

    -- Scheduling configuration (can be dynamically modified via API)
    max_tasks       INT DEFAULT 5,
    cpu_threshold   INT DEFAULT 85,
    mem_threshold   INT DEFAULT 85,
    disk_threshold  INT DEFAULT 90,

    -- Stable non-secret registration attribution.
    registration_token_id  INT NOT NULL REFERENCES registration_token(id) ON DELETE RESTRICT,

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_runtime_status (
    agent_id            INT PRIMARY KEY REFERENCES agent(id) ON DELETE CASCADE,
    session_id          VARCHAR(64),
    session_epoch       BIGINT DEFAULT 0,
    observed_hostname   VARCHAR(255),
    connection_ip       VARCHAR(45) NOT NULL DEFAULT '',
    observed_source_ip  VARCHAR(45) NOT NULL DEFAULT '',
    observed_ip_generation BIGINT NOT NULL DEFAULT 0,
    agent_version       VARCHAR(20),
    operating_system    VARCHAR(32),
    architecture        VARCHAR(32),
    container_runtime_ready BOOLEAN NOT NULL DEFAULT FALSE,
    supported_engine_api_majors JSONB NOT NULL DEFAULT '[]'::jsonb,
    connected_at        TIMESTAMPTZ,
    last_heartbeat      TIMESTAMPTZ,
    health_state        VARCHAR(20) DEFAULT 'healthy',
    health_reason       VARCHAR(64),
    health_message      VARCHAR(256),
    health_since        TIMESTAMPTZ,
    cpu_usage           DOUBLE PRECISION DEFAULT 0,
    mem_usage           DOUBLE PRECISION DEFAULT 0,
    disk_usage          DOUBLE PRECISION DEFAULT 0,
    running_tasks       INT DEFAULT 0,
    task_slots_used     INT DEFAULT 0,
    uptime_seconds      BIGINT DEFAULT 0,
    updated_at          TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT agent_runtime_status_observed_ip_generation_nonnegative CHECK (observed_ip_generation >= 0)
);

CREATE TABLE IF NOT EXISTS agent_location (
    agent_id             INT PRIMARY KEY REFERENCES agent(id) ON DELETE CASCADE,
    latitude             DOUBLE PRECISION NOT NULL,
    longitude            DOUBLE PRECISION NOT NULL,
    accuracy_radius_km   DOUBLE PRECISION,
    source_observed_ip   VARCHAR(45) NOT NULL,
    provider_key         VARCHAR(32) NOT NULL,
    resolved_at          TIMESTAMPTZ NOT NULL,
    forced_expired       BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT agent_location_latitude_range CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT agent_location_longitude_range CHECK (longitude >= -180 AND longitude <= 180),
    CONSTRAINT agent_location_accuracy_nonnegative CHECK (accuracy_radius_km IS NULL OR accuracy_radius_km >= 0),
    CONSTRAINT agent_location_source_nonempty CHECK (source_observed_ip <> ''),
    CONSTRAINT agent_location_provider_nonempty CHECK (provider_key <> '')
);

CREATE TABLE IF NOT EXISTS server_location_snapshot (
    singleton_id         SMALLINT PRIMARY KEY DEFAULT 1,
    observed_egress_ip   VARCHAR(45) NOT NULL,
    latitude             DOUBLE PRECISION NOT NULL,
    longitude            DOUBLE PRECISION NOT NULL,
    accuracy_radius_km   DOUBLE PRECISION,
    provider_key         VARCHAR(32) NOT NULL,
    resolved_at          TIMESTAMPTZ NOT NULL,
    forced_expired       BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT server_location_snapshot_singleton CHECK (singleton_id = 1),
    CONSTRAINT server_location_snapshot_latitude_range CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT server_location_snapshot_longitude_range CHECK (longitude >= -180 AND longitude <= 180),
    CONSTRAINT server_location_snapshot_accuracy_nonnegative CHECK (accuracy_radius_km IS NULL OR accuracy_radius_km >= 0),
    CONSTRAINT server_location_snapshot_source_nonempty CHECK (observed_egress_ip <> ''),
    CONSTRAINT server_location_snapshot_provider_nonempty CHECK (provider_key <> '')
);

-- Indexes for agent table
CREATE INDEX IF NOT EXISTS idx_agent_status ON agent(status);
CREATE INDEX IF NOT EXISTS idx_agent_display_name_trgm ON agent USING GIN (display_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_agent_created_at_id ON agent(created_at, id);
CREATE INDEX IF NOT EXISTS idx_agent_authentication_token ON agent(authentication_token);
CREATE INDEX IF NOT EXISTS idx_agent_registration_token_id ON agent(registration_token_id);
CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_last_heartbeat ON agent_runtime_status(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_observed_hostname_trgm ON agent_runtime_status USING GIN (observed_hostname gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_connection_ip_trgm ON agent_runtime_status USING GIN (connection_ip gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_observed_source_ip_trgm ON agent_runtime_status USING GIN (observed_source_ip gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_agent_runtime_status_health_state_agent_id ON agent_runtime_status(health_state, agent_id);

-- Indexes for registration_token table
CREATE INDEX IF NOT EXISTS idx_registration_token_token ON registration_token(token);
CREATE INDEX IF NOT EXISTS idx_registration_token_expires ON registration_token(expires_at);

-- task_progress_log
CREATE TABLE IF NOT EXISTS task_progress_log (
    id BIGSERIAL NOT NULL,
    scan_id INTEGER NOT NULL,
    task_id INTEGER NOT NULL,
    request_id VARCHAR(100) NOT NULL DEFAULT '',
    sequence BIGINT NOT NULL DEFAULT 0,
    level VARCHAR(10) NOT NULL DEFAULT 'info',
    content TEXT NOT NULL DEFAULT '',
    emitted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (scan_id, id),
    CONSTRAINT task_progress_log_task_scan_fk
        FOREIGN KEY (scan_id, task_id) REFERENCES scan_task(scan_id, id) ON DELETE CASCADE
) PARTITION BY RANGE (scan_id);
CREATE INDEX IF NOT EXISTS idx_task_progress_log_task ON task_progress_log(scan_id, task_id, id);
CREATE INDEX IF NOT EXISTS idx_task_progress_log_created_at ON task_progress_log(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_task_progress_log_request ON task_progress_log(scan_id, task_id, request_id, sequence);
CREATE TABLE IF NOT EXISTS task_progress_log_p00000000 PARTITION OF task_progress_log
    FOR VALUES FROM (0) TO (10000);
CREATE TABLE IF NOT EXISTS task_progress_log_p00010000 PARTITION OF task_progress_log
    FOR VALUES FROM (10000) TO (20000);

-- ============================================
-- Global fingerprint libraries
-- ============================================
-- resource_id values are generated by the Go application. The database keeps
-- no UUID default so an import cannot accidentally replace an existing
-- canonical resource identity during an identity-key conflict update.

CREATE TABLE IF NOT EXISTS fingerprint_library_state (
    library TEXT NOT NULL PRIMARY KEY,
    source_generation BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fingerprint_library_state_generation_nonnegative CHECK (source_generation >= 0)
);
INSERT INTO fingerprint_library_state (library, source_generation)
VALUES ('fingerprinthub', 0)
ON CONFLICT (library) DO NOTHING;

CREATE TABLE IF NOT EXISTS fingerprint_library_artifact (
    id BIGSERIAL PRIMARY KEY,
    library TEXT NOT NULL,
    source_generation BIGINT NOT NULL,
    sha256_digest TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    record_count BIGINT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fingerprint_library_artifact_generation_key UNIQUE (library, source_generation),
    CONSTRAINT fingerprint_library_artifact_size_nonnegative CHECK (size_bytes >= 0),
    CONSTRAINT fingerprint_library_artifact_count_nonnegative CHECK (record_count >= 0),
    CONSTRAINT fingerprint_library_artifact_digest_format CHECK (sha256_digest ~ '^sha256:[0-9a-f]{64}$')
);
CREATE INDEX IF NOT EXISTS idx_fingerprint_library_artifact_digest ON fingerprint_library_artifact(sha256_digest);

CREATE TABLE IF NOT EXISTS fingerprint_fingerprinthub (
    resource_id UUID NOT NULL PRIMARY KEY,
    identity_key TEXT NOT NULL,
    content_hash BYTEA NOT NULL,
    payload JSONB NOT NULL,
    fingerprint_id TEXT NOT NULL,
    name TEXT NOT NULL,
    severity TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fingerprint_fingerprinthub_identity_key_key UNIQUE (identity_key)
);
CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_name_trgm ON fingerprint_fingerprinthub USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_severity ON fingerprint_fingerprinthub(severity);
CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_created_at_resource_id ON fingerprint_fingerprinthub(created_at DESC, resource_id DESC);
CREATE INDEX IF NOT EXISTS idx_fingerprint_fingerprinthub_name_resource_id ON fingerprint_fingerprinthub(name, resource_id);

-- Comments for agent tables
COMMENT ON TABLE agent IS 'Agent metadata and configuration';
COMMENT ON TABLE registration_token IS 'Registration tokens for agent self-registration';
COMMENT ON TABLE agent_location IS 'Last successful Agent GeoIP snapshots inferred from observed public egress';
COMMENT ON TABLE server_location_snapshot IS 'Last successful single-Server outbound egress GeoIP snapshot';
COMMENT ON COLUMN agent.status IS 'Agent status: online/offline';
COMMENT ON COLUMN agent.authentication_token IS '8-character hex string for authentication';
COMMENT ON COLUMN agent.registration_token_id IS 'Stable non-secret registration-token attribution';
COMMENT ON COLUMN agent_runtime_status.connection_ip IS 'Current Agent control-connection address resolved by Server; it may be private and is never a reachability guarantee';
COMMENT ON COLUMN agent_runtime_status.observed_source_ip IS 'Strict public projection of the current connection observation, used only for GeoIP provenance and never Agent-reported';
COMMENT ON COLUMN agent_runtime_status.observed_ip_generation IS 'Monotonic fence incremented whenever the normalized observed source changes';
COMMENT ON COLUMN registration_token.expires_at IS 'Token expiration time (default 1 hour after creation)';
COMMENT ON COLUMN registration_token.ever_attributed_at IS 'First successful Agent attribution marker retained after Agent deletion';
COMMENT ON COLUMN scan.scan_workflow_id IS 'Scan workflow ID (e.g. subdomain_discovery)';
COMMENT ON COLUMN scheduled_scan.scan_workflow_id IS 'Scheduled scan workflow ID (e.g. subdomain_discovery)';
COMMENT ON COLUMN scan.configuration IS 'Canonical workflow configuration root stored as JSON object';
COMMENT ON COLUMN scan.failure_kind IS 'Machine-readable scan failure summary projected from canonical failed task';
COMMENT ON TABLE scan_task IS 'Executable scan task source for workflow step engine execution';
COMMENT ON COLUMN scan_task.task_execution_config IS 'Canonical per-task operation config stored as JSON object';
COMMENT ON COLUMN scan_task.resolved_execution_plan IS 'Immutable protobuf execution plan compiled and persisted at scan creation; empty only for planning-time skipped tasks after the scan-create transaction commits';
COMMENT ON COLUMN scan_task.assigned_session_epoch IS 'Server-issued claim epoch bound after planning; never part of the immutable execution plan';
COMMENT ON COLUMN scan_task.assigned_request_id IS 'Session-scoped RequestTask correlation persisted with the claim; never part of the immutable execution plan';
COMMENT ON COLUMN scan_task.status IS 'Task status: blocked/pending/running/succeeded/skipped/failed/cancelled';
COMMENT ON COLUMN scan_task.error_message IS 'Error message (truncated by Agent, max 4KB)';
COMMENT ON COLUMN scan_task.failure_kind IS 'Machine-readable task failure classification for assignment/bootstrap/runtime failures';
COMMENT ON COLUMN scan_task.failure_detail IS 'Optional controlled customer-facing diagnostic for a task failure';
COMMENT ON COLUMN scan_task.engine_diagnostics IS 'Bounded Agent-owned Engine terminal diagnostic snapshot; null before terminal submission and never a log or result-item store';
COMMENT ON COLUMN scan_task.skip_reason IS 'Non-failure reason explaining why an intentionally non-applicable workflow task was skipped';
COMMENT ON INDEX idx_scan_task_pending_order IS 'Supports task pull queries over executable scan tasks';
