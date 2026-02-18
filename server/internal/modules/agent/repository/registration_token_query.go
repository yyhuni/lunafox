package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

// FindValid returns a non-expired token by value.
func (r *registrationTokenRepository) FindValid(ctx context.Context, token string, now time.Time) (*agentdomain.RegistrationToken, error) {
	var item model.RegistrationToken
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, now).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return modelTokenToDomain(&item), nil
}
