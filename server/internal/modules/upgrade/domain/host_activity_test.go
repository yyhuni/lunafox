package domain

import (
	"testing"
	"time"
)

func TestValidateHostActivityAcceptsOnlyOwnedActiveSnapshots(t *testing.T) {
	base := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		action HostAction
		status Status
		mode   ExecutionMode
	}{
		{name: "preflight", action: HostActionPreflight, status: StatusPreflight, mode: ExecutionModeFull},
		{name: "pull", action: HostActionPullImages, status: StatusUpdating, mode: ExecutionModeFull},
		{name: "update", action: HostActionUpdateServices, status: StatusUpdating, mode: ExecutionModeFrontendOnly},
		{name: "migration", action: HostActionDatabaseMigration, status: StatusMigrating, mode: ExecutionModeFull},
		{name: "resident agent", action: HostActionUpdateResidentAgent, status: StatusRestarting, mode: ExecutionModeFull},
		{name: "full health", action: HostActionWaitForServiceHealth, status: StatusAgentVerifying, mode: ExecutionModeFull},
		{name: "frontend health", action: HostActionWaitForServiceHealth, status: StatusVerifying, mode: ExecutionModeFrontendOnly},
		{name: "runtime images", action: HostActionVerifyRuntimeImages, status: StatusVerifying, mode: ExecutionModeFull},
		{name: "frontend container", action: HostActionVerifyFrontendContainer, status: StatusVerifying, mode: ExecutionModeFrontendOnly},
		{name: "frontend edge", action: HostActionVerifyFrontendEdge, status: StatusVerifying, mode: ExecutionModeFrontendOnly},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity := &HostActivity{
				Action:          test.action,
				StartedAt:       base.Add(time.Minute),
				LastHeartbeatAt: base.Add(2 * time.Minute),
			}
			if err := ValidateHostActivity(activity, test.status, test.mode, base, base.Add(3*time.Minute), base.Add(4*time.Minute)); err != nil {
				t.Fatalf("ValidateHostActivity() error = %v", err)
			}
		})
	}

	if err := ValidateHostActivity(nil, StatusUpdating, ExecutionModeFull, base, base, base); err != nil {
		t.Fatalf("legacy nil activity rejected: %v", err)
	}
}

func TestValidateHostActivityRejectsInvalidLifecycleAndTimestamps(t *testing.T) {
	base := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	valid := HostActivity{Action: HostActionPullImages, StartedAt: base.Add(time.Minute), LastHeartbeatAt: base.Add(2 * time.Minute)}
	tests := []struct {
		name       string
		activity   HostActivity
		status     Status
		mode       ExecutionMode
		startedAt  time.Time
		observedAt time.Time
		now        time.Time
	}{
		{name: "terminal", activity: valid, status: StatusSucceeded, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "unknown action", activity: HostActivity{Action: "shell", StartedAt: valid.StartedAt, LastHeartbeatAt: valid.LastHeartbeatAt}, status: StatusUpdating, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "wrong stage", activity: valid, status: StatusPreflight, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "wrong mode", activity: HostActivity{Action: HostActionVerifyRuntimeImages, StartedAt: valid.StartedAt, LastHeartbeatAt: valid.LastHeartbeatAt}, status: StatusVerifying, mode: ExecutionModeFrontendOnly, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "inverted timestamps", activity: HostActivity{Action: valid.Action, StartedAt: base.Add(2 * time.Minute), LastHeartbeatAt: base.Add(time.Minute)}, status: StatusUpdating, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "before operation", activity: HostActivity{Action: valid.Action, StartedAt: base.Add(-time.Minute), LastHeartbeatAt: base.Add(time.Minute)}, status: StatusUpdating, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(4 * time.Minute)},
		{name: "after observation", activity: HostActivity{Action: valid.Action, StartedAt: valid.StartedAt, LastHeartbeatAt: base.Add(4 * time.Minute)}, status: StatusUpdating, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(3 * time.Minute), now: base.Add(5 * time.Minute)},
		{name: "future clock", activity: HostActivity{Action: valid.Action, StartedAt: valid.StartedAt, LastHeartbeatAt: base.Add(11 * time.Minute)}, status: StatusUpdating, mode: ExecutionModeFull, startedAt: base, observedAt: base.Add(12 * time.Minute), now: base.Add(5 * time.Minute)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateHostActivity(&test.activity, test.status, test.mode, test.startedAt, test.observedAt, test.now); err == nil {
				t.Fatal("ValidateHostActivity() accepted invalid activity")
			}
		})
	}
}

func TestValidateHostActionProgressEventUsesClosedMetadata(t *testing.T) {
	if err := ValidateHostActionProgressEvent(HostActionProgressStarted, "Host upgrade action started", map[string]string{"action": string(HostActionPullImages)}); err != nil {
		t.Fatalf("valid start event rejected: %v", err)
	}
	if err := ValidateHostActionProgressEvent(HostActionProgressFailed, "Host upgrade action failed", map[string]string{"action": string(HostActionPullImages), "reason": string(HostActionFailureCommandFailed)}); err != nil {
		t.Fatalf("valid failure event rejected: %v", err)
	}
	for _, metadata := range []map[string]string{
		{"action": "shell"},
		{"action": string(HostActionPullImages), "reason": "stderr"},
		{"action": string(HostActionPullImages), "unexpected": "value"},
	} {
		if err := ValidateHostActionProgressEvent(HostActionProgressFailed, "Host upgrade action failed", metadata); err == nil {
			t.Fatalf("invalid metadata accepted: %#v", metadata)
		}
	}
}
