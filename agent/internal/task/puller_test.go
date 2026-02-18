package task

import (
	"math/rand"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/agent/internal/domain"
)

func TestPullerUpdateConfig(t *testing.T) {
	p := NewPuller(nil, nil, nil, 5, 85, 86, 87)
	max, cpu, mem, disk := p.currentConfig()
	if max != 5 || cpu != 85 || mem != 86 || disk != 87 {
		t.Fatalf("unexpected initial config")
	}

	maxUpdate := 8
	cpuUpdate := 70
	p.UpdateConfig(&maxUpdate, &cpuUpdate, nil, nil)
	max, cpu, mem, disk = p.currentConfig()
	if max != 8 || cpu != 70 || mem != 86 || disk != 87 {
		t.Fatalf("unexpected updated config")
	}
}

func TestPullerPause(t *testing.T) {
	p := NewPuller(nil, nil, nil, 1, 1, 1, 1)
	p.Pause()
	if !p.paused.Load() {
		t.Fatalf("expected paused")
	}
}

func TestPullerEnsureTaskHandler(t *testing.T) {
	p := NewPuller(nil, nil, nil, 1, 1, 1, 1)
	if err := p.EnsureTaskHandler(); err == nil {
		t.Fatalf("expected error when handler missing")
	}
	p.SetOnTask(func(*domain.Task) {})
	if err := p.EnsureTaskHandler(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPullerNextEmptyDelay(t *testing.T) {
	p := NewPuller(nil, nil, nil, 1, 1, 1, 1)
	p.emptyBackoff = []time.Duration{5 * time.Second, 10 * time.Second}

	if delay := p.nextEmptyDelay(8 * time.Second); delay != 8*time.Second {
		t.Fatalf("expected delay to honor load interval, got %v", delay)
	}
	if delay := p.nextEmptyDelay(1 * time.Second); delay != 10*time.Second {
		t.Fatalf("expected backoff delay, got %v", delay)
	}
	if p.emptyIdx != 2 {
		t.Fatalf("expected empty index to advance")
	}
	p.resetEmptyBackoff()
	if p.emptyIdx != 0 {
		t.Fatalf("expected empty index reset")
	}
}

func TestPullerErrorBackoff(t *testing.T) {
	p := NewPuller(nil, nil, nil, 1, 1, 1, 1)
	p.randSrc = rand.New(rand.NewSource(1))

	first := p.nextErrorBackoff()
	if first < time.Second || first > time.Second+(time.Second/5) {
		t.Fatalf("unexpected backoff %v", first)
	}
	if p.errorBackoff != 2*time.Second {
		t.Fatalf("expected backoff to double")
	}

	second := p.nextErrorBackoff()
	if second < 2*time.Second || second > 2*time.Second+(2*time.Second/5) {
		t.Fatalf("unexpected backoff %v", second)
	}
	if p.errorBackoff != 4*time.Second {
		t.Fatalf("expected backoff to double")
	}

	p.resetErrorBackoff()
	if p.errorBackoff != time.Second {
		t.Fatalf("expected error backoff reset")
	}
}

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
