package task

import "testing"

func TestCounterIncDec(t *testing.T) {
	var counter Counter

	counter.Inc()
	counter.Inc()
	if got := counter.Count(); got != 2 {
		t.Fatalf("expected count 2, got %d", got)
	}

	counter.Dec()
	if got := counter.Count(); got != 1 {
		t.Fatalf("expected count 1, got %d", got)
	}
}
