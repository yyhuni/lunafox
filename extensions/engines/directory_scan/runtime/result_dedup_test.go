package directoryscanruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestDirectoryResultDedupProductionBoundsAreFixed(t *testing.T) {
	if directoryResultChunkByteBudget != 8_388_608 {
		t.Fatalf("chunk byte budget = %d, want 8388608", directoryResultChunkByteBudget)
	}
	if directoryResultMergeFanIn != 8 {
		t.Fatalf("merge fan-in = %d, want 8", directoryResultMergeFanIn)
	}
}

func TestDirectoryResultDedupKeepsGreatestSourceOrdinalAcrossChunksAndPasses(t *testing.T) {
	stager := newDirectoryResultDedupTestStager(t, 1, 2)
	observations := []DirectoryObservation{
		testDirectoryObservation("https://example.com/same", 100, 0, 0),
		testDirectoryObservation("https://example.com/A", 201, 9, 0),
		testDirectoryObservation("https://example.com/a", 202, 8, 0),
		testDirectoryObservation("https://example.com/a?x=1", 203, 7, 0),
		testDirectoryObservation("https://example.com/a#x", 204, 6, 0),
		testDirectoryObservation("https://example.com/a/", 205, 5, 0),
		testDirectoryObservation("https://example.com//a", 206, 4, 0),
		testDirectoryObservation("https://example.com/same", 300, 5, 0),
		testDirectoryObservation("https://example.com/b", 207, 3, 0),
		testDirectoryObservation("https://example.com/c", 208, 2, 0),
		testDirectoryObservation("https://example.com/same", 200, 4, 99),
	}
	for _, observation := range observations {
		if err := stager.Add(context.Background(), observation); err != nil {
			t.Fatalf("Add(%q): %v", observation.Item.URL, err)
		}
	}

	winners, stats, err := stager.Finalize(context.Background())
	if err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if stats != (directoryDedupStats{InputItems: 11, WinnerItems: 9, SkippedDuplicate: 2}) {
		t.Fatalf("dedup stats = %#v", stats)
	}
	if stager.mergePasses < 2 || stager.maxOpenReaders > 2 || stager.maxLoadedRecords > 2 || stager.maxBufferedRecords > 1 {
		t.Fatalf("bounds: passes=%d open=%d loaded=%d buffered=%d", stager.mergePasses, stager.maxOpenReaders, stager.maxLoadedRecords, stager.maxBufferedRecords)
	}
	if _, err := os.Stat(winners.path); err != nil {
		t.Fatalf("winner set was not finalized before streaming: %v", err)
	}

	var got []enginecontract.Directory
	if err := winners.Stream(context.Background(), func(item enginecontract.Directory) error {
		got = append(got, item)
		return nil
	}); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(got) != 9 || !sort.SliceIsSorted(got, func(left, right int) bool { return got[left].URL < got[right].URL }) {
		t.Fatalf("winner order = %#v", got)
	}
	for _, item := range got {
		if item.URL == "https://example.com/same" && item.Status != 300 {
			t.Fatalf("same-URL winner = %#v, want candidate ordinal 5", item)
		}
	}
	if err := winners.Stream(context.Background(), func(enginecontract.Directory) error { return nil }); err == nil {
		t.Fatal("winner set allowed a second stream")
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestDirectoryResultDedupRejectsDuplicateSourceOrdinal(t *testing.T) {
	stager := newDirectoryResultDedupTestStager(t, 1, 2)
	first := testDirectoryObservation("https://example.com/same", 200, 1, 2)
	second := testDirectoryObservation("https://example.com/same", 201, 1, 2)
	for _, observation := range []DirectoryObservation{first, second} {
		if err := stager.Add(context.Background(), observation); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := stager.Finalize(context.Background()); err == nil || !strings.Contains(err.Error(), "duplicate source ordinal") {
		t.Fatalf("Finalize() error = %v", err)
	}
	if err := stager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDirectoryResultDedupWinnerSetIsChunkingInvariant(t *testing.T) {
	records := []DirectoryObservation{
		testDirectoryObservation("https://example.com/b", 200, 0, 0),
		testDirectoryObservation("https://example.com/a", 201, 1, 0),
		testDirectoryObservation("https://example.com/b", 202, 2, 0),
		testDirectoryObservation("https://example.com/c", 203, 3, 0),
		testDirectoryObservation("https://example.com/a", 204, 4, 0),
	}
	finalize := func(chunkBudget int64, fanIn int) ([]enginecontract.Directory, directoryDedupStats) {
		stager := newDirectoryResultDedupTestStager(t, chunkBudget, fanIn)
		for _, record := range records {
			if err := stager.Add(context.Background(), record); err != nil {
				t.Fatal(err)
			}
		}
		winners, stats, err := stager.Finalize(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		var items []enginecontract.Directory
		if err := winners.Stream(context.Background(), func(item enginecontract.Directory) error {
			items = append(items, item)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := stager.Close(); err != nil {
			t.Fatal(err)
		}
		return items, stats
	}

	smallItems, smallStats := finalize(1, 2)
	largeItems, largeStats := finalize(1<<20, 8)
	if !reflect.DeepEqual(smallItems, largeItems) || smallStats != largeStats {
		t.Fatalf("chunk-dependent winners: small=%#v/%#v large=%#v/%#v", smallItems, smallStats, largeItems, largeStats)
	}
}

func TestDirectoryResultDedupPropagatesStagingMergeStreamAndCancellationFaults(t *testing.T) {
	wantCreateErr := errors.New("create failed")
	t.Run("staging", func(t *testing.T) {
		stager := newDirectoryResultDedupTestStager(t, 1, 2)
		stager.files.create = func(string) (directoryDedupWriteCloser, error) { return nil, wantCreateErr }
		if err := stager.Add(context.Background(), testDirectoryObservation("https://example.com/a", 200, 0, 0)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := stager.Finalize(context.Background()); !errors.Is(err, wantCreateErr) {
			t.Fatalf("Finalize() error = %v", err)
		}
		_ = stager.Close()
	})

	t.Run("merge", func(t *testing.T) {
		stager := newDirectoryResultDedupTestStager(t, 1, 2)
		for ordinal := uint64(0); ordinal < 5; ordinal++ {
			if err := stager.Add(context.Background(), testDirectoryObservation("https://example.com/"+string(rune('a'+ordinal)), 200, ordinal, 0)); err != nil {
				t.Fatal(err)
			}
		}
		realCreate := stager.files.create
		stager.files.create = func(path string) (directoryDedupWriteCloser, error) {
			if strings.HasPrefix(filepath.Base(path), "merge-") {
				return nil, wantCreateErr
			}
			return realCreate(path)
		}
		if _, _, err := stager.Finalize(context.Background()); !errors.Is(err, wantCreateErr) {
			t.Fatalf("Finalize() error = %v", err)
		}
		_ = stager.Close()
	})

	t.Run("cancellation", func(t *testing.T) {
		stager := newDirectoryResultDedupTestStager(t, 1, 2)
		if err := stager.Add(context.Background(), testDirectoryObservation("https://example.com/a", 200, 0, 0)); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, _, err := stager.Finalize(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Finalize() error = %v", err)
		}
		_ = stager.Close()
	})

	t.Run("winner read", func(t *testing.T) {
		stager := newDirectoryResultDedupTestStager(t, 1024, 2)
		if err := stager.Add(context.Background(), testDirectoryObservation("https://example.com/a", 200, 0, 0)); err != nil {
			t.Fatal(err)
		}
		winners, _, err := stager.Finalize(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(winners.path, 1); err != nil {
			t.Fatal(err)
		}
		if err := winners.Stream(context.Background(), func(enginecontract.Directory) error { return nil }); err == nil {
			t.Fatal("Stream() accepted a truncated winner file")
		}
		_ = stager.Close()
	})

	t.Run("emit", func(t *testing.T) {
		stager := newDirectoryResultDedupTestStager(t, 1024, 2)
		if err := stager.Add(context.Background(), testDirectoryObservation("https://example.com/a", 200, 0, 0)); err != nil {
			t.Fatal(err)
		}
		winners, _, err := stager.Finalize(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		wantErr := errors.New("emit failed")
		if err := winners.Stream(context.Background(), func(enginecontract.Directory) error { return wantErr }); !errors.Is(err, wantErr) {
			t.Fatalf("Stream() error = %v", err)
		}
		_ = stager.Close()
	})
}

func TestDirectoryResultDedupCleanupIsPrivateIdempotentAndRetryable(t *testing.T) {
	workspace := t.TempDir()
	rawPath := filepath.Join(workspace, "ffuf-00000000000000000000.jsonl")
	if err := os.WriteFile(rawPath, []byte("raw"), 0o600); err != nil {
		t.Fatal(err)
	}
	stager, err := newDirectoryDedupStager(directoryDedupOptions{Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	tempDir := stager.tempDir
	realRemoveAll := stager.files.removeAll
	wantErr := errors.New("cleanup failed")
	calls := 0
	stager.files.removeAll = func(path string) error {
		calls++
		if calls == 1 {
			return wantErr
		}
		return realRemoveAll(path)
	}
	if err := stager.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("first Close() error = %v", err)
	}
	if _, err := os.Stat(tempDir); err != nil {
		t.Fatalf("failed cleanup unexpectedly removed private directory: %v", err)
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("retry Close() error = %v", err)
	}
	if err := stager.Close(); err != nil {
		t.Fatalf("idempotent Close() error = %v", err)
	}
	if _, err := os.Stat(tempDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("private directory remains: %v", err)
	}
	if payload, err := os.ReadFile(rawPath); err != nil || string(payload) != "raw" {
		t.Fatalf("raw artifact changed: %q, %v", payload, err)
	}
}

func newDirectoryResultDedupTestStager(t *testing.T, chunkBudget int64, mergeFanIn int) *directoryDedupStager {
	t.Helper()
	stager, err := newDirectoryDedupStager(directoryDedupOptions{
		Workspace:       t.TempDir(),
		ChunkByteBudget: chunkBudget,
		MergeFanIn:      mergeFanIn,
	})
	if err != nil {
		t.Fatalf("newDirectoryDedupStager() error = %v", err)
	}
	return stager
}

func testDirectoryObservation(url string, status int, candidateOrdinal, physicalOrdinal uint64) DirectoryObservation {
	return DirectoryObservation{
		Item: enginecontract.Directory{
			URL: url, Status: status, ContentLength: int64(status), ContentType: "text/plain", Duration: int64(status) + 1,
		},
		CandidateOrdinal:      candidateOrdinal,
		PhysicalRecordOrdinal: physicalOrdinal,
	}
}
