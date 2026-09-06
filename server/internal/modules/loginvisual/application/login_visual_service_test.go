package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
)

func TestServiceKeepsDraftPrivateUntilPublished(t *testing.T) {
	t.Parallel()

	media := &fakeMediaStore{saved: domain.Media{ID: "draft-1", Kind: domain.MediaKindImage, ContentType: "image/png", StorageKey: "draft-1"}}
	service := NewService(&fakeSettingsStore{}, fakeAuthorizer{allowed: true}, &fakeDiscoverabilityStore{}, media)
	ctx := context.Background()

	settings, err := service.Upload(ctx, 7, []byte("valid"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if settings.Draft == nil || settings.Published != nil {
		t.Fatalf("upload settings = %#v, want private draft only", settings)
	}

	visual, err := service.Public(ctx)
	if err != nil {
		t.Fatalf("public draft: %v", err)
	}
	if visual.HasVisual || media.openCalls != 0 {
		t.Fatalf("draft became public: visual=%#v opens=%d", visual, media.openCalls)
	}

	settings, err = service.Publish(ctx, 7)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if settings.Draft != nil || settings.Published == nil {
		t.Fatalf("publish settings = %#v, want published media only", settings)
	}

	visual, err = service.Public(ctx)
	if err != nil {
		t.Fatalf("public published: %v", err)
	}
	if !visual.HasVisual || visual.MediaURL == "" || media.closed != 1 {
		t.Fatalf("published visual = %#v, closed=%d", visual, media.closed)
	}

	settings, err = service.RestoreDefault(ctx, 7)
	if err != nil {
		t.Fatalf("restore default: %v", err)
	}
	if settings.Draft != nil || settings.Published != nil {
		t.Fatalf("restore settings = %#v, want built-in state", settings)
	}
}

func TestServiceRejectsNonSuperuserBeforeWritingMedia(t *testing.T) {
	t.Parallel()

	media := &fakeMediaStore{saved: domain.Media{ID: "draft-1"}}
	service := NewService(&fakeSettingsStore{}, fakeAuthorizer{}, &fakeDiscoverabilityStore{}, media)

	_, err := service.Upload(context.Background(), 7, []byte("valid"))
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("upload error = %v, want permission denied", err)
	}
	if media.savedCalls != 0 {
		t.Fatalf("media save calls = %d, want 0", media.savedCalls)
	}
}

func TestServicePersistsDiscoverabilityWithoutGrantingMediaAuthority(t *testing.T) {
	t.Parallel()

	discovery := &fakeDiscoverabilityStore{}
	media := &fakeMediaStore{saved: domain.Media{ID: "draft-1"}}
	service := NewService(&fakeSettingsStore{}, fakeAuthorizer{}, discovery, media)

	unlocked, err := service.IsDiscoverabilityUnlocked(context.Background(), 7)
	if err != nil || unlocked {
		t.Fatalf("initial discoverability = %t, %v; want false, nil", unlocked, err)
	}
	if err := service.UnlockDiscoverability(context.Background(), 7); err != nil {
		t.Fatalf("unlock discoverability: %v", err)
	}
	unlocked, err = service.IsDiscoverabilityUnlocked(context.Background(), 7)
	if err != nil || !unlocked {
		t.Fatalf("unlocked discoverability = %t, %v; want true, nil", unlocked, err)
	}

	if _, err := service.Upload(context.Background(), 7, []byte("valid")); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("upload error = %v, want permission denied", err)
	}
	if media.savedCalls != 0 {
		t.Fatalf("media save calls = %d, want 0", media.savedCalls)
	}
}

func TestServiceRejectsDiscoverabilityWithoutAuthenticatedUser(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeSettingsStore{}, fakeAuthorizer{}, &fakeDiscoverabilityStore{}, &fakeMediaStore{})
	if err := service.UnlockDiscoverability(context.Background(), 0); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("unlock error = %v, want permission denied", err)
	}
}

type fakeAuthorizer struct {
	allowed bool
	err     error
}

type fakeDiscoverabilityStore struct{ unlocked bool }

func (store *fakeDiscoverabilityStore) IsUnlocked(context.Context, int) (bool, error) {
	return store.unlocked, nil
}

func (store *fakeDiscoverabilityStore) Unlock(context.Context, int) error {
	store.unlocked = true
	return nil
}

func (authorizer fakeAuthorizer) IsActiveSuperuser(context.Context, int) (bool, error) {
	return authorizer.allowed, authorizer.err
}

type fakeSettingsStore struct{ settings domain.Settings }

func (store *fakeSettingsStore) Get(context.Context) (domain.Settings, error) {
	return store.settings, nil
}

func (store *fakeSettingsStore) SaveDraft(_ context.Context, media domain.Media) (domain.Settings, error) {
	store.settings.Draft = &media
	return store.settings, nil
}

func (store *fakeSettingsStore) PublishDraft(context.Context) (domain.Settings, error) {
	if store.settings.Draft == nil {
		return domain.Settings{}, ErrNoDraft
	}
	store.settings.Published = store.settings.Draft
	store.settings.Draft = nil
	return store.settings, nil
}

func (store *fakeSettingsStore) RestoreDefault(context.Context) (domain.Settings, error) {
	store.settings = domain.Settings{}
	return store.settings, nil
}

type fakeMediaStore struct {
	saved      domain.Media
	savedCalls int
	openCalls  int
	closed     int
}

func (store *fakeMediaStore) Save(context.Context, []byte) (domain.Media, error) {
	store.savedCalls++
	return store.saved, nil
}

func (store *fakeMediaStore) Open(context.Context, domain.Media, bool) (io.ReadCloser, error) {
	store.openCalls++
	return &countingReadCloser{Reader: bytes.NewReader(nil), closed: &store.closed}, nil
}

func (store *fakeMediaStore) Delete(context.Context, domain.Media) error { return nil }

type countingReadCloser struct {
	io.Reader
	closed *int
}

func (reader *countingReadCloser) Close() error {
	(*reader.closed)++
	return nil
}
