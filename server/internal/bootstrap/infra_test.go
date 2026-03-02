package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResolveAgentImageRef(t *testing.T) {
	t.Setenv("AGENT_IMAGE_REF", "")
	if _, err := resolveAgentImageRef(); err == nil {
		t.Fatalf("expected error for missing AGENT_IMAGE_REF")
	}

	t.Setenv("AGENT_IMAGE_REF", "docker.io/yyhuni/lunafox-agent")
	if _, err := resolveAgentImageRef(); err == nil {
		t.Fatalf("expected error for AGENT_IMAGE_REF without tag or digest")
	}

	t.Setenv("AGENT_IMAGE_REF", "docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if got, err := resolveAgentImageRef(); err != nil || got == "" {
		t.Fatalf("expected valid AGENT_IMAGE_REF, got %q, err=%v", got, err)
	}
}

func TestResolveAgentVersion(t *testing.T) {
	t.Setenv("AGENT_VERSION", "")
	if _, err := resolveAgentVersion(); err == nil {
		t.Fatalf("expected error for missing AGENT_VERSION")
	}

	t.Setenv("AGENT_VERSION", "v1.2")
	if _, err := resolveAgentVersion(); err == nil {
		t.Fatalf("expected error for invalid AGENT_VERSION")
	}

	t.Setenv("AGENT_VERSION", "v1.2.3")
	if got, err := resolveAgentVersion(); err != nil || got != "1.2.3" {
		t.Fatalf("expected normalized AGENT_VERSION, got %q, err=%v", got, err)
	}
}

func TestResolveWorkerImageRef(t *testing.T) {
	t.Setenv("WORKER_IMAGE_REF", "")
	if _, err := resolveWorkerImageRef(); err == nil {
		t.Fatalf("expected error for missing WORKER_IMAGE_REF")
	}

	t.Setenv("WORKER_IMAGE_REF", "docker.io/yyhuni/lunafox-worker")
	if _, err := resolveWorkerImageRef(); err == nil {
		t.Fatalf("expected error for WORKER_IMAGE_REF without tag or digest")
	}

	t.Setenv("WORKER_IMAGE_REF", "docker.io/yyhuni/lunafox-worker@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if got, err := resolveWorkerImageRef(); err != nil || got == "" {
		t.Fatalf("expected valid WORKER_IMAGE_REF, got %q, err=%v", got, err)
	}
}

func TestResolveWorkerVersion(t *testing.T) {
	t.Setenv("WORKER_VERSION", "")
	if _, err := resolveWorkerVersion(); err == nil {
		t.Fatalf("expected error for missing WORKER_VERSION")
	}

	t.Setenv("WORKER_VERSION", "v1.2")
	if _, err := resolveWorkerVersion(); err == nil {
		t.Fatalf("expected error for invalid WORKER_VERSION")
	}

	t.Setenv("WORKER_VERSION", "v1.2.3")
	if got, err := resolveWorkerVersion(); err != nil || got != "1.2.3" {
		t.Fatalf("expected normalized WORKER_VERSION, got %q, err=%v", got, err)
	}
}

func TestEnsureRuntimeVersionConsistency(t *testing.T) {
	if err := ensureRuntimeVersionConsistency("v1.2.3", "1.2.3", "1.2.3"); err != nil {
		t.Fatalf("expected versions to be treated as consistent, got %v", err)
	}

	if err := ensureRuntimeVersionConsistency("v1.2.4", "1.2.3", "1.2.3"); err == nil {
		t.Fatalf("expected mismatch between IMAGE_TAG and AGENT_VERSION")
	}

	if err := ensureRuntimeVersionConsistency("v1.2.3", "1.2.3", "1.2.4"); err == nil {
		t.Fatalf("expected mismatch between AGENT_VERSION and WORKER_VERSION")
	}
}

func TestResolveSharedDataVolumeBind(t *testing.T) {
	t.Setenv("LUNAFOX_SHARED_DATA_VOLUME_BIND", "")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected error for missing LUNAFOX_SHARED_DATA_VOLUME_BIND")
	}

	t.Setenv("LUNAFOX_SHARED_DATA_VOLUME_BIND", "lunafox_data")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected error for invalid bind format")
	}

	t.Setenv("LUNAFOX_SHARED_DATA_VOLUME_BIND", "/data/lunafox:/opt/lunafox")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected error for host path bind mount")
	}

	t.Setenv("LUNAFOX_SHARED_DATA_VOLUME_BIND", "lunafox_data:/tmp")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected error for invalid target path")
	}

	t.Setenv("LUNAFOX_SHARED_DATA_VOLUME_BIND", "lunafox_data:/opt/lunafox")
	if got, err := resolveSharedDataVolumeBind(); err != nil || got != "lunafox_data:/opt/lunafox" {
		t.Fatalf("expected valid bind, got %q, err=%v", got, err)
	}
}

func TestWaitForLokiReadyRetriesUntilSuccess(t *testing.T) {
	t.Parallel()

	var attempts int
	err := waitForLokiReady(func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("loki unavailable: /ready status=503")
		}
		return nil
	}, 200*time.Millisecond, 40*time.Millisecond, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("expected readiness success after retries, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestWaitForLokiReadyTimeout(t *testing.T) {
	t.Parallel()

	var attempts int
	err := waitForLokiReady(func(ctx context.Context) error {
		attempts++
		return errors.New("loki unavailable: /ready status=503")
	}, 120*time.Millisecond, 40*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected readiness timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error message, got %v", err)
	}
	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts before timeout, got %d", attempts)
	}
}
