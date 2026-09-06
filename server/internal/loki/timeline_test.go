package loki

import (
	"strings"
	"testing"
)

func TestBuildTimelineEntriesNormalizesStableOrderAndIDs(t *testing.T) {
	entries := BuildTimelineEntries([]StreamResult{
		{
			Stream: map[string]string{"source": "stderr"},
			Values: []StreamValue{
				{TsNs: "1740381601000000000", Line: "same\n"},
				{TsNs: "1740381601000000000", Line: "same"},
			},
		},
		{
			Stream: map[string]string{},
			Values: []StreamValue{
				{TsNs: "1740381600000000000", Line: "oldest"},
			},
		},
	}, TimelineBuildOptions{
		IDPrefix:     "src:lunafox",
		MaxLineBytes: 8,
	}, "BACKWARD")

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Line != "oldest" || entries[0].Stream != "stdout" {
		t.Fatalf("expected default stdout oldest entry first, got %+v", entries[0])
	}
	if entries[1].Line != "same" || entries[2].Line != "same" {
		t.Fatalf("expected newline-trimmed duplicate lines, got %+v", entries)
	}
	if entries[1].ID == entries[2].ID {
		t.Fatalf("expected duplicate log lines to receive unique IDs")
	}
	if !strings.HasPrefix(entries[1].ID, "src:lunafox:1740381601000000000:stderr:") {
		t.Fatalf("unexpected ID prefix: %q", entries[1].ID)
	}
}

func TestTimelineCursorFiltersUseSameOrderingKeyAsEntries(t *testing.T) {
	entries := BuildTimelineEntries([]StreamResult{
		{
			Stream: map[string]string{"source": "stdout"},
			Values: []StreamValue{
				{TsNs: "1740381601000000000", Line: "line-a"},
				{TsNs: "1740381601000000000", Line: "line-a"},
				{TsNs: "1740381601000000001", Line: "line-b"},
			},
		},
	}, TimelineBuildOptions{
		IDPrefix:     "src:lunafox",
		MaxLineBytes: 16,
	}, "FORWARD")

	after := FilterTimelineEntriesAfterCursor(entries, TimelineCursorKey{
		LastTsNs:       entries[0].TSNs,
		LastID:         entries[0].ID,
		LastStream:     entries[0].Stream,
		LastLineHash:   entries[0].LineHash,
		LastOccurrence: entries[0].Occurrence,
	})
	if len(after) != 2 {
		t.Fatalf("expected two entries after first duplicate, got %d", len(after))
	}

	before := FilterTimelineEntriesBeforeCursor(entries, TimelineCursorKey{
		LastTsNs:       entries[2].TSNs,
		LastID:         entries[2].ID,
		LastStream:     entries[2].Stream,
		LastLineHash:   entries[2].LineHash,
		LastOccurrence: entries[2].Occurrence,
	})
	if len(before) != 2 {
		t.Fatalf("expected two entries before final line, got %d", len(before))
	}
}
