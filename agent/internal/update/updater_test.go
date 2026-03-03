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
	payload := domain.UpdateRequiredPayload{
		AgentVersion:   "v1.0.0",
		AgentImageRef:  "yyhuni/lunafox-agent:v1.0.0",
		WorkerImageRef: "yyhuni/lunafox-worker:v1.0.0",
		WorkerVersion:  "1.0.0",
	}

	err := updater.updateOnce(payload)
	if err == nil {
		t.Fatalf("expected error when docker client is nil")
	}
	if !strings.Contains(err.Error(), "docker client unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWorkerTargetPrefersPayload(t *testing.T) {
	imageRef, version, err := resolveWorkerTarget(domain.UpdateRequiredPayload{
		WorkerImageRef: "ghcr.io/acme/lunafox-worker:v2.1.0",
		WorkerVersion:  "2.1.0",
	})
	if err != nil {
		t.Fatalf("resolve worker target failed: %v", err)
	}
	if imageRef != "ghcr.io/acme/lunafox-worker:v2.1.0" || version != "2.1.0" {
		t.Fatalf("unexpected target: %s %s", imageRef, version)
	}
}

func TestResolveWorkerTargetMissingPayloadReturnsError(t *testing.T) {
	_, _, err := resolveWorkerTarget(domain.UpdateRequiredPayload{})
	if err == nil {
		t.Fatalf("expected error when worker target missing")
	}
	if !strings.Contains(err.Error(), "worker image ref is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveWorkerTargetMissingWorkerVersionReturnsError(t *testing.T) {
	_, _, err := resolveWorkerTarget(domain.UpdateRequiredPayload{
		WorkerImageRef: "ghcr.io/acme/lunafox-worker:v2.1.0",
		WorkerVersion:  "",
	})
	if err == nil {
		t.Fatalf("expected error when worker version missing")
	}
	if !strings.Contains(err.Error(), "worker version is required") {
		t.Fatalf("unexpected error: %v", err)
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

func TestResolveRuntimeVolumeName(t *testing.T) {
	t.Setenv(runtimeVolumeNameEnvKey, "")
	if got, err := resolveRuntimeVolumeName(); err != nil || got != "lunafox_runtime" {
		t.Fatalf("expected default runtime volume name, got=%q err=%v", got, err)
	}

	t.Setenv(runtimeVolumeNameEnvKey, "custom_runtime")
	if got, err := resolveRuntimeVolumeName(); err != nil || got != "lunafox_runtime" {
		t.Fatalf("expected runtime volume name from code default, got=%q err=%v", got, err)
	}

	t.Setenv(runtimeVolumeNameEnvKey, "/host/path")
	if got, err := resolveRuntimeVolumeName(); err != nil || got != "lunafox_runtime" {
		t.Fatalf("expected runtime volume name from code default, got=%q err=%v", got, err)
	}
}
