package application

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapScanInvalidWorkflowNames_PreservesSentinelAndDetail(t *testing.T) {
	source := invalidWorkflowNamesf("workflowNames[0] must not be empty")
	err := wrapScanInvalidWorkflowNames(source)
	if !errors.Is(err, ErrScanInvalidWorkflowNames) {
		t.Fatalf("expected ErrScanInvalidWorkflowNames sentinel, got: %v", err)
	}
	if !strings.Contains(err.Error(), "workflowNames[0] must not be empty") {
		t.Fatalf("expected detail to be preserved, got: %v", err)
	}
}

func TestWrapScanInvalidWorkflowNames_FallbackToSentinel(t *testing.T) {
	err := wrapScanInvalidWorkflowNames(ErrCreateInvalidWorkflowNames)
	if !errors.Is(err, ErrScanInvalidWorkflowNames) {
		t.Fatalf("expected ErrScanInvalidWorkflowNames sentinel, got: %v", err)
	}
	if err.Error() != ErrScanInvalidWorkflowNames.Error() {
		t.Fatalf("expected generic scan error without detail, got: %v", err)
	}
}
