package repository

import (
	"context"

	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const administratorUsername = "admin"

// ResetAdminPassword changes the existing administrator credential and revokes
// its issued tokens in one transaction. The row lock serializes concurrent resets.
func (r *UserRepository) ResetAdminPassword(ctx context.Context, hashedPassword string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var admin model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").
			Where("username = ?", administratorUsername).
			First(&admin).Error; err != nil {
			return err
		}

		result := tx.Model(&model.User{}).
			Where("id = ?", admin.ID).
			Updates(map[string]interface{}{
				"password":      hashedPassword,
				"token_version": gorm.Expr("token_version + ?", 1),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
