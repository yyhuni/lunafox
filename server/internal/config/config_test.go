package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestConfigDefaults tests that default values are correctly set
// Property 4: default configuration correctness
// For any missing environment variable, the config system should return the predefined default value.
// Verification target: requirement 2.4
func TestConfigDefaults(t *testing.T) {
	// Clear all relevant environment variables
	envVars := []string{
		"SERVER_PORT", "SERVER_GRPC_PORT", "GIN_MODE",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "DB_TIMEZONE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
		"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
		"LOKI_URL",
		"LOG_LEVEL",
		"ENGINE_PACKAGE_CACHE_ROOT",
		"WORKFLOW_DEFINITIONS_ROOT",
		"JWT_SECRET",
		"PUBLIC_URL",
		"SCAN_HISTORY_RETENTION_MODE",
		"SCAN_HISTORY_RETENTION_INTERVAL",
		"SCAN_HISTORY_RETENTION_DURATION",
		"SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE",
		"SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE",
		"SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN",
		"SCAN_HISTORY_RETENTION_MAX_RUN_DURATION",
		"TARGET_CLEANUP_BATCH_SIZE",
		"TARGET_CLEANUP_MAX_BATCHES_PER_RUN",
		"TARGET_CLEANUP_MAX_RUN_DURATION",
	}
	for _, env := range envVars {
		if err := os.Unsetenv(env); err != nil {
			t.Logf("Warning: failed to unset %s: %v", env, err)
		}
	}
	if err := os.Setenv("JWT_SECRET", "jwt-secret-for-config-defaults-test"); err != nil {
		t.Fatalf("Failed to set JWT_SECRET: %v", err)
	}
	if err := os.Setenv("PUBLIC_URL", "https://public.example.com:8083"); err != nil {
		t.Fatalf("Failed to set PUBLIC_URL: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("PUBLIC_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	defaults := GetDefaults()

	// Test Server defaults
	if cfg.Server.Port != defaults.Server.Port {
		t.Errorf("Server.Port: expected %d, got %d", defaults.Server.Port, cfg.Server.Port)
	}
	if cfg.Server.GRPCPort != defaults.Server.GRPCPort {
		t.Errorf("Server.GRPCPort: expected %d, got %d", defaults.Server.GRPCPort, cfg.Server.GRPCPort)
	}
	if cfg.Server.Mode != defaults.Server.Mode {
		t.Errorf("Server.Mode: expected %s, got %s", defaults.Server.Mode, cfg.Server.Mode)
	}

	// Test Database defaults
	if cfg.Database.Host != defaults.Database.Host {
		t.Errorf("Database.Host: expected %s, got %s", defaults.Database.Host, cfg.Database.Host)
	}
	if cfg.Database.Port != defaults.Database.Port {
		t.Errorf("Database.Port: expected %d, got %d", defaults.Database.Port, cfg.Database.Port)
	}
	if cfg.Database.User != defaults.Database.User {
		t.Errorf("Database.User: expected %s, got %s", defaults.Database.User, cfg.Database.User)
	}
	if cfg.Database.Name != defaults.Database.Name {
		t.Errorf("Database.Name: expected %s, got %s", defaults.Database.Name, cfg.Database.Name)
	}
	if cfg.Database.SSLMode != defaults.Database.SSLMode {
		t.Errorf("Database.SSLMode: expected %s, got %s", defaults.Database.SSLMode, cfg.Database.SSLMode)
	}
	if cfg.Database.TimeZone != defaults.Database.TimeZone {
		t.Errorf("Database.TimeZone: expected %s, got %s", defaults.Database.TimeZone, cfg.Database.TimeZone)
	}
	if cfg.Database.MaxOpenConns != defaults.Database.MaxOpenConns {
		t.Errorf("Database.MaxOpenConns: expected %d, got %d", defaults.Database.MaxOpenConns, cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != defaults.Database.MaxIdleConns {
		t.Errorf("Database.MaxIdleConns: expected %d, got %d", defaults.Database.MaxIdleConns, cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != defaults.Database.ConnMaxLifetime {
		t.Errorf("Database.ConnMaxLifetime: expected %d, got %d", defaults.Database.ConnMaxLifetime, cfg.Database.ConnMaxLifetime)
	}

	// Test Redis defaults
	if cfg.Redis.Host != defaults.Redis.Host {
		t.Errorf("Redis.Host: expected %s, got %s", defaults.Redis.Host, cfg.Redis.Host)
	}
	if cfg.Redis.Port != defaults.Redis.Port {
		t.Errorf("Redis.Port: expected %d, got %d", defaults.Redis.Port, cfg.Redis.Port)
	}
	if cfg.Redis.DB != defaults.Redis.DB {
		t.Errorf("Redis.DB: expected %d, got %d", defaults.Redis.DB, cfg.Redis.DB)
	}
	if cfg.LokiURL != defaults.LokiURL {
		t.Errorf("LokiURL: expected %s, got %s", defaults.LokiURL, cfg.LokiURL)
	}

	// Test Log defaults
	if cfg.Log.Level != defaults.Log.Level {
		t.Errorf("Log.Level: expected %s, got %s", defaults.Log.Level, cfg.Log.Level)
	}
	if cfg.Storage.EnginePackageCacheRoot != defaults.Storage.EnginePackageCacheRoot {
		t.Errorf("Storage.EnginePackageCacheRoot: expected %s, got %s", defaults.Storage.EnginePackageCacheRoot, cfg.Storage.EnginePackageCacheRoot)
	}
	if cfg.Storage.WorkflowDefinitionsRoot != defaults.Storage.WorkflowDefinitionsRoot {
		t.Errorf("Storage.WorkflowDefinitionsRoot: expected %s, got %s", defaults.Storage.WorkflowDefinitionsRoot, cfg.Storage.WorkflowDefinitionsRoot)
	}
	if cfg.PublicURL != "https://public.example.com:8083" {
		t.Errorf("PublicURL: expected %s, got %s", "https://public.example.com:8083", cfg.PublicURL)
	}
	if cfg.ScanHistoryRetention.Mode != "enforce" ||
		cfg.ScanHistoryRetention.Retention != 30*24*time.Hour ||
		cfg.ScanHistoryRetention.TaskDeleteBatchSize != 1000 ||
		cfg.ScanHistoryRetention.MaxTaskDeleteBatchesPerRange != 100 ||
		cfg.ScanHistoryRetention.MaxRangesPerRun != 1 ||
		cfg.ScanHistoryRetention.MaxRunDuration != 5*time.Minute {
		t.Errorf("unexpected scan-history retention defaults: %+v", cfg.ScanHistoryRetention)
	}
	if cfg.TargetCleanup.BatchSize != 1000 ||
		cfg.TargetCleanup.MaxBatchesPerRun != 100 ||
		cfg.TargetCleanup.MaxRunDuration != 5*time.Minute {
		t.Errorf("unexpected target-cleanup defaults: %+v", cfg.TargetCleanup)
	}
}

func TestLoadDatabaseConfigDoesNotRequireServerOnlySettings(t *testing.T) {
	for _, key := range []string{"JWT_SECRET", "PUBLIC_URL"} {
		t.Setenv(key, "")
	}

	cfg, err := LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("load database-only config: %v", err)
	}
	if cfg.Host == "" || cfg.Port == 0 || cfg.User == "" || cfg.Name == "" {
		t.Fatalf("unexpected incomplete database config: %+v", cfg)
	}
}

func TestLoadValidatesScanHistoryRetentionConfiguration(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret-for-scan-history-retention-test")
	t.Setenv("PUBLIC_URL", "https://public.example")

	t.Run("report mode is accepted", func(t *testing.T) {
		t.Setenv("SCAN_HISTORY_RETENTION_MODE", "report")
		if _, err := Load(); err != nil {
			t.Fatalf("Load report retention mode: %v", err)
		}
	})

	for _, tt := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "unknown mode", key: "SCAN_HISTORY_RETENTION_MODE", value: "delete-now"},
		{name: "not thirty days", key: "SCAN_HISTORY_RETENTION_DURATION", value: "719h"},
		{name: "non-positive task batch", key: "SCAN_HISTORY_RETENTION_TASK_DELETE_BATCH_SIZE", value: "0"},
		{name: "non-positive task batch budget", key: "SCAN_HISTORY_RETENTION_MAX_TASK_DELETE_BATCHES_PER_RANGE", value: "0"},
		{name: "non-positive range budget", key: "SCAN_HISTORY_RETENTION_MAX_RANGES_PER_RUN", value: "0"},
		{name: "non-positive run duration", key: "SCAN_HISTORY_RETENTION_MAX_RUN_DURATION", value: "0s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load must reject %s=%q", tt.key, tt.value)
			}
		})
	}
}

