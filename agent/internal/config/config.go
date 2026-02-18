package config

import (
	"errors"
	"fmt"
)

// Config represents runtime settings for the agent.
type Config struct {
	ServerURL     string
	APIKey        string
	AgentVersion  string
	MaxTasks      int
	CPUThreshold  int
	MemThreshold  int
	DiskThreshold int
}

// Validate ensures config values are usable.
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return errors.New("server URL is required")
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
	if _, err := BuildWebSocketURL(c.ServerURL); err != nil {
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
