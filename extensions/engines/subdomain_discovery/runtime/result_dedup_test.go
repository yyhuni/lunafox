package subdomaindiscoveryruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestResultDedupStreamDeduplicatesAcrossChunksAndKeepsLatestSourceRecord(t *testing.T) {
	stager := newTestResultDedupStager(t, 1, 2)
	addResultDedupTestRecords(t, stager,
		resultDedupTestRecord{key: "b.example", payload: "b-first"},
		resultDedupTestRecord{key: "a.example", payload: "a-first"},
		resultDedupTestRecord{key: "b.example", payload: "b-later"},
		resultDedupTestRecord{key: "c.example", payload: "c-only"},
		resultDedupTestRecord{key: "a.example", payload: "a-later"},
	)

	var got []resultDedupRecord
	stats, err := stager.stream(context.Background(), func(record resultDedupRecord) error {
		got = append(got, record)
		return nil
	})
	if err != nil {
		t.Fatalf("stream records: %v", err)
	}

	want := []resultDedupRecord{
		{key: "a.example", payload: []byte("a-later")},
		{key: "b.example", payload: []byte("b-later")},
		{key: "c.example", payload: []byte("c-only")},
	}
	assertResultDedupRecordsEqual(t, want, got)
	if wantStats := (resultDedupStats{inputCount: 5, uniqueCount: 3, duplicateCount: 2}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupStreamBoundsMergeReadersAndLoadedPayloads(t *testing.T) {
	stager := newTestResultDedupStager(t, 1, 2)
	payload := strings.Repeat("x", 128*1024)
	for index := 8; index >= 0; index-- {
		if err := stager.add(context.Background(), fmt.Sprintf("key-%02d", index), []byte(payload)); err != nil {
			t.Fatalf("add record %d: %v", index, err)
		}
	}

	var got []string
	stats, err := stager.stream(context.Background(), func(record resultDedupRecord) error {
		got = append(got, record.key)
		return nil
	})
	if err != nil {
		t.Fatalf("stream records: %v", err)
	}
	if wantStats := (resultDedupStats{inputCount: 9, uniqueCount: 9}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	if stager.mergePasses < 2 {
		t.Fatalf("expected multi-pass reduction, passes=%d", stager.mergePasses)
	}
	if stager.maxOpenChunkReaders != 2 {
		t.Fatalf("opened %d chunk readers, want configured fan-in 2", stager.maxOpenChunkReaders)
	}
	if stager.maxLoadedPayloads != 1 {
		t.Fatalf("loaded %d payloads at once, want 1", stager.maxLoadedPayloads)
	}
	want := []string{"key-00", "key-01", "key-02", "key-03", "key-04", "key-05", "key-06", "key-07", "key-08"}
	if !resultDedupStringsEqual(got, want) {
		t.Fatalf("unexpected stream order: got %#v, want %#v", got, want)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupStreamCancellationCleansTemporaryDirectory(t *testing.T) {
	stager := newTestResultDedupStager(t, 1, 2)
	addResultDedupTestRecords(t, stager,
		resultDedupTestRecord{key: "a.example", payload: "a"},
		resultDedupTestRecord{key: "b.example", payload: "b"},
		resultDedupTestRecord{key: "c.example", payload: "c"},
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	emitted := 0
	_, err := stager.stream(ctx, func(resultDedupRecord) error {
		emitted++
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if emitted != 1 {
		t.Fatalf("expected one emitted record before cancellation, got %d", emitted)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupStreamEmitErrorCleansTemporaryDirectory(t *testing.T) {
	stager := newTestResultDedupStager(t, 1, 2)
	addResultDedupTestRecords(t, stager, resultDedupTestRecord{key: "a.example", payload: "a"})

	expected := errors.New("typed submission failed")
	_, err := stager.stream(context.Background(), func(resultDedupRecord) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected emit error, got %v", err)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupAddCancellationCleansTemporaryDirectory(t *testing.T) {
	stager := newTestResultDedupStager(t, 1, 2)
	if err := stager.add(context.Background(), "a.example", []byte("a")); err != nil {
		t.Fatalf("stage first record: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stager.add(ctx, "b.example", []byte("b")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultSubmissionFailureAggregatesStreamAndCleanupErrors(t *testing.T) {
	submissionErr := errors.New("submission failed")
	streamErr := errors.New("stream failed")
	cleanupErr := errors.New("cleanup failed")

	err := resultSubmissionFailure(submissionErr, streamErr, cleanupErr)
	for _, expected := range []error{submissionErr, streamErr, cleanupErr} {
		if !errors.Is(err, expected) {
			t.Fatalf("aggregated error %v does not contain %v", err, expected)
		}
	}
}

func TestResultSubmissionFailureSuppressesExpectedCancellationOnly(t *testing.T) {
	submissionErr := errors.New("submission failed")
	err := resultSubmissionFailure(submissionErr, context.Canceled, nil)
	if !errors.Is(err, submissionErr) {
		t.Fatalf("aggregated error %v does not contain submission error", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("aggregated error unexpectedly contains cancellation: %v", err)
	}
}

type resultDedupTestRecord struct {
	key     string
	payload string
}

func newTestResultDedupStager(t *testing.T, chunkByteBudget int64, mergeFanIn int) *resultDedupStager {
	t.Helper()
	stager, err := newResultDedupStager(resultDedupOptions{
		workspaceDir:    t.TempDir(),
		chunkByteBudget: chunkByteBudget,
		mergeFanIn:      mergeFanIn,
	})
	if err != nil {
		t.Fatalf("new result dedup stager: %v", err)
	}
	return stager
}

func addResultDedupTestRecords(t *testing.T, stager *resultDedupStager, records ...resultDedupTestRecord) {
	t.Helper()
	for _, record := range records {
		if err := stager.add(context.Background(), record.key, []byte(record.payload)); err != nil {
			t.Fatalf("add %q: %v", record.key, err)
		}
	}
}

func assertResultDedupRecordsEqual(t *testing.T, want, got []resultDedupRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected record count: got %d, want %d; got %#v", len(got), len(want), got)
	}
	for index := range want {
		if want[index].key != got[index].key || string(want[index].payload) != string(got[index].payload) {
			t.Fatalf("record %d: got %#v, want %#v", index, got[index], want[index])
		}
	}
}

func assertResultDedupTempDirRemoved(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be removed, stat error=%v", path, err)
	}
}

func resultDedupStringsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
