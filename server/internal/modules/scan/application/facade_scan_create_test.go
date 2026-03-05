package application

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapScanInvalidWorkflowIDs_PreservesSentinelAndDetail(t *testing.T) {
	source := invalidWorkflowIDsf("workflowIds[0] must not be empty")
	err := wrapScanInvalidWorkflowIDs(source)
	if !errors.Is(err, ErrScanInvalidWorkflowIDs) {
		t.Fatalf("expected ErrScanInvalidWorkflowIDs sentinel, got: %v", err)
	}
	if !strings.Contains(err.Error(), "workflowIds[0] must not be empty") {
		t.Fatalf("expected detail to be preserved, got: %v", err)
	}
}

func TestWrapScanInvalidWorkflowIDs_FallbackToSentinel(t *testing.T) {
	err := wrapScanInvalidWorkflowIDs(ErrCreateInvalidWorkflowIDs)
	if !errors.Is(err, ErrScanInvalidWorkflowIDs) {
		t.Fatalf("expected ErrScanInvalidWorkflowIDs sentinel, got: %v", err)
	}
	if err.Error() != ErrScanInvalidWorkflowIDs.Error() {
		t.Fatalf("expected generic scan error without detail, got: %v", err)
	}
}
