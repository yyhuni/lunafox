package config

import "testing"

func TestUpdaterApplyAndSnapshot(t *testing.T) {
	cfg := Config{
		RuntimeGRPCURL: "https://runtime.example.com",
		APIKey:         "key",
		MaxTasks:       2,
		CPUThreshold:   70,
		MemThreshold:   80,
		DiskThreshold:  90,
	}

	updater := NewUpdater(cfg)
	snapshot := updater.Snapshot()
	if snapshot.MaxTasks != 2 || snapshot.CPUThreshold != 70 {
		t.Fatalf("unexpected snapshot values")
	}

	invalid := 0
	update := Update{MaxTasks: &invalid, CPUThreshold: &invalid}
	snapshot = updater.Apply(update)
	if snapshot.MaxTasks != 2 || snapshot.CPUThreshold != 70 {
		t.Fatalf("expected invalid update to be ignored")
	}

	maxTasks := 5
	cpu := 85
	mem := 60
	snapshot = updater.Apply(Update{
		MaxTasks:     &maxTasks,
		CPUThreshold: &cpu,
		MemThreshold: &mem,
	})
	if snapshot.MaxTasks != 5 || snapshot.CPUThreshold != 85 || snapshot.MemThreshold != 60 {
		t.Fatalf("unexpected applied update")
	}
}
