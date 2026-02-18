package repository

import (
	"context"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

// Create inserts a new registration token.
func (r *registrationTokenRepository) Create(ctx context.Context, token *agentdomain.RegistrationToken) error {
	return r.db.WithContext(ctx).Create(domainTokenToModel(token)).Error
}

// DeleteExpired removes all expired registration tokens.
func (r *registrationTokenRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).
		Where("expires_at <= ?", now).
		Delete(&model.RegistrationToken{}).Error
}
