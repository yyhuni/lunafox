package screenshotruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/screenshot/contract"
	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

type progressTestReporter struct{ messages []string }

func (reporter *progressTestReporter) Report(_ context.Context, message string) error {
	reporter.messages = append(reporter.messages, message)
	return nil
}

type screenshotTestSubmitter struct{ items []contractresults.Screenshot }

func (submitter *screenshotTestSubmitter) Submit(_ context.Context, items <-chan contractresults.Screenshot) error {
	for item := range items {
		submitter.items = append(submitter.items, item)
	}
	return nil
}

type scriptedPassExecutor struct {
	passes [][]string
	paths  []string
	inputs []string
}

func (executor *scriptedPassExecutor) Run(_ context.Context, command httpxCommand, onRecord func([]byte) error) error {
	for index, arg := range command.args {
		if arg == "-list" && index+1 < len(command.args) {
			executor.paths = append(executor.paths, command.args[index+1])
			payload, err := os.ReadFile(command.args[index+1])
			if err != nil {
				return err
			}
			executor.inputs = append(executor.inputs, string(payload))
		}
	}
	pass := len(executor.paths) - 1
	if pass >= len(executor.passes) {
		return nil
	}
	for _, record := range executor.passes[pass] {
		if err := onRecord([]byte(record)); err != nil {
			return err
		}
	}
	return nil
}

type scriptedCWebP struct{ image []byte }

func (runner scriptedCWebP) Run(_ context.Context, args []string) error {
	return os.WriteFile(args[len(args)-1], runner.image, 0o600)
}

type failingCWebP struct{}

func (failingCWebP) Run(_ context.Context, _ []string) error { return os.ErrPermission }

func newScreenshotExecution(workspace, facts string, progress *progressTestReporter, submitter *screenshotTestSubmitter, retries int64) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:    enginecontract.Config{Capture: enginecontract.CaptureConfig{Enabled: true, PageTimeout: 15, Concurrency: 5, Retries: retries}},
		Workspace: workspace,
		Progress:  progress,
		Results:   enginecontract.Results{Screenshots: submitter},
	}
}

func screenshotRow(fields map[string]any) string {
	payload, _ := json.Marshal(fields)
	return string(payload)
}

func TestRuntimeUsesObservedURLAndRunsOneEnginePass(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, []byte("https://example.com/path\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(workspace, "capture.png")
	writeTestPNG(t, pngPath, 320, 200)
	executor := &scriptedPassExecutor{passes: [][]string{{
		screenshotRow(map[string]any{"input": "http://example.com", "url": "https://example.com/final", "status_code": 302, "screenshot_path": pngPath}),
	}}}
	progress := &progressTestReporter{}
	submitter := &screenshotTestSubmitter{}
	execution := newScreenshotExecution(workspace, facts, progress, submitter, 3)
	runtime := &Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(320, 200)}}
	if err := runtime.Execute(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	if len(executor.paths) != 1 {
		t.Fatalf("HTTPX launch count = %d, want one Engine pass", len(executor.paths))
	}
	if len(submitter.items) != 1 || submitter.items[0].URL != "https://example.com/final" {
		t.Fatalf("submitted observed URL = %#v", submitter.items)
	}
	if submitter.items[0].StatusCode == nil || *submitter.items[0].StatusCode != 302 {
		t.Fatalf("status code = %#v", submitter.items[0].StatusCode)
	}
	if !strings.Contains(progress.messages[len(progress.messages)-1], "attempts=1") {
		t.Fatalf("progress = %#v", progress.messages)
	}
}

func TestRuntimeAllowsMissingUnknownAndDuplicateInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	rows := []string{
		screenshotRow(map[string]any{"url": "https://example.com/one", "failed": true}),
		screenshotRow(map[string]any{"input": "unknown", "url": "https://example.com/two", "failed": true}),
		screenshotRow(map[string]any{"input": "unknown", "url": "https://example.com/two", "failed": true}),
	}
	executor := &scriptedPassExecutor{passes: [][]string{rows}}
	progress := &progressTestReporter{}
	submitter := &screenshotTestSubmitter{}
	if err := (&Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), newScreenshotExecution(workspace, facts, progress, submitter, 0)); err != nil {
		t.Fatal(err)
	}
	if len(submitter.items) != 0 || len(executor.paths) != 1 {
		t.Fatalf("failed observations should be accepted without submission: items=%#v paths=%d", submitter.items, len(executor.paths))
	}
}

