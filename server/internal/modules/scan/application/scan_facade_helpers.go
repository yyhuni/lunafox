package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func mapScanBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if dberrors.IsRecordNotFound(err) {
		return ErrScanNotFound
	}
	return err
}

func mapScanStopBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if dberrors.IsRecordNotFound(err) {
		return ErrScanNotFound
	}
	if errors.Is(err, scandomain.ErrScanCannotStop) || errors.Is(err, scandomain.ErrInvalidStatusChange) {
		return ErrScanCannotStop
	}
	return err
}

func mapTaskStatusBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrScanTaskNotFound) {
		return ErrScanTaskNotFound
	}
	if errors.Is(err, ErrScanTaskNotOwned) {
		return ErrScanTaskNotOwned
	}
	if errors.Is(err, ErrScanTaskInvalidTransition) {
		return ErrScanTaskInvalidTransition
	}
	if errors.Is(err, ErrScanTaskInvalidUpdate) {
		return ErrScanTaskInvalidUpdate
	}
	return err
}
