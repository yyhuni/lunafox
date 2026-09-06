package repository

import (
	"context"
	"fmt"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
)

// Create inserts a new registration token.
func (r *registrationTokenRepository) Create(ctx context.Context, token *agentdomain.RegistrationToken) error {
	if token == nil {
		return fmt.Errorf("registration token is required")
	}
	record := domainTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(record).Error; err != nil {
		return err
	}
	token.ID = record.ID
	return nil
}

// DeleteNeverAttributedBefore removes only tokens whose minimum post-expiry retention elapsed.
func (r *registrationTokenRepository) DeleteNeverAttributedBefore(ctx context.Context, expiredBefore time.Time) error {
	return r.db.WithContext(ctx).
		Where("expires_at <= ? AND ever_attributed_at IS NULL", expiredBefore.UTC()).
		Where("NOT EXISTS (SELECT 1 FROM agent WHERE agent.registration_token_id = registration_token.id)").
		Delete(&model.RegistrationToken{}).Error
}
