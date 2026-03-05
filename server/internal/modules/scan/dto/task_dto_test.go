package dto

import (
	"encoding/json"
	"testing"
)

func TestTaskAssignmentJSONUsesWorkflowConfigYAMLField(t *testing.T) {
	payload, err := json.Marshal(TaskAssignment{
		TaskID:             1,
		WorkflowConfigYAML: "recon:\n  enabled: true\n",
	})
	if err != nil {
		t.Fatalf("marshal task assignment: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal task assignment: %v", err)
	}

	if _, ok := got["workflowConfigYAML"]; !ok {
		t.Fatalf("expected workflowConfigYAML json key, got: %v", got)
	}
	if _, exists := got["config"]; exists {
		t.Fatalf("unexpected legacy config json key, got: %v", got)
	}
}
