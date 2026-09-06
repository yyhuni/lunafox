package idempotency

import (
	"time"

	"gorm.io/datatypes"
)

// ReplayRecord is the database-neutral replay ledger shared by MCP business
// mutations. The primary request ID intentionally spans actions: a caller
// cannot accidentally reuse one key for unrelated side effects.
type ReplayRecord struct {
	RequestID          string         `gorm:"column:request_id;primaryKey;size:128"`
	Action             string         `gorm:"column:action;size:80;not null"`
	RequestFingerprint string         `gorm:"column:request_fingerprint;size:64;not null"`
	Response           datatypes.JSON `gorm:"column:response;type:jsonb;not null"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime"`
	ExpiresAt          time.Time      `gorm:"column:expires_at;not null;index"`
}

// TableName keeps replay storage independent from any one application module.
func (ReplayRecord) TableName() string { return "mcp_request_replay" }
