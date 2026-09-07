package application

import (
	"context"
	"errors"
	"fmt"
	"math"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

// ResultIngestFacade is the single Server application boundary for task result
// persistence. Per-result decoding and materialization remain behind its closed
// registry.
type ResultIngestFacade struct {
	registry        resultIngestRegistry
	scanSummary     ScanResultSummaryUpdater
	materialization ResultMaterializationCoordinator
}

// ResultIngestFacadeDependencies wires the three canonical persistence
// implementations. The field types are package-private so callers cannot use
// the per-result seams as application entrypoints.
type ResultIngestFacadeDependencies struct {
	Subdomains          subdomainResultMaterializer
	HostPorts           hostPortResultMaterializer
	Websites            websiteResultMaterializer
	WebsiteTechnologies websiteTechnologyResultMaterializer
	Endpoints           endpointResultMaterializer
	Directories         directoryResultMaterializer
	Screenshots         screenshotResultMaterializer
	Vulnerabilities     vulnerabilityResultMaterializer
	ScanSummary         ScanResultSummaryUpdater
	Materialization     ResultMaterializationCoordinator
}

// NewResultIngestFacade creates the only result-ingest application boundary.
func NewResultIngestFacade(deps ResultIngestFacadeDependencies) *ResultIngestFacade {
	return &ResultIngestFacade{
		registry:        newResultIngestRegistry(deps),
		scanSummary:     deps.ScanSummary,
		materialization: deps.Materialization,
	}
}

// Ingest validates the batch envelope before any write, dispatches each
// locatable item through the closed Server registry, and synchronously
// completes all projections.
func (facade *ResultIngestFacade) Ingest(ctx context.Context, command ResultIngestCommand) (ResultIngestOutcome, error) {
	if ctx == nil {
		return ResultIngestOutcome{}, ErrResultIngestContextRequired
	}
	if err := ctx.Err(); err != nil {
		return ResultIngestOutcome{}, err
	}
	if facade == nil || command.TaskID <= 0 || command.ScanID <= 0 || command.TargetID <= 0 {
		return ResultIngestOutcome{}, ErrInvalidResultScope
	}

	batch, err := contractresults.ValidateEncodedBatchEnvelope(command.ResultType, command.Items, contractresults.DefaultBatchLimits())
	if err != nil {
		return ResultIngestOutcome{}, fmt.Errorf("%w: %v", ErrInvalidResultItems, err)
	}
	handler, ok := facade.registry.lookup(batch.ResultType)
	if !ok {
		return ResultIngestOutcome{}, fmt.Errorf("%w: %s", ErrUnsupportedResultType, batch.ResultType)
	}
	prepared, err := handler.prepare(ctx, batch.Items)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ResultIngestOutcome{}, err
		}
		return ResultIngestOutcome{}, fmt.Errorf("%w: %v", ErrInvalidResultItems, err)
	}
	if err := ctx.Err(); err != nil {
		return ResultIngestOutcome{}, err
	}
	if isNilResultDependency(facade.materialization) {
		return ResultIngestOutcome{}, ErrResultMaterializerUnavailable
	}
	if !prepared.currentOnly && isNilResultDependency(facade.scanSummary) {
		return ResultIngestOutcome{}, ErrResultMaterializerUnavailable
	}

	var (
		summary materializationSummary
		outcome ResultIngestOutcome
	)
	materialize := func(materializeCtx context.Context) error {
		var err error
		summary, err = prepared.materialize(materializeCtx, resultIngestScope{
			taskID: command.TaskID, scanID: command.ScanID, targetID: command.TargetID,
		})
		if err != nil {
			return err
		}
		// Materializers report rejected records instead of mutating them. This
		// check is deliberately inside the coordinator callback so a transaction
		// rolls back any accepted writes before the batch-level error escapes.
		if summary.invalidItems > 0 || summary.unsupportedItems > 0 {
			return ErrInvalidResultItems
		}
		if summary.scopeFilteredItems > 0 {
			return ErrResultNotAuthorized
		}
		outcome, err = resultIngestOutcome(prepared.receivedItems, summary, true)
		if err != nil {
			return err
		}
		if prepared.currentOnly {
			return nil
		}
		return facade.scanSummary.RefreshScanResultSummary(materializeCtx, command.ScanID, command.TargetID)
	}
	materializeErr := facade.materialization.Materialize(ctx, ResultMaterializationScope{
		TaskID:       command.TaskID,
		ScanID:       command.ScanID,
		TargetID:     command.TargetID,
		AgentID:      command.AgentID,
		SessionID:    command.SessionID,
		SessionEpoch: command.SessionEpoch,
	}, materialize)
	if materializeErr != nil {
		return ResultIngestOutcome{}, materializeErr
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	return outcome, nil
}

func resultIngestOutcome(received int, summary materializationSummary, complete bool) (ResultIngestOutcome, error) {
	if received <= 0 || summary.receivedItems != received || summary.invalidItems < 0 || summary.scopeFilteredItems < 0 || summary.unsupportedItems < 0 || summary.duplicateItems < 0 || summary.currentOnlyCount < 0 || summary.snapshotCount < 0 || summary.assetCount < 0 {
		return ResultIngestOutcome{}, ErrResultMaterializationInvariant
	}
	if summary.snapshotCount > int64(math.MaxInt) {
		return ResultIngestOutcome{}, ErrResultMaterializationInvariant
	}
	rejected := summary.invalidItems + summary.scopeFilteredItems + summary.unsupportedItems
	classified := summary.duplicateItems + rejected + int(summary.snapshotCount) + summary.currentOnlyCount
	if classified > received || (complete && classified != received) {
		return ResultIngestOutcome{}, ErrResultMaterializationInvariant
	}
	return ResultIngestOutcome{
		ReceivedItems:      received,
		RejectedItems:      rejected,
		DuplicateItems:     summary.duplicateItems,
		ScopeFilteredItems: summary.scopeFilteredItems,
		UnsupportedItems:   summary.unsupportedItems,
		SnapshotCount:      summary.snapshotCount,
		AssetCount:         summary.assetCount,
	}, nil
}
