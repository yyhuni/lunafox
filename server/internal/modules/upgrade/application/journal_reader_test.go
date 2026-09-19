package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const journalReaderTestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestFileJournalEventReaderCarriesStageTimestampAndProgressEvents(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	journal := hostJournal{
		SchemaVersion: hostJournalSchema, OperationID: "11111111-1111-4111-8111-111111111111", ManifestDigest: journalReaderTestDigest,
		Stage: "updating", StartedAt: base, UpdatedAt: base.Add(2 * time.Minute), StageUpdatedAt: base,
		ProgressEvents: []hostProgressEvent{{
			Timestamp: base.Add(time.Minute), Stage: "updating", MessageKey: "pullImagesStarted",
			Message: "Pulling release images", Metadata: map[string]string{"scope": "release"},
		}},
	}
	writeJournalReaderFixture(t, root, journal, false)

	reader, err := NewFileJournalEventReader(root)
	if err != nil {
		t.Fatal(err)
	}
	event, err := reader.ReadCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !event.FromJournal || !event.StageUpdatedAt.Equal(base) || len(event.ProgressEvents) != 1 {
		t.Fatalf("host event = %#v", event)
	}
	if event.ProgressEvents[0].MessageKey != "pullImagesStarted" || event.ProgressEvents[0].Metadata["scope"] != "release" {
		t.Fatalf("progress event = %#v", event.ProgressEvents)
	}
}

func TestFileJournalEventReaderAcceptsLegacyJournalWithoutProgressFields(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	journal := hostJournal{
		SchemaVersion: hostJournalSchema, OperationID: "22222222-2222-4222-8222-222222222222", ManifestDigest: journalReaderTestDigest,
		Stage: "updating", StartedAt: base, UpdatedAt: base.Add(time.Minute),
	}
	writeJournalReaderFixture(t, root, journal, true)

	reader, err := NewFileJournalEventReader(root)
	if err != nil {
		t.Fatal(err)
	}
	event, err := reader.ReadCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !event.StageUpdatedAt.IsZero() || len(event.ProgressEvents) != 0 {
		t.Fatalf("legacy journal event = %#v", event)
	}
}

func TestFileJournalEventReaderRejectsConflictingSameTimeProgressHistory(t *testing.T) {
	root := t.TempDir()
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	journal := hostJournal{
		SchemaVersion: hostJournalSchema, OperationID: "33333333-3333-4333-8333-333333333333", ManifestDigest: journalReaderTestDigest,
		Stage: "updating", StartedAt: base, UpdatedAt: base.Add(2 * time.Minute), StageUpdatedAt: base,
		ProgressEvents: []hostProgressEvent{{Timestamp: base.Add(time.Minute), Stage: "updating", MessageKey: "pullImagesStarted", Message: "Pulling release images", Metadata: map[string]string{}}},
	}
	writeJournalReaderFixture(t, root, journal, false)
	conflictingHistory := journal
	conflictingHistory.ProgressEvents = []hostProgressEvent{{Timestamp: base.Add(time.Minute), Stage: "updating", MessageKey: "servicesUpdateStarted", Message: "Updating core services", Metadata: map[string]string{}}}
	encoded, err := json.Marshal(conflictingHistory)
	if err != nil {
		t.Fatal(err)
	}
	historyPath := filepath.Join(root, hostJournalDirectory, hostJournalHistoryDir, journal.OperationID+".json")
	if err := os.WriteFile(historyPath, encoded, 0o600); err != nil {
		t.Fatal(err)
	}

	reader, err := NewFileJournalEventReader(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadCurrent(context.Background()); err == nil {
		t.Fatal("conflicting same-time journal history was accepted")
	}
}

func writeJournalReaderFixture(t *testing.T, root string, journal hostJournal, legacy bool) {
	t.Helper()
	journalDirectory := filepath.Join(root, hostJournalDirectory)
	for _, directory := range []string{
		journalDirectory,
		filepath.Join(journalDirectory, hostJournalHistoryDir),
		filepath.Join(journalDirectory, hostJournalReceiptDir),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := json.Marshal(journal)
	if err != nil {
		t.Fatal(err)
	}
	if legacy {
		var document map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &document); err != nil {
			t.Fatal(err)
		}
		delete(document, "stageUpdatedAt")
		delete(document, "progressEvents")
		encoded, err = json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{
		filepath.Join(journalDirectory, hostJournalCurrentFile),
		filepath.Join(journalDirectory, hostJournalHistoryDir, journal.OperationID+".json"),
	} {
		if err := os.WriteFile(path, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
