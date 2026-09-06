package websitediscoveryruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
	resultparser "github.com/yyhuni/lunafox/engines/website_discovery/internal/results"
)

func (r *Runtime) ReportResults(ctx context.Context, progress ProgressReporter, websiteResults websitediscoverycontract.WebsiteSubmitter, artifacts *resultArtifacts) error {
	if artifacts == nil || artifacts.resultArtifactPath == "" {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("result context is required")
	}
	if websiteResults == nil {
		return fmt.Errorf("typed Website result submitter is required")
	}
	if progress == nil {
		return fmt.Errorf("engine progress port is required")
	}

	stager, err := newResultDedupStager(resultDedupOptions{workspaceDir: artifacts.workspaceDir})
	if err != nil {
		return fmt.Errorf("create website result deduplication staging: %w", err)
	}

	summary, err := resultparser.StreamHTTPXWebsitesWithSummary(ctx, artifacts.resultArtifactPath, func(item websitediscoverycontract.Website) error {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("encode observed website result: %w", err)
		}
		return stager.Add(ctx, item.URL, payload)
	})
	if err != nil {
		if closeErr := stager.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("stage website results: %w", err),
				fmt.Errorf("clean up website result deduplication staging: %w", closeErr),
			)
		}
		return fmt.Errorf("stage website results: %w", err)
	}
	if summary.ParsedItems == 0 && summary.SourceRecords > 0 && summary.SkippedFailed == 0 &&
		(summary.SkippedMalformed > 0 || summary.SkippedInvalid > 0 || summary.SkippedOversized > 0) {
		if closeErr := stager.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("httpx output contains no valid observable record"),
				fmt.Errorf("clean up website result deduplication staging: %w", closeErr),
			)
		}
		return fmt.Errorf("httpx output contains no valid observable record")
	}

	streamContext, cancel := context.WithCancel(ctx)
	defer cancel()
	items := make(chan websitediscoverycontract.Website)
	type deduplicationResult struct {
		stats resultDedupStats
		err   error
	}
	streamedResults := make(chan deduplicationResult, 1)
	go func() {
		defer close(items)
		stats, err := stager.Stream(streamContext, func(record resultDedupRecord) error {
			var item websitediscoverycontract.Website
			if err := json.Unmarshal(record.payload, &item); err != nil {
				return fmt.Errorf("decode staged website result: %w", err)
			}
			select {
			case items <- item:
				return nil
			case <-streamContext.Done():
				return streamContext.Err()
			}
		})
		streamedResults <- deduplicationResult{stats: stats, err: err}
	}()

	if err := websiteResults.Submit(streamContext, items); err != nil {
		cancel()
		streamed := <-streamedResults
		submissionErr := fmt.Errorf("submit typed website results: %w", err)
		return joinWebsiteResultSubmissionErrors(submissionErr, streamed.err, stager.Close())
	}
	streamed := <-streamedResults
	if streamed.err != nil {
		return fmt.Errorf("deduplicate website results: %w", streamed.err)
	}
	summary.SkippedDuplicate += streamed.stats.duplicateCount
	log.Printf("results reported websites=%d", streamed.stats.uniqueCount)
	if err := progress.Report(ctx, fmt.Sprintf("complete result reporting sourceRecords=%d parsedItems=%d submittedItems=%d skippedMalformed=%d skippedInvalid=%d skippedFailed=%d skippedDuplicate=%d skippedOversized=%d",
		summary.SourceRecords,
		summary.ParsedItems,
		streamed.stats.uniqueCount,
		summary.SkippedMalformed,
		summary.SkippedInvalid,
		summary.SkippedFailed,
		summary.SkippedDuplicate,
		summary.SkippedOversized,
	)); err != nil {
		return fmt.Errorf("report website result progress: %w", err)
	}
	return nil
}

func joinWebsiteResultSubmissionErrors(submissionErr, streamErr, cleanupErr error) error {
	errs := []error{submissionErr}
	if streamErr != nil && !isContextTermination(streamErr) {
		errs = append(errs, fmt.Errorf("deduplicate website results after typed submission failure: %w", streamErr))
	}
	if cleanupErr != nil {
		errs = append(errs, fmt.Errorf("clean up website result deduplication staging: %w", cleanupErr))
	}
	return errors.Join(errs...)
}
