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
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
		JWT: JWTConfig{
			Secret:        v.GetString("JWT_SECRET"),
			AccessExpire:  v.GetDuration("JWT_ACCESS_EXPIRE"),
			RefreshExpire: v.GetDuration("JWT_REFRESH_EXPIRE"),
		},
		Storage: StorageConfig{
			WordlistsBasePath: v.GetString("WORDLISTS_BASE_PATH"),
		},
		Worker: WorkerConfig{
			Token: v.GetString("WORKER_TOKEN"),
		},
		PublicURL: v.GetString("PUBLIC_URL"),
	}
}
