package directoryscanruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

var (
	// ErrAllWebsitesTimedOut is returned only after complete pre-deadline
	// records from every timed-out invocation have been parsed.
	ErrAllWebsitesTimedOut = errors.New("all Website FFUF invocations timed out")
	// ErrAllFFUFRecordsRejected distinguishes corrupt or invalid source output
	// from a valid zero-record FFUF run.
	ErrAllFFUFRecordsRejected = errors.New("all FFUF source records were rejected")
)

type DirectoryScanSummary struct {
	Websites WebsiteScanSummary
	Records  FFUFParseSummary
}

// ScanAndParseDirectories defers the all-timeout failure until every complete
// pre-deadline record has been parsed. Callers can therefore stage those valid
// observations before returning the terminal timeout error.
func ScanAndParseDirectories(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
	visit func(DirectoryObservation) error,
) (DirectoryScanSummary, error) {
	return scanAndParseDirectories(ctx, plan, config, workspace, visit, websiteSchedulerOptions{
		runner:         newContainerFFUFProcessRunner(),
		websiteTimeout: time.Duration(config.Timeout) * time.Second,
	})
}

func scanAndParseDirectories(
	ctx context.Context,
	plan *CandidatePlan,
	config enginecontract.FfufConfig,
	workspace string,
	visit func(DirectoryObservation) error,
	options websiteSchedulerOptions,
) (DirectoryScanSummary, error) {
	if visit == nil {
		return DirectoryScanSummary{}, errors.New("Directory observation callback is required")
	}
	websiteSummary, err := runWebsiteScans(ctx, plan, config, workspace, options)
	summary := DirectoryScanSummary{Websites: websiteSummary}
	if err != nil {
		return summary, err
	}
	for ordinal := uint64(0); ordinal < websiteSummary.WebsiteCandidates; ordinal++ {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		path, err := rawFFUFArtifactPath(workspace, ordinal)
		if err != nil {
			return summary, err
		}
		parsed, err := ParseFFUFArtifact(ctx, path, ordinal, visit)
		summary.Records.add(parsed)
		if err != nil {
			return summary, fmt.Errorf("parse FFUF artifact %d: %w", ordinal, err)
		}
	}
	if websiteSummary.AllWebsitesTimedOut() {
		return summary, ErrAllWebsitesTimedOut
	}
	if summary.Records.AllRecordsRejected() {
		return summary, ErrAllFFUFRecordsRejected
	}
	return summary, nil
}
