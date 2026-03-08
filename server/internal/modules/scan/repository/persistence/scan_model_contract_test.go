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

func TestScanModelConfigurationUsesJSONBColumn(t *testing.T) {
	field, ok := reflect.TypeOf(Scan{}).FieldByName("Configuration")
	if !ok {
		t.Fatalf("scan model must define Configuration field")
	}

	tag := field.Tag.Get("gorm")
	if !strings.Contains(tag, "column:configuration") {
		t.Fatalf("Configuration must map to configuration column, got tag: %q", tag)
	}
	if !strings.Contains(tag, "type:jsonb") {
		t.Fatalf("Configuration must use jsonb column type, got tag: %q", tag)
	}
	if got := field.Tag.Get("json"); got != "configuration" {
		t.Fatalf("Configuration must use configuration json tag, got %q", got)
	}
}

func TestScanModelDoesNotPersistLegacyYAMLConfigurationField(t *testing.T) {
	if _, ok := reflect.TypeOf(Scan{}).FieldByName("YAMLConfiguration"); ok {
		t.Fatalf("legacy scan.yaml_configuration field should be removed from persistence model")
	}
}
