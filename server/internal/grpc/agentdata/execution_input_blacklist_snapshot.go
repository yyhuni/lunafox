package agentdata

import (
	"context"
	"errors"
	"fmt"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

// ExecutionInputBlacklistSnapshotSource resolves the immutable matcher for one
// Server-owned execution-input materialization. It deliberately has no path
// to the editable BlacklistPolicy resource.
type ExecutionInputBlacklistSnapshotSource interface {
	ResolveExecutionInputBlacklistFilter(context.Context, int) (*scanapp.ExecutionInputBlacklistFilter, error)
}

type executionInputBlacklistSnapshotSource struct {
	snapshots scanapp.ScanBlacklistSnapshotStore
}

// NewExecutionInputBlacklistSnapshotSource adapts the Scan-private snapshot
// store to the execution artifact boundary without exposing it to Agent or
// Engine contracts.
func NewExecutionInputBlacklistSnapshotSource(snapshots scanapp.ScanBlacklistSnapshotStore) ExecutionInputBlacklistSnapshotSource {
	if snapshots == nil {
		return nil
	}
	return &executionInputBlacklistSnapshotSource{snapshots: snapshots}
}

func (source *executionInputBlacklistSnapshotSource) ResolveExecutionInputBlacklistFilter(ctx context.Context, scanID int) (*scanapp.ExecutionInputBlacklistFilter, error) {
	if source == nil || source.snapshots == nil {
		return nil, unavailableError("read frozen Scan blacklist snapshot", errors.New("snapshot source is unavailable"))
	}
	if ctx == nil {
		return nil, dataLossError(errors.New("execution input snapshot context is required"))
	}
	if scanID <= 0 {
		return nil, dataLossError(fmt.Errorf("execution input snapshot scan id is required"))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	patterns, err := source.snapshots.LoadBlacklistSnapshot(ctx, scanID)
	if err != nil {
		switch {
		case errors.Is(err, scanapp.ErrScanBlacklistSnapshotNotFound), errors.Is(err, scanapp.ErrScanBlacklistSnapshotDataIntegrity):
			return nil, dataLossError(fmt.Errorf("read frozen Scan blacklist snapshot: %w", err))
		default:
			return nil, unavailableError("read frozen Scan blacklist snapshot", err)
		}
	}
	// A repository adapter normally validates this too. Rechecking here keeps
	// the stream fail-closed even when another internal store implementation is
	// injected at this boundary.
	if err := blacklistdomain.ValidateCanonicalEffectivePatterns(patterns); err != nil {
		return nil, dataLossError(fmt.Errorf("validate frozen Scan blacklist snapshot: %w", err))
	}
	matcher, err := blacklistdomain.CompileMatcher(patterns)
	if err != nil {
		return nil, dataLossError(fmt.Errorf("compile frozen Scan blacklist snapshot: %w", err))
	}
	filter, err := scanapp.NewExecutionInputBlacklistFilter(matcher)
	if err != nil {
		return nil, dataLossError(err)
	}
	return filter, nil
}

var _ ExecutionInputBlacklistSnapshotSource = (*executionInputBlacklistSnapshotSource)(nil)
