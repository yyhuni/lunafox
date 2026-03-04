-- Rollback initial schema migration
-- Drop tables in reverse dependency order

-- Statistics tables
DROP TABLE IF EXISTS notification CASCADE;
DROP TABLE IF EXISTS statistics_history CASCADE;
DROP TABLE IF EXISTS asset_statistics CASCADE;

-- Snapshot tables
DROP TABLE IF EXISTS vulnerability_snapshot CASCADE;
DROP TABLE IF EXISTS screenshot_snapshot CASCADE;
DROP TABLE IF EXISTS directory_snapshot CASCADE;
DROP TABLE IF EXISTS endpoint_snapshot CASCADE;
DROP TABLE IF EXISTS website_snapshot CASCADE;
DROP TABLE IF EXISTS host_port_mapping_snapshot CASCADE;
DROP TABLE IF EXISTS subdomain_snapshot CASCADE;

-- Asset tables
DROP TABLE IF EXISTS vulnerability CASCADE;
DROP TABLE IF EXISTS screenshot CASCADE;
DROP TABLE IF EXISTS directory CASCADE;
DROP TABLE IF EXISTS endpoint CASCADE;
DROP TABLE IF EXISTS website CASCADE;
DROP TABLE IF EXISTS host_port_mapping CASCADE;
DROP TABLE IF EXISTS subdomain CASCADE;

-- Scan related tables
DROP TABLE IF EXISTS scheduled_scan CASCADE;
DROP TABLE IF EXISTS scan_log CASCADE;
DROP TABLE IF EXISTS scan CASCADE;

-- Settings tables
DROP TABLE IF EXISTS subfinder_provider_settings CASCADE;
DROP TABLE IF EXISTS notification_settings CASCADE;
DROP TABLE IF EXISTS blacklist_rule CASCADE;
DROP TABLE IF EXISTS nuclei_template_repo CASCADE;
DROP TABLE IF EXISTS wordlist CASCADE;
DROP TABLE IF EXISTS worker_node CASCADE;

-- Core tables
DROP TABLE IF EXISTS organization_target CASCADE;
DROP TABLE IF EXISTS scan_engine CASCADE;
DROP TABLE IF EXISTS target CASCADE;
DROP TABLE IF EXISTS organization CASCADE;
DROP TABLE IF EXISTS django_session CASCADE;
DROP TABLE IF EXISTS auth_user CASCADE;
