package repository

import (
	model "github.com/yyhuni/lunafox/server/internal/modules/scan/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

// GetByIDNotDeleted finds a scan by ID (excluding soft deleted).
func (r *ScanRepository) GetByIDNotDeleted(id int) (*ScanRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToRecord(&scan), nil
}

// GetQueryByID finds a query scan by ID with target preloaded.
func (r *ScanRepository) GetQueryByID(id int) (*ScanRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Target").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToRecord(&scan), nil
}

// GetTaskRuntimeByID finds a runtime scan by ID with target preloaded.
func (r *ScanRepository) GetTaskRuntimeByID(id int) (*TaskRuntimeScanRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Target").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToTaskRuntimeRecord(&scan), nil
}

// FindScanLogScanRefByID finds the minimal scan projection required by scan-log flows.
func (r *ScanRepository) FindScanLogScanRefByID(id int) (*ScanLogScanRefRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Select("id", "status").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	return scanModelToScanLogScanRefRecord(&scan), nil
}

// FindByIDs finds scans by IDs (excluding soft deleted).
func (r *ScanRepository) FindByIDs(ids []int) ([]ScanRecord, error) {
	if len(ids) == 0 {
		return []ScanRecord{}, nil
	}

	var scans []model.Scan
	err := r.db.Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&scans).Error
	if err != nil {
		return nil, err
	}

	return scanModelListToRecord(scans), nil
}

// FindAll finds all scans with pagination and filters (excluding soft deleted).
func (r *ScanRepository) FindAll(page, pageSize int, targetID int, status, search string) ([]ScanRecord, int64, error) {
	var scans []model.Scan
	var total int64

	baseQuery := r.db.Model(&model.Scan{}).Where("scan.deleted_at IS NULL")

	if targetID > 0 {
		baseQuery = baseQuery.Where("scan.target_id = ?", targetID)
	}
	if status != "" {
		baseQuery = baseQuery.Where("scan.status = ?", status)
	}
	if search != "" {
		baseQuery = baseQuery.Joins("LEFT JOIN target ON target.id = scan.target_id").
			Where("target.name ILIKE ?", "%"+search+"%")
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Preload("Target").
		Scopes(
			scope.WithPagination(page, pageSize),
			scope.OrderByCreatedAtDesc(),
		).
		Find(&scans).Error
	if err != nil {
		return nil, 0, err
	}

	return scanModelListToRecord(scans), total, nil
}

// GetStatistics returns scan statistics.
func (r *ScanRepository) GetStatistics() (*ScanStatistics, error) {
	stats := &ScanStatistics{}

	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL").
		Count(&stats.Total).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusRunning).
		Count(&stats.Running).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusCompleted).
		Count(&stats.Completed).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL AND status = ?", scanStatusFailed).
		Count(&stats.Failed).Error; err != nil {
		return nil, err
	}

	type sumResult struct {
		TotalVulns      int64
		TotalSubdomains int64
		TotalEndpoints  int64
		TotalWebsites   int64
	}
	var sums sumResult
	if err := r.db.Model(&model.Scan{}).Where("deleted_at IS NULL").
		Select(`
			COALESCE(SUM(cached_vulns_total), 0) as total_vulns,
			COALESCE(SUM(cached_subdomains_count), 0) as total_subdomains,
			COALESCE(SUM(cached_endpoints_count), 0) as total_endpoints,
			COALESCE(SUM(cached_websites_count), 0) as total_websites
		`).
		Scan(&sums).Error; err != nil {
		return nil, err
	}

	stats.TotalVulns = sums.TotalVulns
	stats.TotalSubdomains = sums.TotalSubdomains
	stats.TotalEndpoints = sums.TotalEndpoints
	stats.TotalWebsites = sums.TotalWebsites
	stats.TotalAssets = sums.TotalSubdomains + sums.TotalEndpoints + sums.TotalWebsites

	return stats, nil
}

// GetTargetRefByScanID returns the target associated with a scan.
func (r *ScanRepository) GetTargetRefByScanID(scanID int) (*ScanTargetRecord, error) {
	var scan model.Scan
	err := r.db.Where("id = ? AND deleted_at IS NULL", scanID).
		Preload("Target").
		First(&scan).Error
	if err != nil {
		return nil, err
	}
	if scan.Target == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return scanTargetModelToRecord(scan.Target), nil
}
