package domain

import (
	"encoding/json"
	"time"
)

// QueryFields are the FingerprintHub payload projections used for list search,
// filtering, and ordering. Payload remains the native export source.
type QueryFields struct {
	DisplayName string
	Severity    *string
	NativeID    string
}

// ImportedRecord is a fully validated native rule ready for transactional
// persistence. IdentityKey and ContentHash are backend-derived only.
type ImportedRecord struct {
	Library     Library
	SourceIndex int
	IdentityKey string
	ContentHash []byte
	Payload     json.RawMessage
	Fields      QueryFields
}

// PersistedRecord is the internal read model for FingerprintHub records.
// ResourceID only becomes visible as part of CanonicalName at the HTTP edge.
type PersistedRecord struct {
	Library     Library
	ResourceID  string
	IdentityKey string
	ContentHash []byte
	Payload     json.RawMessage
	Fields      QueryFields
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (record PersistedRecord) CanonicalName() string {
	return record.Library.CanonicalName(record.ResourceID)
}

// FingerPrintHubRecord represents one native Nuclei-compatible HTTP template.
type FingerPrintHubRecord struct {
	ID       string
	Name     string
	Severity *string
	Payload  json.RawMessage
}
