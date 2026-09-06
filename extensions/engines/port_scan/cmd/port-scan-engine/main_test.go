package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func TestRunPortScanBuildsTargetBaselineForEmptySubdomains(t *testing.T) {
	workspace := t.TempDir()
	hosts := writeReadOnlyInput(t, "subdomains.txt", nil)
	marker := filepath.Join(t.TempDir(), "naabu.started")
	candidates := filepath.Join(t.TempDir(), "naabu.candidates")
	binDir := t.TempDir()
	writeExecutable(t, filepath.Join(binDir, "naabu"), `#!/bin/sh
set -eu
list=""; out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -list) list="$2"; shift 2 ;;
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
touch "$NAABU_STARTED_MARKER"
cp "$list" "$NAABU_CANDIDATE_CAPTURE"
printf '%s\n' '{"host":"example.com","ip":"192.0.2.10","port":443}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NAABU_STARTED_MARKER", marker)
	t.Setenv("NAABU_CANDIDATE_CAPTURE", candidates)
	results := &captureHostPorts{}
	execution := portScanExecution(workspace, hosts, results)

	if err := runPortScan(context.Background(), execution); err != nil {
		t.Fatalf("runPortScan() error = %v", err)
	}
	if results.calls != 1 {
		t.Fatalf("empty Subdomains submitted %d result batches, want 1 baseline result", results.calls)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("naabu did not start for zero-byte Subdomains: %v", err)
	}
	if got, err := os.ReadFile(candidates); err != nil || string(got) != "example.com\n" {
		t.Fatalf("Naabu candidate file = %q, err=%v; want Target baseline only", got, err)
	}
}

func TestRunPortScanRejectsMissingContextBeforeEmptyNoOp(t *testing.T) {
	workspace := t.TempDir()
	hosts := filepath.Join(workspace, "subdomains.txt")
	if err := os.WriteFile(hosts, nil, 0o444); err != nil {
		t.Fatal(err)
	}

	err := runPortScan(nil, portScanExecution(workspace, hosts, &captureHostPorts{}))
	if err == nil || !strings.Contains(err.Error(), "execution context is required") {
		t.Fatalf("runPortScan() error = %v, want context validation", err)
	}
}

func TestRunPortScanRejectsCustomPortsConfigBeforeEmptyNoOp(t *testing.T) {
	workspace := t.TempDir()
	hosts := filepath.Join(workspace, "subdomains.txt")
	if err := os.WriteFile(hosts, nil, 0o444); err != nil {
		t.Fatal(err)
	}
	execution := portScanExecution(workspace, hosts, &captureHostPorts{})
	execution.Config.NaabuActive.Ports = ""

	err := runPortScan(context.Background(), execution)
	if err == nil || !strings.Contains(err.Error(), "ports is required") {
		t.Fatalf("runPortScan() error = %v, want custom ports validation", err)
	}
}

func TestRunPortScanDoesNotTreatNonZeroWhitespaceSubdomainsAsEmpty(t *testing.T) {
	workspace := t.TempDir()
	hosts := writeReadOnlyInput(t, "subdomains.txt", []byte(" \n"))

	err := runPortScan(context.Background(), portScanExecution(workspace, hosts, &captureHostPorts{}))
	if err == nil || !strings.Contains(err.Error(), "invalid candidate line") {
		t.Fatalf("runPortScan() error = %v, want malformed non-zero input failure", err)
	}
}

func TestRunPortScanRejectsMissingSubdomains(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing-subdomains.txt")

	err := runPortScan(context.Background(), portScanExecution(t.TempDir(), missing, &captureHostPorts{}))
	if err == nil || !strings.Contains(err.Error(), "open subdomains facts") || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runPortScan() error = %v, want missing Subdomains open failure", err)
	}
}

func TestRunPortScanRejectsNonRegularSubdomains(t *testing.T) {
	directory := t.TempDir()

	err := runPortScan(context.Background(), portScanExecution(t.TempDir(), directory, &captureHostPorts{}))
	if err == nil || !strings.Contains(err.Error(), "subdomains facts must be a regular file") {
		t.Fatalf("runPortScan() error = %v, want non-regular Subdomains failure", err)
	}
}

func TestRunPortScanRunsImageOwnedNaabuAndSubmitsTypedResults(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	subdomains := []byte("api.example.com\n")
	hosts := writeReadOnlyInput(t, "subdomains.txt", subdomains)
	writeExecutable(t, filepath.Join(binDir, "naabu"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"host":"api.example.com","ip":"192.0.2.10","port":443}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	results := &captureHostPorts{}
	execution := portScanExecution(workspace, hosts, results)

	if err := runPortScan(context.Background(), execution); err != nil {
		t.Fatalf("runPortScan() error = %v", err)
	}
	if results.calls != 1 || len(results.items) != 1 || results.items[0].Port != 443 {
		t.Fatalf("typed HostPorts submission = calls:%d items:%+v", results.calls, results.items)
	}
	if _, err := os.Stat(filepath.Join(workspace, "naabu_active.jsonl")); err != nil {
		t.Fatalf("naabu did not write its output in the writable workspace: %v", err)
	}
	assertReadOnlyInputUnchanged(t, hosts, subdomains)
}

func TestRunPortScanPropagatesTypedResultAcknowledgementFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	hosts := writeReadOnlyInput(t, "subdomains.txt", []byte("api.example.com\n"))
	writeExecutable(t, filepath.Join(binDir, "naabu"), `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"host":"api.example.com","ip":"192.0.2.10","port":443}' > "$out"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ackErr := errors.New("result acknowledgement unavailable")
	results := &captureHostPorts{err: ackErr}

	err := runPortScan(context.Background(), portScanExecution(workspace, hosts, results))
	if !errors.Is(err, ackErr) {
		t.Fatalf("runPortScan() error = %v, want typed result acknowledgement failure", err)
	}
	if results.calls != 1 || len(results.items) != 1 {
		t.Fatalf("typed HostPorts submission = calls:%d items:%+v", results.calls, results.items)
	}
}

