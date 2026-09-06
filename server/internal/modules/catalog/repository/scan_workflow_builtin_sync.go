package repository

import (
	"errors"
	"fmt"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/gorm"
)

const scanWorkflowBuiltinSyncAdvisoryLock int64 = 0x4c554e4157464c57

// SynchronizeBuiltinScanWorkflows applies one release definition set as a
// single transaction. It never infers deletion from files absent in a release.
func (repository *ScanWorkflowRepository) SynchronizeBuiltinScanWorkflows(workflows []catalogdomain.ManagedScanWorkflow) error {
	if repository == nil || repository.db == nil {
		return fmt.Errorf("scan workflow repository is not configured")
	}
	seen := make(map[string]struct{}, len(workflows))
	for index := range workflows {
		workflow := workflows[index]
		if !workflow.IsBuiltin {
			return fmt.Errorf("release workflow %q must be built-in", workflow.ScanWorkflowID)
		}
		if err := workflow.Validate(); err != nil {
			return fmt.Errorf("validate release workflow %q: %w", workflow.ScanWorkflowID, err)
		}
		if _, exists := seen[workflow.ScanWorkflowID]; exists {
			return fmt.Errorf("duplicate release workflow %q", workflow.ScanWorkflowID)
		}
		seen[workflow.ScanWorkflowID] = struct{}{}
	}

	return repository.db.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", scanWorkflowBuiltinSyncAdvisoryLock).Error; err != nil {
				return fmt.Errorf("acquire built-in workflow synchronization lock: %w", err)
			}
		}
		for index := range workflows {
			if err := validateBuiltinWorkflowEngineReferences(tx, workflows[index]); err != nil {
				return err
			}
		}
		for index := range workflows {
			if err := synchronizeOneBuiltinScanWorkflow(tx, &workflows[index]); err != nil {
				return err
			}
		}
		return nil
	})
}

// validateBuiltinWorkflowEngineReferences runs inside the synchronization
// transaction so release validation and persisted ownership changes use one
// database snapshot. A missing Engine must abort startup before any row writes.
func validateBuiltinWorkflowEngineReferences(tx *gorm.DB, workflow catalogdomain.ManagedScanWorkflow) error {
	for _, stage := range workflow.Stages {
		for _, step := range stage.Steps {
			engineID := strings.TrimSpace(step.EngineID)
			var engine model.Engine
			if err := tx.Select("engine_id").Where("engine_id = ?", engineID).First(&engine).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("release workflow %q step %q engineId %q does not resolve to an installed Engine", workflow.ScanWorkflowID, step.StepID, engineID)
				}
				return fmt.Errorf("resolve release workflow %q step %q engineId %q: %w", workflow.ScanWorkflowID, step.StepID, engineID, err)
			}
		}
	}
	return nil
}

func synchronizeOneBuiltinScanWorkflow(tx *gorm.DB, workflow *catalogdomain.ManagedScanWorkflow) error {
	var existing model.ScanWorkflow
	err := tx.Where("scan_workflow_id = ?", workflow.ScanWorkflowID).First(&existing).Error
	if err == nil {
		if !existing.IsBuiltin {
			return fmt.Errorf("release workflow %q collides with user-owned resource", workflow.ScanWorkflowID)
		}
		if existing.DefinitionDigest != nil && *existing.DefinitionDigest == workflow.DefinitionDigest {
			return nil
		}
		record, mapErr := scanWorkflowDomainToModel(workflow)
		if mapErr != nil {
			return mapErr
		}
		return tx.Model(&model.ScanWorkflow{}).Where("scan_workflow_id = ? AND is_builtin = TRUE", workflow.ScanWorkflowID).Updates(map[string]any{
			"display_name":      record.DisplayName,
			"description":       record.Description,
			"stages":            record.Stages,
			"definition_digest": record.DefinitionDigest,
			"version":           gorm.Expr("version + 1"),
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	record, mapErr := scanWorkflowDomainToModel(workflow)
	if mapErr != nil {
		return mapErr
	}
	if record.Version == 0 {
		record.Version = 1
	}
	return tx.Create(record).Error
}
