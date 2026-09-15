package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
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
	if err := resolveSecretFile(v, "DB_PASSWORD", "DB_PASSWORD_FILE"); err != nil {
		return nil, err
	}
	if err := resolveSecretFile(v, "JWT_SECRET", "JWT_SECRET_FILE"); err != nil {
		return nil, err
	}
	if err := resolvePublicURL(v); err != nil {
		return nil, err
	}

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
	if err := resolveSecretFile(v, "DB_PASSWORD", "DB_PASSWORD_FILE"); err != nil {
		return nil, err
	}

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

func resolvePublicURL(v *viper.Viper) error {
	if strings.TrimSpace(v.GetString("PUBLIC_URL")) != "" {
		return nil
	}

	host := strings.TrimSpace(v.GetString("PUBLIC_HOST"))
	port := strings.TrimSpace(v.GetString("PUBLIC_PORT"))
	if host == "" && port == "" {
		return nil
	}
	if host == "" {
		return fmt.Errorf("PUBLIC_HOST is required when PUBLIC_URL is not set")
	}
	if port == "" {
		return fmt.Errorf("PUBLIC_PORT is required when PUBLIC_URL is not set")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("PUBLIC_PORT must be an integer between 1 and 65535")
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	if host == "" || strings.ContainsAny(host, "/?#@") {
		return fmt.Errorf("PUBLIC_HOST must be a hostname or IP address without scheme, port, or path")
	}
	if net.ParseIP(host) == nil && strings.Contains(host, ":") {
		return fmt.Errorf("PUBLIC_HOST must not include a port")
	}

	// net.JoinHostPort preserves hostnames and brackets IPv6 literals correctly.
	v.Set("PUBLIC_URL", (&url.URL{Scheme: "https", Host: net.JoinHostPort(host, port)}).String())
	return nil
}

func resolveSecretFile(v *viper.Viper, valueKey, fileKey string) error {
	path := strings.TrimSpace(v.GetString(fileKey))
	if path == "" {
		return nil
	}
	if v.GetString(valueKey) != "" {
		return fmt.Errorf("%s and %s cannot both be set", valueKey, fileKey)
	}

	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", fileKey, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s must reference a regular file", fileKey)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s must not be accessible by group or others", fileKey)
	}
	if info.Size() == 0 || info.Size() > 64*1024 {
		return fmt.Errorf("%s must contain between 1 and 65536 bytes", fileKey)
	}

	value, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", fileKey, err)
	}
	if strings.TrimSpace(string(value)) == "" {
		return fmt.Errorf("%s must not be empty", fileKey)
	}
	if strings.ContainsAny(string(value), "\r\n\x00") {
		return fmt.Errorf("%s must contain exactly one line", fileKey)
	}

	// Set has higher precedence than environment and file configuration, so all
	// commands consume the validated file as the single secret source.
	v.Set(valueKey, string(value))
	return nil
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
		Upgrade: UpgradeConfig{
			DeploymentRoot:      v.GetString("LUNAFOX_UPGRADE_DEPLOYMENT_ROOT"),
			ManifestPath:        v.GetString("RELEASE_MANIFEST_PATH"),
			MigrationPolicyPath: v.GetString("MIGRATION_POLICY_PATH"),
			SocketPath:          v.GetString("LUNAFOX_UPGRADE_SOCKET_PATH"),
		},
		PublicURL: v.GetString("PUBLIC_URL"),
	}
}
