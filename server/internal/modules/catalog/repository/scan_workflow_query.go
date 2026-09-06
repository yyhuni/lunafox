package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/gorm"
)

func (repository *ScanWorkflowRepository) GetScanWorkflowByID(scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error) {
	return repository.GetScanWorkflowByIDContext(context.Background(), scanWorkflowID)
}

func (repository *ScanWorkflowRepository) GetScanWorkflowByIDContext(ctx context.Context, scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error) {
	if repository == nil || repository.db == nil {
		return nil, fmt.Errorf("scan workflow repository is not configured")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var record model.ScanWorkflow
	if err := repository.db.WithContext(ctx).Where("scan_workflow_id = ?", strings.TrimSpace(scanWorkflowID)).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %q", catalogdomain.ErrScanWorkflowNotFound, scanWorkflowID)
		}
		return nil, err
	}
	return scanWorkflowModelToDomain(&record)
}

func (repository *ScanWorkflowRepository) FindScanWorkflowByRequestID(requestID string) (*catalogdomain.ManagedScanWorkflow, error) {
	trimmed := strings.TrimSpace(requestID)
	if trimmed == "" {
		return nil, nil
	}
	var record model.ScanWorkflow
	if err := repository.db.Where("request_id = ?", trimmed).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return scanWorkflowModelToDomain(&record)
}

func (repository *ScanWorkflowRepository) ListScanWorkflows(filter catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error) {
	return repository.ListScanWorkflowsContext(context.Background(), filter)
}

// ListScanWorkflowsContext preserves the caller-owned cancellation and
// deadline for MCP catalog discovery.
func (repository *ScanWorkflowRepository) ListScanWorkflowsContext(ctx context.Context, filter catalogdomain.ScanWorkflowListFilter) ([]catalogdomain.ManagedScanWorkflow, int64, error) {
	if repository == nil || repository.db == nil {
		return nil, 0, fmt.Errorf("scan workflow repository is not configured")
	}
	if ctx == nil {
		return nil, 0, context.Canceled
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		return nil, 0, fmt.Errorf("pageSize must be positive")
	}
	query := applyScanWorkflowTextFilter(repository.db.WithContext(ctx).Model(&model.ScanWorkflow{}), filter.NormalizedFilter())
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []model.ScanWorkflow
	if err := query.Order("is_builtin DESC").Order("display_name ASC").Order("scan_workflow_id ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	workflows, err := scanWorkflowModelListToDomain(records)
	if err != nil {
		return nil, 0, err
	}
	return workflows, total, nil
}

func applyScanWorkflowTextFilter(query *gorm.DB, filter string) *gorm.DB {
	if filter == "" {
		return query
	}
	pattern := "%" + strings.ToLower(filter) + "%"
	return query.Where("LOWER(scan_workflow_id) LIKE ? OR LOWER(display_name) LIKE ? OR LOWER(description) LIKE ?", pattern, pattern, pattern)
}
