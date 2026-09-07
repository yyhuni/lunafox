package application

import (
	"errors"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
)

var (
	ErrOrganizationNotFound = identitydomain.ErrOrganizationNotFound
	ErrOrganizationExists   = identitydomain.ErrOrganizationNameExist
	ErrTargetNotFound       = identitydomain.ErrTargetNotFound

	ErrUnsupportedOrganizationFilter  = errors.New("unsupported organization filter")
	ErrUnsupportedOrganizationOrderBy = errors.New("unsupported organization orderBy")
	ErrInvalidOrganizationPageToken   = errors.New("invalid organization pageToken")
	ErrInvalidOrganizationQuery       = errors.New("invalid organization query")
)
