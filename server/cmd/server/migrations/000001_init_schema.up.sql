-- Initial schema migration for Go backend
-- This migration creates all tables from scratch

-- ============================================
-- Core tables (no dependencies)
-- ============================================

-- auth_user (Django compatible)
CREATE TABLE IF NOT EXISTS auth_user (
    id SERIAL PRIMARY KEY,
    password VARCHAR(128) NOT NULL,
    last_login TIMESTAMPTZ,
    is_superuser BOOLEAN NOT NULL DEFAULT FALSE,
    username VARCHAR(150) NOT NULL,
    first_name VARCHAR(150) NOT NULL DEFAULT '',
    last_name VARCHAR(150) NOT NULL DEFAULT '',
    email VARCHAR(254) NOT NULL DEFAULT '',
    is_staff BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    date_joined TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_user_username ON auth_user(username);

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

-- organization_target (many-to-many)
CREATE TABLE IF NOT EXISTS organization_target (
    organization_id INTEGER NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    PRIMARY KEY (organization_id, target_id)
);
CREATE INDEX IF NOT EXISTS idx_org_target_org ON organization_target(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_target_target ON organization_target(target_id);

-- scan_engine
CREATE TABLE IF NOT EXISTS scan_engine (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    configuration VARCHAR(10000) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_engine_name ON scan_engine(name);
CREATE INDEX IF NOT EXISTS idx_scan_engine_created_at ON scan_engine(created_at);


-- worker_node
CREATE TABLE IF NOT EXISTS worker_node (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ip_address INET,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    username VARCHAR(50) NOT NULL DEFAULT 'root',
    password VARCHAR(200) NOT NULL DEFAULT '',
    is_local BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_worker_name ON worker_node(name);
CREATE INDEX IF NOT EXISTS idx_worker_node_created_at ON worker_node(created_at);

-- wordlist
CREATE TABLE IF NOT EXISTS wordlist (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description VARCHAR(200) NOT NULL DEFAULT '',
    file_path VARCHAR(500) NOT NULL DEFAULT '',
    file_size BIGINT NOT NULL DEFAULT 0,
    line_count INTEGER NOT NULL DEFAULT 0,
    file_hash VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_wordlist_name ON wordlist(name);
CREATE INDEX IF NOT EXISTS idx_wordlist_created_at ON wordlist(created_at);

-- nuclei_template_repo
CREATE TABLE IF NOT EXISTS nuclei_template_repo (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    repo_url VARCHAR(500) NOT NULL DEFAULT '',
    local_path VARCHAR(500) NOT NULL DEFAULT '',
    commit_hash VARCHAR(40) NOT NULL DEFAULT '',
    last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS unique_nuclei_repo_name ON nuclei_template_repo(name);

-- blacklist_rule
CREATE TABLE IF NOT EXISTS blacklist_rule (
    id SERIAL PRIMARY KEY,
    pattern VARCHAR(255) NOT NULL,
    rule_type VARCHAR(20) NOT NULL,
    scope VARCHAR(20) NOT NULL DEFAULT 'global',
    target_id INTEGER REFERENCES target(id) ON DELETE CASCADE,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_blacklist_scope ON blacklist_rule(scope);
CREATE INDEX IF NOT EXISTS idx_blacklist_target ON blacklist_rule(target_id);
CREATE INDEX IF NOT EXISTS idx_blacklist_created_at ON blacklist_rule(created_at);

-- notification_settings (singleton)
CREATE TABLE IF NOT EXISTS notification_settings (
    id SERIAL PRIMARY KEY,
    discord_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    discord_webhook_url VARCHAR(500) NOT NULL DEFAULT '',
    wecom_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    wecom_webhook_url VARCHAR(500) NOT NULL DEFAULT '',
    categories JSONB NOT NULL DEFAULT '{"scan": true, "vulnerability": true, "asset": true, "system": false}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- subfinder_provider_settings (singleton)
CREATE TABLE IF NOT EXISTS subfinder_provider_settings (
    id SERIAL PRIMARY KEY,
    providers JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- Scan related tables (depends on target)
-- ============================================

-- scan
CREATE TABLE IF NOT EXISTS scan (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    engine_ids INTEGER[] NOT NULL DEFAULT '{}',
    engine_names JSONB NOT NULL DEFAULT '[]',
    yaml_configuration TEXT NOT NULL DEFAULT '',
    scan_mode VARCHAR(10) NOT NULL DEFAULT 'full',
    status VARCHAR(20) NOT NULL DEFAULT 'initiated',
    results_dir VARCHAR(100) NOT NULL DEFAULT '',
    container_ids VARCHAR(100)[] NOT NULL DEFAULT '{}',
    worker_id INTEGER,
    error_message VARCHAR(2000) NOT NULL DEFAULT '',
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


-- scan_input_target
CREATE TABLE IF NOT EXISTS scan_input_target (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    value VARCHAR(2000) NOT NULL,
    input_type VARCHAR(10) NOT NULL DEFAULT 'domain',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_input_target_scan ON scan_input_target(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_input_target_type ON scan_input_target(input_type);

-- scan_log
CREATE TABLE IF NOT EXISTS scan_log (
    id BIGSERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    level VARCHAR(10) NOT NULL DEFAULT 'info',
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_log_scan ON scan_log(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_log_created_at ON scan_log(created_at);

-- scheduled_scan
CREATE TABLE IF NOT EXISTS scheduled_scan (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    engine_ids INTEGER[] NOT NULL DEFAULT '{}',
    engine_names JSONB NOT NULL DEFAULT '[]',
    yaml_configuration TEXT NOT NULL DEFAULT '',
    organization_id INTEGER REFERENCES organization(id) ON DELETE SET NULL,
    target_id INTEGER REFERENCES target(id) ON DELETE SET NULL,
    cron_expression VARCHAR(100) NOT NULL DEFAULT '0 2 * * *',
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    run_count INTEGER NOT NULL DEFAULT 0,
    last_run_time TIMESTAMPTZ,
    next_run_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_name ON scheduled_scan(name);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_enabled ON scheduled_scan(is_enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_scan_created_at ON scheduled_scan(created_at);

-- ============================================
-- Asset tables (depends on target)
-- ============================================

-- subdomain
CREATE TABLE IF NOT EXISTS subdomain (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    name VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subdomain_target ON subdomain(target_id);
CREATE INDEX IF NOT EXISTS idx_subdomain_name ON subdomain(name);
CREATE INDEX IF NOT EXISTS idx_subdomain_created_at ON subdomain(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_subdomain_name_target ON subdomain(name, target_id);

-- host_port_mapping
CREATE TABLE IF NOT EXISTS host_port_mapping (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    host VARCHAR(1000) NOT NULL,
    ip INET NOT NULL,
    port INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_hpm_target ON host_port_mapping(target_id);
CREATE INDEX IF NOT EXISTS idx_hpm_host ON host_port_mapping(host);
CREATE INDEX IF NOT EXISTS idx_hpm_ip ON host_port_mapping(ip);
CREATE INDEX IF NOT EXISTS idx_hpm_port ON host_port_mapping(port);
CREATE INDEX IF NOT EXISTS idx_hpm_created_at ON host_port_mapping(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_target_host_ip_port ON host_port_mapping(target_id, host, ip, port);

-- website
CREATE TABLE IF NOT EXISTS website (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT '',
    tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
    status_code INTEGER,
    content_length INTEGER,
    vhost BOOLEAN,
    response_headers TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_website_target ON website(target_id);
CREATE INDEX IF NOT EXISTS idx_website_url ON website(url);
CREATE INDEX IF NOT EXISTS idx_website_host ON website(host);
CREATE INDEX IF NOT EXISTS idx_website_title ON website(title);
CREATE INDEX IF NOT EXISTS idx_website_status_code ON website(status_code);
CREATE INDEX IF NOT EXISTS idx_website_created_at ON website(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_website_url_target ON website(url, target_id);

-- endpoint
CREATE TABLE IF NOT EXISTS endpoint (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    host VARCHAR(253) NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    title TEXT NOT NULL DEFAULT '',
    webserver TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	tech VARCHAR(100)[] NOT NULL DEFAULT '{}',
	status_code INTEGER,
	content_length INTEGER,
	vhost BOOLEAN,
	response_headers TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_endpoint_target ON endpoint(target_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_url ON endpoint(url);
CREATE INDEX IF NOT EXISTS idx_endpoint_host ON endpoint(host);
CREATE INDEX IF NOT EXISTS idx_endpoint_title ON endpoint(title);
CREATE INDEX IF NOT EXISTS idx_endpoint_status_code ON endpoint(status_code);
CREATE INDEX IF NOT EXISTS idx_endpoint_created_at ON endpoint(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_endpoint_url_target ON endpoint(url, target_id);


-- directory
CREATE TABLE IF NOT EXISTS directory (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status INTEGER,
    content_length INTEGER,
    content_type VARCHAR(200) NOT NULL DEFAULT '',
    duration INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_directory_target ON directory(target_id);
CREATE INDEX IF NOT EXISTS idx_directory_url ON directory(url);
CREATE INDEX IF NOT EXISTS idx_directory_status ON directory(status);
CREATE INDEX IF NOT EXISTS idx_directory_created_at ON directory(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_directory_url_target ON directory(target_id, url);

-- screenshot
CREATE TABLE IF NOT EXISTS screenshot (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    status_code SMALLINT,
    image BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_screenshot_target ON screenshot(target_id);
CREATE INDEX IF NOT EXISTS idx_screenshot_created_at ON screenshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_screenshot_per_target ON screenshot(target_id, url);

-- vulnerability
CREATE TABLE IF NOT EXISTS vulnerability (
    id SERIAL PRIMARY KEY,
    target_id INTEGER NOT NULL REFERENCES target(id) ON DELETE CASCADE,
    url TEXT NOT NULL DEFAULT '',
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
CREATE INDEX IF NOT EXISTS idx_vuln_url ON vulnerability(url);
CREATE INDEX IF NOT EXISTS idx_vuln_type ON vulnerability(vuln_type);
CREATE INDEX IF NOT EXISTS idx_vuln_severity ON vulnerability(severity);
CREATE INDEX IF NOT EXISTS idx_vuln_source ON vulnerability(source);
CREATE INDEX IF NOT EXISTS idx_vuln_created_at ON vulnerability(created_at);
CREATE INDEX IF NOT EXISTS idx_vuln_target_reviewed ON vulnerability(target_id, reviewed);

-- ============================================
-- Snapshot tables (depends on scan)
-- ============================================

-- subdomain_snapshot
CREATE TABLE IF NOT EXISTS subdomain_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    name VARCHAR(1000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_scan ON subdomain_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_name ON subdomain_snapshot(name);
CREATE INDEX IF NOT EXISTS idx_subdomain_snap_created_at ON subdomain_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_subdomain_per_scan_snapshot ON subdomain_snapshot(scan_id, name);

-- host_port_mapping_snapshot
CREATE TABLE IF NOT EXISTS host_port_mapping_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    host VARCHAR(1000) NOT NULL,
    ip INET NOT NULL,
    port INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_scan ON host_port_mapping_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_host ON host_port_mapping_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_ip ON host_port_mapping_snapshot(ip);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_port ON host_port_mapping_snapshot(port);
CREATE INDEX IF NOT EXISTS idx_hpm_snap_created_at ON host_port_mapping_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_scan_host_ip_port_snapshot ON host_port_mapping_snapshot(scan_id, host, ip, port);

-- website_snapshot
CREATE TABLE IF NOT EXISTS website_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_website_snap_scan ON website_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_website_snap_url ON website_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_website_snap_host ON website_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_website_snap_title ON website_snapshot(title);
CREATE INDEX IF NOT EXISTS idx_website_snap_created_at ON website_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_website_per_scan_snapshot ON website_snapshot(scan_id, url);


-- endpoint_snapshot
CREATE TABLE IF NOT EXISTS endpoint_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
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
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_scan ON endpoint_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_url ON endpoint_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_host ON endpoint_snapshot(host);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_title ON endpoint_snapshot(title);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_status_code ON endpoint_snapshot(status_code);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_webserver ON endpoint_snapshot(webserver);
CREATE INDEX IF NOT EXISTS idx_endpoint_snap_created_at ON endpoint_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_endpoint_per_scan_snapshot ON endpoint_snapshot(scan_id, url);

-- directory_snapshot
CREATE TABLE IF NOT EXISTS directory_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url VARCHAR(2000) NOT NULL,
    status INTEGER,
    content_length INTEGER,
    content_type VARCHAR(200) NOT NULL DEFAULT '',
    duration INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_directory_snap_scan ON directory_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_directory_snap_url ON directory_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_directory_snap_status ON directory_snapshot(status);
CREATE INDEX IF NOT EXISTS idx_directory_snap_content_type ON directory_snapshot(content_type);
CREATE INDEX IF NOT EXISTS idx_directory_snap_created_at ON directory_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_directory_per_scan_snapshot ON directory_snapshot(scan_id, url);

-- screenshot_snapshot
CREATE TABLE IF NOT EXISTS screenshot_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    status_code SMALLINT,
    image BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_scan ON screenshot_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_screenshot_snap_created_at ON screenshot_snapshot(created_at);
CREATE UNIQUE INDEX IF NOT EXISTS unique_screenshot_per_scan_snapshot ON screenshot_snapshot(scan_id, url);

-- vulnerability_snapshot
CREATE TABLE IF NOT EXISTS vulnerability_snapshot (
    id SERIAL PRIMARY KEY,
    scan_id INTEGER NOT NULL REFERENCES scan(id) ON DELETE CASCADE,
    url TEXT NOT NULL DEFAULT '',
    vuln_type VARCHAR(200) NOT NULL DEFAULT '',
    severity VARCHAR(20) NOT NULL DEFAULT 'unknown',
    source VARCHAR(100) NOT NULL DEFAULT '',
    cvss_score DECIMAL(3,1) NOT NULL DEFAULT 0.0,
    description TEXT NOT NULL DEFAULT '',
    raw_output JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_scan ON vulnerability_snapshot(scan_id);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_url ON vulnerability_snapshot(url);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_type ON vulnerability_snapshot(vuln_type);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_severity ON vulnerability_snapshot(severity);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_source ON vulnerability_snapshot(source);
CREATE INDEX IF NOT EXISTS idx_vuln_snap_created_at ON vulnerability_snapshot(created_at);

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

-- notification
CREATE TABLE IF NOT EXISTS notification (
    id SERIAL PRIMARY KEY,
    category VARCHAR(20) NOT NULL DEFAULT 'system',
    level VARCHAR(20) NOT NULL DEFAULT 'low',
    title VARCHAR(200) NOT NULL DEFAULT '',
    message VARCHAR(2000) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_notification_category ON notification(category);
CREATE INDEX IF NOT EXISTS idx_notification_level ON notification(level);
CREATE INDEX IF NOT EXISTS idx_notification_created_at ON notification(created_at);
CREATE INDEX IF NOT EXISTS idx_notification_is_read ON notification(is_read);

-- ============================================
-- GIN Indexes for array fields
-- ============================================

-- GIN index for website.tech array
CREATE INDEX IF NOT EXISTS idx_website_tech_gin ON website USING GIN (tech);

-- GIN index for endpoint.tech array
CREATE INDEX IF NOT EXISTS idx_endpoint_tech_gin ON endpoint USING GIN (tech);

-- GIN index for scan.engine_ids array
CREATE INDEX IF NOT EXISTS idx_scan_engine_ids_gin ON scan USING GIN (engine_ids);

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

-- ============================================
-- Subfinder Provider Settings (singleton)
-- ============================================
CREATE TABLE IF NOT EXISTS subfinder_provider_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    providers JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Insert default row with all providers disabled
INSERT INTO subfinder_provider_settings (id, providers) VALUES (1, '{
    "fofa": {"enabled": false, "email": "", "api_key": ""},
    "hunter": {"enabled": false, "api_key": ""},
    "shodan": {"enabled": false, "api_key": ""},
    "censys": {"enabled": false, "api_id": "", "api_secret": ""},
    "zoomeye": {"enabled": false, "api_key": ""},
    "securitytrails": {"enabled": false, "api_key": ""},
    "threatbook": {"enabled": false, "api_key": ""},
    "quake": {"enabled": false, "api_key": ""}
}'::jsonb) ON CONFLICT (id) DO NOTHING;

-- ============================================
-- WebSocket Agent System tables
-- ============================================

-- agent
CREATE TABLE IF NOT EXISTS agent (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    api_key         VARCHAR(8) NOT NULL UNIQUE,
    status          VARCHAR(20) DEFAULT 'offline',
    health_state    VARCHAR(20) DEFAULT 'ok',
    health_reason   VARCHAR(64),
    health_message  VARCHAR(256),
    health_since    TIMESTAMPTZ,
    hostname        VARCHAR(255),
    ip_address      VARCHAR(45),
    agent_version   VARCHAR(20),
    worker_version  VARCHAR(64),

    -- Scheduling configuration (can be dynamically modified via API)
    max_tasks       INT DEFAULT 5,
    cpu_threshold   INT DEFAULT 85,
    mem_threshold   INT DEFAULT 85,
    disk_threshold  INT DEFAULT 90,

    -- Self-registration related
    registration_token  VARCHAR(8),

    -- Timestamps
    connected_at    TIMESTAMPTZ,
    last_heartbeat  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- registration_token
CREATE TABLE IF NOT EXISTS registration_token (
    id              SERIAL PRIMARY KEY,
    token           VARCHAR(8) NOT NULL UNIQUE,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 hour'),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- scan_task
CREATE TABLE IF NOT EXISTS scan_task (
    id              SERIAL PRIMARY KEY,
    scan_id         INT NOT NULL,
    stage           INT NOT NULL DEFAULT 0,
    workflow_name   VARCHAR(100) NOT NULL,
    workflow_api_version VARCHAR(16),
    workflow_schema_version VARCHAR(64),
    status          VARCHAR(20) DEFAULT 'pending',

    -- Assignment information
    agent_id        INT REFERENCES agent(id),
    config          TEXT,
    error_message   VARCHAR(4096),

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ
);

-- Indexes for agent table
CREATE INDEX IF NOT EXISTS idx_agent_status ON agent(status);
CREATE INDEX IF NOT EXISTS idx_agent_api_key ON agent(api_key);

-- Indexes for registration_token table
CREATE INDEX IF NOT EXISTS idx_registration_token_token ON registration_token(token);
CREATE INDEX IF NOT EXISTS idx_registration_token_expires ON registration_token(expires_at);

-- Indexes for scan_task table
CREATE INDEX IF NOT EXISTS idx_scan_task_pending_order ON scan_task(status, stage DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_scan_task_agent_id ON scan_task(agent_id);
CREATE INDEX IF NOT EXISTS idx_scan_task_scan_id ON scan_task(scan_id);

-- Comments for agent tables
COMMENT ON TABLE agent IS 'Agent metadata and configuration';
COMMENT ON TABLE registration_token IS 'Registration tokens for agent self-registration';
COMMENT ON TABLE scan_task IS 'Task queue supporting priority scheduling';
COMMENT ON COLUMN agent.status IS 'Agent status: online/offline';
COMMENT ON COLUMN agent.api_key IS '8-character hex string for authentication';
COMMENT ON COLUMN registration_token.expires_at IS 'Token expiration time (default 1 hour after creation)';
COMMENT ON COLUMN scan_task.stage IS 'Stage order index (shared for parallel tasks)';
COMMENT ON COLUMN scan_task.workflow_name IS 'Workflow name (e.g. subdomain_discovery)';
COMMENT ON COLUMN scan_task.status IS 'Task status: blocked/pending/running/completed/failed/cancelled';
COMMENT ON COLUMN scan_task.workflow_api_version IS 'Precomputed workflow API version tuple used by scheduler compatibility gate';
COMMENT ON COLUMN scan_task.workflow_schema_version IS 'Precomputed workflow schema version tuple used by scheduler compatibility gate';
COMMENT ON COLUMN scan_task.error_message IS 'Error message (truncated by Agent, max 4KB)';
COMMENT ON INDEX idx_scan_task_pending_order IS 'Supports task pull queries (ordered by stage DESC to prioritize completing existing scans, created_at ASC)';
COMMENT ON INDEX idx_scan_task_agent_id IS 'Supports querying tasks by agent';
COMMENT ON INDEX idx_scan_task_scan_id IS 'Supports querying tasks by scan';
