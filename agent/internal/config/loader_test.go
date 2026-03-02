package config

import (
	"strings"
	"testing"
)

func TestLoadConfigFromEnvAndFlags(t *testing.T) {
	t.Setenv("RUNTIME_GRPC_URL", "https://runtime.example.com:8443")
	t.Setenv("API_KEY", "abc12345")
	t.Setenv("AGENT_VERSION", "v1.2.3")
	t.Setenv("LUNAFOX_AGENT_MAX_TASKS", "5")
	t.Setenv("LUNAFOX_AGENT_CPU_THRESHOLD", "80")
	t.Setenv("LUNAFOX_AGENT_MEM_THRESHOLD", "81")
	t.Setenv("LUNAFOX_AGENT_DISK_THRESHOLD", "82")
	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker:1.0.0")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.RuntimeGRPCURL != "https://runtime.example.com:8443" {
		t.Fatalf("expected runtime grpc url from env")
	}
	if cfg.MaxTasks != 5 {
		t.Fatalf("expected max tasks from env")
	}
	if cfg.WorkerVersion != "1.0.0" {
		t.Fatalf("expected worker version parsed from image ref, got %q", cfg.WorkerVersion)
	}

	args := []string{
		"--runtime-grpc-url=https://runtime-override.example.com:9443",
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
	if cfg.RuntimeGRPCURL != "https://runtime-override.example.com:9443" {
		t.Fatalf("expected runtime grpc url from args")
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

func TestLoadConfigWorkerVersionPrefersExplicitEnv(t *testing.T) {
	t.Setenv("RUNTIME_GRPC_URL", "https://runtime.example.com:8443")
	t.Setenv("API_KEY", "abc12345")
	t.Setenv("AGENT_VERSION", "v1.2.3")
	t.Setenv("WORKER_VERSION", "2.3.4")
	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker:1.0.0")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.WorkerVersion != "2.3.4" {
		t.Fatalf("expected explicit WORKER_VERSION, got %q", cfg.WorkerVersion)
	}
}

func TestLoadConfigMissingRequired(t *testing.T) {
	t.Setenv("RUNTIME_GRPC_URL", "")
	t.Setenv("API_KEY", "")
	t.Setenv("AGENT_VERSION", "v1.2.3")

	_, err := Load([]string{})
	if err == nil {
		t.Fatalf("expected error when required values missing")
	}
}

func TestLoadConfigInvalidEnvValue(t *testing.T) {
	t.Setenv("RUNTIME_GRPC_URL", "https://runtime.example.com")
	t.Setenv("API_KEY", "abc")
	t.Setenv("AGENT_VERSION", "v1.2.3")
	t.Setenv("LUNAFOX_AGENT_MAX_TASKS", "nope")

	_, err := Load([]string{})
	if err == nil {
		t.Fatalf("expected error for invalid LUNAFOX_AGENT_MAX_TASKS")
	}
}

func TestLoadConfigRequiresExplicitRuntimeGRPCURL(t *testing.T) {
	t.Setenv("RUNTIME_GRPC_URL", "")
	t.Setenv("API_KEY", "abc12345")
	t.Setenv("AGENT_VERSION", "v1.2.3")

	_, err := Load([]string{})
	if err == nil {
		t.Fatalf("expected error when runtime url is missing")
	}
	if !strings.Contains(err.Error(), "runtime gRPC URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigDoesNotRequireServerURL(t *testing.T) {
	t.Setenv("SERVER_URL", "")
	t.Setenv("RUNTIME_GRPC_URL", "https://runtime.example.com:8443")
	t.Setenv("API_KEY", "abc12345")
	t.Setenv("AGENT_VERSION", "v1.2.3")

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("expected load success without SERVER_URL: %v", err)
	}
	if cfg.RuntimeGRPCURL != "https://runtime.example.com:8443" {
		t.Fatalf("expected runtime grpc url from env")
	}
}
