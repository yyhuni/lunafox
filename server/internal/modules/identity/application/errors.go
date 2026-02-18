package application

import (
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

var (
	ErrUserNotFound    = identitydomain.ErrUserNotFound
	ErrUsernameExists  = identitydomain.ErrUsernameExists
	ErrInvalidPassword = identitydomain.ErrInvalidPassword

	ErrOrganizationNotFound = identitydomain.ErrOrganizationNotFound
	ErrOrganizationExists   = identitydomain.ErrOrganizationNameExist
	ErrTargetNotFound       = identitydomain.ErrTargetNotFound

	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUserDisabled        = errors.New("user account is disabled")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)
