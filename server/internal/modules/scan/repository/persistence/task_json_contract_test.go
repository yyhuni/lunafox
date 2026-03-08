package model

import (
	"reflect"
	"testing"
)

func TestScanTaskModelJSONTagsUseCamelCase(t *testing.T) {
	tests := map[string]string{
		"ScanID":         "scanId",
		"WorkflowID":     "workflowId",
		"AgentID":        "agentId,omitempty",
		"WorkflowConfig": "workflowConfig",
		"ErrorMessage":   "errorMessage,omitempty",
		"CreatedAt":      "createdAt",
		"StartedAt":      "startedAt,omitempty",
		"CompletedAt":    "completedAt,omitempty",
	}

	for fieldName, expectedTag := range tests {
		field, ok := reflect.TypeOf(ScanTask{}).FieldByName(fieldName)
		if !ok {
			t.Fatalf("scan task model must define %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != expectedTag {
			t.Fatalf("%s must use %s json tag, got %q", fieldName, expectedTag, got)
		}
	}
}
