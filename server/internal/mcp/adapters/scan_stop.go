package adapters

import (
	"context"
	"errors"
	"strings"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type scanStopAdapter struct{ facade *scanapp.ScanFacade }

// NewScanStopper projects the canonical Scan stop services onto stop_scan and
// batch_stop_scans. Stops stay synchronous, reuse the shared stop transaction,
// and never create an MCP operation resource.
func NewScanStopper(facade *scanapp.ScanFacade) tools.ScanStopper {
	return &scanStopAdapter{facade: facade}
}

func (adapter *scanStopAdapter) Stop(ctx context.Context, input tools.ScanStopInput) (tools.ScanStopOutput, error) {
	if adapter == nil || adapter.facade == nil {
		return tools.ScanStopOutput{}, mcpErrors.ErrInternal
	}
	scanID, err := parseCanonicalScanName(input.Scan)
	if err != nil {
		return tools.ScanStopOutput{}, mcpErrors.ErrInvalidInput
	}
	revokedTaskCount, err := adapter.facade.Stop(ctx, scanID)
	if err != nil {
		return tools.ScanStopOutput{}, mapMCPScanStopError(err)
	}
	return tools.ScanStopOutput{
		Scan:             resourcenames.Scan(scanID),
		Status:           "cancelled",
		RevokedTaskCount: revokedTaskCount,
	}, nil
}

func (adapter *scanStopAdapter) BatchStop(ctx context.Context, input tools.BatchScanStopInput) (tools.BatchScanStopOutput, error) {
	if adapter == nil || adapter.facade == nil {
		return tools.BatchScanStopOutput{}, mcpErrors.ErrInternal
	}
	if len(input.Scans) == 0 || len(input.Scans) > tools.MaxBatchScanStopItems {
		return tools.BatchScanStopOutput{}, mcpErrors.ErrInvalidInput
	}

	scanIDs := make([]int, 0, len(input.Scans))
	seen := make(map[int]struct{}, len(input.Scans))
	for _, name := range input.Scans {
		scanID, err := parseCanonicalScanName(name)
		if err != nil {
			return tools.BatchScanStopOutput{}, mcpErrors.ErrInvalidInput
		}
		if _, duplicate := seen[scanID]; duplicate {
			return tools.BatchScanStopOutput{}, mcpErrors.ErrInvalidInput
		}
		seen[scanID] = struct{}{}
		scanIDs = append(scanIDs, scanID)
	}
	outcome, err := adapter.facade.BatchStop(ctx, scanIDs)
	if err != nil {
		return tools.BatchScanStopOutput{}, mapMCPScanStopError(err)
	}
	if outcome == nil {
		return tools.BatchScanStopOutput{}, mcpErrors.ErrInternal
	}
	return tools.BatchScanStopOutput{
		StoppedCount:     outcome.StoppedCount,
		SkippedCount:     outcome.SkippedCount,
		RevokedTaskCount: outcome.RevokedTaskCount,
	}, nil
}

// parseCanonicalScanName rejects any non-canonical scans/{id} shape; lenient
// alias resolution is deliberately absent so input stays single-form.
func parseCanonicalScanName(name string) (int, error) {
	scanID, err := resourcenames.ParseScan(strings.TrimSpace(name))
	if err != nil || scanID <= 0 || resourcenames.Scan(scanID) != strings.TrimSpace(name) {
		return 0, mcpErrors.ErrInvalidInput
	}
	return scanID, nil
}

// mapMCPScanStopError keeps stop failures inside the existing stable category
// vocabulary: a terminal scan is a business command failure, a missing scan is
// not-found, and internal causes are never surfaced as new categories.
func mapMCPScanStopError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, scanapp.ErrScanNotFound):
		return mcpErrors.ErrNotFound
	case errors.Is(err, scanapp.ErrScanCannotStop):
		return mcpErrors.ErrCommandFailed
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	default:
		return mcpErrors.ErrCommandFailed
	}
}
