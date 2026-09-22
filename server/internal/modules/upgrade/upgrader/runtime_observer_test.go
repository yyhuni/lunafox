package upgrader

import (
	"context"
	"strings"
	"testing"
)

type runtimeObserverRunner struct {
	config             string
	healthy            bool
	bootstrapContainer string
	engineInventory    string
	commands           []string
}

func (runner *runtimeObserverRunner) Run(_ context.Context, _ string, args []string, _ string) (RunResult, error) {
	joined := strings.Join(args, " ")
	runner.commands = append(runner.commands, joined)
	switch {
	case strings.Contains(joined, "ps --format json nginx"):
		if runner.healthy {
			return RunResult{Stdout: `[{"Service":"nginx","State":"running","Health":"healthy"}]`}, nil
		}
		return RunResult{Stdout: `[{"Service":"nginx","State":"exited","Health":"unhealthy"}]`}, nil
	case strings.Contains(joined, "ps -a -q bootstrap"):
		containerID := runner.bootstrapContainer
		if containerID == "" {
			containerID = "fedcba012345"
		}
		return RunResult{Stdout: containerID + "\n"}, nil
	case strings.Contains(joined, "ps -q "):
		service := strings.TrimSpace(joined[strings.LastIndex(joined, "ps -q ")+len("ps -q "):])
		ids := map[string]string{
			"server":   "abcdef012345",
			"frontend": "bcdefa012345",
			"nginx":    "cdefab012345",
			"agent":    "defabc012345",
		}
		return RunResult{Stdout: ids[service] + "\n"}, nil
	case strings.Contains(joined, "inspect --format"):
		return RunResult{Stdout: `"ghcr.io/yyhuni/lunafox@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`}, nil
	case strings.Contains(joined, "exec ") && strings.HasSuffix(joined, "/usr/local/bin/server engine-inventory"):
		inventory := runner.engineInventory
		if inventory == "" {
			inventory = `{"schemaVersion":1,"components":[{"id":"engine.port-scan.package","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"id":"engine.port-scan.runtime","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`
		}
		return RunResult{Stdout: inventory}, nil
	case strings.Contains(joined, "exec ") && strings.HasSuffix(joined, " nginx -T"):
		return RunResult{Stdout: runner.config}, nil
	default:
		return RunResult{}, nil
	}
}

const validDynamicNginxDump = `
# resolver frontend:3000 without resolve must not count
resolver 127.0.0.11 valid=10s ipv6=off;
upstream frontend {
  zone frontend 64k;
  server frontend:3000 resolve;
}
`

func newRuntimeObserverForTest(t *testing.T, runner CommandRunner) *DockerRuntimeObserver {
	t.Helper()
	executor, err := NewPublicComposeExecutor(runner, "ghcr.io")
	if err != nil {
		t.Fatal(err)
	}
	observer := NewDockerRuntimeObserver(executor)
	if observer == nil {
		t.Fatal("runtime observer was not created")
	}
	observer.SetDeploymentRoot("/deployment")
	return observer
}

func TestDockerRuntimeObserverRequiresHealthyNginxAndDynamicConfig(t *testing.T) {
	runner := &runtimeObserverRunner{config: validDynamicNginxDump, healthy: true}
	observation, err := newRuntimeObserverForTest(t, runner).ObserveRuntime(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !observation.NginxHealthy || observation.NginxConfigDigest == "" {
		t.Fatalf("observation omitted nginx evidence: %#v", observation)
	}
	if err := observation.Validate(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runner.commands, "\n"), "exec cdefab012345 nginx -T") {
		t.Fatalf("observer did not inspect live nginx config: %#v", runner.commands)
	}
	trace := strings.Join(runner.commands, "\n")
	if !strings.Contains(trace, "ps -a -q bootstrap") {
		t.Fatalf("observer did not inspect the completed bootstrap container: %#v", runner.commands)
	}
	if !strings.Contains(trace, "exec abcdef012345 /usr/local/bin/server engine-inventory") {
		t.Fatalf("observer did not read the fixed Server Engine inventory: %#v", runner.commands)
	}
	for _, componentID := range []string{runtimeBootstrapComponent, "engine.port-scan.package", "engine.port-scan.runtime"} {
		if _, found := observation.Images[componentID]; !found {
			t.Fatalf("observation omitted %s: %#v", componentID, observation.Images)
		}
	}
}

func TestDockerRuntimeObserverRejectsUnhealthyNginx(t *testing.T) {
	runner := &runtimeObserverRunner{config: validDynamicNginxDump, healthy: false}
	if _, err := newRuntimeObserverForTest(t, runner).ObserveRuntime(context.Background()); err == nil || !strings.Contains(err.Error(), "nginx health") {
		t.Fatalf("unhealthy nginx error = %v", err)
	}
}

func TestDockerRuntimeObserverRejectsStaticFrontendUpstream(t *testing.T) {
	runner := &runtimeObserverRunner{
		config:  "resolver 127.0.0.11; upstream frontend { zone frontend 64k; server frontend:3000; }",
		healthy: true,
	}
	if _, err := newRuntimeObserverForTest(t, runner).ObserveRuntime(context.Background()); err == nil || !strings.Contains(err.Error(), "dynamically resolvable") {
		t.Fatalf("static frontend upstream error = %v", err)
	}
}

func TestNormalizeNginxDynamicFrontendConfigBindsCompleteLiveConfig(t *testing.T) {
	base, err := normalizeNginxDynamicFrontendConfig(validDynamicNginxDump)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := normalizeNginxDynamicFrontendConfig(validDynamicNginxDump + "\nclient_max_body_size 8m;\n")
	if err != nil {
		t.Fatal(err)
	}
	if base == changed {
		t.Fatalf("live nginx configuration digest did not change after a config edit: %q", base)
	}
	commentOnly, err := normalizeNginxDynamicFrontendConfig("# generated header\n" + validDynamicNginxDump)
	if err != nil {
		t.Fatal(err)
	}
	if base != commentOnly {
		t.Fatalf("comment-only nginx change altered normalized digest: base=%q commentOnly=%q", base, commentOnly)
	}
}

func TestDockerRuntimeObserverRejectsMissingBootstrapOrMalformedEngineInventory(t *testing.T) {
	tests := []struct {
		name   string
		runner *runtimeObserverRunner
		want   string
	}{
		{
			name:   "missing bootstrap",
			runner: &runtimeObserverRunner{config: validDynamicNginxDump, healthy: true, bootstrapContainer: "missing"},
			want:   "bootstrap container returned an invalid identity",
		},
		{
			name:   "malformed engine inventory",
			runner: &runtimeObserverRunner{config: validDynamicNginxDump, healthy: true, engineInventory: `{"schemaVersion":1,"components":[{"id":"engine.port-scan.runtime","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`},
			want:   "Engine inventory",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newRuntimeObserverForTest(t, test.runner).ObserveRuntime(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ObserveRuntime() error = %v, want %q", err, test.want)
			}
		})
	}
}
