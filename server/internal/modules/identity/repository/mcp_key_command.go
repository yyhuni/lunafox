package repository

import (
	"context"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Replace atomically replaces the user's active digest and returns its record.
// The user uniqueness constraint makes concurrent rotations converge on one row.
func (r *MCPKeyRepository) Replace(ctx context.Context, userID int, digest string) (*identitydomain.MCPKey, error) {
	var key model.MCPKey
	err := dbtx.Resolve(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{"key_digest": digest, "updated_at": gorm.Expr("CURRENT_TIMESTAMP")}),
		}).Create(&model.MCPKey{UserID: userID, KeyDigest: digest}).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Where("user_id = ?", userID).First(&key).Error
	})
	if err != nil {
		return nil, err
	}
	return mcpKeyModelToDomain(&key), nil
}
