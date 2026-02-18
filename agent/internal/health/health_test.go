package health

import "testing"

func TestManagerSetTransitions(t *testing.T) {
	mgr := NewManager()
	initial := mgr.Get()
	if initial.State != "ok" || initial.Since != nil {
		t.Fatalf("expected initial ok status")
	}

	mgr.Set("paused", "update", "waiting")
	status := mgr.Get()
	if status.State != "paused" || status.Since == nil {
		t.Fatalf("expected paused state with timestamp")
	}
	prevSince := status.Since

	mgr.Set("paused", "still", "waiting more")
	status = mgr.Get()
	if status.Since == nil || !status.Since.Equal(*prevSince) {
		t.Fatalf("expected unchanged since on same state")
	}
	if status.Reason != "still" || status.Message != "waiting more" {
		t.Fatalf("expected updated reason/message")
	}

	mgr.Set("ok", "ignored", "ignored")
	status = mgr.Get()
	if status.State != "ok" || status.Since != nil || status.Reason != "" || status.Message != "" {
		t.Fatalf("expected ok reset to clear fields")
	}
}
