package metrics

import "testing"

func TestCollectorSample(t *testing.T) {
	c := NewCollector()
	cpu, mem, disk := c.Sample()
	if cpu < 0 || mem < 0 || disk < 0 {
		t.Fatalf("expected non-negative metrics")
	}
}
