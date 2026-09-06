package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

// Runtime owns the Directory-specific lifecycle around the shared Engine API
// ports. Agent terminal reporting remains the sole owner of Task state.
type Runtime struct {
	aggregation directoryAggregationOptions
}

func New() *Runtime {
	return &Runtime{}
}

func (runtime *Runtime) Execute(ctx context.Context, execution *enginecontract.Execution) error {
	if runtime == nil {
		return errors.New("Directory runtime is required")
	}
	if ctx == nil {
		return errors.New("execution context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if execution == nil {
		return errors.New("execution is required")
	}
	if execution.Progress == nil {
		return errors.New("Directory progress port is required")
	}
	if execution.Results.Directories == nil {
		return errors.New("typed Directory result port is required")
	}
	if err := ValidateConfig(execution.Config.Ffuf); err != nil {
		return err
	}
	if _, err := requireDirectoryWorkspace(execution.Workspace); err != nil {
		return err
	}
	if execution.Input.WebsiteURLs == nil {
		return errors.New("typed WebsiteURLs input handle is required")
	}
	websiteURLsPath, err := execution.Input.WebsiteURLs.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize WebsiteURLs input: %w", err)
	}

	plan, err := PreflightCandidates(ctx, execution.Target, websiteURLsPath)
	if err != nil {
		return fmt.Errorf("preflight Directory candidates: %w", err)
	}
	websiteCandidates := plan.CandidateCount()
	if err := execution.Progress.Report(ctx, fmt.Sprintf("input_ready websiteCandidates=%d", websiteCandidates)); err != nil {
		return directoryProgressError(ctx, "input_ready", err)
	}

	options := runtime.aggregation
	options.beforeScheduling = func(reportCtx context.Context) error {
		if err := execution.Progress.Report(reportCtx, fmt.Sprintf("scan_started websiteCandidates=%d", websiteCandidates)); err != nil {
			return directoryProgressError(reportCtx, "scan_started", err)
		}
		return nil
	}
	summary, aggregationErr := scanDeduplicateAndSubmitDirectories(
		ctx,
		plan,
		execution.Config.Ffuf,
		execution.Workspace,
		execution.Results.Directories,
		options,
	)
	if aggregationErr != nil {
		if errors.Is(aggregationErr, ErrAllFFUFRecordsRejected) && ctx.Err() == nil {
			message := fmt.Sprintf(
				"aggregation_failed malformedRecords=%d invalidRecords=%d oversizedRecords=%d",
				summary.Records.MalformedRecords,
				summary.Records.InvalidRecords,
				summary.Records.OversizedRecords,
			)
			if reportErr := execution.Progress.Report(ctx, message); reportErr != nil {
				return errors.Join(aggregationErr, directoryProgressError(ctx, "aggregation_failed", reportErr))
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return aggregationErr
	}
	if err := validateDirectoryCompletionSummary(websiteCandidates, summary); err != nil {
		return err
	}

	message := fmt.Sprintf(
		"aggregation-completed websiteCandidates=%d normalCompletedWebsites=%d timedOutWebsites=%d skippedDuplicate=%d malformedRecords=%d invalidRecords=%d oversizedRecords=%d",
		websiteCandidates,
		summary.Websites.NormalCompletedWebsites,
		summary.Websites.TimedOutWebsites,
		summary.SkippedDuplicate,
		summary.Records.MalformedRecords,
		summary.Records.InvalidRecords,
		summary.Records.OversizedRecords,
	)
	if err := execution.Progress.Report(ctx, message); err != nil {
		return directoryProgressError(ctx, "aggregation-completed", err)
	}
	return nil
}

func validateDirectoryCompletionSummary(websiteCandidates uint64, summary DirectoryAggregationSummary) error {
	if summary.Websites.WebsiteCandidates != websiteCandidates {
		return errors.New("Directory aggregation candidate count changed after preflight")
	}
	if summary.Websites.TimedOutWebsites > websiteCandidates ||
		summary.Websites.NormalCompletedWebsites != websiteCandidates-summary.Websites.TimedOutWebsites {
		return errors.New("Directory Website outcome counts do not match candidate count")
	}
	return nil
}

func directoryProgressError(ctx context.Context, stage string, err error) error {
	if ctx != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
	}
	return fmt.Errorf("report Directory %s progress: %w", stage, err)
}
