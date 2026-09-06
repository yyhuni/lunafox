package application

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func mapSnapshotScanBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScanNotFoundForSnapshot
	}
	return err
}

func mapSnapshotSaveBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScanNotFoundForSnapshot
	}
	if errors.Is(err, ErrSnapshotTargetMismatch) {
		return ErrTargetMismatch
	}
	return err
}

func mapSubdomainSnapshotSaveBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScanNotFoundForSnapshot
	}
	if errors.Is(err, ErrSnapshotTargetMismatch) {
		return ErrTargetMismatch
	}
	if errors.Is(err, ErrSubdomainSnapshotInvalidTargetType) {
		return ErrInvalidTargetType
	}
	return err
}

func mapSubdomainSnapshotListBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScanNotFoundForSnapshot
	}
	return err
}

func mapScreenshotSnapshotBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSnapshotScanNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrScanNotFoundForSnapshot
	}
	if errors.Is(err, ErrScreenshotSnapshotNotFound) {
		return ErrScreenshotSnapshotNotFound
	}
	return err
}

func mapVulnerabilitySnapshotBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrVulnerabilitySnapshotNotFound) || dberrors.IsRecordNotFound(err) {
		return ErrVulnerabilitySnapshotNotFound
	}
	return err
}
