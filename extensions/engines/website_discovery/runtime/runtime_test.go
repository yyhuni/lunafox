package websitediscoveryruntime

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	args  []string
}

func (executor *recordingHTTPXExecutor) run(_ context.Context, command httpxCommand) error {
	executor.calls++
	executor.args = append([]string(nil), command.args...)
	return nil
}

type discardingProgress struct{}

func (discardingProgress) Report(context.Context, string) error { return nil }

func TestHTTPXStageMasksHeadersOnlyInProgress(t *testing.T) {
	headers := []string{"Cookie: secret-cookie", "Authorization: Bearer secret-token"}
	progress := &recordingProgress{}
	executor := &recordingHTTPXExecutor{}
	run := &websiteDiscoveryRun{
		config:      websitediscoverycontract.Config{HTTPX: websitediscoverycontract.HTTPXConfig{Enabled: true, Timeout: 60, Headers: headers}},
		urlListPath: "urls.txt", urlCount: 1, workDir: t.TempDir(), progress: progress, executor: executor,
	}
	if _, err := New().runHTTPXStage(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	want := []string{"-H", headers[0], "-H", headers[1]}
	if !reflect.DeepEqual(executor.args[len(executor.args)-len(want):], want) {
		t.Fatalf("executor did not receive actual headers: %#v", executor.args)
	}
	text := strings.Join(progress.messages, "\n")
	if strings.Contains(text, "secret-cookie") || strings.Contains(text, "secret-token") || strings.Count(text, "-H [REDACTED]") != 2 {
		t.Fatalf("progress failed to mask headers: %s", text)
	}
}

type recordingProgress struct{ messages []string }

func (progress *recordingProgress) Report(_ context.Context, message string) error {
	progress.messages = append(progress.messages, message)
	return nil
}
