package config

import "time"

// Config holds all configuration for the application.
type Config struct {
	Server               ServerConfig
	Database             DatabaseConfig
	Redis                RedisConfig
	LokiURL              string
	Log                  LogConfig
	JWT                  JWTConfig
	Storage              StorageConfig
	EngineInstall        EngineInstallConfig
	ScanHistoryRetention ScanHistoryRetentionConfig
	TargetCleanup        TargetCleanupConfig
	Notification         NotificationConfig
	PublicURL            string
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port     int    `mapstructure:"SERVER_PORT"`
	GRPCPort int    `mapstructure:"SERVER_GRPC_PORT"`
	Mode     string `mapstructure:"GIN_MODE"`
}

// DatabaseConfig holds database-related configuration.
type DatabaseConfig struct {
	Host            string `mapstructure:"DB_HOST"`
	Port            int    `mapstructure:"DB_PORT"`
	User            string `mapstructure:"DB_USER"`
	Password        string `mapstructure:"DB_PASSWORD"`
	Name            string `mapstructure:"DB_NAME"`
	SSLMode         string `mapstructure:"DB_SSLMODE"`
	TimeZone        string `mapstructure:"DB_TIMEZONE"`
	MaxOpenConns    int    `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int    `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime int    `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

// RedisConfig holds Redis-related configuration.
type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     int    `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// LogConfig holds logging-related configuration.
type LogConfig struct {
	Level string `mapstructure:"LOG_LEVEL"`
}

// JWTConfig holds JWT-related configuration.
type JWTConfig struct {
	Secret        string        `mapstructure:"JWT_SECRET"`
	AccessExpire  time.Duration `mapstructure:"JWT_ACCESS_EXPIRE"`
	RefreshExpire time.Duration `mapstructure:"JWT_REFRESH_EXPIRE"`
}

// StorageConfig holds storage-related configuration.
type StorageConfig struct {
	WordlistsBasePath            string `mapstructure:"WORDLISTS_BASE_PATH"`
	FingerprintArtifactsBasePath string `mapstructure:"FINGERPRINT_ARTIFACTS_BASE_PATH"`
	LoginVisualsBasePath         string `mapstructure:"LOGIN_VISUALS_BASE_PATH"`
	EnginePackageCacheRoot       string `mapstructure:"ENGINE_PACKAGE_CACHE_ROOT"`
	WorkflowDefinitionsRoot      string `mapstructure:"WORKFLOW_DEFINITIONS_ROOT"`
	NucleiPocWorkspaceRoot       string `mapstructure:"NUCLEI_POC_WORKSPACE_ROOT"`
}

// EngineInstallConfig controls the bootstrap-only OCI installation boundary.
// It is intentionally absent from task/session configuration.
type EngineInstallConfig struct {
	InventoryPath                            string `mapstructure:"ENGINE_INSTALL_INVENTORY_PATH"`
	MaxPackageBytes                          int64  `mapstructure:"ENGINE_INSTALL_MAX_PACKAGE_BYTES"`
	DevelopmentMode                          bool   `mapstructure:"ENGINE_INSTALL_DEVELOPMENT_MODE"`
	AllowPlainHTTP                           bool   `mapstructure:"ENGINE_INSTALL_ALLOW_PLAIN_HTTP"`
	CFAcceleration                           bool   `mapstructure:"ENGINE_INSTALL_CF_ACCELERATION"`
	Registry                                 string `mapstructure:"ENGINE_INSTALL_REGISTRY"`
	DevelopmentRuntimeImageIdentityRegistry  string `mapstructure:"ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY"`
	DevelopmentRuntimeImageTransportRegistry string `mapstructure:"ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY"`
}

// ScanHistoryRetentionConfig controls backend-only history partition lifecycle.
// It intentionally has no user- or tenant-facing configuration surface.
type ScanHistoryRetentionConfig struct {
	Mode                         string        `mapstructure:"SCAN_HISTORY_RETENTION_MODE"`
	Interval                     time.Duration `mapstructure:"SCAN_HISTORY_RETENTION_INTERVAL"`
	Retention                    time.Duration `mapstructure:"SCAN_HISTORY_RETENTION_DURATION"`
	TaskDeleteBatchSize          int           `mapstructure:"SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE"`
	MaxTaskDeleteBatchesPerRange int           `mapstructure:"SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE"`
	MaxRangesPerRun              int           `mapstructure:"SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN"`
	MaxRunDuration               time.Duration `mapstructure:"SCAN_HISTORY_RETENTION_MAX_RUN_DURATION"`
}

// TargetCleanupConfig bounds one internal Target cleanup reconciliation run.
// It intentionally has no runtime API or hot-reload surface.
type TargetCleanupConfig struct {
	BatchSize        int           `mapstructure:"TARGET_CLEANUP_BATCH_SIZE"`
	MaxBatchesPerRun int           `mapstructure:"TARGET_CLEANUP_MAX_BATCHES_PER_RUN"`
	MaxRunDuration   time.Duration `mapstructure:"TARGET_CLEANUP_MAX_RUN_DURATION"`
}

// NotificationConfig holds first-phase, server-owned notification policy.
// It intentionally has no runtime mutation or per-destination threshold.
type NotificationConfig struct {
	VulnerabilityThreshold string `mapstructure:"NOTIFICATION_VULNERABILITY_THRESHOLD"`
}
