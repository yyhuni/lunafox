package application

import (
	"errors"

	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func (service *ScanFacade) CreateNormal(req *CreateNormalRequest) (*QueryScan, error) {
	if service.createService == nil {
		return nil, ErrScanInvalidConfig
	}

	scan, err := service.createService.CreateNormal(&CreateNormalInput{TargetID: req.TargetID, EngineIDs: req.EngineIDs, EngineNames: req.EngineNames, Configuration: req.Configuration})
	if err != nil {
		if dberrors.IsRecordNotFound(err) || errors.Is(err, ErrCreateTargetNotFound) {
			return nil, ErrTargetNotFound
		}
		if errors.Is(err, ErrCreateInvalidConfig) || errors.Is(err, ErrCreateTargetLookupNotReady) {
			return nil, ErrScanInvalidConfig
		}
		if errors.Is(err, ErrCreateInvalidEngineNames) {
			return nil, ErrScanInvalidEngineNames
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
	result := &QueryScan{ID: scan.ID, TargetID: scan.TargetID, EngineIDs: scan.EngineIDs, EngineNames: scan.EngineNames, YamlConfiguration: scan.YamlConfiguration, ScanMode: scan.ScanMode, Status: scan.Status, CreatedAt: scan.CreatedAt}
	if scan.Target != nil {
		result.Target = &QueryTargetRef{ID: scan.Target.ID, Name: scan.Target.Name, Type: scan.Target.Type}
	}
	return result
}
