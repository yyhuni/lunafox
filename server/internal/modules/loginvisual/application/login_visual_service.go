package application

import (
	"context"
	"fmt"
	"io"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
)

// Service keeps draft state private until Publish succeeds. The public handler
// only asks it for Published media, so a stored draft cannot become routable.
type Service struct {
	settings        SettingsStore
	authorizer      ActiveSuperuserAuthorizer
	discoverability DiscoverabilityStore
	media           MediaStore
}

func NewService(settings SettingsStore, authorizer ActiveSuperuserAuthorizer, discoverability DiscoverabilityStore, media MediaStore) *Service {
	if settings == nil || authorizer == nil || discoverability == nil || media == nil {
		panic("login visual service dependencies are required")
	}
	return &Service{settings: settings, authorizer: authorizer, discoverability: discoverability, media: media}
}

// IsDiscoverabilityUnlocked intentionally does not use active-superuser
// authorization. This is an account-owned cosmetic state, never media access.
func (service *Service) IsDiscoverabilityUnlocked(ctx context.Context, userID int) (bool, error) {
	if userID <= 0 {
		return false, ErrPermissionDenied
	}
	return service.discoverability.IsUnlocked(ctx, userID)
}

// UnlockDiscoverability is idempotent so repeated GitHub-link activations and
// retries cannot create multiple records for the same authenticated account.
func (service *Service) UnlockDiscoverability(ctx context.Context, userID int) error {
	if userID <= 0 {
		return ErrPermissionDenied
	}
	return service.discoverability.Unlock(ctx, userID)
}

func (service *Service) GetSettings(ctx context.Context, userID int) (domain.Settings, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Settings{}, err
	}
	return service.settings.Get(ctx)
}

func (service *Service) Upload(ctx context.Context, userID int, contents []byte) (domain.Settings, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Settings{}, err
	}
	media, err := service.media.Save(ctx, contents)
	if err != nil {
		return domain.Settings{}, err
	}
	settings, err := service.settings.SaveDraft(ctx, media)
	if err != nil {
		_ = service.media.Delete(ctx, media)
		return domain.Settings{}, fmt.Errorf("save login visual draft: %w", err)
	}
	return settings, nil
}

func (service *Service) Publish(ctx context.Context, userID int) (domain.Settings, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Settings{}, err
	}
	return service.settings.PublishDraft(ctx)
}

func (service *Service) RestoreDefault(ctx context.Context, userID int) (domain.Settings, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Settings{}, err
	}
	return service.settings.RestoreDefault(ctx)
}

func (service *Service) Public(ctx context.Context) (domain.PublicVisual, error) {
	settings, err := service.settings.Get(ctx)
	if err != nil || settings.Published == nil {
		return domain.PublicVisual{}, nil
	}
	media := settings.Published
	reader, err := service.media.Open(ctx, *media, false)
	if err != nil {
		return domain.PublicVisual{}, nil
	}
	_ = reader.Close()
	result := domain.PublicVisual{HasVisual: true, Kind: media.Kind, MediaURL: "/v1/loginVisual/current/media"}
	if media.Kind == domain.MediaKindVideo {
		result.PosterURL = "/v1/loginVisual/current/poster"
	}
	return result, nil
}

func (service *Service) OpenPublished(ctx context.Context, poster bool) (domain.Media, io.ReadCloser, error) {
	settings, err := service.settings.Get(ctx)
	if err != nil || settings.Published == nil {
		return domain.Media{}, nil, ErrMediaUnavailable
	}
	reader, err := service.media.Open(ctx, *settings.Published, poster)
	if err != nil {
		return domain.Media{}, nil, ErrMediaUnavailable
	}
	return *settings.Published, reader, nil
}

func (service *Service) OpenPreview(ctx context.Context, userID int) (domain.Media, io.ReadCloser, error) {
	if err := service.authorize(ctx, userID); err != nil {
		return domain.Media{}, nil, err
	}
	settings, err := service.settings.Get(ctx)
	if err != nil {
		return domain.Media{}, nil, err
	}
	media := settings.Draft
	if media == nil {
		media = settings.Published
	}
	if media == nil {
		return domain.Media{}, nil, ErrMediaUnavailable
	}
	reader, err := service.media.Open(ctx, *media, false)
	if err != nil {
		return domain.Media{}, nil, ErrMediaUnavailable
	}
	return *media, reader, nil
}

func (service *Service) authorize(ctx context.Context, userID int) error {
	if userID <= 0 {
		return ErrPermissionDenied
	}
	allowed, err := service.authorizer.IsActiveSuperuser(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve login visual authority: %w", err)
	}
	if !allowed {
		return ErrPermissionDenied
	}
	return nil
}
