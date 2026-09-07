package model

import "time"

// ScanOperation is the durable MCP polling handle for one immutable Scan.
// Lifecycle state is deliberately projected from Scan rather than written as
// a second execution state machine.
type ScanOperation struct {
	ID                 string    `gorm:"column:id;primaryKey;size:36"`
	ScanID             int       `gorm:"column:scan_id;not null;uniqueIndex"`
	TargetID           int       `gorm:"column:target_id;not null;index"`
	RequestID          *string   `gorm:"column:request_id;size:128;index"`
	RequestFingerprint string    `gorm:"column:request_fingerprint;size:64;not null"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ScanOperation) TableName() string { return "scan_operation" }
