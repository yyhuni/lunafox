package application

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func mapTargetBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if isRecordNotFound(err) {
		return ErrTargetNotFound
	}
	return err
}

func mapWordlistBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrWordlistNotFound) || isRecordNotFound(err) {
		return ErrWordlistNotFound
	}
	return err
}

func mapWordlistFileBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrWordlistNotFound) || isRecordNotFound(err) {
		return ErrWordlistNotFound
	}
	if errors.Is(err, ErrFileNotFound) {
		return ErrFileNotFound
	}
	return err
}

func isRecordNotFound(err error) bool {
	return dberrors.IsRecordNotFound(err)
}
