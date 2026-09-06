package application

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func mapAssetTargetBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrTargetNotFound
	}
	return err
}

func mapAssetTargetDomainError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTargetNotFound) {
		return ErrTargetNotFound
	}
	return err
}

func mapSubdomainCreateBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTargetNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrTargetNotFound
	}
	if errors.Is(err, ErrSubdomainInvalidTargetType) {
		return ErrInvalidTargetType
	}
	return err
}

func mapEndpointRecordBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if dberrors.IsRecordNotFound(err) {
		return ErrEndpointNotFound
	}
	return err
}

func mapEndpointDeleteBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrEndpointNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrEndpointNotFound
	}
	return err
}

func mapScreenshotRecordBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrScreenshotNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScreenshotNotFound
	}
	return err
}

func mapWebsiteRecordBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrWebsiteNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrWebsiteNotFound
	}
	return err
}
