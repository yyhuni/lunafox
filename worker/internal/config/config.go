package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

var (
	ErrMissingServerURL    = errors.New("missing required configuration: SERVER_URL")
	ErrMissingServerToken  = errors.New("missing required configuration: SERVER_TOKEN")
	ErrMissingScanID       = errors.New("missing required configuration: SCAN_ID")
	ErrMissingTargetID     = errors.New("missing required configuration: TARGET_ID")
	ErrMissingTargetName   = errors.New("missing required configuration: TARGET_NAME")
	ErrMissingTargetType   = errors.New("missing required configuration: TARGET_TYPE")
	ErrMissingWorkflowName = errors.New("missing required configuration: WORKFLOW_NAME")
	ErrMissingConfig       = errors.New("missing required configuration: CONFIG")
)

// Config holds all configuration for the worker
type Config struct {
	// Server connection
	ServerURL   string
	ServerToken string

	// Task parameters (from environment variables)
	ScanID       int
	TargetID     int
	TargetName   string
	TargetType   string // "domain", "ip", "cidr", "url"
	WorkflowName string // e.g., "subdomain_discovery", "website_scan"
	WorkspaceDir string // Base directory for workflow execution
	Config       map[string]any

	// Paths
	LogLevel string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (for local development)
	_ = godotenv.Load()

	cfg := &Config{
		ServerURL:    os.Getenv("SERVER_URL"),
		ServerToken:  os.Getenv("SERVER_TOKEN"),
		ScanID:       getEnvAsInt("SCAN_ID", 0),
		TargetID:     getEnvAsInt("TARGET_ID", 0),
		TargetName:   os.Getenv("TARGET_NAME"),
		TargetType:   os.Getenv("TARGET_TYPE"),
		WorkflowName: os.Getenv("WORKFLOW_NAME"),
		WorkspaceDir: getEnvOrDefault("WORKSPACE_DIR", "/opt/lunafox/workspace"),
		LogLevel:     getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Parse YAML config from environment variable
	configYAML := os.Getenv("CONFIG")
	if configYAML != "" {
		var config map[string]any
		if err := yaml.Unmarshal([]byte(configYAML), &config); err != nil {
			return nil, err
		}
		cfg.Config = config
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// InputURL returns the URL to download input data for this scan
func (c *Config) InputURL() string {
	return c.ServerURL + "/api/scans/" + strconv.Itoa(c.ScanID) + "/input/"
}

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return ErrMissingServerURL
	}
	if c.ServerToken == "" {
		return ErrMissingServerToken
	}
	if c.ScanID == 0 {
		return ErrMissingScanID
	}
	if c.TargetID == 0 {
		return ErrMissingTargetID
	}
	if c.TargetName == "" {
		return ErrMissingTargetName
	}
	if c.TargetType == "" {
		return ErrMissingTargetType
	}
	if c.WorkflowName == "" {
		return ErrMissingWorkflowName
	}
	if c.Config == nil {
		return ErrMissingConfig
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
