package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanModelWorkflowIDsColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(Scan{}).FieldByName("WorkflowIDs")
	if !ok {
		t.Fatalf("scan model must define WorkflowIDs field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:workflow_ids") {
		t.Fatalf("WorkflowIDs must map to workflow_ids column, got tag: %q", tag)
	}
	if strings.Contains(tag, "column:engine_names") {
		t.Fatalf("legacy engine_names column tag should not remain, got tag: %q", tag)
	}
}
