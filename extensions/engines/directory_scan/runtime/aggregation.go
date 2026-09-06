package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

// DirectoryAggregationSummary retains the exact scan, rejection, duplicate,
// winner, and Server-acknowledgement facts needed by lifecycle reporting.
type DirectoryAggregationSummary struct {
	Websites         WebsiteScanSummary
	Records          FFUFParseSummary
	WinnerItems      uint64
	SkippedDuplicate uint64
}

type directoryAggregationOptions struct {
	scheduler        websiteSchedulerOptions
	dedup            directoryDedupOptions
	beforeScheduling func(context.Context) error
}

// ScanDeduplicateAndSubmitDirectories makes the complete exact-URL winner set
// immutable before the first typed submission. The all-Website-timeout path is
// the sole scan failure allowed to submit complete pre-deadline observations.
func ScanDeduplicateAndSubmitDirectories(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
	submitter enginecontract.DirectorySubmitter,
) (DirectoryAggregationSummary, error) {
	return scanDeduplicateAndSubmitDirectories(ctx, plan, config, workspace, submitter, directoryAggregationOptions{
		scheduler: websiteSchedulerOptions{
			runner:         newContainerFFUFProcessRunner(),
			websiteTimeout: time.Duration(config.Timeout) * time.Second,
		},
	})
}

func scanDeduplicateAndSubmitDirectories(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
	submitter enginecontract.DirectorySubmitter,
	options directoryAggregationOptions,
) (summary DirectoryAggregationSummary, returnErr error) {
	if ctx == nil {
		return summary, errors.New("Directory aggregation context is required")
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	if plan == nil {
		return summary, errors.New("candidate plan is required")
	}
	if submitter == nil {
		return summary, errors.New("typed Directory result submitter is required")
	}
	if err := ValidateConfig(config); err != nil {
		return summary, err
	}
	if options.scheduler.runner == nil {
		options.scheduler.runner = newContainerFFUFProcessRunner()
	}
	if options.scheduler.websiteTimeout <= 0 {
		options.scheduler.websiteTimeout = time.Duration(config.Timeout) * time.Second
	}
	if options.dedup.Workspace != "" && options.dedup.Workspace != workspace {
		return summary, errors.New("Directory deduplication workspace conflicts with execution workspace")
	}
	options.dedup.Workspace = workspace
	stager, err := newDirectoryDedupStager(options.dedup)
	if err != nil {
		return summary, err
	}
	defer func() {
		cleanupErr := stager.Close()
		// Parent cancellation remains authoritative even when synchronous private
		// cleanup also fails; Agent terminal mapping must not see partial success.
		if contextErr := ctx.Err(); contextErr != nil {
			if cleanupErr != nil {
				returnErr = errors.Join(contextErr, cleanupErr)
			} else {
				returnErr = contextErr
			}
			return
		}
		if cleanupErr != nil {
			if returnErr == nil {
				returnErr = cleanupErr
			} else {
				returnErr = errors.Join(returnErr, cleanupErr)
			}
		}
	}()
	if options.beforeScheduling != nil {
		if err := options.beforeScheduling(ctx); err != nil {
			return summary, err
		}
	}

	scanSummary, scanErr := scanAndParseDirectories(ctx, plan, config, workspace, func(observation DirectoryObservation) error {
		return stager.Add(ctx, observation)
	}, options.scheduler)
	summary.Websites = scanSummary.Websites
	summary.Records = scanSummary.Records
	allWebsitesTimedOut := errors.Is(scanErr, ErrAllWebsitesTimedOut)
	if scanErr != nil && !allWebsitesTimedOut {
		return summary, scanErr
	}

	winners, dedupStats, err := stager.Finalize(ctx)
	summary.WinnerItems = dedupStats.WinnerItems
	summary.SkippedDuplicate = dedupStats.SkippedDuplicate
	if err != nil {
		return summary, fmt.Errorf("finalize Directory result winners: %w", err)
	}
	if dedupStats.InputItems != summary.Records.ParsedItems {
		return summary, errors.New("Directory staged result count does not match parsed item count")
	}
	if summary.WinnerItems > 0 {
		if err := submitDirectoryWinners(ctx, submitter, winners); err != nil {
			return summary, err
		}
	}
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	if allWebsitesTimedOut {
		return summary, ErrAllWebsitesTimedOut
	}
	return summary, nil
}

func submitDirectoryWinners(
	ctx context.Context,
	submitter enginecontract.DirectorySubmitter,
	winners *directoryWinnerSet,
) error {
	streamContext, cancelStream := context.WithCancel(ctx)
	items := make(chan enginecontract.Directory)
	streamDone := make(chan error, 1)
	go func() {
		defer close(items)
		streamDone <- winners.Stream(streamContext, func(item enginecontract.Directory) error {
			select {
			case <-streamContext.Done():
				return streamContext.Err()
			case items <- item:
				return nil
			}
		})
	}()

	submitErr := submitter.Submit(streamContext, items)
	cancelStream()
	streamErr := <-streamDone
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	var failures []error
	if submitErr != nil {
		failures = append(failures, fmt.Errorf("submit typed Directory results: %w", submitErr))
	}
	if streamErr != nil && !isOnlyDirectoryContextTermination(streamErr) {
		failures = append(failures, fmt.Errorf("stream finalized Directory winners: %w", streamErr))
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	return nil
}

func isOnlyDirectoryContextTermination(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		causes := joined.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !isOnlyDirectoryContextTermination(cause) {
				return false
			}
		}
		return true
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		if cause := wrapped.Unwrap(); cause != nil {
			return isOnlyDirectoryContextTermination(cause)
		}
	}
	return err == context.Canceled || err == context.DeadlineExceeded
}
