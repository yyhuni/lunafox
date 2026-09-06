package urlcollectionruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

// ProgressReporter is the message-only Engine API port used by URL
// Collection. A failed acknowledgement is deliberately terminal: continuing
// after losing the required execution-state channel could report a false
// successful task while Server has no reliable progress record.
type ProgressReporter interface {
	Report(context.Context, string) error
}

// Runtime owns URL Collection's image-local tool pipeline. It receives only
// generated Engine inputs and ports; workflow state, Server repositories, and
// Agent materialization capabilities are intentionally outside this boundary.
type Runtime struct {
	executor toolExecutor
}

func New() *Runtime { return &Runtime{executor: containerToolExecutor{}} }

// Execute validates the typed execution and runs the strict collection
// pipeline. Tool results are staged only after every enabled stage succeeds.
func (r *Runtime) Execute(ctx context.Context, execution *enginecontract.Execution) error {
	if ctx == nil {
		return errors.New("execution context is required")
	}
	if execution == nil {
		return errors.New("execution is required")
	}
	if execution.Progress == nil {
		return errors.New("engine progress port is required")
	}
	if execution.Results.Endpoints == nil {
		return errors.New("typed Endpoint result submitter is required")
	}
	if execution.Workspace == "" {
		return errors.New("workspace is required")
	}
	if r == nil || r.executor == nil {
		return errors.New("URL Collection tool executor is required")
	}
	if info, err := os.Stat(execution.Workspace); err != nil || !info.IsDir() {
		if err != nil {
			return fmt.Errorf("URL Collection workspace: %w", err)
		}
		return errors.New("URL Collection workspace is not a directory")
	}

	paths := pipelinePaths{workspace: execution.Workspace}
	if err := paths.prepare(); err != nil {
		return err
	}
	if execution.Input.WebsiteURLs == nil {
		return errors.New("typed WebsiteURLs input handle is required")
	}
	websiteURLsPath, err := execution.Input.WebsiteURLs.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize WebsiteURLs input: %w", err)
	}
	seedPath, seedRecords, err := prepareTargetSeeds(ctx, execution.Target, websiteURLsPath, execution.Workspace)
	if err != nil {
		return err
	}
	paths.websiteSeeds = seedPath
	if err := execution.Progress.Report(ctx, fmt.Sprintf("input_ready urlSeedRecords=%d", seedRecords)); err != nil {
		return fmt.Errorf("report input_ready progress: %w", err)
	}
	collectorPaths, err := r.runCollectors(ctx, execution, paths)
	if err != nil {
		return err
	}
	candidates, err := admitCollectorOutputs(ctx, execution.Target, collectorPaths, paths.collectorCandidates)
	if err != nil {
		return err
	}
	if err := execution.Progress.Report(ctx, candidates.collectorProgress(seedRecords)); err != nil {
		return fmt.Errorf("report collector progress: %w", err)
	}
	if candidates.unique == 0 {
		return reportEmptyDownstreamCompletion(ctx, execution.Progress, seedRecords, 0)
	}

	probeInput, uroSummary, err := r.runUro(ctx, execution, paths)
	if err != nil {
		return err
	}
	if execution.Config.Uro.Enabled {
		if err := execution.Progress.Report(ctx, uroSummary.progressMessage()); err != nil {
			return fmt.Errorf("report uro progress: %w", err)
		}
	}
	if probeInput == "" {
		return reportEmptyDownstreamCompletion(ctx, execution.Progress, seedRecords, candidates.unique)
	}

	if err := execution.Progress.Report(ctx, "result_staging started"); err != nil {
		return fmt.Errorf("report result staging start: %w", err)
	}
	staged, httpxSummary, err := r.runHTTPXOrStageMinimal(ctx, execution, paths, probeInput)
	if err != nil {
		return err
	}
	if execution.Config.HTTPX.Enabled {
		if err := execution.Progress.Report(ctx, httpxSummary.progressMessage()); err != nil {
			return fmt.Errorf("report httpx progress: %w", err)
		}
	}
	if err := execution.Progress.Report(ctx, fmt.Sprintf("result_staging completed stagedEndpoints=%d", httpxSummary.stagedEndpoints)); err != nil {
		return fmt.Errorf("report result staging completion: %w", err)
	}
	if httpxSummary.stagedEndpoints == 0 {
		if err := execution.Progress.Report(ctx, "result_submission skipped reason=empty_input"); err != nil {
			return fmt.Errorf("report empty result submission: %w", err)
		}
		return reportCompletion(ctx, execution.Progress, seedRecords, candidates.unique, uroSummary.probeCandidates, 0)
	}
	if err := execution.Progress.Report(ctx, "result_submission started"); err != nil {
		return fmt.Errorf("report result submission start: %w", err)
	}
	if err := submitStagedEndpoints(ctx, execution.Results.Endpoints, staged); err != nil {
		return err
	}
	if err := execution.Progress.Report(ctx, "result_submission completed"); err != nil {
		return fmt.Errorf("report result submission completion: %w", err)
	}
	return reportCompletion(ctx, execution.Progress, seedRecords, candidates.unique, uroSummary.probeCandidates, httpxSummary.stagedEndpoints)
}

