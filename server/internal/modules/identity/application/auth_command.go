package application

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         LoginUser
}

type LoginUser struct {
	ID       int
	Username string
	Email    string
}

type RefreshResult struct {
	AccessToken string
	ExpiresIn   int64
}

type CurrentUser struct {
	ID       int
	Username string
	Email    string
}

type AuthCommandService struct {
	users    AuthUserStore
	password PasswordVerifier
	tokens   TokenProvider
}

func NewAuthCommandService(users AuthUserStore, password PasswordVerifier, tokens TokenProvider) *AuthCommandService {
	return &AuthCommandService{
		users:    users,
		password: password,
		tokens:   tokens,
	}
}

func (service *AuthCommandService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	_ = ctx

	user, err := service.users.FindAuthUserByUsername(username)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserDisabled
	}

	if !service.password.VerifyPassword(password, user.Password) {
		return nil, ErrInvalidCredentials
	}

	tokenPair, err := service.tokens.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User: LoginUser{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		},
	}, nil
}

func (service *AuthCommandService) RefreshToken(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	_ = ctx

	claims, err := service.tokens.ValidateToken(refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrExpiredToken) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	accessToken, expiresIn, err := service.tokens.GenerateAccessToken(claims.UserID, claims.Username)
	if err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	}, nil
}

func (service *AuthCommandService) GetCurrentUser(ctx context.Context, userID int) (*CurrentUser, error) {
	_ = ctx

	user, err := service.users.GetAuthUserByID(userID)
	if err != nil {
		return nil, err
	}

	return &CurrentUser{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}, nil
}
