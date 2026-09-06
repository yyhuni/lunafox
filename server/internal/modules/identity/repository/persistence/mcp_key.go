package model

import "time"

// MCPKey is the database projection for a single active MCP credential.
// KeyDigest is intentionally the only credential material stored here.
type MCPKey struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int       `gorm:"column:user_id;not null;uniqueIndex:idx_mcp_key_user" json:"userId"`
	KeyDigest string    `gorm:"column:key_digest;size:64;not null;uniqueIndex:idx_mcp_key_digest" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (MCPKey) TableName() string {
	return "mcp_key"
}
