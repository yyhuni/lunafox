package domain

import (
	"testing"
)

func TestValidateAgentConfigRejectsZeroValues(t *testing.T) {
	cases := []struct {
		name  string
		cfg   AgentConfig
		field string
	}{
		{"zero maxTasks", AgentConfig{MaxTasks: 0, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 85}, "maxTasks"},
		{"zero cpuThreshold", AgentConfig{MaxTasks: 5, CPUThreshold: 0, MemThreshold: 80, DiskThreshold: 85}, "cpuThreshold"},
		{"zero memThreshold", AgentConfig{MaxTasks: 5, CPUThreshold: 80, MemThreshold: 0, DiskThreshold: 85}, "memThreshold"},
		{"zero diskThreshold", AgentConfig{MaxTasks: 5, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 0}, "diskThreshold"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAgentConfig(tc.cfg); err == nil {
				t.Fatalf("expected validation error for %s, got nil", tc.field)
			}
		})
	}
}

func TestValidateAgentConfigRejectsOutOfRangeValues(t *testing.T) {
	cases := []struct {
		name string
		cfg  AgentConfig
	}{
		{"negative maxTasks", AgentConfig{MaxTasks: -1, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 85}},
		{"cpuThreshold too high", AgentConfig{MaxTasks: 5, CPUThreshold: 101, MemThreshold: 80, DiskThreshold: 85}},
		{"memThreshold too low", AgentConfig{MaxTasks: 5, CPUThreshold: 80, MemThreshold: 0, DiskThreshold: 85}},
		{"diskThreshold negative", AgentConfig{MaxTasks: 5, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAgentConfig(tc.cfg); err == nil {
				t.Fatalf("expected validation error, got nil")
			}
		})
	}
}

func TestValidateAgentConfigAcceptsLargeMaxTasks(t *testing.T) {
	cases := []struct {
		name     string
		maxTasks int
	}{
		{"maxTasks 100", 100},
		{"maxTasks 500", 500},
		{"maxTasks 1000", 1000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := AgentConfig{MaxTasks: tc.maxTasks, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 85}
			if err := ValidateAgentConfig(cfg); err != nil {
				t.Fatalf("expected valid config with maxTasks=%d, got error: %v", tc.maxTasks, err)
			}
		})
	}
}

func TestValidateAgentConfigAcceptsValidValues(t *testing.T) {
	cases := []struct {
		name string
		cfg  AgentConfig
	}{
		{"typical config", AgentConfig{MaxTasks: 5, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 85}},
		{"large maxTasks", AgentConfig{MaxTasks: 200, CPUThreshold: 80, MemThreshold: 80, DiskThreshold: 85}},
		{"maxTasks 1", AgentConfig{MaxTasks: 1, CPUThreshold: 1, MemThreshold: 1, DiskThreshold: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAgentConfig(tc.cfg); err != nil {
				t.Fatalf("expected valid config, got error: %v", err)
			}
		})
	}
}

func TestApplyRegistrationDefaultsFillsAllNilFields(t *testing.T) {
	opts := ApplyRegistrationDefaults(AgentRegistrationOptions{})

	if *opts.MaxTasks != DefaultMaxTasks {
		t.Fatalf("expected default max tasks %d, got %d", DefaultMaxTasks, *opts.MaxTasks)
	}
	if *opts.CPUThreshold != DefaultCPUThreshold {
		t.Fatalf("expected default cpu threshold %d, got %d", DefaultCPUThreshold, *opts.CPUThreshold)
	}
	if *opts.MemThreshold != DefaultMemThreshold {
		t.Fatalf("expected default mem threshold %d, got %d", DefaultMemThreshold, *opts.MemThreshold)
	}
	if *opts.DiskThreshold != DefaultDiskThreshold {
		t.Fatalf("expected default disk threshold %d, got %d", DefaultDiskThreshold, *opts.DiskThreshold)
	}
}

func TestApplyRegistrationDefaultsPreservesExplicitValues(t *testing.T) {
	maxTasks := 3
	opts := ApplyRegistrationDefaults(AgentRegistrationOptions{MaxTasks: &maxTasks})

	if *opts.MaxTasks != 3 {
		t.Fatalf("expected explicit max tasks 3, got %d", *opts.MaxTasks)
	}
	if *opts.CPUThreshold != DefaultCPUThreshold {
		t.Fatalf("expected default cpu threshold %d, got %d", DefaultCPUThreshold, *opts.CPUThreshold)
	}
}

func TestApplyAgentConfigOnlyOverridesProvidedFields(t *testing.T) {
	mem := 70
	got := ApplyAgentConfig(AgentConfig{
		MaxTasks:      4,
		CPUThreshold:  75,
		MemThreshold:  80,
		DiskThreshold: 85,
	}, AgentConfigUpdate{
		MemThreshold: &mem,
	})

	if got.MaxTasks != 4 {
		t.Fatalf("expected max tasks unchanged, got %d", got.MaxTasks)
	}
	if got.CPUThreshold != 75 {
		t.Fatalf("expected cpu threshold unchanged, got %d", got.CPUThreshold)
	}
	if got.MemThreshold != 70 {
		t.Fatalf("expected mem threshold updated, got %d", got.MemThreshold)
	}
	if got.DiskThreshold != 85 {
		t.Fatalf("expected disk threshold unchanged, got %d", got.DiskThreshold)
	}
}

func TestApplyAgentConfigDoesNotPatchZeroValues(t *testing.T) {
	got := ApplyAgentConfig(AgentConfig{}, AgentConfigUpdate{})

	if got.MaxTasks != 0 || got.CPUThreshold != 0 || got.MemThreshold != 0 || got.DiskThreshold != 0 {
		t.Fatalf("expected zero config to pass through unchanged, got %+v", got)
	}
}
