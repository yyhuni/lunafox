package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	defaultMaxTasks      = 5
	defaultCPUThreshold  = 85
	defaultMemThreshold  = 85
	defaultDiskThreshold = 90
)

// Load parses configuration from environment variables and CLI flags.
func Load(args []string) (*Config, error) {
	maxTasks, err := readEnvInt("LUNAFOX_AGENT_MAX_TASKS", defaultMaxTasks)
	if err != nil {
		return nil, err
	}
	cpuThreshold, err := readEnvInt("LUNAFOX_AGENT_CPU_THRESHOLD", defaultCPUThreshold)
	if err != nil {
		return nil, err
	}
	memThreshold, err := readEnvInt("LUNAFOX_AGENT_MEM_THRESHOLD", defaultMemThreshold)
	if err != nil {
		return nil, err
	}
	diskThreshold, err := readEnvInt("LUNAFOX_AGENT_DISK_THRESHOLD", defaultDiskThreshold)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ServerURL:     strings.TrimSpace(os.Getenv("SERVER_URL")),
		APIKey:        strings.TrimSpace(os.Getenv("API_KEY")),
		AgentVersion:  strings.TrimSpace(os.Getenv("AGENT_VERSION")),
		MaxTasks:      maxTasks,
		CPUThreshold:  cpuThreshold,
		MemThreshold:  memThreshold,
		DiskThreshold: diskThreshold,
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	serverURL := fs.String("server-url", cfg.ServerURL, "Server base URL (e.g. https://1.1.1.1:8080)")
	apiKey := fs.String("api-key", cfg.APIKey, "Agent API key")
	maxTasksFlag := fs.Int("max-tasks", cfg.MaxTasks, "Maximum concurrent tasks")
	cpuThresholdFlag := fs.Int("cpu-threshold", cfg.CPUThreshold, "CPU threshold percentage")
	memThresholdFlag := fs.Int("mem-threshold", cfg.MemThreshold, "Memory threshold percentage")
	diskThresholdFlag := fs.Int("disk-threshold", cfg.DiskThreshold, "Disk threshold percentage")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.ServerURL = strings.TrimSpace(*serverURL)
	cfg.APIKey = strings.TrimSpace(*apiKey)
	cfg.MaxTasks = *maxTasksFlag
	cfg.CPUThreshold = *cpuThresholdFlag
	cfg.MemThreshold = *memThresholdFlag
	cfg.DiskThreshold = *diskThresholdFlag

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func readEnvInt(key string, fallback int) (int, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}
