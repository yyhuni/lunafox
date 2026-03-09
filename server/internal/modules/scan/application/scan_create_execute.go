package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	workflowdefaulting "github.com/yyhuni/lunafox/server/internal/workflow/defaulting"
	workflowschema "github.com/yyhuni/lunafox/server/internal/workflow/schema"
)

func (service *ScanCreateService) CreateNormal(input *CreateNormalInput) (*CreateScan, error) {
	if input == nil {
		return nil, ErrCreateInvalidConfig
	}
	if input.TargetID == 0 {
		return nil, ErrCreateTargetNotFound
	}
	if input.Configuration == nil {
		return nil, ErrCreateInvalidConfig
	}

	root := input.Configuration
	workflowIds := append([]string(nil), input.WorkflowIDs...)
	if err := validateWorkflowIDsStrict(workflowIds); err != nil {
		return nil, err
	}
	if err := service.validateRequestedWorkflows(workflowIds); err != nil {
		return nil, err
	}

	canonicalRoot, err := workflowdefaulting.NormalizeRootConfig(root, workflowIds)
	if err != nil {
		return nil, WrapSchemaInvalid("", "failed to normalize workflow configuration", err)
	}

	for _, workflow := range workflowIds {
		workflow = strings.TrimSpace(workflow)
		if workflow == "" {
			continue
		}
		config, err := extractWorkflowConfig(canonicalRoot, workflow)
		if err != nil {
			return nil, err
		}
		if err := workflowschema.ValidateConfigMap(workflow, config); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, WrapSchemaInvalid(
					workflow,
					fmt.Sprintf("workflow %s does not support this configuration version", workflow),
					err,
				)
			}
			return nil, WrapSchemaInvalid(
				workflow,
				fmt.Sprintf("workflow %s configuration failed schema validation", workflow),
				err,
			)
		}
	}

	if service.targetLookup == nil {
		return nil, ErrCreateTargetLookupNotReady
	}
	target, err := service.targetLookup(input.TargetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, ErrCreateTargetNotFound
	}

	workflowIdsJSON, err := json.Marshal(workflowIds)
	if err != nil {
		return nil, err
	}

	scan := &CreateScan{
		TargetID:      input.TargetID,
		WorkflowIDs:   workflowIdsJSON,
		Configuration: canonicalRoot,
		ScanMode:      CreateScanModeFull,
		Status:        CreateScanStatusPending,
	}

	tasks, err := buildScanTasks(workflowIds, canonicalRoot)
	if err != nil {
		return nil, err
	}
	if err := service.scanStore.CreateWithTasks(scan, tasks); err != nil {
		return nil, err
	}
	scan.Target = &TargetRef{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
	}

	return scan, nil
}
