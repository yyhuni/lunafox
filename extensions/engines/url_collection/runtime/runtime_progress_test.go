package urlcollectionruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestExecuteReportsEachStageOnceAndFinalFunnel(t *testing.T) {
	workspace := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, []byte("https://example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	progress := &capturedProgress{}
	results := &captureEndpointSubmitter{}
	execution := runtimeProgressExecution(workspace, seedPath, progress, results)
	runtime := Runtime{executor: scriptToolExecutor{collectorURL: "https://example.com/crawl"}}

	if err := runtime.Execute(context.Background(), execution); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, message := range []string{
		"input_ready urlSeedRecords=2",
		"waymore skipped reason=disabled",
		"katana started",
		"katana completed",
		"collectors completed urlSeedRecords=2 waymoreRecords=0 katanaRecords=1 collectorUniqueURLs=1",
		"uro started",
		"uro completed uroRecords=1 probeCandidateURLs=1",
		"result_staging started",
		"httpx started",
		"httpx completed httpxRecords=1 stagedEndpoints=1",
		"result_staging completed stagedEndpoints=1",
		"result_submission started",
		"result_submission completed",
		"completed urlSeedRecords=2 collectorUniqueURLs=1 probeCandidateURLs=1 stagedEndpoints=1",
	} {
		if !containsProgressPrefix(progress.messages, message) {
			t.Fatalf("missing progress %q in %#v", message, progress.messages)
		}
	}
	if countProgressPrefix(progress.messages, "uro completed") != 1 || countProgressPrefix(progress.messages, "httpx completed") != 1 {
		t.Fatalf("Uro and HTTPX must each report one completion: %#v", progress.messages)
	}
	if results.calls != 1 || len(results.items) != 1 {
		t.Fatalf("unexpected result submission: calls=%d items=%#v", results.calls, results.items)
	}
}

func TestExecuteSkipsDownstreamAndSubmissionForZeroCollectorResults(t *testing.T) {
	workspace := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, []byte("https://example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	progress := &capturedProgress{}
	results := &captureEndpointSubmitter{}
	execution := runtimeProgressExecution(workspace, seedPath, progress, results)
	runtime := Runtime{executor: scriptToolExecutor{}}

	if err := runtime.Execute(context.Background(), execution); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, message := range []string{
		"uro skipped reason=empty_input",
		"httpx skipped reason=empty_input",
		"result_staging skipped reason=empty_input",
		"result_submission skipped reason=empty_input",
		"completed urlSeedRecords=2 collectorUniqueURLs=0 probeCandidateURLs=0 stagedEndpoints=0",
	} {
		if !containsProgress(progress.messages, message) {
			t.Fatalf("missing progress %q in %#v", message, progress.messages)
		}
	}
	if results.calls != 0 {
		t.Fatalf("zero collector results must not submit an empty batch, got %d calls", results.calls)
	}
}

type capturedProgress struct{ messages []string }

func (progress *capturedProgress) Report(_ context.Context, message string) error {
	progress.messages = append(progress.messages, message)
	return nil
}

type scriptToolExecutor struct{ collectorURL string }

func (executor scriptToolExecutor) run(_ context.Context, command toolCommand) error {
	output, err := toolArgument(command.args, "-o")
	if err != nil {
		return err
	}
	switch command.name {
	case "katana":
		return os.WriteFile(output, []byte(executor.collectorURL+"\n"), 0o600)
	case "uro":
		input, err := toolArgument(command.args, "-i")
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		return os.WriteFile(output, contents, 0o600)
	case "httpx":
		input, err := toolArgument(command.args, "-list")
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		url := strings.TrimSpace(string(contents))
		return os.WriteFile(output, []byte(fmt.Sprintf(`{"input":%q,"host":"example.com"}`, url)+"\n"), 0o600)
	default:
		return fmt.Errorf("unexpected tool %q", command.name)
	}
}

func runtimeProgressExecution(workspace, seedPath string, progress ProgressReporter, results *captureEndpointSubmitter) *enginecontract.Execution {
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{WebsiteURLs: testInputPath(seedPath)},
		Workspace: workspace,
		Config: enginecontract.Config{
			Waymore: enginecontract.WaymoreConfig{Enabled: false, Timeout: 60},
			Katana:  enginecontract.KatanaConfig{Enabled: true, Timeout: 60, Depth: 1, Concurrency: 1, RateLimit: 1, RequestTimeout: 1},
			Uro:     enginecontract.UroConfig{Enabled: true, Timeout: 60},
			HTTPX:   enginecontract.HTTPXConfig{Enabled: true, Timeout: 60, Threads: 1, RateLimit: 1, RequestTimeout: 1},
		},
		Progress: progress,
		Results:  enginecontract.Results{Endpoints: results},
	}
}

func toolArgument(arguments []string, name string) (string, error) {
	for index, argument := range arguments {
		if argument == name && index+1 < len(arguments) {
			return arguments[index+1], nil
		}
	}
	return "", fmt.Errorf("missing %s argument", name)
}

func containsProgress(messages []string, want string) bool {
	for _, message := range messages {
		if message == want {
			return true
		}
	}
	return false
}

func containsProgressPrefix(messages []string, prefix string) bool {
	for _, message := range messages {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}

func countProgressPrefix(messages []string, prefix string) int {
	count := 0
	for _, message := range messages {
		if strings.HasPrefix(message, prefix) {
			count++
		}
	}
	return count
}