func reportEmptyDownstreamCompletion(ctx context.Context, progress ProgressReporter, seedRecords uint64, collectorUniqueURLs int) error {
	for _, message := range []string{
		"uro skipped reason=empty_input",
		"httpx skipped reason=empty_input",
		"result_staging skipped reason=empty_input",
		"result_submission skipped reason=empty_input",
	} {
		if err := progress.Report(ctx, message); err != nil {
			return fmt.Errorf("report empty downstream stage: %w", err)
		}
	}
	return reportCompletion(ctx, progress, seedRecords, collectorUniqueURLs, 0, 0)
}

func reportCompletion(ctx context.Context, progress ProgressReporter, seedRecords uint64, collectorUniqueURLs, probeCandidateURLs, stagedEndpoints int) error {
	if err := progress.Report(ctx, fmt.Sprintf("completed urlSeedRecords=%d collectorUniqueURLs=%d probeCandidateURLs=%d stagedEndpoints=%d", seedRecords, collectorUniqueURLs, probeCandidateURLs, stagedEndpoints)); err != nil {
		return fmt.Errorf("report URL Collection completion: %w", err)
	}
	return nil
}

type pipelinePaths struct {
	workspace           string
	websiteSeeds        string
	waymore             string
	katana              string
	collectorRaw        string
	collectorCandidates string
	uro                 string
	probeCandidates     string
	httpx               string
	endpointStage       string
}

func (paths *pipelinePaths) prepare() error {
	paths.waymore = filepath.Join(paths.workspace, "waymore.txt")
	paths.katana = filepath.Join(paths.workspace, "katana.txt")
	paths.collectorRaw = filepath.Join(paths.workspace, "collector-raw.txt")
	paths.collectorCandidates = filepath.Join(paths.workspace, "collector-candidates.txt")
	paths.uro = filepath.Join(paths.workspace, "uro.txt")
	paths.probeCandidates = filepath.Join(paths.workspace, "probe-candidates.txt")
	paths.httpx = filepath.Join(paths.workspace, "httpx.jsonl")
	paths.endpointStage = filepath.Join(paths.workspace, "endpoint-stage.jsonl")
	return nil
}

type collectorOutput struct {
	name string
	path string
}

