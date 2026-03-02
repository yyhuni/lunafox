package model

import (
	"reflect"
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
