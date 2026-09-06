package repository

import (
	"fmt"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

func (repository *ScanWorkflowRepository) CreateScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow) error {
	record, err := scanWorkflowDomainToModel(workflow)
	if err != nil {
		return err
	}
	if record.Version == 0 {
		record.Version = 1
	}
	if err := repository.db.Create(record).Error; err != nil {
		return err
	}
	stored, err := scanWorkflowModelToDomain(record)
	if err != nil {
		return err
	}
	*workflow = *stored
	return nil
}

// UpdateUserScanWorkflow advances a user row only if the expected version is
// still current. Callers re-read a false result to distinguish conflicts.
func (repository *ScanWorkflowRepository) UpdateUserScanWorkflow(workflow *catalogdomain.ManagedScanWorkflow, expectedVersion int64) (bool, error) {
	if workflow == nil || workflow.IsBuiltin {
		return false, fmt.Errorf("user scan workflow is required")
	}
	if expectedVersion <= 0 {
		return false, fmt.Errorf("expected workflow version must be positive")
	}
	record, err := scanWorkflowDomainToModel(workflow)
	if err != nil {
		return false, err
	}
	updates := map[string]any{
		"display_name": record.DisplayName,
		"description":  record.Description,
		"stages":       record.Stages,
		"version":      expectedVersion + 1,
	}
	result := repository.db.Model(&model.ScanWorkflow{}).
		Where("scan_workflow_id = ? AND version = ? AND is_builtin = FALSE", record.ScanWorkflowID, expectedVersion).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, nil
	}
	workflow.Version = expectedVersion + 1
	return true, nil
}
