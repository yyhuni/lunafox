package urlcollectionruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestRunUroFiltersOutOfScopeToolOutputWithoutServerURLValidation(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.collectorCandidates, []byte("https://example.com/a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{executor: outputToolExecutor{content: "not a URL\n"}}
	path, summary, err := runtime.runUro(context.Background(), pipelineTestExecution(workspace), paths)
	if err != nil || path != "" || summary.uroRecords != 1 || summary.scopeFilteredBeforeProbe != 1 {
		t.Fatalf("runUro() = path=%q summary=%#v error=%v", path, summary, err)
	}
}

func TestRunUroFailsOnToolErrorWithoutUsingCandidateFallback(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	candidates := []byte("https://example.com/a\n")
	if err := os.WriteFile(paths.collectorCandidates, candidates, 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{executor: outputToolExecutor{err: context.DeadlineExceeded}}
	if _, _, err := runtime.runUro(context.Background(), pipelineTestExecution(workspace), paths); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runUro() error = %v, want deadline failure", err)
	}
	if _, err := os.Stat(paths.uro); !os.IsNotExist(err) {
		t.Fatalf("Uro failure published a fallback output: %v", err)
	}
	got, err := os.ReadFile(paths.collectorCandidates)
	if err != nil || string(got) != string(candidates) {
		t.Fatalf("Uro failure changed candidates: %q, %v", got, err)
	}
}

func TestRunHTTPXAcceptsExplicitNoFindingRows(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.probeCandidates, []byte("https://example.com/a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{executor: outputToolExecutor{content: `{"input":"https://example.com/a","host":"example.com","failed":true}` + "\n"}}
	stage, summary, err := runtime.runHTTPXOrStageMinimal(context.Background(), pipelineTestExecution(workspace), paths, paths.probeCandidates)
	if err != nil {
		t.Fatalf("runHTTPXOrStageMinimal() error = %v", err)
	}
	if stage != paths.endpointStage || summary.httpxRecords != 1 || summary.skippedHTTPXFailed != 1 || summary.stagedEndpoints != 0 {
		t.Fatalf("HTTPX summary = %#v, stage = %q", summary, stage)
	}
}

func TestRunHTTPXAcceptsNoFindingAlongsideMalformedRows(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.probeCandidates, []byte("https://example.com/a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{executor: outputToolExecutor{content: "not-json\n" + `{"input":"https://example.com/a","host":"example.com","failed":true}` + "\n"}}
	stage, summary, err := runtime.runHTTPXOrStageMinimal(context.Background(), pipelineTestExecution(workspace), paths, paths.probeCandidates)
	if err != nil {
		t.Fatalf("runHTTPXOrStageMinimal() error = %v", err)
	}
	if stage != paths.endpointStage || summary.httpxRecords != 2 || summary.skippedMalformedJSON != 1 || summary.skippedHTTPXFailed != 1 || summary.stagedEndpoints != 0 {
		t.Fatalf("HTTPX summary = %#v, stage = %q", summary, stage)
	}
}

func TestRunHTTPXStagesHostMismatchForServerValidation(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.probeCandidates, []byte("https://example.com/a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{executor: outputToolExecutor{content: `{"input":"https://example.com/a","host":"other.example.com"}` + "\n"}}
	stage, summary, err := runtime.runHTTPXOrStageMinimal(context.Background(), pipelineTestExecution(workspace), paths, paths.probeCandidates)
	if err != nil || stage != paths.endpointStage || summary.stagedEndpoints != 1 {
		t.Fatalf("runHTTPXOrStageMinimal() = stage=%q summary=%#v error=%v", stage, summary, err)
	}
	line, err := os.ReadFile(paths.endpointStage)
	if err != nil || !strings.Contains(string(line), `"host":"other.example.com"`) {
		t.Fatalf("staged endpoint changed host evidence: %q, %v", line, err)
	}
}

type outputToolExecutor struct {
	content string
	err     error
}

func (executor outputToolExecutor) run(_ context.Context, command toolCommand) error {
	if executor.err != nil {
		return executor.err
	}
	output := ""
	for index, argument := range command.args {
		if (argument == "-o" || argument == "-oU") && index+1 < len(command.args) {
			output = command.args[index+1]
			break
		}
	}
	if output == "" {
		return errors.New("test tool output path is missing")
	}
	return os.WriteFile(filepath.Clean(output), []byte(executor.content), 0o600)
}

func pipelineTestExecution(workspace string) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target: enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Config: enginecontract.Config{
			Uro:   enginecontract.UroConfig{Enabled: true, Timeout: 60},
			HTTPX: enginecontract.HTTPXConfig{Enabled: true, Timeout: 60, Threads: 1, RateLimit: 1, RequestTimeout: 1},
		},
		Workspace: workspace,
		Progress:  noOpProgress{},
	}
}

type noOpProgress struct{}

func (noOpProgress) Report(context.Context, string) error { return nil }
