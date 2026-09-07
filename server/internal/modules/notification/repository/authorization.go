package repository

import (
	"context"
	"fmt"

	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"gorm.io/gorm"
)

// ActiveSuperuserRepository reads only the authoritative auth_user fields
// required to protect complete installation Webhook credentials.
type ActiveSuperuserRepository struct {
	db *gorm.DB
}

// NewActiveSuperuserRepository creates the settings authorization reader.
func NewActiveSuperuserRepository(db *gorm.DB) *ActiveSuperuserRepository {
	if db == nil {
		panic("notification authorization database is required")
	}
	return &ActiveSuperuserRepository{db: db}
}

// IsActiveSuperuser never trusts a caller-provided role or stale JWT claim.
func (repository *ActiveSuperuserRepository) IsActiveSuperuser(ctx context.Context, userID int) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var count int64
	if err := repository.db.WithContext(ctx).Table("auth_user").
		Where("id = ? AND is_active = ? AND is_superuser = ?", userID, true, true).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("read notification settings authority: %w", err)
	}
	return count == 1, nil
}

var _ notificationapp.ActiveSuperuserAuthorizer = (*ActiveSuperuserRepository)(nil)
