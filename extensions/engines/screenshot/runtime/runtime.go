package screenshotruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

type ProgressReporter interface {
	Report(context.Context, string) error
}

type Runtime struct {
	passExecutor httpxPassExecutor
	cwebpRunner  cwebpRunner
}

type screenshotAggregate struct {
	processed, submitted, skippedNavigation, skippedScreenshot, skippedInvalidPNG, skippedConversion, skippedImageBudget uint64
}

// passResult contains only observations actually emitted by HTTPX. In
// particular, it has no candidate set or replay state: missing tool rows are
// intentionally absent from the result stream.
type passResult struct {
	processed     uint64
	submitted     uint64
	malformedRows uint64
}

func New() *Runtime {
	return &Runtime{passExecutor: containerHTTPXExecutor{}, cwebpRunner: containerCWebPRunner{}}
}

func (runtime *Runtime) Execute(ctx context.Context, execution *enginecontract.Execution) (err error) {
	if ctx == nil {
		return errors.New("execution context is required")
	}
	if execution == nil {
		return errors.New("execution is required")
	}
	if execution.Progress == nil || execution.Results.Screenshots == nil {
		return errors.New("Screenshot progress and result ports are required")
	}
	config := execution.Config.Capture
	if err := validateScreenshotConfig(config); err != nil {
		return err
	}
	if !config.Enabled {
		return nil
	}
	if execution.Input.WebsiteURLs == nil {
		return errors.New("typed WebsiteURLs input handle is required")
	}
	websiteURLsPath, err := execution.Input.WebsiteURLs.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize WebsiteURLs input: %w", err)
	}
	candidatePath, candidateCount, err := materializeCandidates(ctx, execution.Target, websiteURLsPath, execution.Workspace)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := cleanupScreenshotArtifacts(execution.Workspace); cleanupErr != nil {
			if err == nil {
				err = cleanupErr
			} else {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	if err := execution.Progress.Report(ctx, fmt.Sprintf("input_ready candidates=%d pageTimeout=%d concurrency=%d retries=%d", candidateCount, config.PageTimeout, config.Concurrency, config.Retries)); err != nil {
		return err
	}
	if runtime == nil || runtime.passExecutor == nil || runtime.cwebpRunner == nil {
		return errors.New("Screenshot runtime executors are not configured")
	}
	if candidateCount == 0 {
		return execution.Progress.Report(ctx, "completed candidates=0 processed=0 submitted=0 attempts=0 skippedNavigation=0 skippedScreenshot=0 skippedInvalidPNG=0 skippedConversion=0 skippedImageBudget=0")
	}
	command, err := buildHTTPXCommand(candidatePath, execution.Workspace, config)
	if err != nil {
		return err
	}
	if err := execution.Progress.Report(ctx, fmt.Sprintf("screenshot_attempt started attempt=1 candidates=%d", candidateCount)); err != nil {
		return err
	}
	var final screenshotAggregate
	result, err := runtime.executePass(ctx, command, execution.Workspace, execution.Results.Screenshots, &final)
	if err != nil {
		return err
	}
	if err := execution.Progress.Report(ctx, fmt.Sprintf("screenshot_attempt completed attempt=1 processed=%d submitted=%d retryPending=0", result.processed, result.submitted)); err != nil {
		return err
	}
	return execution.Progress.Report(ctx, fmt.Sprintf("completed candidates=%d processed=%d submitted=%d attempts=1 skippedNavigation=%d skippedScreenshot=%d skippedInvalidPNG=%d skippedConversion=%d skippedImageBudget=%d malformedRows=%d", candidateCount, final.processed, final.submitted, final.skippedNavigation, final.skippedScreenshot, final.skippedInvalidPNG, final.skippedConversion, final.skippedImageBudget, result.malformedRows))
}

// executePass consumes one bounded HTTPX JSONL stream. Its identity and URL
// semantics come solely from each observed row; candidate input is not an
// authorization or attribution ledger.
func (runtime *Runtime) executePass(ctx context.Context, command httpxCommand, workspace string, submitter enginecontract.ScreenshotSubmitter, final *screenshotAggregate) (passResult, error) {
	if ctx == nil || workspace == "" || submitter == nil || final == nil {
		return passResult{}, errors.New("Screenshot pass inputs are required")
	}
	result := passResult{}
	outputRecords := uint64(0)
	validRecords := uint64(0)
	processRow := func(payload []byte) error {
		if outputRecords == ^uint64(0) {
			return errors.New("HTTPX output record count overflow")
		}
		outputRecords++
		row, err := decodeHTTPXRow(payload)
		if err != nil {
			if result.malformedRows == ^uint64(0) || result.processed == ^uint64(0) {
				return errors.New("HTTPX malformed record count overflow")
			}
			result.malformedRows++
			result.processed++
			return nil
		}
		if validRecords == ^uint64(0) {
			return errors.New("HTTPX valid record count overflow")
		}
		validRecords++

		outcome := outcomeScreenshot
		var encoded []byte
		if row.Failed {
			outcome = outcomeNavigation
			if row.ScreenshotPath != "" {
				if err := validatePNGReclaimPath(workspace, row.ScreenshotPath); err != nil {
					return err
				}
				if err := reclaimPNG(row.ScreenshotPath); err != nil {
					return err
				}
			}
		} else if row.ScreenshotPath != "" {
			outputBase := filepath.Join(workspace, ".screenshot-"+safeCandidateID(row.Identity))
			encoded, outcome, err = processScreenshotPNG(ctx, workspace, row.ScreenshotPath, runtime.cwebpRunner, outputBase)
			if err != nil {
				return err
			}
			if reclaimErr := reclaimPNG(row.ScreenshotPath); reclaimErr != nil {
				return reclaimErr
			}
		}
		if result.processed == ^uint64(0) {
			return errors.New("HTTPX processed record count overflow")
		}
		result.processed++
		if outcome == outcomeSubmitted {
			item := enginecontract.Screenshot{URL: row.Identity, StatusCode: validStatusPointer(row.StatusCode), Image: encoded}
			if err := submitScreenshot(ctx, submitter, item); err != nil {
				return err
			}
			if final.processed == ^uint64(0) || final.submitted == ^uint64(0) || result.submitted == ^uint64(0) {
				return errors.New("Screenshot aggregate count overflow")
			}
			final.processed++
			final.submitted++
			result.submitted++
			return nil
		}
		addFinalOutcome(final, outcome)
		return nil
	}
	if err := runtime.passExecutor.Run(ctx, command, processRow); err != nil {
		return passResult{}, err
	}
	if outputRecords > 0 && validRecords == 0 {
		return passResult{}, errors.New("HTTPX output contains no valid observed URL row")
	}
	return result, nil
}

func addFinalOutcome(final *screenshotAggregate, outcome screenshotAttemptOutcome) {
	if final == nil {
		return
	}
	final.processed++
	switch outcome {
	case outcomeNavigation:
		final.skippedNavigation++
	case outcomeScreenshot:
		final.skippedScreenshot++
	case outcomeInvalidPNG:
		final.skippedInvalidPNG++
	case outcomeConversion:
		final.skippedConversion++
	case outcomeImageBudget:
		final.skippedImageBudget++
	}
}

func submitScreenshot(ctx context.Context, submitter enginecontract.ScreenshotSubmitter, item enginecontract.Screenshot) error {
	if submitter == nil {
		return errors.New("Screenshot result submitter is required")
	}
	items := make(chan enginecontract.Screenshot, 1)
	items <- item
	close(items)
	return submitter.Submit(ctx, items)
}

func validStatusPointer(value *int) *int {
	if value == nil || *value < 100 || *value > 599 {
		return nil
	}
	copy := *value
	return &copy
}

func safeCandidateID(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func cleanupScreenshotArtifacts(workspace string) error {
	if workspace == "" {
		return nil
	}
	var result error
	if err := os.Remove(filepath.Join(workspace, "httpx-candidates.txt")); err != nil && !errors.Is(err, os.ErrNotExist) {
		result = errors.Join(result, err)
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return errors.Join(result, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".screenshot-") {
			if err := os.RemoveAll(filepath.Join(workspace, name)); err != nil {
				result = errors.Join(result, err)
			}
		}
	}
	return result
}
