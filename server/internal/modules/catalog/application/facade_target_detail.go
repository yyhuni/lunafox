package application

import (
	"context"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"

	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

// GetDetailByID returns a target with asset summary by ID.
func (service *TargetFacade) GetDetailByID(id int) (*Target, *dto.TargetSummary, error) {
	target, summary, err := service.queryService.GetTargetDetailByID(context.Background(), id)
	if err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, nil, ErrTargetNotFound
		}
		return nil, nil, err
	}

	responseSummary := &dto.TargetSummary{
		Subdomains:  summary.Subdomains,
		Websites:    summary.Websites,
		Endpoints:   summary.Endpoints,
		IPs:         summary.IPs,
		Directories: summary.Directories,
		Screenshots: summary.Screenshots,
		Vulnerabilities: &dto.VulnerabilitySummary{
			Total:    summary.Vulnerabilities.Total,
			Critical: summary.Vulnerabilities.Critical,
			High:     summary.Vulnerabilities.High,
			Medium:   summary.Vulnerabilities.Medium,
			Low:      summary.Vulnerabilities.Low,
		},
	}

	return target, responseSummary, nil
}
