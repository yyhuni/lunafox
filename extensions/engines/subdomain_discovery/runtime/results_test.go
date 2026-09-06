package subdomaindiscoveryruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	subdomaincontract "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

const testLogicalSubmissionBatchSize = 3

var _ func(
	*Runtime,
	context.Context,
	ProgressReporter,
	subdomaincontract.SubdomainSubmitter,
	*resultArtifacts,
) error = (*Runtime).ReportResults

func TestReportResultsDeduplicatesCanonicalSubdomainsWithinFinalArtifact(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "resolve.txt")
	if err := os.WriteFile(artifact, []byte("A.example.com.\nb.example.com\na.example.com\nC.example.com\nb.example.com\n"), 0o600); err != nil {
		t.Fatalf("write final artifact: %v", err)
	}

	port := &captureSubdomainResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if got, want := subdomainNames(port.items), []string{"a.example.com", "b.example.com", "c.example.com"}; !sameStrings(got, want) {
		t.Fatalf("submitted subdomains = %v, want %v", got, want)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "sourceRecords=5") || !strings.Contains(progress.messages[0], "parsedItems=5") || !strings.Contains(progress.messages[0], "submittedItems=3") || !strings.Contains(progress.messages[0], "skippedDuplicate=2") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsReportsOversizedSubdomainRecords(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "resolve.txt")
	content := strings.Repeat("a", 4*1024*1024+1) + "\napi.example.com\n"
	if err := os.WriteFile(artifact, []byte(content), 0o600); err != nil {
		t.Fatalf("write final artifact: %v", err)
	}

	port := &captureSubdomainResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if got, want := subdomainNames(port.items), []string{"api.example.com"}; !sameStrings(got, want) {
		t.Fatalf("submitted subdomains = %v, want %v", got, want)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "skippedOversized=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
}

