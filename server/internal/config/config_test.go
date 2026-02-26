package config

import (
	"os"
	"testing"
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
		"LOG_LEVEL", "LOG_FORMAT",
		"PUBLIC_URL",
	}
	for _, env := range envVars {
		if err := os.Unsetenv(env); err != nil {
			t.Logf("Warning: failed to unset %s: %v", env, err)
		}
	}

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
	if cfg.Log.Format != defaults.Log.Format {
		t.Errorf("Log.Format: expected %s, got %s", defaults.Log.Format, cfg.Log.Format)
	}
	if cfg.PublicURL != defaults.PublicURL {
		t.Errorf("PublicURL: expected %s, got %s", defaults.PublicURL, cfg.PublicURL)
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
	if err := os.Setenv("LOKI_URL", "http://custom-loki:3100"); err != nil {
		t.Fatalf("Failed to set LOKI_URL: %v", err)
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
		_ = os.Unsetenv("LOKI_URL")
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
	if cfg.LokiURL != "http://custom-loki:3100" {
		t.Errorf("LokiURL: expected http://custom-loki:3100, got %s", cfg.LokiURL)
	}
	if cfg.PublicURL != "https://public.example" {
		t.Errorf("PublicURL: expected https://public.example, got %s", cfg.PublicURL)
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
