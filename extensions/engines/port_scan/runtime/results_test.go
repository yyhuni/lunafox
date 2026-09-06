package portscanruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

var _ func(
	*Runtime,
	context.Context,
	ProgressReporter,
	portscancontract.HostPortSubmitter,
	*resultArtifacts,
) error = (*Runtime).ReportResults

func TestReportResultsDeduplicatesCanonicalHostPortsAcrossArtifacts(t *testing.T) {
	workspace := t.TempDir()
	active := filepath.Join(workspace, "active.jsonl")
	passive := filepath.Join(workspace, "passive.jsonl")
	if err := os.WriteFile(active, []byte(`{"host":" Api.Example.COM. ","ip":"192.0.2.10","port":443}`+"\n"+`{"host":"db.example.com","ip":"192.0.2.11","port":5432}`+"\n"), 0o600); err != nil {
		t.Fatalf("write active artifact: %v", err)
	}
	if err := os.WriteFile(passive, []byte(`{"host":"api.example.com","ip":"192.0.2.10","port":443}`+"\n"), 0o600); err != nil {
		t.Fatalf("write passive artifact: %v", err)
	}

	port := &captureHostPortResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPaths: []string{active, passive},
		workspaceDir:        workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if len(port.items) != 2 {
		t.Fatalf("submitted host ports = %#v, want 2 items", port.items)
	}
	got := make(map[string]portscancontract.HostPort, len(port.items))
	for _, item := range port.items {
		got[canonicalHostPortKey(item)] = item
	}
	for _, want := range []portscancontract.HostPort{
		{Host: "api.example.com", IP: "192.0.2.10", Port: 443},
		{Host: "db.example.com", IP: "192.0.2.11", Port: 5432},
	} {
		if got[canonicalHostPortKey(want)] != want {
			t.Fatalf("canonical host port was changed or lost: want %#v got %#v", want, got)
		}
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "sourceRecords=3") || !strings.Contains(progress.messages[0], "parsedItems=3") || !strings.Contains(progress.messages[0], "submittedItems=2") || !strings.Contains(progress.messages[0], "skippedDuplicate=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsSkipsInvalidRowsAndSubmitsLaterValidRows(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "active.jsonl")
	content := `{"host":"api.example.com","ip":"192.0.2.10","port":443}` + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.11","port":0}` + "\n" +
		`{"host":"","ip":"192.0.2.12","port":8443}` + "\n"
	if err := os.WriteFile(artifact, []byte(content), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	port := &encodingHostPortResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPaths: []string{artifact},
		workspaceDir:        workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if want := []portscancontract.HostPort{
		{Host: "api.example.com", IP: "192.0.2.10", Port: 443},
		{Host: "192.0.2.12", IP: "192.0.2.12", Port: 8443},
	}; !sameHostPortItems(port.items, want) {
		t.Fatalf("submitted host ports = %#v, want %#v", port.items, want)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "parsedItems=2") || !strings.Contains(progress.messages[0], "submittedItems=2") || !strings.Contains(progress.messages[0], "skippedInvalid=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestCanonicalHostPortKeyIncludesEachIdentityField(t *testing.T) {
	base := portscancontract.HostPort{Host: "api.example.com", IP: "192.0.2.10", Port: 443}
	keys := []string{
		canonicalHostPortKey(base),
		canonicalHostPortKey(portscancontract.HostPort{Host: "admin.example.com", IP: "192.0.2.10", Port: 443}),
		canonicalHostPortKey(portscancontract.HostPort{Host: "api.example.com", IP: "192.0.2.11", Port: 443}),
		canonicalHostPortKey(portscancontract.HostPort{Host: "api.example.com", IP: "192.0.2.10", Port: 8443}),
	}
	for index := 1; index < len(keys); index++ {
		if keys[index] == keys[0] {
			t.Fatalf("identity field %d did not affect canonical host-port key", index)
		}
	}
}

func TestReportResultsReportsOversizedHostPortRecords(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "active.jsonl")
	content := strings.Repeat("x", 4*1024*1024+1) + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.10","port":443}` + "\n"
	if err := os.WriteFile(artifact, []byte(content), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	port := &captureHostPortResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPaths: []string{artifact},
		workspaceDir:        workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if len(port.items) != 1 || port.items[0].Host != "api.example.com" {
		t.Fatalf("submitted host ports = %#v", port.items)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "skippedOversized=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
}

func TestReportResultsCleansStagingAfterArtifactReadFailure(t *testing.T) {
	workspace := t.TempDir()
	err := New().ReportResults(context.Background(), &captureResultProgress{}, &captureHostPortResults{}, &resultArtifacts{
		resultArtifactPaths: []string{filepath.Join(workspace, "missing.jsonl")},
		workspaceDir:        workspace,
	})
	if err == nil {
		t.Fatal("ReportResults() succeeded for a missing artifact")
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsRemovesStagingAfterImmediateSubmissionFailure(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "active.jsonl")
	if err := os.WriteFile(artifact, []byte(`{"host":"api.example.com","ip":"192.0.2.10","port":443}`+"\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	ackErr := errors.New("result acknowledgement failed")
	err := New().ReportResults(context.Background(), &captureResultProgress{}, rejectingHostPortResults{err: ackErr}, &resultArtifacts{
		resultArtifactPaths: []string{artifact},
		workspaceDir:        workspace,
	})
	if !errors.Is(err, ackErr) {
		t.Fatalf("ReportResults() error = %v, want %v", err, ackErr)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

type captureHostPortResults struct {
	items []portscancontract.HostPort
}

func (capture *captureHostPortResults) Submit(_ context.Context, items <-chan portscancontract.HostPort) error {
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return nil
}

// encodingHostPortResults mirrors the generated result adapter's per-item
// encoding guard, so this test reaches the failure that previously terminated
// result submission after Naabu emitted port=0.
type encodingHostPortResults struct {
	items []portscancontract.HostPort
}

func (capture *encodingHostPortResults) Submit(_ context.Context, items <-chan portscancontract.HostPort) error {
	for item := range items {
		if _, err := portscancontract.EncodeHostPort(item); err != nil {
			return err
		}
		capture.items = append(capture.items, item)
	}
	return nil
}

func sameHostPortItems(got, want []portscancontract.HostPort) bool {
	if len(got) != len(want) {
		return false
	}
	gotByKey := make(map[string]portscancontract.HostPort, len(got))
	for _, item := range got {
		gotByKey[canonicalHostPortKey(item)] = item
	}
	for _, item := range want {
		if gotByKey[canonicalHostPortKey(item)] != item {
			return false
		}
	}
	return true
}

type rejectingHostPortResults struct {
	err error
}

func (rejecting rejectingHostPortResults) Submit(context.Context, <-chan portscancontract.HostPort) error {
	return rejecting.err
}

type captureResultProgress struct {
	messages []string
}

func (capture *captureResultProgress) Report(_ context.Context, message string) error {
	capture.messages = append(capture.messages, message)
	return nil
}

func assertNoResultDedupTempDirs(t *testing.T, workspace string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(workspace, ".result-dedup-*"))
	if err != nil {
		t.Fatalf("glob result dedup temp directories: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("result dedup temp directories remain: %v", paths)
	}
}
