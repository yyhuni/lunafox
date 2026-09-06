package urlcollectionruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSortAndDeduplicateURLsUsesDeterministicExternalMerge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidates.txt")
	lines := make([]string, 0, 40)
	for index := 39; index >= 0; index-- {
		lines = append(lines, "https://example.com/"+string(rune('a'+index%4)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	unique, duplicates, err := sortAndDeduplicateURLs(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if unique != 4 || duplicates != 36 {
		t.Fatalf("sort summary = unique:%d duplicates:%d", unique, duplicates)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "https://example.com/a\nhttps://example.com/b\nhttps://example.com/c\nhttps://example.com/d\n"; got != want {
		t.Fatalf("sorted candidates = %q, want %q", got, want)
	}
}

func TestSortURLLinesUsesMultipleChunkAndMergeRounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidates.txt")
	lines := make([]string, 0, 18)
	for index := 17; index >= 0; index-- {
		lines = append(lines, "https://example.com/"+string(rune('a'+index%6)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	unique, total, err := sortURLLinesWithOptions(context.Background(), path, true, urlSortOptions{chunkByteBudget: 24, mergeFanIn: 2})
	if err != nil {
		t.Fatal(err)
	}
	if unique != 6 || total != 18 {
		t.Fatalf("sort summary = unique:%d total:%d", unique, total)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "https://example.com/a\nhttps://example.com/b\nhttps://example.com/c\nhttps://example.com/d\nhttps://example.com/e\nhttps://example.com/f\n"; got != want {
		t.Fatalf("sorted candidates = %q, want %q", got, want)
	}
}
