package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/auth"
)

var ErrAuthUserNotFound = ErrUserNotFound

// AuthFacade handles authentication workflows.
type AuthFacade struct {
	commandService *AuthCommandService
}

// NewAuthFacade creates a new auth service.
func NewAuthFacade(userStore AuthUserStore, jwtManager *auth.JWTManager) *AuthFacade {
	return &AuthFacade{
		commandService: NewAuthCommandService(userStore, authPasswordVerifier{}, jwtManager),
	}
}

func (service *AuthFacade) Login(username, password string) (*LoginResult, error) {
	result, err := service.commandService.Login(context.Background(), username, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, ErrUserDisabled) {
			return nil, ErrUserDisabled
		}
		return nil, err
	}
	return result, nil
}

func (service *AuthFacade) RefreshToken(refreshToken string) (*RefreshResult, error) {
	result, err := service.commandService.RefreshToken(context.Background(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}
	return result, nil
}

func (service *AuthFacade) GetCurrentUser(userID int) (*CurrentUser, error) {
	result, err := service.commandService.GetCurrentUser(context.Background(), userID)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrAuthUserNotFound
		}
		return nil, err
	}
	return result, nil
}

type authPasswordVerifier struct{}

func (authPasswordVerifier) VerifyPassword(password, hashed string) bool {
	return auth.VerifyPassword(password, hashed)
}
