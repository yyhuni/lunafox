package websitediscoveryruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestInitializeWebsiteDiscoveryBuildsBaselineForEmptyHostPorts(t *testing.T) {
	hostPortsPath := filepath.Join(t.TempDir(), "host-ports.jsonl")
	if err := os.WriteFile(hostPortsPath, nil, 0o600); err != nil {
		t.Fatalf("write empty hostPorts: %v", err)
	}
	executor := &recordingHTTPXExecutor{}
	run, err := initializeWebsiteDiscovery(
		context.Background(),
		websitediscoverycontract.Config{HTTPX: websitediscoverycontract.HTTPXConfig{Enabled: true, Timeout: 60}},
		websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeDomain, Value: "example.com"},
		hostPortsPath,
		t.TempDir(),
		discardingProgress{},
		executor,
	)
	if err != nil {
		t.Fatalf("initializeWebsiteDiscovery() error = %v", err)
	}
	if run.urlCount != 2 {
		t.Fatalf("url count = %d, want 2", run.urlCount)
	}
	if _, err := New().runHTTPXStage(context.Background(), run); err != nil {
		t.Fatalf("runHTTPXStage() error = %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("httpx calls = %d, want 1", executor.calls)
	}
}

type recordingHTTPXExecutor struct {
	calls int
}

func (executor *recordingHTTPXExecutor) run(context.Context, httpxCommand) error {
	executor.calls++
	return nil
}

type discardingProgress struct{}

func (discardingProgress) Report(context.Context, string) error { return nil }
