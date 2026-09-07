package catalogwiring

import (
	"context"
	"errors"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
)

type scanWorkflowEngineResolver struct{ query installedengines.Query }

func (resolver scanWorkflowEngineResolver) HasEngine(ctx context.Context, engineID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if resolver.query == nil {
		return false, errors.New("installed Engine Package query is not configured")
	}
	_, err := resolver.query.GetInstalledEnginePackage(engineID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, catalogdomain.ErrEngineNotFound) {
		return false, nil
	}
	return false, err
}

func (resolver scanWorkflowEngineResolver) ExecutionDefinition(ctx context.Context, engineID string) (engineexecution.ExecutionDefinition, bool, error) {
	if err := ctx.Err(); err != nil {
		return engineexecution.ExecutionDefinition{}, false, err
	}
	if resolver.query == nil {
		return engineexecution.ExecutionDefinition{}, false, errors.New("installed Engine Package query is not configured")
	}
	loaded, err := resolver.query.GetInstalledEnginePackage(engineID)
	if errors.Is(err, catalogdomain.ErrEngineNotFound) {
		return engineexecution.ExecutionDefinition{}, false, nil
	}
	if err != nil {
		return engineexecution.ExecutionDefinition{}, false, err
	}
	return loaded.Layout.Definition.EngineDefinition.Execution, true, nil
}

func NewScanWorkflowManagementService(repo *catalogrepo.ScanWorkflowRepository, query installedengines.Query) (*catalogapp.ScanWorkflowManagementService, error) {
	return catalogapp.NewScanWorkflowManagementService(repo, scanWorkflowEngineResolver{query: query})
}

func NewScanWorkflowProfileService(repo *catalogrepo.ScanWorkflowRepository, query installedengines.Query) (*catalogapp.ScanWorkflowProfileService, error) {
	return catalogapp.NewScanWorkflowProfileService(repo, scanWorkflowEngineResolver{query: query})
}
