package application

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func (service *ScanFacade) CreateNormal(req *CreateNormalRequest) (*QueryScan, error) {
	if service.createService == nil {
		return nil, ErrScanInvalidConfig
	}

	scan, err := service.createService.CreateNormal(&CreateNormalInput{
		TargetID:      req.TargetID,
		WorkflowIDs:   req.WorkflowIDs,
		Configuration: req.Configuration,
	})
	if err != nil {
		if _, ok := AsWorkflowError(err); ok {
			return nil, err
		}
		if dberrors.IsRecordNotFound(err) || errors.Is(err, ErrCreateTargetNotFound) {
			return nil, ErrTargetNotFound
		}
		if errors.Is(err, ErrCreateInvalidConfig) || errors.Is(err, ErrCreateTargetLookupNotReady) {
			return nil, ErrScanInvalidConfig
		}
		if errors.Is(err, ErrCreateInvalidWorkflowIDs) {
			return nil, wrapScanInvalidWorkflowIDs(err)
		}
		if errors.Is(err, ErrCreateNoWorkflows) {
			return nil, ErrScanNoWorkflows
		}
		return nil, err
	}

	return createScanToQueryScan(scan), nil
}

func createScanToQueryScan(scan *CreateScan) *QueryScan {
	if scan == nil {
		return nil
	}

	result := &QueryScan{
		ID:            scan.ID,
		TargetID:      scan.TargetID,
		WorkflowIDs:   scan.WorkflowIDs,
		Configuration: scan.Configuration,
		ScanMode:      scan.ScanMode,
		Status:        scan.Status,
		CreatedAt:     scan.CreatedAt,
	}
	if scan.Target != nil {
		result.Target = &QueryTargetRef{
			ID:   scan.Target.ID,
			Name: scan.Target.Name,
			Type: scan.Target.Type,
		}
	}

	return result
}

func wrapScanInvalidWorkflowIDs(err error) error {
	if err == nil {
		return ErrScanInvalidWorkflowIDs
	}

	detail := strings.TrimSpace(strings.TrimPrefix(err.Error(), ErrCreateInvalidWorkflowIDs.Error()))
	detail = strings.TrimSpace(strings.TrimPrefix(detail, ":"))
	if detail == "" {
		return ErrScanInvalidWorkflowIDs
	}

	return fmt.Errorf("%w: %s", ErrScanInvalidWorkflowIDs, detail)
}
