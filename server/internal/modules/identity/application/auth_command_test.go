package application

import (
	"context"
	"errors"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/auth"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"gorm.io/gorm"
)

type authUserStoreStub struct {
	byID       map[int]*identitydomain.User
	byUsername map[string]*identitydomain.User
	errByID    error
	errByName  error
}

func (stub *authUserStoreStub) GetAuthUserByID(id int) (*identitydomain.User, error) {
	if stub.errByID != nil {
		return nil, stub.errByID
	}
	user, ok := stub.byID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

func (stub *authUserStoreStub) FindAuthUserByUsername(username string) (*identitydomain.User, error) {
	if stub.errByName != nil {
		return nil, stub.errByName
	}
	user, ok := stub.byUsername[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

type passwordVerifierStub struct {
	verify bool
}

func (stub passwordVerifierStub) VerifyPassword(password, hashed string) bool {
	_ = password
	_ = hashed
	return stub.verify
}

type tokenProviderStub struct {
	pair       *auth.TokenPair
	claims     *auth.Claims
	access     string
	expiresIn  int64
	errPair    error
	errVerify  error
	errAccess  error
	lastUserID int
	lastName   string
}

func (stub *tokenProviderStub) GenerateTokenPair(userID int, username string) (*auth.TokenPair, error) {
	stub.lastUserID = userID
	stub.lastName = username
	if stub.errPair != nil {
		return nil, stub.errPair
	}
	return stub.pair, nil
}

func (stub *tokenProviderStub) ValidateToken(token string) (*auth.Claims, error) {
	_ = token
	if stub.errVerify != nil {
		return nil, stub.errVerify
	}
	return stub.claims, nil
}

func (stub *tokenProviderStub) GenerateAccessToken(userID int, username string) (string, int64, error) {
	stub.lastUserID = userID
	stub.lastName = username
	if stub.errAccess != nil {
		return "", 0, stub.errAccess
	}
	return stub.access, stub.expiresIn, nil
}

func TestAuthCommandServiceLogin(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{byUsername: map[string]*identitydomain.User{}},
			passwordVerifierStub{verify: true},
			&tokenProviderStub{},
		)

		_, err := service.Login(context.Background(), "alice", "pass")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("user disabled", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{byUsername: map[string]*identitydomain.User{"alice": {Username: "alice", IsActive: false}}},
			passwordVerifierStub{verify: true},
			&tokenProviderStub{},
		)

		_, err := service.Login(context.Background(), "alice", "pass")
		if !errors.Is(err, ErrUserDisabled) {
			t.Fatalf("expected ErrUserDisabled, got %v", err)
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{byUsername: map[string]*identitydomain.User{"alice": {Username: "alice", IsActive: true, Password: "hash"}}},
			passwordVerifierStub{verify: false},
			&tokenProviderStub{},
		)

		_, err := service.Login(context.Background(), "alice", "bad")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("login succeeds", func(t *testing.T) {
		tokens := &tokenProviderStub{
			pair: &auth.TokenPair{AccessToken: "a", RefreshToken: "r", ExpiresIn: 3600},
		}
		service := NewAuthCommandService(
			&authUserStoreStub{byUsername: map[string]*identitydomain.User{"alice": {ID: 7, Username: "alice", Email: "a@x.com", IsActive: true, Password: "hash"}}},
			passwordVerifierStub{verify: true},
			tokens,
		)

		result, err := service.Login(context.Background(), "alice", "pass")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		if result.AccessToken != "a" || result.User.ID != 7 {
			t.Fatalf("unexpected login result: %+v", result)
		}
	})
}

func TestAuthCommandServiceRefreshAndCurrentUser(t *testing.T) {
	t.Run("invalid refresh token", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{},
			passwordVerifierStub{verify: true},
			&tokenProviderStub{errVerify: auth.ErrInvalidToken},
		)

		_, err := service.RefreshToken(context.Background(), "bad")
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
		}
	})

	t.Run("refresh succeeds", func(t *testing.T) {
		tokens := &tokenProviderStub{
			claims:    &auth.Claims{UserID: 9, Username: "bob"},
			access:    "new-access",
			expiresIn: 1800,
		}
		service := NewAuthCommandService(&authUserStoreStub{}, passwordVerifierStub{verify: true}, tokens)

		result, err := service.RefreshToken(context.Background(), "ok")
		if err != nil {
			t.Fatalf("refresh failed: %v", err)
		}
		if result.AccessToken != "new-access" || result.ExpiresIn != 1800 {
			t.Fatalf("unexpected refresh result: %+v", result)
		}
	})

	t.Run("current user succeeds", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{byID: map[int]*identitydomain.User{2: {ID: 2, Username: "u2", Email: "u2@x.com"}}},
			passwordVerifierStub{verify: true},
			&tokenProviderStub{},
		)

		user, err := service.GetCurrentUser(context.Background(), 2)
		if err != nil {
			t.Fatalf("get current user failed: %v", err)
		}
		if user.ID != 2 || user.Username != "u2" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})
}
