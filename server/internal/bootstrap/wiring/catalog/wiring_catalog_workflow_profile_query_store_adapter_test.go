package catalogwiring

import (
	"testing"

	workflowprofile "github.com/yyhuni/lunafox/server/internal/workflow/profile"
)

func TestCloneWorkflowConfig_DeepCopiesNestedMapsAndSlices(t *testing.T) {
	original := workflowprofile.WorkflowConfig{
		"subdomain_discovery": map[string]any{
			"recon": map[string]any{
				"enabled": true,
				"tools": []any{
					map[string]any{"name": "subfinder", "enabled": true},
				},
			},
		},
	}

	cloned := cloneWorkflowConfig(original)

	clonedWorkflow := cloned["subdomain_discovery"].(map[string]any)
	clonedRecon := clonedWorkflow["recon"].(map[string]any)
	clonedRecon["enabled"] = false
	clonedTools := clonedRecon["tools"].([]any)
	clonedTool := clonedTools[0].(map[string]any)
	clonedTool["enabled"] = false

	originalWorkflow := original["subdomain_discovery"].(map[string]any)
	originalRecon := originalWorkflow["recon"].(map[string]any)
	if got := originalRecon["enabled"]; got != true {
		t.Fatalf("expected original nested map value unchanged, got %#v", got)
	}
	originalTools := originalRecon["tools"].([]any)
	originalTool := originalTools[0].(map[string]any)
	if got := originalTool["enabled"]; got != true {
		t.Fatalf("expected original nested slice item unchanged, got %#v", got)
	}
}