func TestRuntimeDoesNotSynthesizeMissingRowsOrReplayFailures(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, []byte("https://example.com/path\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &scriptedPassExecutor{passes: [][]string{{screenshotRow(map[string]any{"url": "http://example.com", "failed": true})}, {screenshotRow(map[string]any{"url": "https://example.com/path", "failed": true})}}}
	progress := &progressTestReporter{}
	submitter := &screenshotTestSubmitter{}
	if err := (&Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), newScreenshotExecution(workspace, facts, progress, submitter, 3)); err != nil {
		t.Fatal(err)
	}
	if len(executor.paths) != 1 || len(submitter.items) != 0 {
		t.Fatalf("missing rows triggered replay or synthetic result: paths=%d items=%#v", len(executor.paths), submitter.items)
	}
	if !strings.Contains(progress.messages[len(progress.messages)-1], "processed=1") {
		t.Fatalf("progress = %#v", progress.messages)
	}
}

func TestRuntimeSkipsMalformedRowsWhenValidObservedRowExists(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	progress := &progressTestReporter{}
	submitter := &screenshotTestSubmitter{}
	executor := &scriptedPassExecutor{passes: [][]string{{"not-json", screenshotRow(map[string]any{"url": "http://example.com", "failed": true})}}}
	if err := (&Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), newScreenshotExecution(workspace, facts, progress, submitter, 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(progress.messages[len(progress.messages)-1], "malformedRows=1") {
		t.Fatalf("progress = %#v", progress.messages)
	}
}

func TestRuntimeRejectsAllMalformedOutput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &scriptedPassExecutor{passes: [][]string{{"not-json"}}}
	execution := newScreenshotExecution(workspace, facts, &progressTestReporter{}, &screenshotTestSubmitter{}, 0)
	if err := (&Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), execution); err == nil || !strings.Contains(err.Error(), "no valid observed URL row") {
		t.Fatalf("all-malformed output error = %v", err)
	}
}

func TestRuntimeTreatsConversionFailureAsFinalLocalSkip(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(workspace, "capture.png")
	writeTestPNG(t, pngPath, 320, 200)
	progress := &progressTestReporter{}
	executor := &scriptedPassExecutor{passes: [][]string{{screenshotRow(map[string]any{"url": "http://example.com", "screenshot_path": pngPath})}}}
	if err := (&Runtime{passExecutor: executor, cwebpRunner: failingCWebP{}}).Execute(context.Background(), newScreenshotExecution(workspace, facts, progress, &screenshotTestSubmitter{}, 3)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(progress.messages[len(progress.messages)-1], "skippedConversion=1") {
		t.Fatalf("progress = %#v", progress.messages)
	}
}

func TestRuntimeRejectsFailedRowPathOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "capture.png")
	if err := os.WriteFile(outside, []byte("not a screenshot"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &scriptedPassExecutor{passes: [][]string{{screenshotRow(map[string]any{"url": "http://example.com", "failed": true, "screenshot_path": outside})}}}
	if err := (&Runtime{passExecutor: executor, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), newScreenshotExecution(workspace, facts, &progressTestReporter{}, &screenshotTestSubmitter{}, 0)); err == nil {
		t.Fatal("failed row path escape must fail the task")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside path was unexpectedly removed: %v", err)
	}
}

func TestRuntimeSkipsDisabledCaptureSectionWithoutStartingHTTPX(t *testing.T) {
	progress := &progressTestReporter{}
	execution := &enginecontract.Execution{
		Target:   enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Config:   enginecontract.Config{Capture: enginecontract.CaptureConfig{PageTimeout: 15, Concurrency: 5, Retries: 1}},
		Progress: progress,
		Results:  enginecontract.Results{Screenshots: &screenshotTestSubmitter{}},
	}
	if err := (&Runtime{passExecutor: &scriptedPassExecutor{}, cwebpRunner: scriptedCWebP{image: testWebP(1, 1)}}).Execute(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	if len(progress.messages) != 0 {
		t.Fatalf("disabled capture progress = %#v", progress.messages)
	}
}
