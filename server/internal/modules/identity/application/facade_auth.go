package application

import (
	"context"
	"errors"
)

var ErrAuthUserNotFound = ErrUserNotFound

// AuthFacade handles authentication workflows.
type AuthFacade struct {
	commandService *AuthCommandService
}

// NewAuthFacade creates a new auth service.
func NewAuthFacade(commandService *AuthCommandService) *AuthFacade {
	return &AuthFacade{commandService: commandService}
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
		return nil, mapAuthUserBoundaryError(err)
	}
	return result, nil
}
