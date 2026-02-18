package config

import (
	"testing"
)

func TestLoadConfigFromEnvAndFlags(t *testing.T) {
	t.Setenv("SERVER_URL", "https://example.com")
	t.Setenv("API_KEY", "abc12345")
	t.Setenv("AGENT_VERSION", "v1.2.3")
	t.Setenv("LUNAFOX_AGENT_MAX_TASKS", "5")
	t.Setenv("LUNAFOX_AGENT_CPU_THRESHOLD", "80")
	t.Setenv("LUNAFOX_AGENT_MEM_THRESHOLD", "81")
	t.Setenv("LUNAFOX_AGENT_DISK_THRESHOLD", "82")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.ServerURL != "https://example.com" {
		t.Fatalf("expected server url from env")
	}
	if cfg.MaxTasks != 5 {
		t.Fatalf("expected max tasks from env")
	}

	args := []string{
		"--server-url=https://override.example.com",
		"--api-key=deadbeef",
		"--max-tasks=9",
		"--cpu-threshold=70",
		"--mem-threshold=71",
		"--disk-threshold=72",
	}
	cfg, err = Load(args)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.ServerURL != "https://override.example.com" {
		t.Fatalf("expected server url from args")
	}
	if cfg.APIKey != "deadbeef" {
		t.Fatalf("expected api key from args")
	}
	if cfg.MaxTasks != 9 {
		t.Fatalf("expected max tasks from args")
	}
	if cfg.CPUThreshold != 70 || cfg.MemThreshold != 71 || cfg.DiskThreshold != 72 {
		t.Fatalf("expected thresholds from args")
	}
}

func TestLoadConfigMissingRequired(t *testing.T) {
	t.Setenv("SERVER_URL", "")
	t.Setenv("API_KEY", "")
	t.Setenv("AGENT_VERSION", "v1.2.3")

	_, err := Load([]string{})
	if err == nil {
		t.Fatalf("expected error when required values missing")
	}
}

func TestLoadConfigInvalidEnvValue(t *testing.T) {
	t.Setenv("SERVER_URL", "https://example.com")
	t.Setenv("API_KEY", "abc")
	t.Setenv("AGENT_VERSION", "v1.2.3")
	t.Setenv("LUNAFOX_AGENT_MAX_TASKS", "nope")

	_, err := Load([]string{})
	if err == nil {
		t.Fatalf("expected error for invalid LUNAFOX_AGENT_MAX_TASKS")
	}
}

