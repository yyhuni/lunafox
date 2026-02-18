package domain

import (
	"fmt"
	"strings"
)

// AgentConfigUpdate represents optional config updates for an agent.
type AgentConfigUpdate struct {
	MaxTasks      *int
	CPUThreshold  *int
	MemThreshold  *int
	DiskThreshold *int
}

// AgentRegistrationOptions represents optional scheduling config for registration.
type AgentRegistrationOptions struct {
	MaxTasks      *int
	CPUThreshold  *int
	MemThreshold  *int
	DiskThreshold *int
}

// AgentConfig is a compact value object for scheduler limits.
type AgentConfig struct {
	MaxTasks      int
	CPUThreshold  int
	MemThreshold  int
	DiskThreshold int
}

const (
	DefaultMaxTasks      = 10
	DefaultCPUThreshold  = 80
	DefaultMemThreshold  = 80
	DefaultDiskThreshold = 85
)

func NormalizeAgentHostname(hostname string) string {
	normalized := strings.TrimSpace(hostname)
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func BuildAgentName(hostname string) string {
	return fmt.Sprintf("agent-%s", NormalizeAgentHostname(hostname))
}

func ApplyAgentConfig(current AgentConfig, update AgentConfigUpdate) AgentConfig {
	if current.MaxTasks <= 0 {
		current.MaxTasks = DefaultMaxTasks
	}
	if current.CPUThreshold <= 0 {
		current.CPUThreshold = DefaultCPUThreshold
	}
	if current.MemThreshold <= 0 {
		current.MemThreshold = DefaultMemThreshold
	}
	if current.DiskThreshold <= 0 {
		current.DiskThreshold = DefaultDiskThreshold
	}

	result := current
	if update.MaxTasks != nil {
		result.MaxTasks = *update.MaxTasks
	}
	if update.CPUThreshold != nil {
		result.CPUThreshold = *update.CPUThreshold
	}
	if update.MemThreshold != nil {
		result.MemThreshold = *update.MemThreshold
	}
	if update.DiskThreshold != nil {
		result.DiskThreshold = *update.DiskThreshold
	}
	return result
}
