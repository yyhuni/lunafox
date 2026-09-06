package application

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func isIdentityRecordNotFound(err error) bool {
	return dberrors.IsRecordNotFound(err)
}

func mapOrganizationBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if isIdentityRecordNotFound(err) {
		return ErrOrganizationNotFound
	}
	return err
}

func mapOrganizationTargetLinkBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if isIdentityRecordNotFound(err) {
		return ErrOrganizationNotFound
	}
	if errors.Is(err, ErrTargetNotFound) {
		return ErrTargetNotFound
	}
	return err
}

func mapUserBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if isIdentityRecordNotFound(err) {
		return ErrUserNotFound
	}
	return err
}

func mapAuthUserBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if isIdentityRecordNotFound(err) {
		return ErrAuthUserNotFound
	}
	return err
}
