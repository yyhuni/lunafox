package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/auth"
	"gorm.io/gorm"
)

type adminPasswordResetStoreStub struct {
	hashedPassword string
	err            error
	calls          int
}

func (stub *adminPasswordResetStoreStub) ResetAdminPassword(context.Context, string) error {
	stub.calls++
	return stub.err
}

type captureAdminPasswordResetStore struct {
	adminPasswordResetStoreStub
	hashedPassword string
}

func (stub *captureAdminPasswordResetStore) ResetAdminPassword(ctx context.Context, hashedPassword string) error {
	stub.calls++
	stub.hashedPassword = hashedPassword
	return stub.err
}

type passwordGeneratorStub struct {
	password string
	err      error
}

func (stub passwordGeneratorStub) Generate() (string, error) {
	return stub.password, stub.err
}

type adminPasswordResetHasherStub struct {
	hash string
	err  error
}

func (stub adminPasswordResetHasherStub) HashPassword(string) (string, error) {
	return stub.hash, stub.err
}

func (adminPasswordResetHasherStub) VerifyPassword(string, string) bool {
	return false
}

func TestAdminPasswordResetService(t *testing.T) {
	t.Run("success returns plaintext only after the store succeeds", func(t *testing.T) {
		const password = "operator-delivery-only"
		store := &captureAdminPasswordResetStore{}
		service := NewAdminPasswordResetService(store, NewAuthPasswordHasher(), passwordGeneratorStub{password: password})

		result, err := service.Reset(context.Background())
		if err != nil {
			t.Fatalf("reset admin password: %v", err)
		}
		if result.Password != password {
			t.Fatalf("result password = %q, want generated password", result.Password)
		}
		if store.calls != 1 || !auth.VerifyPassword(password, store.hashedPassword) {
			t.Fatalf("store did not receive a bcrypt hash: calls=%d hash=%q", store.calls, store.hashedPassword)
		}
	})

	t.Run("missing admin is a typed fast-fail", func(t *testing.T) {
		store := &adminPasswordResetStoreStub{err: gorm.ErrRecordNotFound}
		service := NewAdminPasswordResetService(store, adminPasswordResetHasherStub{hash: "hash"}, passwordGeneratorStub{password: "unused"})

		result, err := service.Reset(context.Background())
		if result != nil || !errors.Is(err, ErrAdminNotFound) {
			t.Fatalf("missing admin result=%+v err=%v, want typed fast-fail", result, err)
		}
		if store.calls != 1 {
			t.Fatalf("reset store calls = %d, want 1", store.calls)
		}
	})

	t.Run("store failure does not return the generated password", func(t *testing.T) {
		store := &adminPasswordResetStoreStub{err: errors.New("transaction failed")}
		service := NewAdminPasswordResetService(store, adminPasswordResetHasherStub{hash: "hash"}, passwordGeneratorStub{password: "must-not-return"})

		result, err := service.Reset(context.Background())
		if result != nil || err == nil {
			t.Fatalf("failed reset result=%+v err=%v, want no result and an error", result, err)
		}
	})

	t.Run("generator failure does not call the store", func(t *testing.T) {
		store := &adminPasswordResetStoreStub{}
		service := NewAdminPasswordResetService(store, adminPasswordResetHasherStub{hash: "hash"}, passwordGeneratorStub{err: errors.New("entropy unavailable")})

		if _, err := service.Reset(context.Background()); err == nil {
			t.Fatal("expected generator error")
		}
		if store.calls != 0 {
			t.Fatalf("store calls = %d, want 0", store.calls)
		}
	})
}

func TestCryptoPasswordGenerator(t *testing.T) {
	generator := NewCryptoPasswordGenerator()
	first, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate first password: %v", err)
	}
	second, err := generator.Generate()
	if err != nil {
		t.Fatalf("generate second password: %v", err)
	}
	if len(first) != 32 || len(second) != 32 {
		t.Fatalf("password lengths = %d and %d, want 32", len(first), len(second))
	}
	if first == second {
		t.Fatal("cryptographic password generator returned the same value twice")
	}
}
