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
