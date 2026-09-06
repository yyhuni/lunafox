package portscanruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
	resultparser "github.com/yyhuni/lunafox/engines/port_scan/internal/results"
)

func (r *Runtime) ReportResults(ctx context.Context, progress ProgressReporter, hostPortResults portscancontract.HostPortSubmitter, artifacts *resultArtifacts) error {
	if artifacts == nil || len(artifacts.resultArtifactPaths) == 0 {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("result context is required")
	}
	if hostPortResults == nil {
		return fmt.Errorf("typed HostPort result submitter is required")
	}
	if progress == nil {
		return fmt.Errorf("engine progress port is required")
	}

	stager, err := newResultDedupStager(resultDedupOptions{WorkspaceDir: artifacts.workspaceDir})
	if err != nil {
		return fmt.Errorf("create host-port result deduplication staging: %w", err)
	}

	summary, err := resultparser.StreamNaabuHostPortsWithSummary(ctx, artifacts.resultArtifactPaths, func(item portscancontract.HostPort) error {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("encode canonical host-port result: %w", err)
		}
		return stager.Add(ctx, canonicalHostPortKey(item), payload)
	})
	if err != nil {
		if closeErr := stager.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("stage host-port results: %w", err),
				fmt.Errorf("clean up host-port result deduplication staging: %w", closeErr),
			)
		}
		return fmt.Errorf("stage host-port results: %w", err)
	}

	streamContext, cancel := context.WithCancel(ctx)
	defer cancel()
	items := make(chan portscancontract.HostPort)
	type deduplicationResult struct {
		stats resultDedupStats
		err   error
	}
	streamedResults := make(chan deduplicationResult, 1)
	go func() {
		defer close(items)
		stats, err := stager.Stream(streamContext, func(record resultDedupRecord) error {
			var item portscancontract.HostPort
			if err := json.Unmarshal(record.Payload, &item); err != nil {
				return fmt.Errorf("decode staged host-port result: %w", err)
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

	if err := hostPortResults.Submit(streamContext, items); err != nil {
		cancel()
		streamed := <-streamedResults
		submissionErr := fmt.Errorf("submit typed host-port results: %w", err)
		return resultDedupSubmissionError(submissionErr, streamed.err, stager.Close())
	}
	streamed := <-streamedResults
	if streamed.err != nil {
		return fmt.Errorf("deduplicate host-port results: %w", streamed.err)
	}
	summary.SkippedDuplicate += streamed.stats.DuplicateCount
	log.Printf("results reported hostPorts=%d", streamed.stats.UniqueCount)
	if err := progress.Report(ctx, fmt.Sprintf("complete result reporting sourceRecords=%d parsedItems=%d submittedItems=%d skippedMalformed=%d skippedInvalid=%d skippedFailed=%d skippedDuplicate=%d skippedOversized=%d",
		summary.SourceRecords,
		summary.ParsedItems,
		streamed.stats.UniqueCount,
		summary.SkippedMalformed,
		summary.SkippedInvalid,
		summary.SkippedFailed,
		summary.SkippedDuplicate,
		summary.SkippedOversized,
	)); err != nil {
		return fmt.Errorf("report host-port result progress: %w", err)
	}
	return nil
}

// canonicalHostPortKey remains independent from transport JSON fields. A
// canonical DNS name or IPv4 literal cannot contain NUL, so the separators
// preserve each normalized identity field during local staging.
func canonicalHostPortKey(item portscancontract.HostPort) string {
	return item.Host + "\x00" + item.IP + "\x00" + strconv.Itoa(item.Port)
}
