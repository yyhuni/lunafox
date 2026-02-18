package config

import (
	"time"

	"github.com/spf13/viper"
)

// setDefaults sets default values for configuration.
func setDefaults(v *viper.Viper) {
	v.SetDefault("SERVER_PORT", 8080)
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

	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	v.SetDefault("JWT_SECRET", "change-me-in-production-use-a-long-random-string")
	v.SetDefault("JWT_ACCESS_EXPIRE", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRE", "168h")

	v.SetDefault("WORDLISTS_BASE_PATH", "/opt/lunafox/wordlists")
	v.SetDefault("WORKER_TOKEN", "change-me-worker-token")
	v.SetDefault("PUBLIC_URL", "")
}

// GetDefaults returns a Config with all default values (for testing).
func GetDefaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
			Mode: "release",
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
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		JWT: JWTConfig{
			Secret:        "change-me-in-production-use-a-long-random-string",
			AccessExpire:  15 * time.Minute,
			RefreshExpire: 168 * time.Hour,
		},
		Storage: StorageConfig{
			WordlistsBasePath: "/opt/lunafox/wordlists",
		},
		Worker: WorkerConfig{
			Token: "change-me-worker-token",
		},
		PublicURL: "",
	}
}
