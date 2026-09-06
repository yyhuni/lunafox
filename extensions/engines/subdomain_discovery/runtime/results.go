package subdomaindiscoveryruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	subdomaincontract "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
	resultparser "github.com/yyhuni/lunafox/engines/subdomain_discovery/internal/results"
)

func (w *Runtime) ReportResults(ctx context.Context, progress ProgressReporter, subdomainResults subdomaincontract.SubdomainSubmitter, artifacts *resultArtifacts) error {
	if artifacts == nil || artifacts.resultArtifactPath == "" {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("result context is required")
	}
	if subdomainResults == nil {
		return fmt.Errorf("typed Subdomain result submitter is required")
	}
	if progress == nil {
		return fmt.Errorf("engine progress port is required")
	}

	stager, err := newResultDedupStager(resultDedupOptions{workspaceDir: artifacts.workspaceDir})
	if err != nil {
		return fmt.Errorf("create subdomain result deduplication staging: %w", err)
	}

	summary, err := resultparser.StreamSubdomainsWithSummary(ctx, artifacts.resultArtifactPath, func(item subdomaincontract.Subdomain) error {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("encode canonical subdomain result: %w", err)
		}
		return stager.add(ctx, item.DNSName, payload)
	})
	if err != nil {
		if closeErr := stager.close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("stage subdomain results: %w", err),
				fmt.Errorf("clean up subdomain result deduplication staging: %w", closeErr),
			)
		}
		return fmt.Errorf("stage subdomain results: %w", err)
	}

	streamContext, cancel := context.WithCancel(ctx)
	defer cancel()
	items := make(chan subdomaincontract.Subdomain)
	type deduplicationResult struct {
		stats resultDedupStats
		err   error
	}
	streamedResults := make(chan deduplicationResult, 1)
	go func() {
		defer close(items)
		stats, err := stager.stream(streamContext, func(record resultDedupRecord) error {
			var item subdomaincontract.Subdomain
			if err := json.Unmarshal(record.payload, &item); err != nil {
				return fmt.Errorf("decode staged subdomain result: %w", err)
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

	if err := subdomainResults.Submit(streamContext, items); err != nil {
		cancel()
		streamed := <-streamedResults
		submissionErr := fmt.Errorf("submit typed subdomain results: %w", err)
		return resultSubmissionFailure(submissionErr, streamed.err, stager.close())
	}
	streamed := <-streamedResults
	if streamed.err != nil {
		return fmt.Errorf("deduplicate subdomain results: %w", streamed.err)
	}
	summary.SkippedDuplicate += streamed.stats.duplicateCount
	log.Printf("results reported subdomains=%d", streamed.stats.uniqueCount)
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
		return fmt.Errorf("report subdomain result progress: %w", err)
	}
	return nil
}

// resultSubmissionFailure keeps an operational streaming or cleanup failure
// observable when typed submission also fails; cancellation caused by that
// submission adds no independent diagnostic.
func resultSubmissionFailure(submissionErr, streamErr, cleanupErr error) error {
	errs := []error{submissionErr}
	if streamErr != nil && !isResultDedupContextTermination(streamErr) {
		errs = append(errs, fmt.Errorf("deduplicate subdomain results after typed submission failure: %w", streamErr))
	}
	if cleanupErr != nil {
		errs = append(errs, fmt.Errorf("clean up subdomain result deduplication staging: %w", cleanupErr))
	}
	return errors.Join(errs...)
}
