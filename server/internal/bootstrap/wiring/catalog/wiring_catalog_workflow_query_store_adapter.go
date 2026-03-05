package catalogwiring

import (
	"context"
	"strings"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/workflowschema"
)

type catalogWorkflowQueryStoreAdapter struct{}

func newCatalogWorkflowQueryStoreAdapter() *catalogWorkflowQueryStoreAdapter {
	return &catalogWorkflowQueryStoreAdapter{}
}

func (adapter *catalogWorkflowQueryStoreAdapter) ListWorkflows(_ context.Context) ([]catalogapp.Workflow, error) {
	items, err := workflowschema.ListWorkflowMetadata()
	if err != nil {
		return nil, err
	}
	workflows := make([]catalogapp.Workflow, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		workflows = append(workflows, catalogapp.Workflow{
			Name:        name,
			Title:       strings.TrimSpace(item.Title),
			Description: strings.TrimSpace(item.Description),
		})
	}
	return workflows, nil
}

func (adapter *catalogWorkflowQueryStoreAdapter) GetWorkflowByName(ctx context.Context, name string) (*catalogapp.Workflow, error) {
	workflows, err := adapter.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	for i := range workflows {
		if workflows[i].Name == name {
			workflow := workflows[i]
			return &workflow, nil
		}
	}
	return nil, catalogapp.ErrWorkflowNotFound
}
