package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	model "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/repository/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DiscoverabilityRepository persists only the account-owned easter-egg state.
// It is intentionally independent from ActiveSuperuserRepository because it
// must never be interpreted as authorization for media management.
type DiscoverabilityRepository struct{ db *gorm.DB }

func NewDiscoverabilityRepository(db *gorm.DB) *DiscoverabilityRepository {
	if db == nil {
		panic("login visual discoverability database is required")
	}
	return &DiscoverabilityRepository{db: db}
}

func (repository *DiscoverabilityRepository) IsUnlocked(ctx context.Context, userID int) (bool, error) {
	if userID <= 0 {
		return false, application.ErrPermissionDenied
	}
	var discovery model.LoginVisualDiscovery
	// A missing row is the normal locked state, so avoid Take's ErrRecordNotFound
	// result being reported as an error by the shared database logger.
	result := repository.db.WithContext(ctx).Where("user_id = ?", userID).Limit(1).Find(&discovery)
	if result.Error != nil {
		return false, fmt.Errorf("read login visual discoverability: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (repository *DiscoverabilityRepository) Unlock(ctx context.Context, userID int) error {
	if userID <= 0 {
		return application.ErrPermissionDenied
	}
	discovery := model.LoginVisualDiscovery{UserID: userID, UnlockedAt: time.Now().UTC()}
	if err := repository.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&discovery).Error; err != nil {
		return fmt.Errorf("persist login visual discoverability: %w", err)
	}
	return nil
}

var _ application.DiscoverabilityStore = (*DiscoverabilityRepository)(nil)
