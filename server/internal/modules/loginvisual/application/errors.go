package application

import "errors"

var (
	ErrPermissionDenied = errors.New("login visual permission denied")
	ErrNoDraft          = errors.New("login visual draft does not exist")
	ErrInvalidMedia     = errors.New("invalid login visual media")
	ErrMediaUnavailable = errors.New("login visual media unavailable")
)