func TestLoadValidatesTargetCleanupConfiguration(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret-for-target-cleanup-test")
	t.Setenv("PUBLIC_URL", "https://public.example")

	for _, tt := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "non-positive batch size", key: "TARGET_CLEANUP_BATCH_SIZE", value: "0"},
		{name: "non-positive batch budget", key: "TARGET_CLEANUP_MAX_BATCHES_PER_RUN", value: "0"},
		{name: "non-positive duration", key: "TARGET_CLEANUP_MAX_RUN_DURATION", value: "0s"},
		{name: "malformed duration", key: "TARGET_CLEANUP_MAX_RUN_DURATION", value: "not-a-duration"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load must reject %s=%q", tt.key, tt.value)
			}
		})
	}
}

// TestConfigFromEnv tests that environment variables override defaults
func TestConfigFromEnv(t *testing.T) {
	// Set custom environment variables
	if err := os.Setenv("SERVER_PORT", "9999"); err != nil {
		t.Fatalf("Failed to set SERVER_PORT: %v", err)
	}
	if err := os.Setenv("SERVER_GRPC_PORT", "19090"); err != nil {
		t.Fatalf("Failed to set SERVER_GRPC_PORT: %v", err)
	}
	if err := os.Setenv("DB_HOST", "custom-host"); err != nil {
		t.Fatalf("Failed to set DB_HOST: %v", err)
	}
	if err := os.Setenv("DB_PORT", "5433"); err != nil {
		t.Fatalf("Failed to set DB_PORT: %v", err)
	}
	if err := os.Setenv("DB_TIMEZONE", "Asia/Shanghai"); err != nil {
		t.Fatalf("Failed to set DB_TIMEZONE: %v", err)
	}
	if err := os.Setenv("LOG_LEVEL", "debug"); err != nil {
		t.Fatalf("Failed to set LOG_LEVEL: %v", err)
	}
	if err := os.Setenv("ENGINE_PACKAGE_CACHE_ROOT", "/srv/lunafox/engine-packages"); err != nil {
		t.Fatalf("Failed to set ENGINE_PACKAGE_CACHE_ROOT: %v", err)
	}
	if err := os.Setenv("WORKFLOW_DEFINITIONS_ROOT", "/srv/lunafox/workflows"); err != nil {
		t.Fatalf("Failed to set WORKFLOW_DEFINITIONS_ROOT: %v", err)
	}
	if err := os.Setenv("LOKI_URL", "http://custom-loki:3100"); err != nil {
		t.Fatalf("Failed to set LOKI_URL: %v", err)
	}
	if err := os.Setenv("JWT_SECRET", "jwt-secret-for-config-from-env-test"); err != nil {
		t.Fatalf("Failed to set JWT_SECRET: %v", err)
	}
	if err := os.Setenv("PUBLIC_URL", "https://public.example"); err != nil {
		t.Fatalf("Failed to set PUBLIC_URL: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("SERVER_PORT")
		_ = os.Unsetenv("SERVER_GRPC_PORT")
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PORT")
		_ = os.Unsetenv("DB_TIMEZONE")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("ENGINE_PACKAGE_CACHE_ROOT")
		_ = os.Unsetenv("WORKFLOW_DEFINITIONS_ROOT")
		_ = os.Unsetenv("LOKI_URL")
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("PUBLIC_URL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port: expected 9999, got %d", cfg.Server.Port)
	}
	if cfg.Server.GRPCPort != 19090 {
		t.Errorf("Server.GRPCPort: expected 19090, got %d", cfg.Server.GRPCPort)
	}
	if cfg.Database.Host != "custom-host" {
		t.Errorf("Database.Host: expected custom-host, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port: expected 5433, got %d", cfg.Database.Port)
	}
	if cfg.Database.TimeZone != "Asia/Shanghai" {
		t.Errorf("Database.TimeZone: expected Asia/Shanghai, got %s", cfg.Database.TimeZone)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level: expected debug, got %s", cfg.Log.Level)
	}
	if cfg.Storage.EnginePackageCacheRoot != "/srv/lunafox/engine-packages" {
		t.Errorf("Storage.EnginePackageCacheRoot: expected %s, got %s", "/srv/lunafox/engine-packages", cfg.Storage.EnginePackageCacheRoot)
	}
	if cfg.Storage.WorkflowDefinitionsRoot != "/srv/lunafox/workflows" {
		t.Errorf("Storage.WorkflowDefinitionsRoot: expected %s, got %s", "/srv/lunafox/workflows", cfg.Storage.WorkflowDefinitionsRoot)
	}
	if cfg.LokiURL != "http://custom-loki:3100" {
		t.Errorf("LokiURL: expected http://custom-loki:3100, got %s", cfg.LokiURL)
	}
	if cfg.PublicURL != "https://public.example" {
		t.Errorf("PublicURL: expected https://public.example, got %s", cfg.PublicURL)
	}
}

func TestConfigFromViperLoadsEngineInstallDevelopmentSettings(t *testing.T) {
	v := viper.New()
	v.Set("ENGINE_INSTALL_INVENTORY_PATH", "/bootstrap/inventory.yaml")
	v.Set("ENGINE_INSTALL_MAX_PACKAGE_BYTES", int64(64<<20))
	v.Set("ENGINE_INSTALL_DEVELOPMENT_MODE", true)
	v.Set("ENGINE_INSTALL_ALLOW_PLAIN_HTTP", true)
	v.Set("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_IDENTITY_REGISTRY", "localhost:5000")
	v.Set("ENGINE_INSTALL_DEVELOPMENT_RUNTIME_IMAGE_TRANSPORT_REGISTRY", "registry:5000")

	got := configFromViper(v).EngineInstall
	if got.InventoryPath != "/bootstrap/inventory.yaml" || got.MaxPackageBytes != int64(64<<20) {
		t.Fatalf("unexpected base Engine install config: %+v", got)
	}
	if !got.DevelopmentMode || !got.AllowPlainHTTP {
		t.Fatalf("development transport flags were not loaded: %+v", got)
	}
	if got.DevelopmentRuntimeImageIdentityRegistry != "localhost:5000" ||
		got.DevelopmentRuntimeImageTransportRegistry != "registry:5000" {
		t.Fatalf("development Runtime Image Registry mapping was not loaded: %+v", got)
	}
}

func TestConfigFromViperLoadsCloudflareAccelerationSetting(t *testing.T) {
	v := viper.New()
	v.Set("ENGINE_INSTALL_CF_ACCELERATION", true)

	if got := configFromViper(v).EngineInstall.CFAcceleration; !got {
		t.Fatal("Cloudflare acceleration setting was not loaded")
	}
}

func TestLoadRejectsMissingPublicURL(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret-for-missing-public-url-test")
	t.Setenv("PUBLIC_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail when PUBLIC_URL is missing")
	}
}

