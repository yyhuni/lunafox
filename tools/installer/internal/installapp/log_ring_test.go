package installapp

import (
	"strings"
	"testing"
)

func TestLogRingOverflowAndClip(t *testing.T) {
	ring := NewLogRing(2, 16)
	ring.Append("first-line\n")
	ring.Append("second-line\n")
	ring.Append("third-line-with-too-long-content\n")

	output := ring.String()
	if strings.Contains(output, "first-line") {
		t.Fatalf("expected oldest line evicted, got: %s", output)
	}
	if !strings.Contains(output, "second-line") {
		t.Fatalf("expected second line retained, got: %s", output)
	}
	if !strings.Contains(output, "...(truncated)") {
		t.Fatalf("expected truncation marker, got: %s", output)
	}
}
