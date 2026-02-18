package domain

import "errors"

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUsernameExists        = errors.New("username already exists")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrOrganizationNotFound  = errors.New("organization not found")
	ErrOrganizationNameExist = errors.New("organization name already exists")
	ErrTargetNotFound        = errors.New("one or more target IDs do not exist")
)
