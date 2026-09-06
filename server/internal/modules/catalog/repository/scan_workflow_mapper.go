package repository

import (
	"encoding/json"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"gorm.io/datatypes"
)

func scanWorkflowModelToDomain(record *model.ScanWorkflow) (*catalogdomain.ManagedScanWorkflow, error) {
	if record == nil {
		return nil, nil
	}
	var stages []scanworkflow.Stage
	if err := json.Unmarshal(record.Stages, &stages); err != nil {
		return nil, fmt.Errorf("decode persisted workflow %q stages: %w", record.ScanWorkflowID, err)
	}
	workflow := &catalogdomain.ManagedScanWorkflow{
		ScanWorkflowID: record.ScanWorkflowID,
		DisplayName:    record.DisplayName,
		Description:    record.Description,
		Stages:         stages,
		IsBuiltin:      record.IsBuiltin,
		Version:        record.Version,
		CreateTime:     timeutil.ToUTC(record.CreateTime),
		UpdateTime:     timeutil.ToUTC(record.UpdateTime),
	}
	if record.DefinitionDigest != nil {
		workflow.DefinitionDigest = *record.DefinitionDigest
	}
	if record.RequestID != nil {
		workflow.RequestID = *record.RequestID
	}
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("validate persisted workflow %q: %w", record.ScanWorkflowID, err)
	}
	return workflow, nil
}

func scanWorkflowDomainToModel(workflow *catalogdomain.ManagedScanWorkflow) (*model.ScanWorkflow, error) {
	if workflow == nil {
		return nil, fmt.Errorf("scan workflow is required")
	}
	if err := workflow.Validate(); err != nil {
		return nil, err
	}
	stages, err := json.Marshal(workflow.Stages)
	if err != nil {
		return nil, fmt.Errorf("encode scan workflow stages: %w", err)
	}
	modelWorkflow := &model.ScanWorkflow{
		ScanWorkflowID: workflow.ScanWorkflowID,
		DisplayName:    workflow.DisplayName,
		Description:    workflow.Description,
		Stages:         datatypes.JSON(stages),
		IsBuiltin:      workflow.IsBuiltin,
		Version:        workflow.Version,
		CreateTime:     timeutil.ToUTC(workflow.CreateTime),
		UpdateTime:     timeutil.ToUTC(workflow.UpdateTime),
	}
	if workflow.DefinitionDigest != "" {
		digest := workflow.DefinitionDigest
		modelWorkflow.DefinitionDigest = &digest
	}
	if workflow.RequestID != "" {
		requestID := workflow.RequestID
		modelWorkflow.RequestID = &requestID
	}
	return modelWorkflow, nil
}

func scanWorkflowModelListToDomain(records []model.ScanWorkflow) ([]catalogdomain.ManagedScanWorkflow, error) {
	workflows := make([]catalogdomain.ManagedScanWorkflow, 0, len(records))
	for index := range records {
		workflow, err := scanWorkflowModelToDomain(&records[index])
		if err != nil {
			return nil, err
		}
		workflows = append(workflows, *workflow)
	}
	return workflows, nil
}
