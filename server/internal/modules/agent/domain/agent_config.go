package domain

import "errors"

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

// AgentConfig holds per-agent scheduling thresholds. These values are pushed
// to the agent via ConfigUpdate and enforced by the agent's own canPull() gate.
// The server does NOT use them for task assignment decisions.
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

// ValidateAgentConfig returns an error if any config field is missing or out of range.
func ValidateAgentConfig(cfg AgentConfig) error {
	if cfg.MaxTasks < 1 {
		return errors.New("maxTasks must be at least 1")
	}
	if cfg.CPUThreshold < 1 || cfg.CPUThreshold > 100 {
		return errors.New("cpuThreshold must be between 1 and 100")
	}
	if cfg.MemThreshold < 1 || cfg.MemThreshold > 100 {
		return errors.New("memThreshold must be between 1 and 100")
	}
	if cfg.DiskThreshold < 1 || cfg.DiskThreshold > 100 {
		return errors.New("diskThreshold must be between 1 and 100")
	}
	return nil
}

// ApplyRegistrationDefaults fills zero-valued fields with built-in defaults.
// Use only during agent registration when the caller omits config options.
func ApplyRegistrationDefaults(opts AgentRegistrationOptions) AgentRegistrationOptions {
	if opts.MaxTasks == nil {
		v := DefaultMaxTasks
		opts.MaxTasks = &v
	}
	if opts.CPUThreshold == nil {
		v := DefaultCPUThreshold
		opts.CPUThreshold = &v
	}
	if opts.MemThreshold == nil {
		v := DefaultMemThreshold
		opts.MemThreshold = &v
	}
	if opts.DiskThreshold == nil {
		v := DefaultDiskThreshold
		opts.DiskThreshold = &v
	}
	return opts
}

// ApplyAgentConfig merges an update onto the current config.
// All fields in current must already be valid; use ValidateAgentConfig to enforce this.
func ApplyAgentConfig(current AgentConfig, update AgentConfigUpdate) AgentConfig {
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
