package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from .env file and environment variables.
// Priority: environment variables > .env file > defaults.
func Load() (*Config, error) {
	v := viper.New()
	setDefaults(v)

	if err := readEnvFile(v); err != nil {
		return nil, err
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := configFromViper(v)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadDatabaseConfig reads only the settings required for a bounded database
// operation. It intentionally does not require HTTP, JWT, Redis, or Loki setup.
func LoadDatabaseConfig() (*DatabaseConfig, error) {
	v := viper.New()
	setDefaults(v)

	if err := readEnvFile(v); err != nil {
		return nil, err
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := DatabaseConfig{
		Host:            v.GetString("DB_HOST"),
		Port:            v.GetInt("DB_PORT"),
		User:            v.GetString("DB_USER"),
		Password:        v.GetString("DB_PASSWORD"),
		Name:            v.GetString("DB_NAME"),
		SSLMode:         v.GetString("DB_SSLMODE"),
		TimeZone:        v.GetString("DB_TIMEZONE"),
		MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
		MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
		ConnMaxLifetime: v.GetInt("DB_CONN_MAX_LIFETIME"),
	}
	if err := validateDatabaseConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func readEnvFile(v *viper.Viper) error {
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./go-backend")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

func configFromViper(v *viper.Viper) *Config {
	return &Config{
		Server: ServerConfig{
			Port:     v.GetInt("SERVER_PORT"),
			GRPCPort: v.GetInt("SERVER_GRPC_PORT"),
			Mode:     v.GetString("GIN_MODE"),
		},
		Database: DatabaseConfig{
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetInt("DB_PORT"),
			User:            v.GetString("DB_USER"),
			Password:        v.GetString("DB_PASSWORD"),
			Name:            v.GetString("DB_NAME"),
			SSLMode:         v.GetString("DB_SSLMODE"),
			TimeZone:        v.GetString("DB_TIMEZONE"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetInt("DB_CONN_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
		},
		LokiURL: v.GetString("LOKI_URL"),
		Log: LogConfig{
			Level: v.GetString("LOG_LEVEL"),
		},
		JWT: JWTConfig{
			Secret:        v.GetString("JWT_SECRET"),
			AccessExpire:  v.GetDuration("JWT_ACCESS_EXPIRE"),
			RefreshExpire: v.GetDuration("JWT_REFRESH_EXPIRE"),
		},
		Storage: StorageConfig{
			WordlistsBasePath:            v.GetString("WORDLISTS_BASE_PATH"),
			FingerprintArtifactsBasePath: v.GetString("FINGERPRINT_ARTIFACTS_BASE_PATH"),
			LoginVisualsBasePath:         v.GetString("LOGIN_VISUALS_BASE_PATH"),
			EnginePackageCacheRoot:       v.GetString("ENGINE_PACKAGE_CACHE_ROOT"),
			WorkflowDefinitionsRoot:      v.GetString("WORKFLOW_DEFINITIONS_ROOT"),
			NucleiPocWorkspaceRoot:       v.GetString("NUCLEI_POC_WORKSPACE_ROOT"),
		},
		EngineInstall: EngineInstallConfig{
			InventoryPath:                            v.GetString("ENGINE_INSTALL_INVENTORY_PATH"),
			MaxPackageBytes:                          v.GetInt64("ENGINE_INSTALL_MAX_PACKAGE_BYTES"),
			DevelopmentMode:                          v.GetBool("ENGINE_INSTALL_DEVELOPMENT_MODE"),
			AllowPlainHTTP:                           v.GetBool("ENGINE_INSTALL_ALLOW_PLAIN_HTTP"),
			CFAcceleration:                           v.GetBool("ENGINE_INSTALL_CF_ACCELERATION"),
			Registry:                                 v.GetString("ENGINE_INSTALL_REGISTRY"),
			DevelopmentRuntimeImageIdentityRegistry:  v.GetString("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY"),
			DevelopmentRuntimeImageTransportRegistry: v.GetString("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY"),
		},
		ScanHistoryRetention: ScanHistoryRetentionConfig{
			Mode:                         v.GetString("SCAN_HISTORY_RETENTION_MODE"),
			Interval:                     v.GetDuration("SCAN_HISTORY_RETENTION_INTERVAL"),
			Retention:                    v.GetDuration("SCAN_HISTORY_RETENTION_DURATION"),
			TaskDeleteBatchSize:          v.GetInt("SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE"),
			MaxTaskDeleteBatchesPerRange: v.GetInt("SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE"),
			MaxRangesPerRun:              v.GetInt("SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN"),
			MaxRunDuration:               v.GetDuration("SCAN_HISTORY_RETENTION_MAX_RUN_DURATION"),
		},
		TargetCleanup: TargetCleanupConfig{
			BatchSize:        v.GetInt("TARGET_CLEANUP_BATCH_SIZE"),
			MaxBatchesPerRun: v.GetInt("TARGET_CLEANUP_MAX_BATCHES_PER_RUN"),
			MaxRunDuration:   v.GetDuration("TARGET_CLEANUP_MAX_RUN_DURATION"),
		},
		Notification: NotificationConfig{
			VulnerabilityThreshold: v.GetString("NOTIFICATION_VULNERABILITY_THRESHOLD"),
		},
		PublicURL: v.GetString("PUBLIC_URL"),
	}
}
