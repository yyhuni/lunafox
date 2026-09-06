package urlcollectionruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestRunCollectorsFailsForIPWhenKatanaIsDisabled(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	paths.websiteSeeds = "seeds.txt"
	execution := &enginecontract.Execution{
		Target: enginecontract.Target{Type: enginecontract.TargetTypeIP, Value: "192.0.2.10"},
		Config: enginecontract.Config{
			Waymore: enginecontract.WaymoreConfig{Enabled: true, Timeout: 60},
			Katana:  enginecontract.KatanaConfig{Enabled: false},
		},
		Input:    enginecontract.Input{WebsiteURLs: testInputPath("unused")},
		Progress: noOpProgress{},
	}
	_, err := (&Runtime{executor: collectorTestExecutor{}}).runCollectors(context.Background(), execution, paths)
	if err == nil || err.Error() != "no enabled URL Collection collector is applicable" {
		t.Fatalf("runCollectors() error = %v", err)
	}
}

func TestRunCollectorsCancelsSiblingAfterCollectorFailure(t *testing.T) {
	workspace := t.TempDir()
	paths := pipelinePaths{workspace: workspace}
	if err := paths.prepare(); err != nil {
		t.Fatal(err)
	}
	paths.websiteSeeds = "seeds.txt"
	katanaStarted := make(chan struct{})
	katanaCancelled := make(chan struct{})
	var once sync.Once
	executor := collectorTestExecutor{runFn: func(ctx context.Context, command toolCommand) error {
		switch command.name {
		case "waymore":
			select {
			case <-katanaStarted:
			case <-time.After(time.Second):
				return errors.New("Katana did not start concurrently")
			}
			return errors.New("waymore failed")
		case "katana":
			once.Do(func() { close(katanaStarted) })
			<-ctx.Done()
			close(katanaCancelled)
			return ctx.Err()
		default:
			return errors.New("unexpected collector")
		}
	}}
	execution := &enginecontract.Execution{
		Target: enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Config: enginecontract.Config{
			Waymore: enginecontract.WaymoreConfig{Enabled: true, Timeout: 60},
			Katana: enginecontract.KatanaConfig{Enabled: true, Timeout: 60, Depth: 1, Concurrency: 1,
				RateLimit: 1, RequestTimeout: 1, Retries: 0},
		},
		Input:    enginecontract.Input{WebsiteURLs: testInputPath("seeds.txt")},
		Progress: noOpProgress{},
	}
	_, err := (&Runtime{executor: executor}).runCollectors(context.Background(), execution, paths)
	if err == nil {
		t.Fatal("runCollectors() accepted a failed collector")
	}
	select {
	case <-katanaCancelled:
	case <-time.After(time.Second):
		t.Fatal("Katana did not observe sibling cancellation")
	}
}

func TestExecuteFailsWhenRequiredProgressAcknowledgementFailsBeforeTools(t *testing.T) {
	workspace := t.TempDir()
	seedPath := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(seedPath, []byte("https://example.com\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	expected := errors.New("progress acknowledgement failed")
	executor := &countingCollectorExecutor{}
	execution := &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{WebsiteURLs: testInputPath(seedPath)},
		Workspace: workspace,
		Config: enginecontract.Config{Waymore: enginecontract.WaymoreConfig{Enabled: true, Timeout: 60},
			Katana: enginecontract.KatanaConfig{Enabled: true, Timeout: 60, Depth: 1, Concurrency: 1, RateLimit: 1, RequestTimeout: 1}},
		Progress: failingProgress{err: expected},
		Results:  enginecontract.Results{Endpoints: &captureEndpointSubmitter{}},
	}
	err := (&Runtime{executor: executor}).Execute(context.Background(), execution)
	if !errors.Is(err, expected) || executor.calls != 0 {
		t.Fatalf("Execute() error/calls = %v/%d, want progress failure before tools", err, executor.calls)
	}
}

type collectorTestExecutor struct {
	runFn func(context.Context, toolCommand) error
}

type countingCollectorExecutor struct{ calls int }

func (executor *countingCollectorExecutor) run(context.Context, toolCommand) error {
	executor.calls++
	return nil
}

type failingProgress struct{ err error }

func (progress failingProgress) Report(context.Context, string) error { return progress.err }

func (executor collectorTestExecutor) run(ctx context.Context, command toolCommand) error {
	if executor.runFn != nil {
		return executor.runFn(ctx, command)
	}
	return nil
}
