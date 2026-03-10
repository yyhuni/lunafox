package catalogwiring

import (
	"context"
	"sort"
	"strings"

	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	workflowmanifest "github.com/yyhuni/lunafox/server/internal/workflow/manifest"
)

type catalogWorkflowQueryStoreAdapter struct {
	listManifests func() ([]workflowmanifest.Manifest, error)
}

func newCatalogWorkflowQueryStoreAdapter() *catalogWorkflowQueryStoreAdapter {
	return &catalogWorkflowQueryStoreAdapter{
		listManifests: workflowmanifest.ListManifests,
	}
}

func (adapter *catalogWorkflowQueryStoreAdapter) ListWorkflows(_ context.Context) ([]catalogapp.Workflow, error) {
	loadManifests := adapter.listManifests
	if loadManifests == nil {
		loadManifests = workflowmanifest.ListManifests
	}
	items, err := loadManifests()
	if err != nil {
		return nil, err
	}
	workflows := make([]catalogapp.Workflow, 0, len(items))
	for _, item := range items {
		workflowID := strings.TrimSpace(item.WorkflowID)
		if workflowID == "" {
			continue
		}
		workflows = append(workflows, catalogapp.Workflow{
			WorkflowID:  workflowID,
			DisplayName: strings.TrimSpace(item.DisplayName),
			Description: strings.TrimSpace(item.Description),
			Executor: catalogapp.WorkflowExecutorBinding{
				Type: strings.TrimSpace(item.Executor.Type),
				Ref:  strings.TrimSpace(item.Executor.Ref),
			},
		})
	}
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].WorkflowID < workflows[j].WorkflowID
	})
	return workflows, nil
}

func (adapter *catalogWorkflowQueryStoreAdapter) GetWorkflowByID(ctx context.Context, workflowID string) (*catalogapp.Workflow, error) {
	workflows, err := adapter.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	for i := range workflows {
		if workflows[i].WorkflowID == workflowID {
			workflow := workflows[i]
			return &workflow, nil
		}
	}
	return nil, catalogapp.ErrWorkflowNotFound
}
