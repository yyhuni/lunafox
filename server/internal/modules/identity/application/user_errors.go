package application

import (
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

var (
	ErrUserNotFound    = identitydomain.ErrUserNotFound
	ErrUsernameExists  = identitydomain.ErrUsernameExists
	ErrInvalidPassword = identitydomain.ErrInvalidPassword
	ErrUserDisabled    = errors.New("user account is disabled")
)
