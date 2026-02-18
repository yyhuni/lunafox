package websocket

import (
	"testing"
	"time"
)

func TestBackoffSequence(t *testing.T) {
	b := NewBackoff(time.Second, 60*time.Second)

	expected := []time.Duration{
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
		16 * time.Second,
		32 * time.Second,
		60 * time.Second,
		60 * time.Second,
	}

	for i, exp := range expected {
		if got := b.Next(); got != exp {
			t.Fatalf("step %d: expected %v, got %v", i, exp, got)
		}
	}

	b.Reset()
	if got := b.Next(); got != time.Second {
		t.Fatalf("after reset expected %v, got %v", time.Second, got)
	}
}
