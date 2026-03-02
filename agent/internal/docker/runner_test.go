package docker

import (
	"strings"
	"testing"

	"github.com/yyhuni/lunafox/agent/internal/domain"
	"github.com/yyhuni/lunafox/contracts/runtimecontract"
)

func TestResolveWorkerImage(t *testing.T) {
	t.Setenv("WORKER_IMAGE_REF", "")
	if _, err := resolveWorkerImage(); err == nil {
		t.Fatalf("expected error for missing WORKER_IMAGE_REF")
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker:v1.9.9")
	if got, err := resolveWorkerImage(); err != nil || got != "ghcr.io/acme/lunafox-worker:v1.9.9" {
		t.Fatalf("expected WORKER_IMAGE_REF image, got %s, err: %v", got, err)
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker")
	if _, err := resolveWorkerImage(); err == nil {
		t.Fatalf("expected error for WORKER_IMAGE_REF without tag/digest")
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker@sha256:abc")
	if got, err := resolveWorkerImage(); err != nil || got != "ghcr.io/acme/lunafox-worker@sha256:abc" {
		t.Fatalf("expected digest WORKER_IMAGE_REF, got %s, err: %v", got, err)
	}
}

func TestBuildWorkerEnv(t *testing.T) {
	spec := &domain.Task{
		ScanID:       1,
		TargetID:     2,
		TargetName:   "example.com",
		TargetType:   "domain",
		WorkflowName: "subdomain_discovery",
		WorkspaceDir: "/opt/lunafox/results",
		Config:       "config-yaml",
	}

	env := buildWorkerEnv(spec, "/run/lunafox/worker-runtime.sock", "task-token", "/opt/lunafox/results/task_config.yaml")
	expected := []string{
		"TASK_ID=0",
		"SCAN_ID=1",
		"TARGET_ID=2",
		"TARGET_NAME=example.com",
		"TARGET_TYPE=domain",
		"WORKFLOW_NAME=subdomain_discovery",
		"WORKSPACE_DIR=/opt/lunafox/results",
		"CONFIG_PATH=/opt/lunafox/results/task_config.yaml",
		"AGENT_SOCKET=/run/lunafox/worker-runtime.sock",
		"TASK_TOKEN=task-token",
	}

	if len(env) != len(expected) {
		t.Fatalf("expected %d env entries, got %d", len(expected), len(env))
	}
	for i, item := range expected {
		if env[i] != item {
			t.Fatalf("expected env[%d]=%s got %s", i, item, env[i])
		}
	}
}

func TestWriteTaskConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath, err := writeTaskConfigFile(dir, "k: v\n")
	if err != nil {
		t.Fatalf("write task config failed: %v", err)
	}
	expected := runtimecontract.BuildTaskConfigPath(dir)
	if configPath != expected {
		t.Fatalf("unexpected config path: %s", configPath)
	}
}

func TestBuildWorkerEnvDoesNotEmbedRawConfigPayload(t *testing.T) {
	spec := &domain.Task{
		ScanID:       1,
		TargetID:     2,
		TargetName:   "example.com",
		TargetType:   "domain",
		WorkflowName: "subdomain_discovery",
		WorkspaceDir: "/opt/lunafox/results/scan_1/task_1",
		Config:       strings.Repeat("a", 20000),
	}
	env := buildWorkerEnv(spec, "/run/lunafox/worker-runtime.sock", "task-token", "/opt/lunafox/results/scan_1/task_1/task_config.yaml")
	for _, item := range env {
		if strings.HasPrefix(item, "CONFIG=") {
			t.Fatalf("raw CONFIG env should not be used in worker container env")
		}
		if strings.Contains(item, spec.Config) {
			t.Fatalf("sensitive/large config content must not be exposed through env vars")
		}
	}
}

func TestResolveSharedDataBind(t *testing.T) {
	t.Run("missing env", func(t *testing.T) {
		t.Setenv(sharedDataVolumeBindEnvKey, "")
		if _, err := resolveSharedDataVolumeBind(); err == nil {
			t.Fatalf("expected missing %s to fail", sharedDataVolumeBindEnvKey)
		}
	})

	t.Run("reject volume name only", func(t *testing.T) {
		t.Setenv(sharedDataVolumeBindEnvKey, "custom_data")
		if _, err := resolveSharedDataVolumeBind(); err == nil {
			t.Fatalf("expected missing target to fail")
		}
	})

	t.Run("full bind with mode", func(t *testing.T) {
		t.Setenv(sharedDataVolumeBindEnvKey, "custom_data:/opt/lunafox:ro")
		got, err := resolveSharedDataVolumeBind()
		if err != nil {
			t.Fatalf("resolve bind with mode: %v", err)
		}
		if got != "custom_data:/opt/lunafox:ro" {
			t.Fatalf("unexpected bind: %s", got)
		}
	})

	t.Run("reject invalid target", func(t *testing.T) {
		t.Setenv(sharedDataVolumeBindEnvKey, "custom_data:/tmp")
		if _, err := resolveSharedDataVolumeBind(); err == nil {
			t.Fatalf("expected invalid target to fail")
		}
	})

	t.Run("reject host path bind mount", func(t *testing.T) {
		t.Setenv(sharedDataVolumeBindEnvKey, "/data/lunafox:/opt/lunafox")
		if _, err := resolveSharedDataVolumeBind(); err == nil {
			t.Fatalf("expected host path bind mount to fail")
		}
	})
}

func TestResolveRuntimeVolumeBind(t *testing.T) {
	t.Run("default volume", func(t *testing.T) {
		t.Setenv(runtimeVolumeNameEnvKey, "")
		got, err := resolveRuntimeVolumeBind()
		if err != nil {
			t.Fatalf("resolve runtime volume bind: %v", err)
		}
		if got != "lunafox_runtime:/run/lunafox:ro" {
			t.Fatalf("unexpected default runtime bind: %s", got)
		}
	})

	t.Run("ignore env override", func(t *testing.T) {
		t.Setenv(runtimeVolumeNameEnvKey, "custom_runtime")
		got, err := resolveRuntimeVolumeBind()
		if err != nil {
			t.Fatalf("resolve runtime volume bind: %v", err)
		}
		if got != "lunafox_runtime:/run/lunafox:ro" {
			t.Fatalf("unexpected runtime bind: %s", got)
		}
	})

	t.Run("ignore invalid env value", func(t *testing.T) {
		t.Setenv(runtimeVolumeNameEnvKey, "/host/path")
		got, err := resolveRuntimeVolumeBind()
		if err != nil {
			t.Fatalf("resolve runtime volume bind: %v", err)
		}
		if got != "lunafox_runtime:/run/lunafox:ro" {
			t.Fatalf("unexpected runtime bind: %s", got)
		}
	})
}
