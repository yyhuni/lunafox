package results

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

type captureSubdomainSink struct {
	items []contractresults.Subdomain
	err   error
}

func (sink *captureSubdomainSink) Submit(item contractresults.Subdomain) error {
	if sink.err != nil {
		return sink.err
	}
	sink.items = append(sink.items, item)
	return nil
}

func TestStreamSubdomainsNormalizesAndStreamsCanonicalRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "results.txt")
	if err := os.WriteFile(path, []byte("A.example.com.\n\n# comment\nbad domain\n127.0.0.1\nb.example.com\na.example.com\nC.Example.COM\n"), 0644); err != nil {
		t.Fatalf("write final result artifact: %v", err)
	}

	sink := &captureSubdomainSink{}
	count, err := StreamSubdomains(context.Background(), path, sink.Submit)
	if err != nil {
		t.Fatalf("StreamSubdomains returned error: %v", err)
	}
	if count != 4 || len(sink.items) != 4 {
		t.Fatalf("expected 4 canonical subdomains, count=%d items=%#v", count, sink.items)
	}
	if sink.items[0].DNSName != "a.example.com" || sink.items[1].DNSName != "b.example.com" || sink.items[2].DNSName != "a.example.com" || sink.items[3].DNSName != "c.example.com" {
		t.Fatalf("unexpected subdomain order: %#v", sink.items)
	}
}

func TestStreamSubdomainsWithSummaryCountsInvalidRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "items.txt")
	if err := os.WriteFile(path, []byte("A.example.com.\n\n# comment\nbad domain\n127.0.0.1\na.example.com\nb.example.com\n"), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.Subdomain
	summary, err := StreamSubdomainsWithSummary(context.Background(), path, func(item contractresults.Subdomain) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSubdomainsWithSummary failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected three canonical items, got %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 7, ParsedItems: 3, SkippedInvalid: 4}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamSubdomainsStopsOnSubmitError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "items.txt")
	if err := os.WriteFile(path, []byte("a.example.com\nb.example.com\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	expected := errors.New("submit failed")

	_, err := StreamSubdomains(context.Background(), path, (&captureSubdomainSink{err: expected}).Submit)
	if !errors.Is(err, expected) {
		t.Fatalf("expected submit error, got %v", err)
	}
}

func TestStreamSubdomainsSkipsOversizedRecordAndContinues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.txt")
	content := strings.Repeat("a", maxResultArtifactRecordBytes+1) + "\napi.example.com\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write long file: %v", err)
	}

	var items []contractresults.Subdomain
	summary, err := StreamSubdomainsWithSummary(context.Background(), path, func(item contractresults.Subdomain) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSubdomainsWithSummary failed: %v", err)
	}
	if len(items) != 1 || items[0].DNSName != "api.example.com" {
		t.Fatalf("unexpected submitted items: %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 2, ParsedItems: 1, SkippedOversized: 1}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamSubdomainsRejectsMissingFinalArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.txt")
	_, err := StreamSubdomains(context.Background(), path, func(contractresults.Subdomain) error { return nil })
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
}