func TestLoadRejectsWeakJWTSecret(t *testing.T) {
	t.Setenv("PUBLIC_URL", "https://public.example.com:8083")

	tests := []struct {
		name      string
		jwtSecret string
	}{
		{
			name:      "jwt-placeholder",
			jwtSecret: "change-me-in-production-use-a-long-random-string",
		},
		{
			name:      "jwt-empty",
			jwtSecret: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.jwtSecret)

			if _, err := Load(); err == nil {
				t.Fatal("expected Load to fail for weak JWT configuration")
			}
		})
	}
}

func TestLoadRejectsNonHTTPSPublicURL(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt-secret-for-public-url-scheme-test")
	t.Setenv("PUBLIC_URL", "http://public.example.com:8083")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail for non-https PUBLIC_URL")
	}
}

// TestDatabaseDSN tests the DSN generation
func TestDatabaseDSN(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		Name:     "testdb",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	expected := "host=localhost port=5432 user=postgres password=secret dbname=testdb sslmode=disable TimeZone=UTC"
	if cfg.DSN() != expected {
		t.Errorf("DSN: expected %s, got %s", expected, cfg.DSN())
	}
}

// TestRedisAddr tests the Redis address generation
func TestRedisAddr(t *testing.T) {
	cfg := &RedisConfig{
		Host: "localhost",
		Port: 6379,
	}

	expected := "localhost:6379"
	if cfg.Addr() != expected {
		t.Errorf("Addr: expected %s, got %s", expected, cfg.Addr())
	}
}
