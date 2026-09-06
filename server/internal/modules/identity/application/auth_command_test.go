package application

import (
	"context"
	"errors"
	"testing"
	"time"

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
	pair        *auth.TokenPair
	claims      *auth.Claims
	access      string
	expiresIn   int64
	errPair     error
	errVerify   error
	errAccess   error
	lastUserID  int
	lastName    string
	lastVersion int
}

func (stub *tokenProviderStub) GenerateTokenPair(userID int, username string, tokenVersion int) (*auth.TokenPair, error) {
	stub.lastUserID = userID
	stub.lastName = username
	stub.lastVersion = tokenVersion
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

func (stub *tokenProviderStub) GenerateAccessToken(userID int, username string, tokenVersion int) (string, int64, error) {
	stub.lastUserID = userID
	stub.lastName = username
	stub.lastVersion = tokenVersion
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
			&authUserStoreStub{byUsername: map[string]*identitydomain.User{"alice": {ID: 7, Username: "alice", Email: "a@x.com", IsActive: true, Password: "hash", TokenVersion: 3}}},
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
		if tokens.lastVersion != 3 {
			t.Fatalf("login token version = %d, want 3", tokens.lastVersion)
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
			claims:    &auth.Claims{UserID: 9, Username: "bob", TokenVersion: 4},
			access:    "new-access",
			expiresIn: 1800,
		}
		service := NewAuthCommandService(&authUserStoreStub{byID: map[int]*identitydomain.User{9: {ID: 9, Username: "bob", TokenVersion: 4}}}, passwordVerifierStub{verify: true}, tokens)

		result, err := service.RefreshToken(context.Background(), "ok")
		if err != nil {
			t.Fatalf("refresh failed: %v", err)
		}
		if result.AccessToken != "new-access" || result.ExpiresIn != 1800 {
			t.Fatalf("unexpected refresh result: %+v", result)
		}
		if tokens.lastVersion != 4 {
			t.Fatalf("refresh access token version = %d, want 4", tokens.lastVersion)
		}
	})

	t.Run("refresh rejects a stale token version", func(t *testing.T) {
		service := NewAuthCommandService(
			&authUserStoreStub{byID: map[int]*identitydomain.User{9: {ID: 9, Username: "bob", TokenVersion: 5}}},
			passwordVerifierStub{verify: true},
			&tokenProviderStub{claims: &auth.Claims{UserID: 9, Username: "bob", TokenVersion: 4}},
		)

		_, err := service.RefreshToken(context.Background(), "stale")
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("expected stale version to be invalid refresh token, got %v", err)
		}
	})

	t.Run("legacy refresh token remains valid at version zero", func(t *testing.T) {
		tokens := &tokenProviderStub{
			claims:    &auth.Claims{UserID: 9, Username: "bob"},
			access:    "new-access",
			expiresIn: 1800,
		}
		service := NewAuthCommandService(&authUserStoreStub{byID: map[int]*identitydomain.User{9: {ID: 9, Username: "bob", TokenVersion: 0}}}, passwordVerifierStub{verify: true}, tokens)

		if _, err := service.RefreshToken(context.Background(), "legacy"); err != nil {
			t.Fatalf("legacy refresh failed: %v", err)
		}
		if tokens.lastVersion != 0 {
			t.Fatalf("legacy refresh version = %d, want 0", tokens.lastVersion)
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

func TestAuthFacade(t *testing.T) {
	newJWTManager := func() *auth.JWTManager {
		return auth.NewJWTManager("test-secret-key-32-chars-long!!", 15*time.Minute, 7*24*time.Hour)
	}

	t.Run("constructor-backed auth flow succeeds", func(t *testing.T) {
		hashedPassword, err := auth.HashPassword("pass123")
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}

		store := &authUserStoreStub{
			byID: map[int]*identitydomain.User{
				7: {
					ID:       7,
					Username: "alice",
					Email:    "alice@example.com",
					IsActive: true,
					Password: hashedPassword,
				},
			},
			byUsername: map[string]*identitydomain.User{
				"alice": {
					ID:       7,
					Username: "alice",
					Email:    "alice@example.com",
					IsActive: true,
					Password: hashedPassword,
				},
			},
		}
		facade := NewAuthFacade(NewAuthCommandService(store, NewAuthPasswordVerifier(), newJWTManager()))

		loginResult, err := facade.Login("alice", "pass123")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
		if loginResult.AccessToken == "" || loginResult.RefreshToken == "" {
			t.Fatalf("expected token pair, got %+v", loginResult)
		}
		if loginResult.User.ID != 7 || loginResult.User.Username != "alice" {
			t.Fatalf("unexpected login user: %+v", loginResult.User)
		}

		refreshResult, err := facade.RefreshToken(loginResult.RefreshToken)
		if err != nil {
			t.Fatalf("refresh failed: %v", err)
		}
		if refreshResult.AccessToken == "" || refreshResult.ExpiresIn != 900 {
			t.Fatalf("unexpected refresh result: %+v", refreshResult)
		}

		currentUser, err := facade.GetCurrentUser(7)
		if err != nil {
			t.Fatalf("get current user failed: %v", err)
		}
		if currentUser.ID != 7 || currentUser.Email != "alice@example.com" {
			t.Fatalf("unexpected current user: %+v", currentUser)
		}
	})

	t.Run("login maps missing user to invalid credentials", func(t *testing.T) {
		facade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{byUsername: map[string]*identitydomain.User{}}, NewAuthPasswordVerifier(), newJWTManager()))

		_, err := facade.Login("missing", "pass123")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("login maps disabled user", func(t *testing.T) {
		facade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{
			byUsername: map[string]*identitydomain.User{
				"alice": {Username: "alice", IsActive: false, Password: "unused"},
			},
		}, NewAuthPasswordVerifier(), newJWTManager()))

		_, err := facade.Login("alice", "pass123")
		if !errors.Is(err, ErrUserDisabled) {
			t.Fatalf("expected ErrUserDisabled, got %v", err)
		}
	})

	t.Run("refresh maps invalid token", func(t *testing.T) {
		facade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{}, NewAuthPasswordVerifier(), newJWTManager()))

		_, err := facade.RefreshToken("invalid-token")
		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
		}
	})

	t.Run("current user maps record not found", func(t *testing.T) {
		facade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{byID: map[int]*identitydomain.User{}}, NewAuthPasswordVerifier(), newJWTManager()))

		_, err := facade.GetCurrentUser(99)
		if !errors.Is(err, ErrAuthUserNotFound) {
			t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
		}
	})

	t.Run("non sentinel upstream errors are passed through", func(t *testing.T) {
		loginStoreErr := errors.New("user store unavailable")
		loginTokenErr := errors.New("token provider unavailable")
		refreshValidateErr := errors.New("refresh backend unavailable")
		refreshAccessErr := errors.New("access token unavailable")
		currentUserErr := errors.New("user lookup unavailable")

		facade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{errByName: loginStoreErr}, NewAuthPasswordVerifier(), newJWTManager()))
		_, err := facade.Login("alice", "pass123")
		if !errors.Is(err, loginStoreErr) {
			t.Fatalf("expected login store error passthrough, got %v", err)
		}

		tokenFacade := &AuthFacade{
			commandService: NewAuthCommandService(
				&authUserStoreStub{
					byUsername: map[string]*identitydomain.User{
						"alice": {ID: 7, Username: "alice", IsActive: true, Password: "hash"},
					},
				},
				passwordVerifierStub{verify: true},
				&tokenProviderStub{errPair: loginTokenErr},
			),
		}
		_, err = tokenFacade.Login("alice", "pass123")
		if !errors.Is(err, loginTokenErr) {
			t.Fatalf("expected token pair error passthrough, got %v", err)
		}

		refreshFacade := &AuthFacade{
			commandService: NewAuthCommandService(
				&authUserStoreStub{},
				passwordVerifierStub{verify: true},
				&tokenProviderStub{errVerify: refreshValidateErr},
			),
		}
		_, err = refreshFacade.RefreshToken("refresh-token")
		if !errors.Is(err, refreshValidateErr) {
			t.Fatalf("expected refresh validate error passthrough, got %v", err)
		}

		accessFacade := &AuthFacade{
			commandService: NewAuthCommandService(
				&authUserStoreStub{byID: map[int]*identitydomain.User{8: {ID: 8, Username: "bob", TokenVersion: 0}}},
				passwordVerifierStub{verify: true},
				&tokenProviderStub{
					claims:    &auth.Claims{UserID: 8, Username: "bob"},
					errAccess: refreshAccessErr,
				},
			),
		}
		_, err = accessFacade.RefreshToken("refresh-token")
		if !errors.Is(err, refreshAccessErr) {
			t.Fatalf("expected access token error passthrough, got %v", err)
		}

		currentUserFacade := NewAuthFacade(NewAuthCommandService(&authUserStoreStub{errByID: currentUserErr}, NewAuthPasswordVerifier(), newJWTManager()))
		_, err = currentUserFacade.GetCurrentUser(7)
		if !errors.Is(err, currentUserErr) {
			t.Fatalf("expected current user error passthrough, got %v", err)
		}
	})
}
