package config

import (
	"time"

	"github.com/spf13/viper"
	"github.com/yyhuni/lunafox/contracts/sharedstorage"
)

const (
	DefaultEnginePackageCacheRoot   = "/opt/lunafox/engine-packages"
	DefaultFingerprintArtifactsRoot = "/opt/lunafox/fingerprint-artifacts"
	DefaultLoginVisualsRoot         = "/opt/lunafox/login-visuals"
	DefaultWorkflowDefinitionsRoot  = "/usr/local/share/lunafox/workflows"
	DefaultNucleiPocWorkspaceRoot   = "/var/lib/lunafox/nuclei-poc-workspaces"

	// Bound untrusted OCI package downloads and cache writes unless deployment
	// configuration explicitly opts into a larger quota.
	defaultEngineInstallMaxPackageBytes = 512 << 20
)

// setDefaults sets default values for configuration.
func setDefaults(v *viper.Viper) {
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_GRPC_PORT", 9090)
	v.SetDefault("GIN_MODE", "release")

	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "")
	v.SetDefault("DB_NAME", "lunafox")
	v.SetDefault("DB_SSLMODE", "disable")
	v.SetDefault("DB_TIMEZONE", "UTC")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 5)
	v.SetDefault("DB_CONN_MAX_LIFETIME", 300)

	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("LOKI_URL", "http://loki:3100")

	v.SetDefault("LOG_LEVEL", "info")

	v.SetDefault("JWT_SECRET", "")
	v.SetDefault("JWT_ACCESS_EXPIRE", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRE", "168h")

	v.SetDefault("WORDLISTS_BASE_PATH", sharedstorage.DefaultWordlistsRoot)
	v.SetDefault("FINGERPRINT_ARTIFACTS_BASE_PATH", DefaultFingerprintArtifactsRoot)
	v.SetDefault("LOGIN_VISUALS_BASE_PATH", DefaultLoginVisualsRoot)
	v.SetDefault("ENGINE_PACKAGE_CACHE_ROOT", DefaultEnginePackageCacheRoot)
	v.SetDefault("ENGINE_INSTALL_MAX_PACKAGE_BYTES", defaultEngineInstallMaxPackageBytes)
	v.SetDefault("WORKFLOW_DEFINITIONS_ROOT", DefaultWorkflowDefinitionsRoot)
	v.SetDefault("NUCLEI_POC_WORKSPACE_ROOT", DefaultNucleiPocWorkspaceRoot)
	v.SetDefault("PUBLIC_URL", "")
	v.SetDefault("SCAN_HISTORY_RETENTION_MODE", "enforce")
	v.SetDefault("SCAN_HISTORY_RETENTION_INTERVAL", "1h")
	v.SetDefault("SCAN_HISTORY_RETENTION_DURATION", "720h")
	v.SetDefault("SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE", 1000)
	v.SetDefault("SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE", 100)
	v.SetDefault("SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN", 1)
	v.SetDefault("SCAN_HISTORY_RETENTION_MAX_RUN_DURATION", "5m")
	v.SetDefault("TARGET_CLEANUP_BATCH_SIZE", 1000)
	v.SetDefault("TARGET_CLEANUP_MAX_BATCHES_PER_RUN", 100)
	v.SetDefault("TARGET_CLEANUP_MAX_RUN_DURATION", "5m")
	v.SetDefault("NOTIFICATION_VULNERABILITY_THRESHOLD", "high")
}

// GetDefaults returns a Config with all default values (for testing).
func GetDefaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     8080,
			GRPCPort: 9090,
			Mode:     "release",
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "",
			Name:            "lunafox",
			SSLMode:         "disable",
			TimeZone:        "UTC",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		LokiURL: "http://loki:3100",
		Log: LogConfig{
			Level: "info",
		},
		JWT: JWTConfig{
			Secret:        "",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 168 * time.Hour,
		},
		Storage: StorageConfig{
			WordlistsBasePath:            sharedstorage.DefaultWordlistsRoot,
			FingerprintArtifactsBasePath: DefaultFingerprintArtifactsRoot,
			LoginVisualsBasePath:         DefaultLoginVisualsRoot,
			EnginePackageCacheRoot:       DefaultEnginePackageCacheRoot,
			WorkflowDefinitionsRoot:      DefaultWorkflowDefinitionsRoot,
			NucleiPocWorkspaceRoot:       DefaultNucleiPocWorkspaceRoot,
		},
		EngineInstall: EngineInstallConfig{MaxPackageBytes: defaultEngineInstallMaxPackageBytes},
		ScanHistoryRetention: ScanHistoryRetentionConfig{
			Mode:                         "enforce",
			Interval:                     time.Hour,
			Retention:                    30 * 24 * time.Hour,
			TaskDeleteBatchSize:          1000,
			MaxTaskDeleteBatchesPerRange: 100,
			MaxRangesPerRun:              1,
			MaxRunDuration:               5 * time.Minute,
		},
		TargetCleanup: TargetCleanupConfig{
			BatchSize:        1000,
			MaxBatchesPerRun: 100,
			MaxRunDuration:   5 * time.Minute,
		},
		Notification: NotificationConfig{VulnerabilityThreshold: "high"},
		PublicURL:    "",
	}
}
