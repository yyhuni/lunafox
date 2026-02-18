package config

import "time"

// Config holds all configuration for the application.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Log       LogConfig
	JWT       JWTConfig
	Storage   StorageConfig
	Worker    WorkerConfig
	PublicURL string
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port int    `mapstructure:"SERVER_PORT"`
	Mode string `mapstructure:"GIN_MODE"`
}

// WorkerConfig holds worker-related configuration.
type WorkerConfig struct {
	Token string `mapstructure:"WORKER_TOKEN"`
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
	Level  string `mapstructure:"LOG_LEVEL"`
	Format string `mapstructure:"LOG_FORMAT"`
}

// JWTConfig holds JWT-related configuration.
type JWTConfig struct {
	Secret        string        `mapstructure:"JWT_SECRET"`
	AccessExpire  time.Duration `mapstructure:"JWT_ACCESS_EXPIRE"`
	RefreshExpire time.Duration `mapstructure:"JWT_REFRESH_EXPIRE"`
}

// StorageConfig holds storage-related configuration.
type StorageConfig struct {
	WordlistsBasePath string `mapstructure:"WORDLISTS_BASE_PATH"`
}
