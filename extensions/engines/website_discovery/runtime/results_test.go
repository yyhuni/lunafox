package websitediscoveryruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

var _ func(
	*Runtime,
	context.Context,
	ProgressReporter,
	websitediscoverycontract.WebsiteSubmitter,
	*resultArtifacts,
) error = (*Runtime).ReportResults

func TestReportResultsDeduplicatesExactURLsAndKeepsLatestValidObservation(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "httpx.jsonl")
	content := `{"input":"https://Api.Example.COM:443/login","url":"https://Api.Example.COM:443/login","host":"api.example.com","title":"first","status_code":200}` + "\n" +
		`{"input":"https://Api.Example.COM:443/login","url":"https://Api.Example.COM:443/login","host":"api.example.com","title":"later","status_code":404}` + "\n" +
		`{"input":"https://api.example.com/login","url":"https://api.example.com/login","host":"api.example.com","title":"distinct","status_code":201}` + "\n" +
		`{"input":"https://www.example.com/","url":"https://www.example.com/","host":"www.example.com","title":"other","status_code":200}` + "\n"
	if err := os.WriteFile(artifact, []byte(content), 0o600); err != nil {
		t.Fatalf("write final website artifact: %v", err)
	}

	port := &captureWebsiteResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if len(port.items) != 3 {
		t.Fatalf("submitted websites = %#v, want 3 items", port.items)
	}
	first := findWebsite(port.items, "https://Api.Example.COM:443/login")
	if first == nil || first.Title != "later" || first.StatusCode == nil || *first.StatusCode != 404 {
		t.Fatalf("latest website observation = %#v", first)
	}
	if findWebsite(port.items, "https://api.example.com/login") == nil {
		t.Fatalf("raw URL spelling was collapsed: %#v", port.items)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "sourceRecords=4") || !strings.Contains(progress.messages[0], "parsedItems=4") || !strings.Contains(progress.messages[0], "submittedItems=3") || !strings.Contains(progress.messages[0], "skippedDuplicate=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsReportsOversizedAndStagesServerValidationCandidates(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "httpx.jsonl")
	content := strings.Repeat("x", 4*1024*1024+1) + "\n" +
		`{"input":"https://api.example.com","host":"bad host","status_code":999,"content_length":-1}` + "\n"
	if err := os.WriteFile(artifact, []byte(content), 0o600); err != nil {
		t.Fatalf("write final website artifact: %v", err)
	}

	port := &captureWebsiteResults{}
	progress := &captureResultProgress{}
	err := New().ReportResults(context.Background(), progress, port, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
	if len(port.items) != 1 || port.items[0].Host != "bad host" || port.items[0].StatusCode == nil || *port.items[0].StatusCode != 999 || port.items[0].ContentLength == nil || *port.items[0].ContentLength != -1 {
		t.Fatalf("submitted websites were changed or dropped: %#v", port.items)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "sourceRecords=2") || !strings.Contains(progress.messages[0], "parsedItems=1") || !strings.Contains(progress.messages[0], "submittedItems=1") || !strings.Contains(progress.messages[0], "skippedOversized=1") {
		t.Fatalf("result progress = %#v", progress.messages)
	}
}

func TestReportResultsRemovesStagingAfterImmediateSubmissionFailure(t *testing.T) {
	workspace := t.TempDir()
	artifact := filepath.Join(workspace, "httpx.jsonl")
	if err := os.WriteFile(artifact, []byte(`{"input":"https://api.example.com","url":"https://api.example.com","host":"api.example.com","status_code":200}`+"\n"), 0o600); err != nil {
		t.Fatalf("write website artifact: %v", err)
	}
	ackErr := errors.New("result acknowledgement failed")
	err := New().ReportResults(context.Background(), &captureResultProgress{}, rejectingWebsiteResults{err: ackErr}, &resultArtifacts{
		resultArtifactPath: artifact,
		workspaceDir:       workspace,
	})
	if !errors.Is(err, ackErr) {
		t.Fatalf("ReportResults() error = %v, want %v", err, ackErr)
	}
	assertNoResultDedupTempDirs(t, workspace)
}

func TestReportResultsSkipsWhenNoFinalArtifactExists(t *testing.T) {
	err := New().ReportResults(context.Background(), nil, nil, &resultArtifacts{workspaceDir: t.TempDir()})
	if err != nil {
		t.Fatalf("ReportResults() error = %v", err)
	}
}

type captureWebsiteResults struct {
	items []websitediscoverycontract.Website
}

func (capture *captureWebsiteResults) Submit(_ context.Context, items <-chan websitediscoverycontract.Website) error {
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return nil
}

type rejectingWebsiteResults struct {
	err error
}

func (rejecting rejectingWebsiteResults) Submit(context.Context, <-chan websitediscoverycontract.Website) error {
	return rejecting.err
}

type captureResultProgress struct {
	messages []string
}

func (capture *captureResultProgress) Report(_ context.Context, message string) error {
	capture.messages = append(capture.messages, message)
	return nil
}

func findWebsite(items []websitediscoverycontract.Website, url string) *websitediscoverycontract.Website {
	for index := range items {
		if items[index].URL == url {
			return &items[index]
		}
	}
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
