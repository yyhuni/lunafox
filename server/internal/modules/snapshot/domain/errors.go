package domain

import "errors"

var (
	ErrSnapshotScanNotFound          = errors.New("scan not found")
	ErrSnapshotTargetMismatch        = errors.New("targetId does not match scan's target")
	ErrVulnerabilitySnapshotNotFound = errors.New("vulnerability snapshot not found")
)
