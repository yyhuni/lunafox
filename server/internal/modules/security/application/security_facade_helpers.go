package application

import "github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

func mapVulnerabilityBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if dberrors.IsRecordNotFound(err) {
		return ErrVulnerabilityNotFound
	}
	return err
}

func mapSecurityTargetBoundaryError(err error) error {
	if err == nil {
		return nil
	}
	if dberrors.IsRecordNotFound(err) {
		return ErrTargetNotFound
	}
	return err
}
