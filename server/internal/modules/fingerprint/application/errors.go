package application

import "errors"

var (
	ErrFingerprintNotFound          = errors.New("fingerprint not found")
	ErrInvalidFingerprintPageToken  = errors.New("invalid fingerprint pageToken")
	ErrUnsupportedFingerprintFilter = errors.New("unsupported fingerprint filter")
	ErrUnsupportedFingerprintFacet  = errors.New("unsupported fingerprint facet")
	ErrUnsupportedFingerprintOrder  = errors.New("unsupported fingerprint orderBy")
	ErrInvalidBatchDelete           = errors.New("invalid fingerprint batch delete request")
	// ErrFingerprintArtifactServiceUnavailable is deliberately explicit: current
	// exports must use the shared immutable artifact service, never rebuild bytes
	// in memory as an unnoticed fallback.
	ErrFingerprintArtifactServiceUnavailable = errors.New("fingerprint artifact service is not configured")
)

const MaxBatchDeleteNames = 1000