func (r *Runtime) runCollectors(ctx context.Context, execution *enginecontract.Execution, paths pipelinePaths) ([]collectorOutput, error) {
	waymoreApplicable := execution.Target.Type == enginecontract.TargetTypeDomain
	katanaApplicable := true
	if (!waymoreApplicable || !execution.Config.Waymore.Enabled) && (!katanaApplicable || !execution.Config.Katana.Enabled) {
		return nil, errors.New("no enabled URL Collection collector is applicable")
	}

	collectorCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		outputs  []collectorOutput
		outputMu sync.Mutex
		firstErr error
		errMu    sync.Mutex
		wait     sync.WaitGroup
	)
	run := func(label string, command toolCommand, output string) {
		defer wait.Done()
		if err := execution.Progress.Report(collectorCtx, label+" started"); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("report %s start: %w", label, err)
				cancel()
			}
			errMu.Unlock()
			return
		}
		if err := r.executor.run(collectorCtx, command); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = err
				cancel()
			}
			errMu.Unlock()
			return
		}
		if err := requireRegularOutput(output); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("%s output: %w", label, err)
				cancel()
			}
			errMu.Unlock()
			return
		}
		if err := execution.Progress.Report(collectorCtx, label+" completed"); err != nil {
			errMu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("report %s completion: %w", label, err)
				cancel()
			}
			errMu.Unlock()
			return
		}
		outputMu.Lock()
		outputs = append(outputs, collectorOutput{name: label, path: output})
		outputMu.Unlock()
	}
	if waymoreApplicable && execution.Config.Waymore.Enabled {
		command, err := buildWaymoreCommand(execution.Target, paths.waymore, execution.Config.Waymore)
		if err != nil {
			return nil, err
		}
		wait.Add(1)
		go run("waymore", command, paths.waymore)
	} else {
		reason := "disabled"
		if !waymoreApplicable {
			reason = "not_applicable"
		}
		if err := execution.Progress.Report(ctx, "waymore skipped reason="+reason); err != nil {
			return nil, err
		}
	}
	if execution.Config.Katana.Enabled {
		command, err := buildKatanaCommand(paths.websiteSeeds, paths.katana, execution.Config.Katana)
		if err != nil {
			return nil, err
		}
		wait.Add(1)
		go run("katana", command, paths.katana)
	} else if err := execution.Progress.Report(ctx, "katana skipped reason=disabled"); err != nil {
		return nil, err
	}
	wait.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return outputs, nil
}

func requireRegularOutput(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("must be a regular file")
	}
	return nil
}

func submitStagedEndpoints(ctx context.Context, submitter enginecontract.EndpointSubmitter, stagePath string) error {
	if stagePath == "" {
		return nil
	}
	_, total, err := sortURLLines(ctx, stagePath, false)
	if err != nil {
		return fmt.Errorf("sort staged Endpoint records: %w", err)
	}
	if total == 0 {
		return nil
	}
	submitContext, cancel := context.WithCancel(ctx)
	defer cancel()
	channel := make(chan enginecontract.Endpoint)
	type outcome struct {
		items int
		err   error
	}
	streamed := make(chan outcome, 1)
	go func() {
		defer close(channel)
		items, err := streamLastStagedEndpoints(submitContext, stagePath, channel)
		streamed <- outcome{items: items, err: err}
	}()
	err = submitter.Submit(submitContext, channel)
	if err != nil {
		cancel()
		streamResult := <-streamed
		if streamResult.err != nil && !errors.Is(streamResult.err, context.Canceled) {
			return errors.Join(fmt.Errorf("submit Endpoint batch: %w", err), streamResult.err)
		}
		return fmt.Errorf("submit Endpoint batch: %w", err)
	}
	streamResult := <-streamed
	if streamResult.err != nil {
		return streamResult.err
	}
	if streamResult.items == 0 {
		return nil
	}
	return nil
}

func streamLastStagedEndpoints(ctx context.Context, stagePath string, output chan<- enginecontract.Endpoint) (int, error) {
	var lastKey string
	var lastItem enginecontract.Endpoint
	hasLast := false
	count := 0
	emit := func() error {
		if !hasLast {
			return nil
		}
		select {
		case output <- lastItem:
			count++
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	err := ReadBoundedLines(stagePath, func(line string, oversized bool) error {
		if oversized {
			return errors.New("endpoint staging record exceeds 4 MiB")
		}
		if strings.TrimSpace(line) == "" {
			return nil
		}
		key, item, err := decodeStagedEndpoint(line)
		if err != nil {
			return err
		}
		if hasLast && key != lastKey {
			if err := emit(); err != nil {
				return err
			}
		}
		lastKey, lastItem, hasLast = key, item, true
		return nil
	})
	if err != nil {
		return count, err
	}
	if err := emit(); err != nil {
		return count, err
	}
	return count, nil
}

func decodeStagedEndpoint(line string) (string, enginecontract.Endpoint, error) {
	var staged stagedEndpoint
	if err := json.Unmarshal([]byte(line), &staged); err != nil {
		return "", enginecontract.Endpoint{}, fmt.Errorf("decode endpoint staging payload: %w", err)
	}
	if len(staged.Ordinal) != 20 || staged.URL != staged.Endpoint.URL {
		return "", enginecontract.Endpoint{}, errors.New("endpoint staging item is invalid")
	}
	return staged.URL, staged.Endpoint, nil
}
