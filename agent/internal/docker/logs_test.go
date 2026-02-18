package docker

import (
	"strings"
	"testing"
)

func TestTruncateErrorMessage(t *testing.T) {
	short := "short message"
	if got := TruncateErrorMessage(short); got != short {
		t.Fatalf("expected message to stay unchanged")
	}

	long := strings.Repeat("x", maxErrorBytes+10)
	got := TruncateErrorMessage(long)
	if len(got) != maxErrorBytes {
		t.Fatalf("expected length %d, got %d", maxErrorBytes, len(got))
	}
	if got != long[:maxErrorBytes] {
		t.Fatalf("unexpected truncation result")
	}
}
