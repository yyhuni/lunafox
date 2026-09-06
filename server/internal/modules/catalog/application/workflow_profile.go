package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/contracts/scanworkflow"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

var ErrScanWorkflowEngineUnavailable = errors.New("scan workflow engine unavailable")

type ScanWorkflowProfileEngineResolver interface {
	ExecutionDefinition(ctx context.Context, engineID string) (engineexecution.ExecutionDefinition, bool, error)
}

type ManagedScanWorkflowProfile struct {
	ScanWorkflowID string
	Configuration  workflowconfig.Configuration
}

type ScanWorkflowProfileService struct {
	store   ManagedScanWorkflowStore
	engines ScanWorkflowProfileEngineResolver
}

func NewScanWorkflowProfileService(store ManagedScanWorkflowStore, engines ScanWorkflowProfileEngineResolver) (*ScanWorkflowProfileService, error) {
	if store == nil || engines == nil {
		return nil, fmt.Errorf("scan workflow Profile dependencies are required")
	}
	return &ScanWorkflowProfileService{store: store, engines: engines}, nil
}

func (service *ScanWorkflowProfileService) GetScanWorkflowProfile(ctx context.Context, scanWorkflowID string) (*ManagedScanWorkflowProfile, error) {
	if service == nil || service.store == nil {
		return nil, fmt.Errorf("scan workflow Profile store is required")
	}
	if ctx == nil {
		return nil, context.Canceled
	}
	var workflow *catalogdomain.ManagedScanWorkflow
	var err error
	if store, ok := service.store.(ManagedScanWorkflowStoreContext); ok {
		workflow, err = store.GetScanWorkflowByIDContext(ctx, scanWorkflowID)
	} else {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		workflow, err = service.store.GetScanWorkflowByID(scanWorkflowID)
	}
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("scan workflow Profile parent is unavailable")
	}
	// A Profile is materialized from one validated topology snapshot. Do not
	// return an empty or partial draft when a corrupt repository row has no
	// usable Step coverage.
	if err := scanworkflow.ValidateTopology(workflow.Stages); err != nil {
		return nil, fmt.Errorf("scan workflow Profile parent topology is invalid: %w", err)
	}
	steps := make(map[string]any)
	for _, stage := range workflow.Stages {
		for _, step := range stage.Steps {
			definition, available, err := service.engines.ExecutionDefinition(ctx, step.EngineID)
			if err != nil {
				return nil, err
			}
			if !available {
				return nil, fmt.Errorf("%w: step %q engine %q", ErrScanWorkflowEngineUnavailable, step.StepID, step.EngineID)
			}
			defaults, err := engineexecution.NormalizeAndValidateConfig(map[string]any{}, definition)
			if err != nil {
				return nil, fmt.Errorf("materialize step %q defaults: %w", step.StepID, err)
			}
			complete, err := engineexecution.ValidateCompleteConfig(defaults, definition)
			if err != nil {
				return nil, fmt.Errorf("materialize complete step %q defaults: %w", step.StepID, err)
			}
			if complete == nil {
				return nil, fmt.Errorf("materialize complete step %q defaults: empty configuration", step.StepID)
			}
			steps[step.StepID] = map[string]any{
				// Profile is the Server-owned initial editing draft. Its outer
				// Step defaults are independent from Engine config-section defaults.
				"enabled":      step.ProfileDefaultEnabled,
				"engineConfig": complete,
			}
		}
	}
	return &ManagedScanWorkflowProfile{ScanWorkflowID: workflow.ScanWorkflowID, Configuration: workflowconfig.Configuration{"steps": steps}}, nil
}
