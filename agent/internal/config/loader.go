package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/contracts/runtimecontract"
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
		RuntimeGRPCURL: strings.TrimSpace(os.Getenv("RUNTIME_GRPC_URL")),
		APIKey:         strings.TrimSpace(os.Getenv(AgentAPIKeyEnv)),
		AgentVersion:   strings.TrimSpace(os.Getenv("AGENT_VERSION")),
		WorkerVersion:  resolveWorkerVersion(),
		MaxTasks:       maxTasks,
		CPUThreshold:   cpuThreshold,
		MemThreshold:   memThreshold,
		DiskThreshold:  diskThreshold,
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	runtimeGRPCURL := fs.String("runtime-grpc-url", cfg.RuntimeGRPCURL, "Runtime gRPC URL (e.g. https://1.1.1.1:18443)")
	apiKey := fs.String("agent-api-key", cfg.APIKey, "Agent API key")
	maxTasksFlag := fs.Int("max-tasks", cfg.MaxTasks, "Maximum concurrent tasks")
	cpuThresholdFlag := fs.Int("cpu-threshold", cfg.CPUThreshold, "CPU threshold percentage")
	memThresholdFlag := fs.Int("mem-threshold", cfg.MemThreshold, "Memory threshold percentage")
	diskThresholdFlag := fs.Int("disk-threshold", cfg.DiskThreshold, "Disk threshold percentage")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.RuntimeGRPCURL = strings.TrimSpace(*runtimeGRPCURL)
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

func resolveWorkerVersion() string {
	// Runtime contract: WORKER_VERSION must be bare SemVer (e.g. 1.2.3),
	// not v-prefixed, to keep a single canonical format across components.
	return runtimecontract.NormalizeVersion(strings.TrimSpace(os.Getenv("WORKER_VERSION")))
}
