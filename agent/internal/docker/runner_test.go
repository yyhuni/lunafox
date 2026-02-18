package docker

import (
	"testing"

	"github.com/yyhuni/lunafox/agent/internal/domain"
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

	env := buildWorkerEnv(spec, "https://server", "token")
	expected := []string{
		"SERVER_URL=https://server",
		"SERVER_TOKEN=token",
		"SCAN_ID=1",
		"TARGET_ID=2",
		"TARGET_NAME=example.com",
		"TARGET_TYPE=domain",
		"WORKFLOW_NAME=subdomain_discovery",
		"WORKSPACE_DIR=/opt/lunafox/results",
		"CONFIG=config-yaml",
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
