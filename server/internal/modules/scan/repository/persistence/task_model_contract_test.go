package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanTaskModelDoesNotContainLegacyVersionField(t *testing.T) {
	if _, ok := reflect.TypeOf(ScanTask{}).FieldByName("Version"); ok {
		t.Fatalf("legacy scan_task.version field should be removed from model contract")
	}
}

func TestScanTaskModelDoesNotContainSchedulerRetryFields(t *testing.T) {
	for _, name := range []string{"RetryCount", "LastRejectReason", "LastRejectAt", "NextEligibleAt"} {
		if _, ok := reflect.TypeOf(ScanTask{}).FieldByName(name); ok {
			t.Fatalf("legacy scheduler retry field should be removed from model contract: %s", name)
		}
	}
}

func TestScanTaskModelWorkflowIDColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(ScanTask{}).FieldByName("WorkflowID")
	if !ok {
		t.Fatalf("scan task model must define WorkflowID field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:workflow_id") {
		t.Fatalf("WorkflowID must map to workflow_id column, got tag: %q", tag)
	}
}
