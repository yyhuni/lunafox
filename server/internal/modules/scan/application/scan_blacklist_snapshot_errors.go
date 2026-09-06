package application

import "errors"

var (
	// ErrScanBlacklistSnapshotNotFound reports a missing snapshot for a Scan
	// that must not fall back to current editable policy.
	ErrScanBlacklistSnapshotNotFound = errors.New("scan blacklist snapshot not found")
	// ErrScanBlacklistSnapshotDataIntegrity reports persisted snapshot bytes that
	// cannot safely become an execution-input matcher.
	ErrScanBlacklistSnapshotDataIntegrity = errors.New("scan blacklist snapshot data integrity failure")
)