func TestRunPortScanPropagatesImageOwnedNaabuFailure(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "naabu.started")
	hosts := filepath.Join(workspace, "subdomains.txt")
	if err := os.WriteFile(hosts, []byte("api.example.com\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(binDir, "naabu"), "#!/bin/sh\n: > \"$NAABU_STARTED_MARKER\"\nexit 7\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NAABU_STARTED_MARKER", marker)
	execution := portScanExecution(workspace, hosts, &captureHostPorts{})

	err := runPortScan(context.Background(), execution)
	if err == nil || !strings.Contains(err.Error(), "exit code 7") {
		t.Fatalf("runPortScan() error = %v, want naabu failure", err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatalf("naabu non-zero fixture did not start: %v", statErr)
	}
}

func TestRunPortScanPropagatesCancellationToNaabu(t *testing.T) {
	workspace := t.TempDir()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "naabu.started")
	hosts := filepath.Join(workspace, "subdomains.txt")
	if err := os.WriteFile(hosts, []byte("api.example.com\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(binDir, "naabu"), "#!/bin/sh\nset -eu\n: > \"$NAABU_STARTED_MARKER\"\nexec sleep 30\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("NAABU_STARTED_MARKER", marker)
	execution := portScanExecution(workspace, hosts, &captureHostPorts{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- runPortScan(ctx, execution) }()

	waitForToolStart(t, marker)
	cancel()
	err := waitForRunResult(t, result)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runPortScan() error = %v, want cancellation", err)
	}
}

func TestPortScanContainerHandlerDoesNotUseLegacyRuntimeCapabilities(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"contracts/engineapi/runtimekit",
		"StageExecutionCapabilities",
		"SubmitHostPortResults",
		"GetExecutionInput",
		"Materialize",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("container handler retains legacy capability %q", forbidden)
		}
	}
}

func portScanExecution(workspace, hosts string, results *captureHostPorts) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{Subdomains: testInputPath(hosts)},
		Workspace: workspace,
		Config: enginecontract.Config{
			NaabuActive: enginecontract.NaabuActiveConfig{
				Enabled: true, Timeout: 60, Threads: 1, PortMode: "custom", Ports: "443", TopPorts: "100", Rate: 1,
			},
			NaabuPassive: enginecontract.NaabuPassiveConfig{Enabled: false, Timeout: 60},
		},
		Progress: captureProgress{},
		Results:  enginecontract.Results{HostPorts: results},
	}
}

type captureProgress struct{}

func (captureProgress) Report(context.Context, string) error { return nil }

type captureHostPorts struct {
	calls int
	items []enginecontract.HostPort
	err   error
}

func (capture *captureHostPorts) Submit(_ context.Context, items <-chan enginecontract.HostPort) error {
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
		t.Fatalf("read-only Subdomains changed: got %q, want %q", got, want)
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
