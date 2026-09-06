package repository

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"gorm.io/gorm"
)

type ActiveSuperuserRepository struct{ db *gorm.DB }

func NewActiveSuperuserRepository(db *gorm.DB) *ActiveSuperuserRepository {
	if db == nil {
		panic("login visual authorization database is required")
	}
	return &ActiveSuperuserRepository{db: db}
}

func (repository *ActiveSuperuserRepository) IsActiveSuperuser(ctx context.Context, userID int) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var count int64
	if err := repository.db.WithContext(ctx).Table("auth_user").Where("id = ? AND is_active = ? AND is_superuser = ?", userID, true, true).Count(&count).Error; err != nil {
		return false, fmt.Errorf("read login visual settings authority: %w", err)
	}
	return count == 1, nil
}

var _ application.ActiveSuperuserAuthorizer = (*ActiveSuperuserRepository)(nil)
