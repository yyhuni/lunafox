package application

import "context"

// EffectiveBlacklistPolicyResolver resolves the already-validated effective
// policy while ScanRepository owns the surrounding create transaction.
type EffectiveBlacklistPolicyResolver interface {
	ResolveEffectivePatternsForScan(context.Context, int) ([]string, error)
}

// ScanBlacklistSnapshotStore loads Scan-owned immutable patterns for internal
// execution-input materialization. It is not a public Scan or Task projection.
type ScanBlacklistSnapshotStore interface {
	LoadBlacklistSnapshot(context.Context, int) ([]string, error)
}
