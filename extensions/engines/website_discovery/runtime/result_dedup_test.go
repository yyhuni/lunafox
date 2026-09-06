package websitediscoveryruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestResultDedupStreamsAcrossChunksAndKeepsLatestRecord(t *testing.T) {
	stager := newWebsiteResultDedupStager(t, 1, 2)
	addWebsiteResultDedupRecords(t, stager,
		websiteResultDedupTestRecord{key: "b.example", payload: "b-first"},
		websiteResultDedupTestRecord{key: "a.example", payload: "a-first"},
		websiteResultDedupTestRecord{key: "b.example", payload: "b-later"},
		websiteResultDedupTestRecord{key: "c.example", payload: "c-only"},
		websiteResultDedupTestRecord{key: "a.example", payload: "a-later"},
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
		{key: "a.example", payload: []byte("a-later")},
		{key: "b.example", payload: []byte("b-later")},
		{key: "c.example", payload: []byte("c-only")},
	}
	assertWebsiteResultDedupRecords(t, want, got)
	if wantStats := (resultDedupStats{inputCount: 5, uniqueCount: 3, duplicateCount: 2}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupBoundsMergeReadersAndLoadedPayloads(t *testing.T) {
	stager := newWebsiteResultDedupStager(t, 1, 4)
	payload := []byte(strings.Repeat("x", 128*1024))
	for index := 0; index < 9; index++ {
		if err := stager.Add(context.Background(), fmt.Sprintf("key-%02d", index), payload); err != nil {
			t.Fatalf("add oversized record %d: %v", index, err)
		}
	}

	stats, err := stager.Stream(context.Background(), func(resultDedupRecord) error { return nil })
	if err != nil {
		t.Fatalf("stream records: %v", err)
	}
	if wantStats := (resultDedupStats{inputCount: 9, uniqueCount: 9}); stats != wantStats {
		t.Fatalf("unexpected stats: got %#v, want %#v", stats, wantStats)
	}
	if stager.mergePasses < 1 {
		t.Fatalf("expected bounded multi-pass reduction, passes=%d", stager.mergePasses)
	}
	if stager.maxOpenChunkReaders != 4 {
		t.Fatalf("opened %d chunk readers, want fan-in 4", stager.maxOpenChunkReaders)
	}
	if stager.maxLoadedPayloads != 1 {
		t.Fatalf("loaded %d payloads concurrently, want 1", stager.maxLoadedPayloads)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupCancellationAndEmitFailureCleanTemporaryDirectory(t *testing.T) {
	t.Run("cancellation", func(t *testing.T) {
		stager := newWebsiteResultDedupStager(t, 1, 2)
		addWebsiteResultDedupRecords(t, stager,
			websiteResultDedupTestRecord{key: "a.example", payload: "a"},
			websiteResultDedupTestRecord{key: "b.example", payload: "b"},
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
			t.Fatalf("stream cancellation error = %v", err)
		}
		if emitted != 1 {
			t.Fatalf("emitted %d records after cancellation, want 1", emitted)
		}
		assertResultDedupTempDirRemoved(t, stager.tempDir)
	})

	t.Run("emit failure", func(t *testing.T) {
		stager := newWebsiteResultDedupStager(t, 1, 2)
		addWebsiteResultDedupRecords(t, stager, websiteResultDedupTestRecord{key: "a.example", payload: "a"})

		expected := errors.New("typed submission failed")
		_, err := stager.Stream(context.Background(), func(resultDedupRecord) error {
			return expected
		})
		if !errors.Is(err, expected) {
			t.Fatalf("stream emit failure error = %v", err)
		}
		assertResultDedupTempDirRemoved(t, stager.tempDir)
	})
}

func TestResultDedupAddCancellationCleansTemporaryDirectory(t *testing.T) {
	stager := newWebsiteResultDedupStager(t, 1, 2)
	if err := stager.Add(context.Background(), "a.example", []byte("a")); err != nil {
		t.Fatalf("stage first record: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := stager.Add(ctx, "b.example", []byte("b")); !errors.Is(err, context.Canceled) {
		t.Fatalf("add cancellation error = %v", err)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestResultDedupCloseRetriesTemporaryCleanupFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not reliably prevent removal on Windows")
	}
	workspace := t.TempDir()
	stager, err := newResultDedupStager(resultDedupOptions{workspaceDir: workspace})
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
		assertResultDedupTempDirRemoved(t, stager.tempDir)
		t.Skip("execution environment permits removal from a read-only parent")
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("retry close after restoring permissions: %v", err)
	}
	assertResultDedupTempDirRemoved(t, stager.tempDir)
}

func TestJoinWebsiteResultSubmissionErrorsPreservesAllOperationalFailures(t *testing.T) {
	submissionErr := errors.New("result acknowledgement failed")
	streamErr := errors.New("decode staged result failed")
	cleanupErr := errors.New("remove staging failed")
	err := joinWebsiteResultSubmissionErrors(submissionErr, streamErr, cleanupErr)
	for _, expected := range []error{submissionErr, streamErr, cleanupErr} {
		if !errors.Is(err, expected) {
			t.Fatalf("joined error %v does not contain %v", err, expected)
		}
	}

	err = joinWebsiteResultSubmissionErrors(submissionErr, context.Canceled, nil)
	if !errors.Is(err, submissionErr) {
		t.Fatalf("joined cancellation error does not contain submission error: %v", err)
	}
	if strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("expected cancellation caused by submission shutdown to be omitted, got %v", err)
	}
}

type websiteResultDedupTestRecord struct {
	key     string
	payload string
}

func newWebsiteResultDedupStager(t *testing.T, chunkByteBudget int64, mergeFanIn int) *resultDedupStager {
	t.Helper()
	stager, err := newResultDedupStager(resultDedupOptions{
		workspaceDir:    t.TempDir(),
		chunkByteBudget: chunkByteBudget,
		mergeFanIn:      mergeFanIn,
	})
	if err != nil {
		t.Fatalf("new stager: %v", err)
	}
	return stager
}

func addWebsiteResultDedupRecords(t *testing.T, stager *resultDedupStager, records ...websiteResultDedupTestRecord) {
	t.Helper()
	for _, record := range records {
		if err := stager.Add(context.Background(), record.key, []byte(record.payload)); err != nil {
			t.Fatalf("add %q: %v", record.key, err)
		}
	}
}

func assertWebsiteResultDedupRecords(t *testing.T, want, got []resultDedupRecord) {
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
