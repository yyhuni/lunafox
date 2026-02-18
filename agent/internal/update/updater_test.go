package update

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

func TestWithJitterRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	delay := 10 * time.Second
	got := withJitter(delay, rng)
	if got < delay {
		t.Fatalf("expected jitter >= delay")
	}
	if got > delay+(delay/5) {
		t.Fatalf("expected jitter <= 20%%")
	}
}

func TestUpdateOnceDockerUnavailable(t *testing.T) {
	updater := &Updater{}
	payload := domain.UpdateRequiredPayload{Version: "v1.0.0", ImageRef: "yyhuni/lunafox-agent:v1.0.0"}

	err := updater.updateOnce(payload)
	if err == nil {
		t.Fatalf("expected error when docker client is nil")
	}
	if !strings.Contains(err.Error(), "docker client unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWorkerImageRef(t *testing.T) {
	t.Setenv("WORKER_IMAGE_REF", "")
	if _, err := resolveWorkerImageRef(); err == nil {
		t.Fatalf("expected missing WORKER_IMAGE_REF error")
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker:v9")
	if got, err := resolveWorkerImageRef(); err != nil || got != "ghcr.io/acme/lunafox-worker:v9" {
		t.Fatalf("expected raw worker image ref passthrough, got %s, err: %v", got, err)
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker")
	if _, err := resolveWorkerImageRef(); err == nil {
		t.Fatalf("expected error for worker image ref without tag or digest")
	}

	t.Setenv("WORKER_IMAGE_REF", "ghcr.io/acme/lunafox-worker@sha256:abc")
	if got, err := resolveWorkerImageRef(); err != nil || got != "ghcr.io/acme/lunafox-worker@sha256:abc" {
		t.Fatalf("expected digest worker image ref passthrough, got %s, err: %v", got, err)
	}
}

func TestResolveSharedDataBind(t *testing.T) {
	t.Setenv(sharedDataVolumeBindEnvKey, "")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected missing shared data bind env error")
	}

	t.Setenv(sharedDataVolumeBindEnvKey, "custom_data")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected missing target error")
	}

	t.Setenv(sharedDataVolumeBindEnvKey, "custom_data:/opt/lunafox:ro")
	if got, err := resolveSharedDataVolumeBind(); err != nil || got != "custom_data:/opt/lunafox:ro" {
		t.Fatalf("expected bind with mode, got %s, err: %v", got, err)
	}

	t.Setenv(sharedDataVolumeBindEnvKey, "custom_data:/tmp")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected invalid target error")
	}

	t.Setenv(sharedDataVolumeBindEnvKey, "/data/lunafox:/opt/lunafox")
	if _, err := resolveSharedDataVolumeBind(); err == nil {
		t.Fatalf("expected host path bind mount error")
	}
}
