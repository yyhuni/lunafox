package application

import (
	"context"
	"time"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

// MCPKeyStore is the persistence boundary for MCP credentials.
type MCPKeyStore interface {
	GetByUserID(ctx context.Context, userID int) (*identitydomain.MCPKey, error)
	GetByDigest(ctx context.Context, digest string) (*identitydomain.MCPKey, error)
	Replace(ctx context.Context, userID int, digest string) (*identitydomain.MCPKey, error)
}

// MCPKeySecretGenerator creates a plaintext secret and its persistence digest.
type MCPKeySecretGenerator interface {
	Generate() (secret, digest string, err error)
	Digest(secret string) string
}

// MCPKeyStatus is safe to return after the one-time plaintext response expires.
type MCPKeyStatus struct {
	Configured bool
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

// MCPKeyGeneration is returned only by a successful generate/rotate action.
type MCPKeyGeneration struct {
	Secret     string
	Configured bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
