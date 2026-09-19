package domain

import (
	"strings"
	"testing"
	"time"
)

func TestMergeProgressEventsOrdersDeduplicatesAndBounds(t *testing.T) {
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	existing := []ProgressEvent{{
		Timestamp: base.Add(time.Minute), Stage: StatusUpdating, MessageKey: "first", Message: "First progress checkpoint", Metadata: map[string]string{},
	}}
	incoming := make([]ProgressEvent, 0, MaxProgressEvents+1)
	incoming = append(incoming, existing[0])
	for index := 0; index < MaxProgressEvents; index++ {
		incoming = append(incoming, ProgressEvent{
			Timestamp: base.Add(time.Duration(index+2) * time.Minute), Stage: StatusUpdating,
			MessageKey: "checkpoint-" + strings.Repeat("x", index%2), Message: "Progress checkpoint reached", Metadata: map[string]string{},
		})
	}

	merged, changed, err := MergeProgressEvents(existing, incoming)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || len(merged) != MaxProgressEvents {
		t.Fatalf("merge result = %#v, changed=%t", merged, changed)
	}
	if !merged[0].Timestamp.Equal(base.Add(2 * time.Minute)) {
		t.Fatalf("oldest retained event = %#v", merged[0])
	}
	for index := 1; index < len(merged); index++ {
		if merged[index].Timestamp.Before(merged[index-1].Timestamp) {
			t.Fatalf("merged events are not ordered: %#v", merged)
		}
	}
	replayed, replayChanged, err := MergeProgressEvents(merged, incoming)
	if err != nil || replayChanged || len(replayed) != len(merged) {
		t.Fatalf("replayed merge = %#v, changed=%t, err=%v", replayed, replayChanged, err)
	}
}

func TestProgressEventRejectsUnsafeTextAndMetadata(t *testing.T) {
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	unsafe := []ProgressEvent{
		{Timestamp: base, Stage: StatusUpdating, MessageKey: "unsafe", Message: "first\nsecond", Metadata: map[string]string{}},
		{Timestamp: base, Stage: StatusUpdating, MessageKey: "unsafe", Message: "Reading /deployment/.env", Metadata: map[string]string{}},
		{Timestamp: base, Stage: StatusUpdating, MessageKey: "unsafe", Message: "docker compose stderr", Metadata: map[string]string{}},
		{Timestamp: base, Stage: StatusUpdating, MessageKey: "unsafe", Message: "Safe event", Metadata: map[string]string{"secret": "value"}},
	}
	for _, event := range unsafe {
		if err := event.Validate(); err == nil {
			t.Fatalf("unsafe progress event accepted: %#v", event)
		}
	}
}
