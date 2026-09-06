package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestRunWebsiteDiscoveryRunsHTTPXForCIDRBaseline(t *testing.T) {
	workspace := t.TempDir()
	urlList := []byte{}
	urls := writeReadOnlyInput(t, "host-ports.jsonl", urlList)
	marker := filepath.Join(t.TempDir(), "httpx.started")
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "httpx"), `#!/bin/sh
set -eu
: > "$HTTPX_STARTED_MARKER"
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"input":"https://192.0.2.0","url":"https://192.0.2.0","host":"192.0.2.0","status_code":200}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HTTPX_STARTED_MARKER", marker)
	results := &captureWebsites{}
	execution := websiteExecution(workspace, urls, results)

	if err := runWebsiteDiscovery(context.Background(), execution); err != nil {
		t.Fatalf("runWebsiteDiscovery() error = %v", err)
	}
	if results.calls != 1 || len(results.items) != 1 || results.items[0].URL != "https://192.0.2.0" {
		t.Fatalf("typed Websites submission = calls:%d items:%+v", results.calls, results.items)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("httpx did not start for CIDR baseline: %v", err)
	}
	assertReadOnlyInputUnchanged(t, urls, urlList)
}

func TestRunWebsiteDiscoveryRejectsMissingContext(t *testing.T) {
	workspace := t.TempDir()
	urls := filepath.Join(workspace, "host-ports.jsonl")
	if err := os.WriteFile(urls, nil, 0o444); err != nil {
		t.Fatal(err)
	}

	err := runWebsiteDiscovery(nil, websiteExecution(workspace, urls, &captureWebsites{}))
	if err == nil || !strings.Contains(err.Error(), "execution context is required") {
		t.Fatalf("runWebsiteDiscovery() error = %v, want context validation", err)
	}
}

func TestRunWebsiteDiscoveryRejectsMissingHostPorts(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-host-ports.jsonl")

	err := runWebsiteDiscovery(context.Background(), websiteExecution(t.TempDir(), missing, &captureWebsites{}))
	if err == nil || !strings.Contains(err.Error(), "open hostPorts facts") || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runWebsiteDiscovery() error = %v, want missing HostPorts open failure", err)
	}
}

func TestRunWebsiteDiscoveryRejectsNonRegularHostPorts(t *testing.T) {
	directory := t.TempDir()

	err := runWebsiteDiscovery(context.Background(), websiteExecution(t.TempDir(), directory, &captureWebsites{}))
	if err == nil || !strings.Contains(err.Error(), "hostPorts facts must be a regular file") {
		t.Fatalf("runWebsiteDiscovery() error = %v, want non-regular HostPorts failure", err)
	}
}

func TestRunWebsiteDiscoveryRunsImageOwnedHTTPXAndSubmitsTypedResults(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	urlList := []byte(`{"host":"api.example.com","ip":"192.0.2.10","port":443}
`)
	urls := writeReadOnlyInput(t, "host-ports.jsonl", urlList)
	writeExecutable(t, filepath.Join(binDir, "httpx"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"input":"https://api.example.com:443/login","url":"https://api.example.com:443/login","host":"api.example.com","title":"API","status_code":200}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	results := &captureWebsites{}
	execution := websiteExecution(workspace, urls, results)

	if err := runWebsiteDiscovery(context.Background(), execution); err != nil {
		t.Fatalf("runWebsiteDiscovery() error = %v", err)
	}
	if results.calls != 1 || len(results.items) != 1 || results.items[0].URL != "https://api.example.com:443/login" {
		t.Fatalf("typed Websites submission = calls:%d items:%+v", results.calls, results.items)
	}
	if _, err := os.Stat(filepath.Join(workspace, "httpx.jsonl")); err != nil {
		t.Fatalf("httpx did not write its output in the writable workspace: %v", err)
	}
	assertReadOnlyInputUnchanged(t, urls, urlList)
}

func TestRunWebsiteDiscoveryPropagatesTypedResultAcknowledgementFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	urls := writeReadOnlyInput(t, "host-ports.jsonl", []byte(`{"host":"api.example.com","ip":"192.0.2.10","port":443}
`))
	writeExecutable(t, filepath.Join(binDir, "httpx"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"input":"https://api.example.com","url":"https://api.example.com","host":"api.example.com","status_code":200}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ackErr := errors.New("result acknowledgement unavailable")
	results := &captureWebsites{err: ackErr}

	err := runWebsiteDiscovery(context.Background(), websiteExecution(workspace, urls, results))
	if !errors.Is(err, ackErr) {
		t.Fatalf("runWebsiteDiscovery() error = %v, want typed result acknowledgement failure", err)
	}
	if results.calls != 1 || len(results.items) != 1 {
		t.Fatalf("typed Websites submission = calls:%d items:%+v", results.calls, results.items)
	}
}

func TestRunWebsiteDiscoveryPropagatesImageOwnedHTTPXFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "httpx.started")
	urls := filepath.Join(workspace, "host-ports.jsonl")
	if err := os.WriteFile(urls, []byte(`{"host":"example.com","ip":"192.0.2.10","port":443}
