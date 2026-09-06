package adapters

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type targetWriter struct {
	facade *catalogapp.TargetFacade
}

// NewTargetWriter projects the shared catalog batch command onto MCP without
// duplicating target canonicalization, merge, or transaction behavior.
func NewTargetWriter(facade *catalogapp.TargetFacade) tools.TargetBatchCreator {
	if facade == nil {
		return nil
	}
	return &targetWriter{facade: facade}
}

func (writer *targetWriter) Create(ctx context.Context, input tools.TargetBatchCreateInput) (tools.TargetBatchCreateOutput, error) {
	result, err := writer.facade.BatchCreateTargetsContext(ctx, input.Names, input.OrganizationID)
	if err != nil {
		return tools.TargetBatchCreateOutput{}, mapCommandError(err)
	}
	if result.NoValidTargets {
		return tools.TargetBatchCreateOutput{}, invalidCommandInput()
	}
	failedTargets := make([]tools.FailedTarget, 0, len(result.FailedTargets))
	for _, target := range result.FailedTargets {
		failedTargets = append(failedTargets, tools.FailedTarget{Name: target.Name, Reason: target.Reason})
	}
	output := tools.TargetBatchCreateOutput{
		CreatedCount:         result.CreatedCount,
		FailedCount:          result.FailedCount,
		FailedTargets:        failedTargets,
		AssociationCompleted: result.AssociationCompleted,
	}
	if input.OrganizationID != nil {
		output.Organization = httpdto.OrganizationName(*input.OrganizationID)
	}
	return output, nil
}
