package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	legacyDefaultJWTSecret = "change-me-in-production-use-a-long-random-string"
)

// validateConfig enforces required runtime contracts and blocks weak placeholder values.
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.Storage.FingerprintArtifactsBasePath) == "" {
		return fmt.Errorf("FINGERPRINT_ARTIFACTS_BASE_PATH is required")
	}
	if strings.TrimSpace(cfg.Storage.LoginVisualsBasePath) == "" {
		return fmt.Errorf("LOGIN_VISUALS_BASE_PATH is required")
	}
	if strings.TrimSpace(cfg.Storage.NucleiPocWorkspaceRoot) == "" {
		return fmt.Errorf("NUCLEI_POC_WORKSPACE_ROOT is required")
	}
	jwtSecret := strings.TrimSpace(cfg.JWT.Secret)
	if jwtSecret == "" || jwtSecret == legacyDefaultJWTSecret {
		return fmt.Errorf("JWT_SECRET is required and cannot use weak placeholder value")
	}

	publicURL := strings.TrimSpace(cfg.PublicURL)
	if publicURL == "" {
		return fmt.Errorf("PUBLIC_URL is required")
	}
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("PUBLIC_URL is invalid: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("PUBLIC_URL must use https scheme")
	}
	if parsed.Host == "" {
		return fmt.Errorf("PUBLIC_URL host is required")
	}

	retention := cfg.ScanHistoryRetention
	if retention.Mode != "disabled" && retention.Mode != "report" && retention.Mode != "enforce" {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_MODE must be disabled, report, or enforce")
	}
	if retention.Interval <= 0 {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_INTERVAL must be positive")
	}
	if retention.Retention != 30*24*time.Hour {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_DURATION must be exactly 30 days")
	}
	if retention.TaskDeleteBatchSize <= 0 {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE must be positive")
	}
	if retention.MaxTaskDeleteBatchesPerRange <= 0 {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE must be positive")
	}
	if retention.MaxRangesPerRun <= 0 {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN must be positive")
	}
	if retention.MaxRunDuration <= 0 {
		return fmt.Errorf("SCAN_HISTORY_RETENTION_MAX_RUN_DURATION must be positive")
	}

	cleanup := cfg.TargetCleanup
	if cleanup.BatchSize <= 0 {
		return fmt.Errorf("TARGET_CLEANUP_BATCH_SIZE must be positive")
	}
	if cleanup.MaxBatchesPerRun <= 0 {
		return fmt.Errorf("TARGET_CLEANUP_MAX_BATCHES_PER_RUN must be positive")
	}
	if cleanup.MaxRunDuration <= 0 {
		return fmt.Errorf("TARGET_CLEANUP_MAX_RUN_DURATION must be positive")
	}

	switch cfg.Notification.VulnerabilityThreshold {
	case "info", "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("NOTIFICATION_VULNERABILITY_THRESHOLD must be info, low, medium, high, or critical")
	}

	return nil
}

func validateDatabaseConfig(cfg *DatabaseConfig) error {
	if cfg == nil {
		return fmt.Errorf("database config is nil")
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if cfg.Port <= 0 {
		return fmt.Errorf("DB_PORT must be positive")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if strings.TrimSpace(cfg.Name) == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if strings.TrimSpace(cfg.SSLMode) == "" {
		return fmt.Errorf("DB_SSLMODE is required")
	}
	if strings.TrimSpace(cfg.TimeZone) == "" {
		return fmt.Errorf("DB_TIMEZONE is required")
	}
	if cfg.MaxOpenConns <= 0 || cfg.MaxIdleConns < 0 || cfg.ConnMaxLifetime <= 0 {
		return fmt.Errorf("database connection pool settings must be valid")
	}
	return nil
}
