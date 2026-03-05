package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanModelWorkflowNamesColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(Scan{}).FieldByName("WorkflowNames")
	if !ok {
		t.Fatalf("scan model must define WorkflowNames field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:workflow_names") {
		t.Fatalf("WorkflowNames must map to workflow_names column, got tag: %q", tag)
	}
	if strings.Contains(tag, "column:engine_names") {
		t.Fatalf("legacy engine_names column tag should not remain, got tag: %q", tag)
	}
}
