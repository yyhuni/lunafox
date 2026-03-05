package catalogwiring

import (
	"context"
	"fmt"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/preset"
)

type catalogWorkflowProfileQueryStoreAdapter struct {
	loader *preset.Loader
}

func newCatalogWorkflowProfileQueryStoreAdapter(loader *preset.Loader) *catalogWorkflowProfileQueryStoreAdapter {
	return &catalogWorkflowProfileQueryStoreAdapter{loader: loader}
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) ListWorkflowProfiles(_ context.Context) ([]catalogapp.WorkflowProfile, error) {
	if adapter.loader == nil {
		return nil, fmt.Errorf("workflow profile loader is not configured")
	}
	presets := adapter.loader.List()
	result := make([]catalogapp.WorkflowProfile, 0, len(presets))
	for _, item := range presets {
		workflowNames, err := preset.ExtractWorkflowNames(item.Configuration)
		if err != nil {
			return nil, fmt.Errorf("extract workflow names from preset %q: %w", item.ID, err)
		}
		result = append(result, catalogapp.WorkflowProfile{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			WorkflowNames: workflowNames,
			Configuration: item.Configuration,
		})
	}
	return result, nil
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) GetWorkflowProfileByID(_ context.Context, id string) (*catalogapp.WorkflowProfile, error) {
	if adapter.loader == nil {
		return nil, fmt.Errorf("workflow profile loader is not configured")
	}
	item := adapter.loader.GetByID(id)
	if item == nil {
		return nil, catalogapp.ErrWorkflowProfileNotFound
	}
	workflowNames, err := preset.ExtractWorkflowNames(item.Configuration)
	if err != nil {
		return nil, fmt.Errorf("extract workflow names from preset %q: %w", item.ID, err)
	}
	return &catalogapp.WorkflowProfile{
		ID:            item.ID,
		Name:          item.Name,
		Description:   item.Description,
		WorkflowNames: workflowNames,
		Configuration: item.Configuration,
	}, nil
}