func TestReportResultsRemovesStagingAfterSubmissionFailure(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "results.txt")
	if err := os.WriteFile(artifact, []byte("api.example.com\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	ackErr := errors.New("result acknowledgement failed")
	port := rejectingSubdomainResults{err: ackErr}

	err := New().ReportResults(context.Background(), &captureResultProgress{}, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if !errors.Is(err, ackErr) {
		t.Fatalf("ReportResults() error = %v, want %v", err, ackErr)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsReportsCleanupFailureAlongsideSubmissionFailure(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "results.txt")
	if err := os.WriteFile(artifact, []byte("api.example.com\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(workspace, 0o700) })
	ackErr := errors.New("result acknowledgement failed")
	port := &readOnlyRejectingSubdomainResults{
		workspace: workspace,
		err:       ackErr,
	}
	err := New().ReportResults(context.Background(), &captureResultProgress{}, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if restoreErr := os.Chmod(workspace, 0o700); restoreErr != nil {
		t.Fatalf("restore workspace permissions: %v", restoreErr)
	}
	if !errors.Is(err, ackErr) {
		t.Fatalf("ReportResults() error = %v, want %v", err, ackErr)
	}
	if port.chmodErr != nil {
		t.Skipf("cannot make workspace read-only: %v", port.chmodErr)
	}
	if !strings.Contains(err.Error(), "clean up subdomain result deduplication staging") {
		assertNoResultDedupTempDirs(t, workspace)
		t.Skip("execution environment permits temporary directory removal from a read-only parent")
	}
	paths, globErr := filepath.Glob(filepath.Join(workspace, ".result-dedup-*"))
	if globErr != nil {
		t.Fatalf("glob result dedup temp directories: %v", globErr)
	}
	if len(paths) == 0 {
		t.Fatal("expected a retained result deduplication temporary directory")
	}
	for _, path := range paths {
		if removeErr := os.RemoveAll(path); removeErr != nil {
			t.Fatalf("remove retained result dedup temporary directory %q: %v", path, removeErr)
		}
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsCancellationStopsStagedSubmissionAndCleansStaging(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "results.txt")
	if err := os.WriteFile(artifact, []byte("api.example.com\nwww.example.com\n"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	port := &cancelAfterFirstSubdomainResult{cancel: cancel}

	err := New().ReportResults(ctx, &captureResultProgress{}, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReportResults() error = %v, want cancellation", err)
	}
	if got, want := subdomainNames(port.items), []string{"api.example.com"}; !sameStrings(got, want) {
		t.Fatalf("submitted subdomains after cancellation = %v, want %v", got, want)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsDeduplicatesBeforeLogicalSubmissionBatchBoundary(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "results.txt")
	var content strings.Builder
	for index := 0; index <= testLogicalSubmissionBatchSize; index++ {
		fmt.Fprintf(&content, "node-%04d.example.com\n", index)
	}
	content.WriteString("node-0000.example.com\n")
	if err := os.WriteFile(artifact, []byte(content.String()), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	port := &logicalBatchSubdomainResults{}
	err := New().ReportResults(context.Background(), &captureResultProgress{}, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if len(port.batches) != 2 || len(port.batches[0]) != testLogicalSubmissionBatchSize || len(port.batches[1]) != 1 {
		t.Fatalf("logical submitted batches = %#v", port.batches)
	}
	submitted := make(map[string]int, testLogicalSubmissionBatchSize+1)
	for _, batch := range port.batches {
		for _, item := range batch {
			submitted[item.DNSName]++
		}
	}
	if len(submitted) != testLogicalSubmissionBatchSize+1 || submitted["node-0000.example.com"] != 1 {
		t.Fatalf("submitted canonical subdomains = %#v", submitted)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsSkipsWhenNoFinalArtifactExists(t *testing.T) {
	err := New().ReportResults(context.Background(), nil, nil, &resultArtifacts{workspaceDir: t.TempDir()})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
}

type captureSubdomainResults struct {
	items []subdomaincontract.Subdomain
}

func (capture *captureSubdomainResults) Submit(_ context.Context, items <-chan subdomaincontract.Subdomain) error {
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return nil
}

type rejectingSubdomainResults struct {
	err error
}

func (rejecting rejectingSubdomainResults) Submit(context.Context, <-chan subdomaincontract.Subdomain) error {
	return rejecting.err
}

type readOnlyRejectingSubdomainResults struct {
	workspace string
	err       error
	chmodErr  error
}

func (rejecting *readOnlyRejectingSubdomainResults) Submit(context.Context, <-chan subdomaincontract.Subdomain) error {
	rejecting.chmodErr = os.Chmod(rejecting.workspace, 0o500)
	return rejecting.err
}

type cancelAfterFirstSubdomainResult struct {
	cancel context.CancelFunc
	items  []subdomaincontract.Subdomain
}

func (capture *cancelAfterFirstSubdomainResult) Submit(_ context.Context, items <-chan subdomaincontract.Subdomain) error {
	for item := range items {
		capture.items = append(capture.items, item)
		capture.cancel()
	}
	return nil
}

type logicalBatchSubdomainResults struct {
	batches [][]subdomaincontract.Subdomain
}

func (capture *logicalBatchSubdomainResults) Submit(_ context.Context, items <-chan subdomaincontract.Subdomain) error {
	batch := make([]subdomaincontract.Subdomain, 0, testLogicalSubmissionBatchSize)
	for item := range items {
		batch = append(batch, item)
		if len(batch) == testLogicalSubmissionBatchSize {
			capture.batches = append(capture.batches, batch)
			batch = make([]subdomaincontract.Subdomain, 0, testLogicalSubmissionBatchSize)
		}
	}
	if len(batch) > 0 {
		capture.batches = append(capture.batches, batch)
	}
	return nil
}

type captureResultProgress struct {
	messages []string
	err      error
}

func (capture *captureResultProgress) Report(_ context.Context, message string) error {
	capture.messages = append(capture.messages, message)
	return capture.err
}

func subdomainNames(items []subdomaincontract.Subdomain) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.DNSName)
	}
	return names
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
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
