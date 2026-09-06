package application

import (
	"context"
	"errors"
	"testing"
	"time"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

type lifecycleKeyStoreStub struct {
	byUser   map[int]*identitydomain.MCPKey
	byDigest map[string]*identitydomain.MCPKey
	nextID   int
}

func newLifecycleKeyStoreStub() *lifecycleKeyStoreStub {
	return &lifecycleKeyStoreStub{
		byUser:   make(map[int]*identitydomain.MCPKey),
		byDigest: make(map[string]*identitydomain.MCPKey),
		nextID:   1,
	}
}

func (store *lifecycleKeyStoreStub) GetByUserID(ctx context.Context, userID int) (*identitydomain.MCPKey, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, ok := store.byUser[userID]
	if !ok {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	copyKey := *key
	return &copyKey, nil
}

func (store *lifecycleKeyStoreStub) GetByDigest(ctx context.Context, digest string) (*identitydomain.MCPKey, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, ok := store.byDigest[digest]
	if !ok {
		return nil, identitydomain.ErrMCPKeyNotFound
	}
	copyKey := *key
	return &copyKey, nil
}

func (store *lifecycleKeyStoreStub) Replace(ctx context.Context, userID int, digest string) (*identitydomain.MCPKey, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	key, exists := store.byUser[userID]
	if exists {
		delete(store.byDigest, key.KeyDigest)
		key.KeyDigest = digest
		key.UpdatedAt = now
	} else {
		key = &identitydomain.MCPKey{ID: store.nextID, UserID: userID, KeyDigest: digest, CreatedAt: now, UpdatedAt: now}
		store.nextID++
		store.byUser[userID] = key
	}
	store.byDigest[digest] = key
	copyKey := *key
	return &copyKey, nil
}

type lifecycleSecretGeneratorStub struct {
	secrets []string
	index   int
}

func (generator *lifecycleSecretGeneratorStub) Generate() (string, string, error) {
	if generator.index >= len(generator.secrets) {
		return "", "", errors.New("no test secret available")
	}
	secret := generator.secrets[generator.index]
	generator.index++
	return secret, "digest:" + secret, nil
}

func (generator *lifecycleSecretGeneratorStub) Digest(secret string) string {
	return "digest:" + secret
}

func TestMCPKeyLifecycleStatusGenerateAndRotate(t *testing.T) {
	store := newLifecycleKeyStoreStub()
	generator := &lifecycleSecretGeneratorStub{secrets: []string{"first-secret", "second-secret"}}
	service := NewMCPKeyLifecycleService(store, generator)

	status, err := service.Status(context.Background(), 7)
	if err != nil {
		t.Fatalf("initial status: %v", err)
	}
	if status.Configured || status.CreatedAt != nil || status.UpdatedAt != nil {
		t.Fatalf("unconfigured status leaked metadata: %+v", status)
	}

	first, err := service.Generate(context.Background(), 7)
	if err != nil {
		t.Fatalf("first generation: %v", err)
	}
	if first.Secret != "first-secret" || !first.Configured || first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
		t.Fatalf("unexpected first generation: %+v", first)
	}
	if got := store.byUser[7].KeyDigest; got == first.Secret || got != "digest:first-secret" {
		t.Fatalf("plaintext was persisted or digest mismatch: %q", got)
	}

	status, err = service.Status(context.Background(), 7)
	if err != nil {
		t.Fatalf("configured status: %v", err)
	}
	if !status.Configured || status.CreatedAt == nil || status.UpdatedAt == nil {
		t.Fatalf("configured status missing metadata: %+v", status)
	}

	if _, err := service.Authenticate(context.Background(), first.Secret); err != nil {
		t.Fatalf("active first key did not authenticate: %v", err)
	}
	second, err := service.Generate(context.Background(), 7)
	if err != nil {
		t.Fatalf("rotation: %v", err)
	}
	if second.Secret != "second-secret" || second.CreatedAt.IsZero() || second.UpdatedAt.IsZero() {
		t.Fatalf("unexpected rotated generation: %+v", second)
	}
	if _, err := service.Authenticate(context.Background(), first.Secret); !errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		t.Fatalf("old key authentication error = %v, want ErrMCPKeyNotFound", err)
	}
	if key, err := service.Authenticate(context.Background(), second.Secret); err != nil || key.UserID != 7 {
		t.Fatalf("new key authentication = %+v, %v", key, err)
	}
	if _, err := service.Authenticate(context.Background(), ""); !errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		t.Fatalf("empty key authentication error = %v", err)
	}
}

func TestMCPKeyLifecycleRequiresContextAndGenerator(t *testing.T) {
	service := NewMCPKeyLifecycleService(newLifecycleKeyStoreStub(), &lifecycleSecretGeneratorStub{secrets: []string{"secret"}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Generate(ctx, 1); err == nil {
		t.Fatal("expected cancelled generation to fail before persistence")
	}
	if _, err := service.Authenticate(ctx, "secret"); err == nil {
		t.Fatal("expected cancelled authentication to fail")
	}
}
