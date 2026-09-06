-- DESTRUCTIVE DEVELOPMENT/TEST TEARDOWN ONLY.
-- This drops all schema data and is not a production data or in-flight task rollback.
-- Drop tables in reverse dependency order

-- Notification delivery and inbox tables
DROP TABLE IF EXISTS notification_delivery_attempt CASCADE;
DROP TABLE IF EXISTS notification_delivery CASCADE;
DROP TABLE IF EXISTS notification_destination_subscription CASCADE;
DROP TABLE IF EXISTS notification_destination CASCADE;
DROP TABLE IF EXISTS notification_inbox CASCADE;
DROP TABLE IF EXISTS notification_recipient CASCADE;
DROP TABLE IF EXISTS notification_fact CASCADE;
DROP TABLE IF EXISTS notification_outbox CASCADE;

-- Statistics tables
DROP TABLE IF EXISTS statistics_history CASCADE;
DROP TABLE IF EXISTS asset_statistics CASCADE;

-- Global fingerprint libraries
DROP TABLE IF EXISTS fingerprint_library_artifact CASCADE;
DROP TABLE IF EXISTS fingerprint_library_state CASCADE;
DROP TABLE IF EXISTS fingerprint_fingerprinthub CASCADE;

-- Snapshot tables
DROP TABLE IF EXISTS vulnerability_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS vulnerability_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS vulnerability_snapshot CASCADE;
DROP TABLE IF EXISTS screenshot_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS screenshot_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS screenshot_snapshot CASCADE;
DROP TABLE IF EXISTS directory_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS directory_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS directory_snapshot CASCADE;
DROP TABLE IF EXISTS endpoint_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS endpoint_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS endpoint_snapshot CASCADE;
DROP TABLE IF EXISTS website_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS website_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS website_snapshot CASCADE;
DROP TABLE IF EXISTS host_port_mapping_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS host_port_mapping_snapshot_p00000000 CASCADE;
DROP TABLE IF EXISTS host_port_mapping_snapshot CASCADE;
DROP TABLE IF EXISTS subdomain_snapshot_p00010000 CASCADE;
DROP TABLE IF EXISTS subdomain_snapshot_p00000000 CASCADE;
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
DROP TABLE IF EXISTS scheduled_scan_occurrence CASCADE;
DROP TABLE IF EXISTS scheduled_scan CASCADE;
DROP TABLE IF EXISTS task_progress_log_p00010000 CASCADE;
DROP TABLE IF EXISTS task_progress_log_p00000000 CASCADE;
DROP TABLE IF EXISTS task_progress_log CASCADE;
DROP TABLE IF EXISTS scan_task CASCADE;
DROP TABLE IF EXISTS scan_blacklist_snapshot CASCADE;
DROP TABLE IF EXISTS mcp_request_replay CASCADE;
DROP TABLE IF EXISTS scan_operation CASCADE;
DROP TABLE IF EXISTS scan CASCADE;
DROP TABLE IF EXISTS scan_workflow CASCADE;

-- Agent runtime tables
DROP TABLE IF EXISTS server_location_snapshot CASCADE;
DROP TABLE IF EXISTS agent_location CASCADE;
DROP TABLE IF EXISTS agent_runtime_status CASCADE;
DROP TABLE IF EXISTS agent CASCADE;
DROP TABLE IF EXISTS registration_token CASCADE;

-- Settings tables
DROP TABLE IF EXISTS login_visual_settings CASCADE;
DROP TABLE IF EXISTS login_visual_media CASCADE;
DROP TABLE IF EXISTS subfinder_provider_settings CASCADE;
DROP TABLE IF EXISTS blacklist_policy CASCADE;
DROP TABLE IF EXISTS nuclei_poc_sync_request_tombstone CASCADE;
DROP TABLE IF EXISTS nuclei_poc CASCADE;
DROP TABLE IF EXISTS nuclei_poc_candidate_import CASCADE;
DROP TABLE IF EXISTS nuclei_poc_sync_task CASCADE;
DROP TABLE IF EXISTS nuclei_poc_source CASCADE;
DROP TABLE IF EXISTS wordlist CASCADE;
DROP TABLE IF EXISTS engine CASCADE;

-- Core tables
DROP TABLE IF EXISTS login_visual_discovery CASCADE;
DROP TABLE IF EXISTS mcp_key CASCADE;
DROP TABLE IF EXISTS organization_target CASCADE;
DROP TABLE IF EXISTS target_cleanup_job CASCADE;
DROP TABLE IF EXISTS target CASCADE;
DROP INDEX IF EXISTS idx_org_active_name_unique;
DROP TABLE IF EXISTS organization CASCADE;
DROP TABLE IF EXISTS django_session CASCADE;
DROP TABLE IF EXISTS auth_user CASCADE;
