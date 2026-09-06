package application

import (
	"context"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

// ActiveSuperuserAuthorizer resolves the authoritative server-side identity
// state because the JWT intentionally does not carry a mutable role claim.
type ActiveSuperuserAuthorizer interface {
	IsActiveSuperuser(context.Context, int) (bool, error)
}

// DestinationSettingsService guards complete installation credentials behind
// the existing active superuser boundary without adding role-management UI.
type DestinationSettingsService struct {
	destinations DestinationStore
	authorizer   ActiveSuperuserAuthorizer
}

// NewDestinationSettingsService creates the authorized settings use case.
func NewDestinationSettingsService(destinations DestinationStore, authorizer ActiveSuperuserAuthorizer) *DestinationSettingsService {
	if destinations == nil || authorizer == nil {
		panic("notification destination settings dependencies are required")
	}
	return &DestinationSettingsService{destinations: destinations, authorizer: authorizer}
}

// Get returns complete credential material only after server-side authority.
func (service *DestinationSettingsService) Get(ctx context.Context, userID int, provider domain.Provider) (domain.Destination, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Destination{}, err
	}
	return service.destinations.Get(ctx, provider)
}

// List returns the fixed first-phase providers with their independent state.
func (service *DestinationSettingsService) List(ctx context.Context, userID int) ([]domain.Destination, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return nil, err
	}
	return service.destinations.List(ctx)
}

// Update persists one provider's complete credential and exact allowlist.
func (service *DestinationSettingsService) Update(ctx context.Context, userID int, destination domain.Destination) (domain.Destination, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Destination{}, err
	}
	if err := domain.ValidateDestination(destination); err != nil {
		return domain.Destination{}, err
	}
	return service.destinations.Update(ctx, destination)
}

func (service *DestinationSettingsService) authorize(ctx context.Context, userID int) error {
	if userID <= 0 {
		return ErrNotificationPermissionDenied
	}
	allowed, err := service.authorizer.IsActiveSuperuser(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve notification destination authority: %w", err)
	}
	if !allowed {
		return ErrNotificationPermissionDenied
	}
	return nil
}
