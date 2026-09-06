package model

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanModelFailureKindColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(Scan{}).FieldByName("FailureKind")
	if !ok {
		t.Fatalf("scan model must define FailureKind field")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:failure_kind") {
		t.Fatalf("FailureKind must map to failure_kind column, got tag: %q", tag)
	}
	if !strings.Contains(tag, "size:100") {
		t.Fatalf("FailureKind must use size:100, got tag: %q", tag)
	}
}

func TestScanTaskModelFailureDetailColumnTag(t *testing.T) {
	field, ok := reflect.TypeOf(ScanTask{}).FieldByName("FailureDetail")
	if !ok {
		t.Fatalf("scan task model must define FailureDetail field")
	}
	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:failure_detail") || !strings.Contains(tag, "size:500") {
		t.Fatalf("FailureDetail must map to failure_detail with size 500, got tag: %q", tag)
	}
}

func TestTaskProgressLogModelHasRequiredTaskOwnership(t *testing.T) {
	modelType := reflect.TypeOf(TaskProgressLog{})
	taskID, ok := modelType.FieldByName("TaskID")
	if !ok {
		t.Fatal("task progress log model must define TaskID")
	}
	if taskID.Type.Kind() != reflect.Int {
		t.Fatalf("TaskID must be required int ownership, got %s", taskID.Type)
	}
	if tag := taskID.Tag.Get("gorm"); !strings.Contains(tag, "column:task_id") || !strings.Contains(tag, "not null") {
		t.Fatalf("TaskID must map to a non-null task_id column, got tag: %q", tag)
	}
	scanID, ok := modelType.FieldByName("ScanID")
	if !ok {
		t.Fatal("task progress log model must persist ScanID for partition routing")
	}
	if scanID.Type.Kind() != reflect.Int {
		t.Fatalf("ScanID must be int, got %s", scanID.Type)
	}
	if tag := scanID.Tag.Get("gorm"); !strings.Contains(tag, "column:scan_id") || !strings.Contains(tag, "primaryKey") || !strings.Contains(tag, "not null") {
		t.Fatalf("ScanID must be a non-null composite primary key component, got tag: %q", tag)
	}
	if _, exists := modelType.FieldByName("Scan"); exists {
		t.Fatal("task progress log model must not expose a direct scan association")
	}
}
