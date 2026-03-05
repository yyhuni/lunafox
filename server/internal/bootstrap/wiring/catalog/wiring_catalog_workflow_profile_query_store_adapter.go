package catalogwiring

import (
	"context"
	"fmt"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	workflowprofile "github.com/yyhuni/lunafox/server/internal/workflow/profile"
)

type catalogWorkflowProfileQueryStoreAdapter struct {
	loader *workflowprofile.Loader
}

func newCatalogWorkflowProfileQueryStoreAdapter(loader *workflowprofile.Loader) *catalogWorkflowProfileQueryStoreAdapter {
	return &catalogWorkflowProfileQueryStoreAdapter{loader: loader}
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) ListWorkflowProfiles(_ context.Context) ([]catalogapp.WorkflowProfile, error) {
	if adapter.loader == nil {
		return nil, fmt.Errorf("workflow profile loader is not configured")
	}
	profiles := adapter.loader.List()
	result := make([]catalogapp.WorkflowProfile, 0, len(profiles))
	for _, item := range profiles {
		workflowIDs, err := workflowprofile.ExtractWorkflowIDs(item.Configuration)
		if err != nil {
			return nil, fmt.Errorf("extract workflow ids from profile %q: %w", item.ID, err)
		}
		result = append(result, catalogapp.WorkflowProfile{
			ID:            item.ID,
			Name:          item.Name,
			Description:   item.Description,
			WorkflowIDs:   workflowIDs,
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
	workflowIDs, err := workflowprofile.ExtractWorkflowIDs(item.Configuration)
	if err != nil {
		return nil, fmt.Errorf("extract workflow ids from profile %q: %w", item.ID, err)
	}
	return &catalogapp.WorkflowProfile{
		ID:            item.ID,
		Name:          item.Name,
		Description:   item.Description,
		WorkflowIDs:   workflowIDs,
		Configuration: item.Configuration,
	}, nil
}
