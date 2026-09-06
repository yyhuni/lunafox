package domain

import "time"

// MCPKey is the non-secret persistence projection for one active user key.
// The plaintext credential intentionally has no field in this type.
type MCPKey struct {
	ID        int
	UserID    int
	KeyDigest string
	CreatedAt time.Time
	UpdatedAt time.Time
}
