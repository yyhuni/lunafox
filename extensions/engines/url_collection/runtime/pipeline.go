package urlcollectionruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

type collectorSummary struct {
	waymoreRecords            int
	katanaRecords             int
	unique                    int
	skippedOversizedRecord    int
	skippedInvalidURL         int
	skippedOversized          int
	scopeFilteredBeforeProbe  int
	skippedCollectorDuplicate int
}

func (summary collectorSummary) collectorProgress(seedRecords uint64) string {
	return fmt.Sprintf("collectors completed urlSeedRecords=%d waymoreRecords=%d katanaRecords=%d collectorUniqueURLs=%d skippedOversizedRecord=%d skippedInvalidURL=%d skippedOversized=%d scopeFilteredBeforeProbe=%d skippedCollectorDuplicate=%d", seedRecords, summary.waymoreRecords, summary.katanaRecords, summary.unique, summary.skippedOversizedRecord, summary.skippedInvalidURL, summary.skippedOversized, summary.scopeFilteredBeforeProbe, summary.skippedCollectorDuplicate)
}

// admitCollectorOutputs applies raw URL admission before any HTTPX
// request. Scope is applied here as an early egress reduction only; Server
// repeats it at result ingest and remains the final authority.
func admitCollectorOutputs(ctx context.Context, target enginecontract.Target, outputs []collectorOutput, outputPath string) (collectorSummary, error) {
	var summary collectorSummary
	if len(outputs) == 0 || outputPath == "" {
		return summary, errors.New("collector outputs and candidate path are required")
	}
	writer, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return summary, fmt.Errorf("create collector candidate file: %w", err)
	}
	writeErr := func() error {
		for _, output := range outputs {
			physical := 0
			err := ReadBoundedLines(output.path, func(line string, oversized bool) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if oversized {
					summary.skippedOversizedRecord++
					return nil
				}
				if line == "" {
					return nil
				}
				physical++
				if !InTargetScope(target, line) {
					summary.scopeFilteredBeforeProbe++
					return nil
				}
				if _, err := writer.WriteString(line + "\n"); err != nil {
					return fmt.Errorf("write collector candidate: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("parse %s output: %w", output.name, err)
			}
			if output.name == "waymore" {
				summary.waymoreRecords += physical
			} else if output.name == "katana" {
				summary.katanaRecords += physical
			}
		}
		return nil
	}()
	closeErr := writer.Close()
	if writeErr != nil {
		return summary, writeErr
	}
	if closeErr != nil {
		return summary, fmt.Errorf("close collector candidate file: %w", closeErr)
	}
	unique, duplicates, err := sortAndDeduplicateURLs(ctx, outputPath)
	if err != nil {
		return summary, err
	}
	summary.unique = unique
	summary.skippedCollectorDuplicate = duplicates
	return summary, nil
}

type uroSummary struct {
	uroRecords               int
	probeCandidates          int
	skippedOversizedRecord   int
	skippedInvalidURL        int
	skippedOversized         int
	scopeFilteredBeforeProbe int
}

func (summary uroSummary) progressMessage() string {
	return fmt.Sprintf("uro completed uroRecords=%d probeCandidateURLs=%d skippedOversizedRecord=%d skippedInvalidURL=%d skippedOversized=%d scopeFilteredBeforeProbe=%d", summary.uroRecords, summary.probeCandidates, summary.skippedOversizedRecord, summary.skippedInvalidURL, summary.skippedOversized, summary.scopeFilteredBeforeProbe)
}

func (r *Runtime) runUro(ctx context.Context, execution *enginecontract.Execution, paths pipelinePaths) (string, uroSummary, error) {
	var summary uroSummary
	if !execution.Config.Uro.Enabled {
		if err := execution.Progress.Report(ctx, "uro skipped reason=disabled"); err != nil {
			return "", summary, err
		}
		count, err := countNonEmptyRecords(paths.collectorCandidates)
		if err != nil {
			return "", summary, err
		}
		summary.probeCandidates = count
		return paths.collectorCandidates, summary, nil
	}
	if err := execution.Progress.Report(ctx, "uro started"); err != nil {
		return "", summary, err
	}
	command, err := buildUroCommand(paths.collectorCandidates, paths.uro, execution.Config.Uro)
	if err != nil {
		return "", summary, err
	}
	if err := r.executor.run(ctx, command); err != nil {
		return "", summary, err
	}
	if err := requireRegularOutput(paths.uro); err != nil {
		return "", summary, fmt.Errorf("uro output: %w", err)
	}
	writer, err := os.OpenFile(paths.probeCandidates, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", summary, fmt.Errorf("create probe candidate file: %w", err)
	}
	parseErr := ReadBoundedLines(paths.uro, func(line string, oversized bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if oversized {
			summary.skippedOversizedRecord++
			return nil
		}
		if line == "" {
			return nil
		}
		summary.uroRecords++
		if !InTargetScope(execution.Target, line) {
			summary.scopeFilteredBeforeProbe++
			return nil
		}
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
		return nil
	})
	closeErr := writer.Close()
	if parseErr != nil {
		return "", summary, fmt.Errorf("parse uro output: %w", parseErr)
	}
	if closeErr != nil {
		return "", summary, fmt.Errorf("close probe candidate file: %w", closeErr)
	}
	unique, _, err := sortAndDeduplicateURLs(ctx, paths.probeCandidates)
	if err != nil {
		return "", summary, err
	}
	summary.probeCandidates = unique
	if unique == 0 {
		return "", summary, nil
	}
	return paths.probeCandidates, summary, nil
}

type httpxSummary struct {
	httpxRecords            int
	stagedEndpoints         int
	skippedOversizedRecord  int
	skippedMalformedJSON    int
	skippedInvalidFieldType int
	skippedHTTPXFailed      int
	skippedInvalidURL       int
	skippedOversized        int
	skippedHostInconsistent int
	recoveredMetadata       int
}

func (summary httpxSummary) progressMessage() string {
	return fmt.Sprintf("httpx completed httpxRecords=%d stagedEndpoints=%d skippedOversizedRecord=%d skippedMalformedJSON=%d skippedInvalidFieldType=%d skippedHTTPXFailed=%d skippedInvalidURL=%d skippedOversized=%d skippedHostInconsistent=%d recoveredMetadata=%d", summary.httpxRecords, summary.stagedEndpoints, summary.skippedOversizedRecord, summary.skippedMalformedJSON, summary.skippedInvalidFieldType, summary.skippedHTTPXFailed, summary.skippedInvalidURL, summary.skippedOversized, summary.skippedHostInconsistent, summary.recoveredMetadata)
}

func (r *Runtime) runHTTPXOrStageMinimal(ctx context.Context, execution *enginecontract.Execution, paths pipelinePaths, probeInput string) (string, httpxSummary, error) {
	var summary httpxSummary
	if !execution.Config.HTTPX.Enabled {
		if err := execution.Progress.Report(ctx, "httpx skipped reason=disabled"); err != nil {
			return "", summary, err
		}
		writer, err := os.OpenFile(paths.endpointStage, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return "", summary, err
		}
		err = ReadBoundedLines(probeInput, func(line string, oversized bool) error {
			if oversized {
				return errors.New("probe candidate exceeds 4 MiB")
			}
			if line == "" {
				return nil
			}
			endpoint, err := observedEndpointFromURL(line)
			if err != nil {
				return err
			}
			return appendStagedEndpoint(writer, endpoint, &summary.stagedEndpoints)
		})
		closeErr := writer.Close()
		if err != nil {
			return "", summary, err
		}
		if closeErr != nil {
			return "", summary, closeErr
		}
		return paths.endpointStage, summary, nil
	}
	if err := execution.Progress.Report(ctx, "httpx started"); err != nil {
		return "", summary, err
	}
	command, err := buildHTTPXCommand(probeInput, paths.httpx, execution.Config.HTTPX)
	if err != nil {
		return "", summary, err
	}
	if err := r.executor.run(ctx, command); err != nil {
		return "", summary, err
	}
	if err := requireRegularOutput(paths.httpx); err != nil {
		return "", summary, fmt.Errorf("httpx output: %w", err)
	}
	writer, err := os.OpenFile(paths.endpointStage, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", summary, err
	}
	parseErr := ReadBoundedLines(paths.httpx, func(line string, oversized bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if oversized {
			summary.skippedOversizedRecord++
			return nil
		}
		if line == "" {
			return nil
		}
		summary.httpxRecords++
		outcome := ParseHTTPXRecord(line)
		if outcome.Skip != "" {
			switch outcome.Skip {
			case SkipMalformedJSON:
				summary.skippedMalformedJSON++
			case SkipInvalidFieldType:
				summary.skippedInvalidFieldType++
			case SkipHTTPXFailed:
				summary.skippedHTTPXFailed++
			case SkipInvalidURL:
				summary.skippedInvalidURL++
			case SkipUnassociatedIdentity:
				return errors.New("httpx output row has no candidate input identity")
			case SkipOversizedURL:
				summary.skippedOversized++
			case SkipHostInconsistent:
				summary.skippedHostInconsistent++
			}
			return nil
		}
		if outcome.RecoveredMetadata {
			summary.recoveredMetadata++
		}
		return appendStagedEndpoint(writer, outcome.Endpoint, &summary.stagedEndpoints)
	})
	closeErr := writer.Close()
	if parseErr != nil {
		return "", summary, fmt.Errorf("parse httpx output: %w", parseErr)
	}
	if closeErr != nil {
		return "", summary, closeErr
	}
	// A failed HTTPX row is a structurally valid no-finding observation. It
	// proves the tool output is usable even when other rows are malformed, so
	// only non-empty output without either an Endpoint or no-finding row fails.
	if summary.httpxRecords > 0 && summary.stagedEndpoints == 0 && summary.skippedHTTPXFailed == 0 &&
		(summary.skippedOversizedRecord > 0 || summary.skippedMalformedJSON > 0 ||
			summary.skippedInvalidFieldType > 0 || summary.skippedInvalidURL > 0 ||
			summary.skippedOversized > 0 || summary.skippedHostInconsistent > 0) {
		return "", summary, errors.New("httpx output contains no valid observable record")
	}
	return paths.endpointStage, summary, nil
}

type stagedEndpoint struct {
	URL      string                  `json:"url"`
	Ordinal  string                  `json:"ordinal"`
	Endpoint enginecontract.Endpoint `json:"endpoint"`
}

func appendStagedEndpoint(writer *os.File, endpoint enginecontract.Endpoint, count *int) error {
	payload, err := json.Marshal(stagedEndpoint{
		URL:      endpoint.URL,
		Ordinal:  fmt.Sprintf("%020d", *count),
		Endpoint: endpoint,
	})
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(payload, '\n')); err != nil {
		return err
	}
	*count++
	return nil
}

func countNonEmptyRecords(path string) (int, error) {
	count := 0
	err := ReadBoundedLines(path, func(line string, oversized bool) error {
		if oversized {
			return errors.New("candidate record exceeds 4 MiB")
		}
		if line != "" {
			count++
		}
		return nil
	})
	return count, err
}
