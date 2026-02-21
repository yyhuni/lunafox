package steps

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/docker"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
	"github.com/yyhuni/lunafox/tools/installer/internal/ui"
)

func TestNormalizeImageCandidates(t *testing.T) {
	list := normalizeImageCandidates(
		[]string{
			"docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"ghcr.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	)

	want := []string{
		"docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"ghcr.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("unexpected candidates: got=%v want=%v", list, want)
	}
}

func TestBuildSortedCandidatesPrefersSuccessfulLowerLatency(t *testing.T) {
	refs := []string{
		"docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"ghcr.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"ccr.ccs.tencentyun.com/yyhuni/lunafox-agent@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	probes := map[string]imageProbeResult{
		refs[0]: {
			success: true,
			latency: 80 * time.Millisecond,
		},
		refs[1]: {
			success: false,
			latency: 10 * time.Millisecond,
		},
		refs[2]: {
			success: true,
			latency: 30 * time.Millisecond,
		},
	}

	candidates := buildSortedCandidates(refs, probes)
	if len(candidates) != 3 {
		t.Fatalf("unexpected candidate length: %d", len(candidates))
	}

	got := []string{candidates[0].ref, candidates[1].ref, candidates[2].ref}
	want := []string{refs[2], refs[0], refs[1]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected ordering: got=%v want=%v", got, want)
	}
}

func TestBuildSortedCandidatesStableOrderForSameLatency(t *testing.T) {
	refs := []string{
		"docker.io/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"ghcr.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"ccr.ccs.tencentyun.com/yyhuni/lunafox-agent@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	probes := map[string]imageProbeResult{
		refs[0]: {
			success: true,
			latency: 50 * time.Millisecond,
		},
		refs[1]: {
			success: true,
			latency: 50 * time.Millisecond,
		},
		refs[2]: {
			success: false,
		},
	}

	candidates := buildSortedCandidates(refs, probes)
	got := []string{candidates[0].ref, candidates[1].ref, candidates[2].ref}
	want := []string{refs[0], refs[1], refs[2]}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected stable ordering: got=%v want=%v", got, want)
	}
}

func TestPullFirstAvailableFallsBackWhenPrimaryFails(t *testing.T) {
	primary := "ccr.ccs.tencentyun.com/yyhuni/lunafox-agent@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	secondary := "docker.io/yyhuni/lunafox-agent@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	runner := &fakePullRunner{
		failures: map[string]bool{
			primary: true,
		},
	}
	installer := &Installer{
		runner:    runner,
		printer:   ui.NewPrinter(io.Discard, io.Discard),
		toolchain: docker.Toolchain{DockerBin: "docker"},
	}

	probes := map[string]imageProbeResult{
		primary: {
			success: false,
			errMsg:  "probe failed",
		},
		secondary: {
			success: true,
			latency: 40 * time.Millisecond,
		},
	}

	got, err := pullFirstAvailable(context.Background(), installer, "Agent", []string{primary, secondary}, probes)
	if err != nil {
		t.Fatalf("pullFirstAvailable failed: %v", err)
	}
	if got != secondary {
		t.Fatalf("expected fallback ref, got %s", got)
	}

	wantPulled := []string{secondary}
	if !reflect.DeepEqual(runner.pulled, wantPulled) {
		t.Fatalf("unexpected pull order: got=%v want=%v", runner.pulled, wantPulled)
	}
}

type fakePullRunner struct {
	failures map[string]bool
	pulled   []string
}

func (runner *fakePullRunner) LookPath(_ string) (string, error) {
	return "docker", nil
}

func (runner *fakePullRunner) Run(_ context.Context, command execx.Command) (execx.Result, error) {
	if command.Name != "docker" || len(command.Args) < 2 || command.Args[0] != "pull" {
		return execx.Result{}, nil
	}
	ref := command.Args[1]
	runner.pulled = append(runner.pulled, ref)
	if runner.failures[ref] {
		result := execx.Result{Stderr: "pull failed", ExitCode: 1}
		return result, &execx.ExecError{
			Command: command,
			Result:  result,
			Cause:   errors.New("exit status 1"),
		}
	}
	return execx.Result{Stdout: "ok", ExitCode: 0}, nil
}
