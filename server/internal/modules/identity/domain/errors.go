package domain

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUsernameExists       = errors.New("username already exists")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrOrganizationNotFound = errors.New("organization not found")
	// ErrOrganizationExists is the stable business error for an active-name
	// conflict. The legacy name remains an alias for existing REST callers.
	ErrOrganizationExists    = errors.New("organization name already exists")
	ErrOrganizationNameExist = ErrOrganizationExists
	ErrTargetNotFound        = errors.New("one or more target IDs do not exist")
	ErrMCPKeyNotFound        = errors.New("mcp key not found")
	ErrMCPKeyGeneration      = errors.New("mcp key generation failed")
)
