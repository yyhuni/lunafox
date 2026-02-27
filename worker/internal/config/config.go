package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

var (
	ErrMissingTaskID       = errors.New("missing required configuration: TASK_ID")
	ErrMissingAgentSocket  = errors.New("missing required configuration: AGENT_SOCKET")
	ErrMissingTaskToken    = errors.New("missing required configuration: TASK_TOKEN")
	ErrMissingScanID       = errors.New("missing required configuration: SCAN_ID")
	ErrMissingTargetID     = errors.New("missing required configuration: TARGET_ID")
	ErrMissingTargetName   = errors.New("missing required configuration: TARGET_NAME")
	ErrMissingTargetType   = errors.New("missing required configuration: TARGET_TYPE")
	ErrMissingWorkflowName = errors.New("missing required configuration: WORKFLOW_NAME")
	ErrMissingConfig       = errors.New("missing required configuration: CONFIG")
)

// Config holds all configuration for the worker
type Config struct {
	// Runtime connection
	TaskID      int
	AgentSocket string
	TaskToken   string

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
		TaskID:       getEnvAsInt("TASK_ID", 0),
		AgentSocket:  os.Getenv("AGENT_SOCKET"),
		TaskToken:    os.Getenv("TASK_TOKEN"),
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

// Validate checks that all required configuration is present
func (c *Config) Validate() error {
	if c.TaskID == 0 {
		return ErrMissingTaskID
	}
	if c.AgentSocket == "" {
		return ErrMissingAgentSocket
	}
	if c.TaskToken == "" {
		return ErrMissingTaskToken
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
