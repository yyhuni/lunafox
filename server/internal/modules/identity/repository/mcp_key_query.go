package repository

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

// GetByUserID returns the active key metadata for a user.
func (r *MCPKeyRepository) GetByUserID(ctx context.Context, userID int) (*identitydomain.MCPKey, error) {
	var key model.MCPKey
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Where("user_id = ?", userID).First(&key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, identitydomain.ErrMCPKeyNotFound
		}
		return nil, err
	}
	return mcpKeyModelToDomain(&key), nil
}

// GetByDigest resolves a bearer credential by its SHA-256 digest.
func (r *MCPKeyRepository) GetByDigest(ctx context.Context, digest string) (*identitydomain.MCPKey, error) {
	var key model.MCPKey
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Where("key_digest = ?", digest).First(&key).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, identitydomain.ErrMCPKeyNotFound
		}
		return nil, err
	}
	return mcpKeyModelToDomain(&key), nil
}

func mcpKeyModelToDomain(key *model.MCPKey) *identitydomain.MCPKey {
	if key == nil {
		return nil
	}
	return &identitydomain.MCPKey{
		ID:        key.ID,
		UserID:    key.UserID,
		KeyDigest: key.KeyDigest,
		CreatedAt: key.CreatedAt,
		UpdatedAt: key.UpdatedAt,
	}
}
