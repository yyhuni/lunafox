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

func TestScanTaskModelWorkflowConfigUsesJSONBColumn(t *testing.T) {
	field, ok := reflect.TypeOf(ScanTask{}).FieldByName("WorkflowConfig")
	if !ok {
		t.Fatalf("scan task model must define WorkflowConfig field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:workflow_config") {
		t.Fatalf("WorkflowConfig must map to workflow_config column, got tag: %q", tag)
	}
	if !strings.Contains(tag, "type:jsonb") {
		t.Fatalf("WorkflowConfig must use jsonb column type, got tag: %q", tag)
	}
}

func TestScanTaskModelDoesNotPersistLegacyWorkflowConfigYAMLField(t *testing.T) {
	if _, ok := reflect.TypeOf(ScanTask{}).FieldByName("WorkflowConfigYAML"); ok {
		t.Fatalf("legacy scan_task.workflow_config_yaml field should be removed from persistence model")
	}
}

func TestScanTaskModelFailureKindColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(ScanTask{}).FieldByName("FailureKind")
	if !ok {
		t.Fatalf("scan task model must define FailureKind field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:failure_kind") {
		t.Fatalf("FailureKind must map to failure_kind column, got tag: %q", tag)
	}
	if !strings.Contains(tag, "type:varchar(100)") {
		t.Fatalf("FailureKind must use varchar(100), got tag: %q", tag)
	}
}
