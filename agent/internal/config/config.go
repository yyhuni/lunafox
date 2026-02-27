package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Config represents runtime settings for the agent.
type Config struct {
	RuntimeGRPCURL string
	APIKey         string
	AgentVersion   string
	MaxTasks       int
	CPUThreshold   int
	MemThreshold   int
	DiskThreshold  int
}

// Validate ensures config values are usable.
func (c *Config) Validate() error {
	if c.RuntimeGRPCURL == "" {
		return errors.New("runtime gRPC URL is required")
	}
	if c.APIKey == "" {
		return errors.New("api key is required")
	}
	if c.AgentVersion == "" {
		return errors.New("AGENT_VERSION environment variable is required")
	}
	if c.MaxTasks < 1 {
		return errors.New("max tasks must be at least 1")
	}
	if err := validatePercent("cpu threshold", c.CPUThreshold); err != nil {
		return err
	}
	if err := validatePercent("mem threshold", c.MemThreshold); err != nil {
		return err
	}
	if err := validatePercent("disk threshold", c.DiskThreshold); err != nil {
		return err
	}
	if err := validateRuntimeGRPCURL(c.RuntimeGRPCURL); err != nil {
		return err
	}
	return nil
}

func validatePercent(name string, value int) error {
	if value < 1 || value > 100 {
		return fmt.Errorf("%s must be between 1 and 100", name)
	}
	return nil
}

func validateRuntimeGRPCURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "ws", "wss":
	default:
		if parsed.Scheme == "" {
			return errors.New("server URL scheme is required")
		}
		return fmt.Errorf("unsupported server URL scheme: %s", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return errors.New("server host is required")
	}
	return nil
}
