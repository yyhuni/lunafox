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

func newCatalogWorkflowProfileQueryStoreAdapter(
	loader *workflowprofile.Loader,
) *catalogWorkflowProfileQueryStoreAdapter {
	return &catalogWorkflowProfileQueryStoreAdapter{loader: loader}
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) ListWorkflowProfiles(
	_ context.Context,
) ([]catalogapp.WorkflowProfile, error) {
	if adapter.loader == nil {
		return nil, fmt.Errorf("workflow profile loader is not configured")
	}

	profiles := adapter.loader.List()
	result := make([]catalogapp.WorkflowProfile, 0, len(profiles))
	for _, item := range profiles {
		profile, err := adapter.toWorkflowProfile(&item)
		if err != nil {
			return nil, err
		}
		result = append(result, *profile)
	}

	return result, nil
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) GetWorkflowProfileByID(
	_ context.Context,
	id string,
) (*catalogapp.WorkflowProfile, error) {
	if adapter.loader == nil {
		return nil, fmt.Errorf("workflow profile loader is not configured")
	}

	item := adapter.loader.GetByID(id)
	if item == nil {
		return nil, catalogapp.ErrWorkflowProfileNotFound
	}

	return adapter.toWorkflowProfile(item)
}

func (adapter *catalogWorkflowProfileQueryStoreAdapter) toWorkflowProfile(
	item *workflowprofile.Profile,
) (*catalogapp.WorkflowProfile, error) {
	if item == nil {
		return nil, catalogapp.ErrWorkflowProfileNotFound
	}

	workflowIDs, err := workflowprofile.ValidateAndExtractWorkflowIDs(item.Configuration)
	if err != nil {
		return nil, fmt.Errorf("extract workflow ids from profile %q: %w", item.ID, err)
	}

	return &catalogapp.WorkflowProfile{
		ID:            item.ID,
		Name:          item.Name,
		Description:   item.Description,
		WorkflowIDs:   workflowIDs,
		Configuration: cloneWorkflowConfig(item.Configuration),
	}, nil
}

func cloneWorkflowConfig(config workflowprofile.WorkflowConfig) map[string]any {
	cloned := make(map[string]any, len(config))
	for key, value := range config {
		cloned[key] = cloneWorkflowConfigValue(value)
	}
	return cloned
}

func cloneWorkflowConfigValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nested := range typed {
			cloned[key] = cloneWorkflowConfigValue(nested)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = cloneWorkflowConfigValue(item)
		}
		return cloned
	default:
		return value
	}
}
