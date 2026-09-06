// Package model contains GORM mappings for the FingerprintHub corpus and its
// current-artifact lifecycle.
package model

import (
	"time"

	"gorm.io/datatypes"
)

type FingerprintRecordFields struct {
	ResourceID  string         `gorm:"column:resource_id;type:uuid;primaryKey"`
	IdentityKey string         `gorm:"column:identity_key;not null;uniqueIndex"`
	ContentHash []byte         `gorm:"column:content_hash;type:bytea;not null"`
	Payload     datatypes.JSON `gorm:"column:payload;type:jsonb;not null"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

type FingerprintLibraryState struct {
	Library          string `gorm:"column:library;primaryKey"`
	SourceGeneration int64  `gorm:"column:source_generation;not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (FingerprintLibraryState) TableName() string { return "fingerprint_library_state" }

type FingerprintLibraryArtifact struct {
	ID               uint64 `gorm:"column:id;primaryKey"`
	Library          string `gorm:"column:library;not null"`
	SourceGeneration int64  `gorm:"column:source_generation;not null"`
	SHA256Digest     string `gorm:"column:sha256_digest;not null"`
	SizeBytes        int64  `gorm:"column:size_bytes;not null"`
	RecordCount      int64  `gorm:"column:record_count;not null"`
	Filename         string `gorm:"column:filename;not null"`
	ContentType      string `gorm:"column:content_type;not null"`
	StorageKey       string `gorm:"column:storage_key;not null"`
	CreatedAt        time.Time
}

func (FingerprintLibraryArtifact) TableName() string { return "fingerprint_library_artifact" }

// FingerPrintHubFingerprint is the persisted projection of a native template.
type FingerPrintHubFingerprint struct {
	FingerprintRecordFields
	FingerprintID string  `gorm:"column:fingerprint_id;not null"`
	Name          string  `gorm:"column:name;not null"`
	Severity      *string `gorm:"column:severity"`
}

func (FingerPrintHubFingerprint) TableName() string { return "fingerprint_fingerprinthub" }