`), 0o444); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(binDir, "httpx"), "#!/bin/sh\n: > \"$HTTPX_STARTED_MARKER\"\nexit 9\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HTTPX_STARTED_MARKER", marker)

	err := runWebsiteDiscovery(context.Background(), websiteExecution(workspace, urls, &captureWebsites{}))
	if err == nil || !strings.Contains(err.Error(), "exit code 9") {
		t.Fatalf("runWebsiteDiscovery() error = %v, want httpx failure", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("httpx non-zero fixture did not start: %v", statErr)
	}
}

func TestRunWebsiteDiscoveryPropagatesCancellationToHTTPX(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "httpx.started")
	urls := filepath.Join(workspace, "host-ports.jsonl")
	if err := os.WriteFile(urls, []byte(`{"host":"example.com","ip":"192.0.2.10","port":443}
`), 0o444); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(binDir, "httpx"), "#!/bin/sh\nset -eu\n: > \"$HTTPX_STARTED_MARKER\"\nexec sleep 30\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HTTPX_STARTED_MARKER", marker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- runWebsiteDiscovery(ctx, websiteExecution(workspace, urls, &captureWebsites{}))
	}()

	waitForToolStart(t, marker)
	cancel()
	err := waitForRunResult(t, result)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runWebsiteDiscovery() error = %v, want cancellation", err)
	}
}

func TestWebsiteDiscoveryContainerHandlerDoesNotUseLegacyRuntimeCapabilities(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"contracts/engineapi/runtimekit",
		"StageExecutionCapabilities",
		"SubmitWebsiteResults",
		"GetExecutionInput",
		"Materialize",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("container handler retains legacy capability %q", forbidden)
		}
	}
}

func TestWebsiteDiscoveryDoesNotInspectPredecessorStateOrEmptyInput(t *testing.T) {
	for _, sourceName := range []string{"main.go", filepath.Join("..", "..", "runtime", "runtime.go")} {
		source, err := os.ReadFile(sourceName)
		if err != nil {
			t.Fatalf("read %s: %v", sourceName, err)
		}
		for _, forbidden := range []string{
			"fileIsZeroBytes",
			"portScanSucceeded",
			"PriorWorkflowStagesSatisfied",
			"PriorEngineTasksSucceeded",
			"portScanEngineID",
			"requirePriorStagesReady",
		} {
			if strings.Contains(string(source), forbidden) {
				t.Fatalf("website discovery source %s must not inspect predecessor state or empty input via %q", sourceName, forbidden)
			}
		}
	}
}

func websiteExecution(workspace, urls string, results *captureWebsites) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/30"},
		Input:     enginecontract.Input{HostPorts: testInputPath(urls)},
		Workspace: workspace,
		Config: enginecontract.Config{HTTPX: enginecontract.HTTPXConfig{
			Enabled: true, Timeout: 60, Threads: 1, RateLimit: 1, RequestTimeout: 1, Retries: 0,
		}},
		Progress: captureProgress{},
		Results:  enginecontract.Results{Websites: results},
	}
}

type captureProgress struct{}

func (captureProgress) Report(context.Context, string) error { return nil }

type captureWebsites struct {
	calls int
	items []enginecontract.Website
	err   error
}

func (capture *captureWebsites) Submit(_ context.Context, items <-chan enginecontract.Website) error {
	capture.calls++
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return capture.err
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeReadOnlyInput(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o444); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertReadOnlyInputUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o222 != 0 {
		t.Fatalf("input %s mode = %o, want no write bits", path, info.Mode().Perm())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("read-only HostPorts changed: got %q, want %q", got, want)
	}
}

func waitForToolStart(t *testing.T, marker string) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(marker); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat tool start marker: %v", err)
		}
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for tool start marker %s", marker)
		case <-ticker.C:
		}
	}
}

func waitForRunResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for cancelled Engine handler")
		return nil
	}
}
