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
