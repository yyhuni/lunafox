package application

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapScanInvalidEngineNames_PreservesSentinelAndDetail(t *testing.T) {
	source := invalidEngineNamesf("engineNames[0] must not be empty")
	err := wrapScanInvalidEngineNames(source)
	if !errors.Is(err, ErrScanInvalidEngineNames) {
		t.Fatalf("expected ErrScanInvalidEngineNames sentinel, got: %v", err)
	}
	if !strings.Contains(err.Error(), "engineNames[0] must not be empty") {
		t.Fatalf("expected detail to be preserved, got: %v", err)
	}
}

func TestWrapScanInvalidEngineNames_FallbackToSentinel(t *testing.T) {
	err := wrapScanInvalidEngineNames(ErrCreateInvalidEngineNames)
	if !errors.Is(err, ErrScanInvalidEngineNames) {
		t.Fatalf("expected ErrScanInvalidEngineNames sentinel, got: %v", err)
	}
	if err.Error() != ErrScanInvalidEngineNames.Error() {
		t.Fatalf("expected generic scan error without detail, got: %v", err)
	}
}
