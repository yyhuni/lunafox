package repository

import (
	"context"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type domainScanRepository struct {
	repo *ScanRepository
}

func NewDomainScanRepository(repo *ScanRepository) scandomain.ScanRepository {
	return &domainScanRepository{repo: repo}
}

func (repository *domainScanRepository) GetByIDNotDeleted(ctx context.Context, id scandomain.ScanID) (*scandomain.Scan, error) {
	scan, err := repository.repo.GetByIDNotDeleted(int(id))
	if err != nil {
		return nil, err
	}
	return scanRecordToDomain(scan), nil
}

func (repository *domainScanRepository) FindAll(ctx context.Context, filter scandomain.ScanFilter) ([]scandomain.Scan, int64, error) {
	scans, total, err := repository.repo.FindAll(filter.Page, filter.PageSize, filter.TargetID, string(filter.Status), filter.Search)
	if err != nil {
		return nil, 0, err
	}

	results := make([]scandomain.Scan, 0, len(scans))
	for index := range scans {
		results = append(results, *scanRecordToDomain(&scans[index]))
	}

	return results, total, nil
}

func (repository *domainScanRepository) Save(ctx context.Context, scan *scandomain.Scan) error {
	if scan == nil || scan.ID <= 0 {
		return scandomain.ErrInvalidScanID
	}

	updates := map[string]any{
		"status":        string(scan.Status),
		"error_message": scan.ErrorMessage,
		"failure_kind":  "",
		"progress":      scan.Progress,
		"current_stage": scan.CurrentStage,
		"worker_id":     scan.WorkerID,
		"stopped_at":    scan.StoppedAt,
	}

	return repository.repo.db.WithContext(ctx).
		Model(&model.Scan{}).
		Where("id = ? AND deleted_at IS NULL", int(scan.ID)).
		Updates(updates).Error
}

func scanRecordToDomain(scan *ScanRecord) *scandomain.Scan {
	if scan == nil {
		return nil
	}

	return &scandomain.Scan{
		ID:           scandomain.ScanID(scan.ID),
		TargetID:     scan.TargetID,
		Mode:         scandomain.ScanMode(scan.ScanMode),
		Status:       scandomain.ScanStatus(scan.Status),
		ErrorMessage: scan.ErrorMessage,
		Progress:     scan.Progress,
		CurrentStage: scan.CurrentStage,
		WorkerID:     scan.WorkerID,
		CreatedAt:    timeutil.ToUTC(scan.CreatedAt),
		StoppedAt:    timeutil.ToUTCPtr(scan.StoppedAt),
	}
}
