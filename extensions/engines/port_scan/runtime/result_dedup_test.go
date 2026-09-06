package portscanruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestResultDedupAddFlushesChunksByByteBudget(t *testing.T) {
	stager := newResultDedupTestStager(t, 100, 2)
	payload := []byte(strings.Repeat("x", 64))
	if err := stager.Add(context.Background(), "first", payload); err != nil {
		t.Fatalf("add first oversized record: %v", err)
	}
	if stager.chunkCount != 0 {
		t.Fatalf("unexpected flushed chunks after one record: %d", stager.chunkCount)
	}
	if err := stager.Add(context.Background(), "second", payload); err != nil {
		t.Fatalf("add second oversized record: %v", err)
	}
	if stager.chunkCount != 1 {
		t.Fatalf("expected first record to flush by byte budget, chunks=%d", stager.chunkCount)
	}
	if len(stager.chunk) != 1 {
		t.Fatalf("expected current chunk to contain second record, records=%d", len(stager.chunk))
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("close stager: %v", err)
	}
}

func TestResultDedupStreamDeduplicatesAcrossChunksAndKeepsLatestSourceRecord(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 2)
	addResultDedupTestRecords(t, stager,
		resultDedupTestRecord{key: "b.example", payload: "b-first"},
		resultDedupTestRecord{key: "a.example", payload: "a-first"},
		resultDedupTestRecord{key: "b.example", payload: "b-later"},
		resultDedupTestRecord{key: "c.example", payload: "c-only"},
		resultDedupTestRecord{key: "a.example", payload: "a-later"},
	)

	var got []resultDedupRecord
	stats, err := stager.Stream(context.Background(), func(record resultDedupRecord) error {
		got = append(got, record)
		return nil
	})
	if err != nil {
		t.Fatalf("stream records: %v", err)
	}

	want := []resultDedupRecord{
		{Key: "a.example", Payload: []byte("a-later")},
		{Key: "b.example", Payload: []byte("b-later")},
		{Key: "c.example", Payload: []byte("c-only")},
	}
	assertResultDedupRecordsEqual(t, want, got)
	if wantStats := (resultDedupStats{InputCount: 5, UniqueCount: 3, DuplicateCount: 2}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupStreamReducesManyChunksWithinConfiguredFanIn(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 2)
	addResultDedupTestRecords(t, stager,
		resultDedupTestRecord{key: "key-08", payload: "eight"},
		resultDedupTestRecord{key: "key-01", payload: "one"},
		resultDedupTestRecord{key: "key-07", payload: "seven"},
		resultDedupTestRecord{key: "key-02", payload: "two"},
		resultDedupTestRecord{key: "key-06", payload: "six"},
		resultDedupTestRecord{key: "key-03", payload: "three"},
		resultDedupTestRecord{key: "key-05", payload: "five"},
		resultDedupTestRecord{key: "key-04", payload: "four"},
		resultDedupTestRecord{key: "key-00", payload: "zero"},
	)

	var got []string
	stats, err := stager.Stream(context.Background(), func(record resultDedupRecord) error {
		got = append(got, record.Key)
		return nil
	})
	if err != nil {
		t.Fatalf("stream records: %v", err)
	}
	if wantStats := (resultDedupStats{InputCount: 9, UniqueCount: 9}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	if stager.mergePasses < 2 {
		t.Fatalf("expected multi-pass reduction, passes=%d", stager.mergePasses)
	}
	if stager.maxOpenChunkReaders > 2 {
		t.Fatalf("opened too many chunk readers: got %d, fan-in=2", stager.maxOpenChunkReaders)
	}
	if stager.maxOpenChunkReaders != 2 {
		t.Fatalf("expected merge to use configured fan-in, got %d readers", stager.maxOpenChunkReaders)
	}
	if want := []string{"key-00", "key-01", "key-02", "key-03", "key-04", "key-05", "key-06", "key-07", "key-08"}; !resultDedupEqualStrings(got, want) {
		t.Fatalf("unexpected stream order: got %#v, want %#v", got, want)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupStreamLoadsOneOversizedPayloadAtATime(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 4)
	payload := []byte(strings.Repeat("x", 128*1024))
	for index := 0; index < 4; index++ {
		if err := stager.Add(context.Background(), fmt.Sprintf("key-%d", index), payload); err != nil {
			t.Fatalf("add oversized record %d: %v", index, err)
		}
	}

	stats, err := stager.Stream(context.Background(), func(resultDedupRecord) error { return nil })
	if err != nil {
		t.Fatalf("stream oversized records: %v", err)
	}
	if wantStats := (resultDedupStats{InputCount: 4, UniqueCount: 4}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	if stager.maxOpenChunkReaders != 4 {
		t.Fatalf("expected a full fan-in merge, got %d readers", stager.maxOpenChunkReaders)
	}
	if stager.maxLoadedPayloads != 1 {
		t.Fatalf("expected one loaded payload at a time, got %d", stager.maxLoadedPayloads)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupStreamCancellationCleansTemporaryDirectory(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 2)
	addResultDedupTestRecords(t, stager,
		resultDedupTestRecord{key: "a.example", payload: "a"},
		resultDedupTestRecord{key: "b.example", payload: "b"},
		resultDedupTestRecord{key: "c.example", payload: "c"},
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	emitted := 0
	_, err := stager.Stream(ctx, func(resultDedupRecord) error {
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
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupAddCancellationCleansTemporaryDirectory(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 2)
	if err := stager.Add(context.Background(), "a.example", []byte("a")); err != nil {
		t.Fatalf("stage first record: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stager.Add(ctx, "b.example", []byte("b")); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupStreamEmitErrorCleansTemporaryDirectory(t *testing.T) {
	stager := newResultDedupTestStager(t, 1, 2)
	addResultDedupTestRecords(t, stager, resultDedupTestRecord{key: "a.example", payload: "a"})

	expected := errors.New("typed submission failed")
	_, err := stager.Stream(context.Background(), func(resultDedupRecord) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected emit error, got %v", err)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestResultDedupCloseCleansTemporaryDirectoryWithoutStreaming(t *testing.T) {
	stager := newResultDedupTestStager(t, 1024, 2)
	tempDir := stager.tempDir
	if err := stager.Close(); err != nil {
		t.Fatalf("close stager: %v", err)
	}
	assertResultDedupRemoved(t, tempDir)
	if err := stager.Close(); err != nil {
		t.Fatalf("close stager a second time: %v", err)
	}
}

func TestResultDedupCloseRetriesTemporaryCleanupFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not reliably prevent removal on Windows")
	}
	workspace := t.TempDir()
	stager, err := newResultDedupStager(resultDedupOptions{WorkspaceDir: workspace})
	if err != nil {
		t.Fatalf("new stager: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(workspace, 0o700) })
	if err := os.Chmod(workspace, 0o500); err != nil {
		t.Fatalf("make workspace read-only: %v", err)
	}

	firstCloseErr := stager.Close()
	if err := os.Chmod(workspace, 0o700); err != nil {
		t.Fatalf("restore workspace permissions: %v", err)
	}
	if firstCloseErr == nil {
		assertResultDedupRemoved(t, stager.tempDir)
		t.Skip("execution environment permits removal from a read-only parent")
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("retry close after restoring permissions: %v", err)
	}
	assertResultDedupRemoved(t, stager.tempDir)
}

func TestIsResultDedupContextTerminationRejectsJoinedOperationalFailures(t *testing.T) {
	operationalErr := errors.New("cleanup failed")
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "canceled", err: context.Canceled, want: true},
		{name: "wrapped deadline", err: fmt.Errorf("stage: %w", context.DeadlineExceeded), want: true},
		{name: "joined terminations", err: errors.Join(context.Canceled, context.DeadlineExceeded), want: true},
		{name: "joined operational failure", err: errors.Join(context.Canceled, operationalErr), want: false},
		{name: "operational failure", err: operationalErr, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isResultDedupContextTermination(test.err); got != test.want {
				t.Fatalf("isResultDedupContextTermination(%v) = %t, want %t", test.err, got, test.want)
			}
		})
	}
}

func TestResultDedupSubmissionErrorRetainsAllOperationalFailures(t *testing.T) {
	submissionErr := errors.New("submission failed")
	streamErr := errors.New("stream failed")
	closeErr := errors.New("cleanup failed")
	err := resultDedupSubmissionError(submissionErr, streamErr, closeErr)
	for _, expected := range []error{submissionErr, streamErr, closeErr} {
		if !errors.Is(err, expected) {
			t.Fatalf("resultDedupSubmissionError() = %v, missing %v", err, expected)
		}
	}

	err = resultDedupSubmissionError(submissionErr, context.Canceled, nil)
	if !errors.Is(err, submissionErr) {
		t.Fatalf("resultDedupSubmissionError() = %v, missing submission error", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("resultDedupSubmissionError() retained expected cancellation: %v", err)
	}
}

type resultDedupTestRecord struct {
	key     string
	payload string
}

func newResultDedupTestStager(t *testing.T, chunkByteBudget int64, mergeFanIn int) *resultDedupStager {
	t.Helper()
	stager, err := newResultDedupStager(resultDedupOptions{
		WorkspaceDir:    t.TempDir(),
		ChunkByteBudget: chunkByteBudget,
		MergeFanIn:      mergeFanIn,
	})
	if err != nil {
		t.Fatalf("new stager: %v", err)
	}
	return stager
}

func addResultDedupTestRecords(t *testing.T, stager *resultDedupStager, records ...resultDedupTestRecord) {
	t.Helper()
	for _, record := range records {
		if err := stager.Add(context.Background(), record.key, []byte(record.payload)); err != nil {
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
		if want[index].Key != got[index].Key || string(want[index].Payload) != string(got[index].Payload) {
			t.Fatalf("record %d: got %#v, want %#v", index, got[index], want[index])
		}
	}
}

func assertResultDedupRemoved(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be removed, stat error=%v", path, err)
	}
}

func resultDedupEqualStrings(left, right []string) bool {
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
