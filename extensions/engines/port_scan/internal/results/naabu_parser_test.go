package results

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func TestStreamNaabuHostPortsNormalizesValidRowsAndSkipsInvalidRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "naabu.jsonl")
	content := "not-json\n" +
		`{"host":" Api.Example.COM. ","ip":" 192.0.2.10 ","port":443}` + "\n" +
		`{"host":" \t ","ip":"192.0.2.11","port":22}` + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.12","port":0}` + "\n" +
		`{"host":"bad","ip":"not-ip","port":80}` + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.13","port":8443}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.HostPort
	count, err := StreamNaabuHostPorts(context.Background(), []string{path}, func(item contractresults.HostPort) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamNaabuHostPorts failed: %v", err)
	}
	if count != 3 || len(items) != 3 {
		t.Fatalf("expected 3 parseable items, count=%d items=%#v", count, items)
	}
	if items[0] != (contractresults.HostPort{Host: "api.example.com", IP: "192.0.2.10", Port: 443}) {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1] != (contractresults.HostPort{Host: "192.0.2.11", IP: "192.0.2.11", Port: 22}) {
		t.Fatalf("unexpected fallback item: %#v", items[1])
	}
	if items[2] != (contractresults.HostPort{Host: "api.example.com", IP: "192.0.2.13", Port: 8443}) {
		t.Fatalf("unexpected trailing valid item: %#v", items[2])
	}
}

func TestStreamNaabuHostPortsWithSummaryCountsMalformedAndInvalidRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "naabu.jsonl")
	content := "not-json\n" +
		`{"host":"api.example.com","ip":"192.0.2.10","port":443}` + "\n" +
		`{"host":"bad","ip":"not-ip","port":80}` + "\n" +
		`{"host":"","ip":"not-ip","port":22}` + "\n" +
		`{"host":"api.example.com","ip":"2001:db8::1","port":443}` + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.12","port":70000}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.HostPort
	summary, err := StreamNaabuHostPortsWithSummary(context.Background(), []string{path}, func(item contractresults.HostPort) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamNaabuHostPortsWithSummary failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one parseable item, got %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 6, ParsedItems: 1, SkippedMalformed: 1, SkippedInvalid: 4}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}

func TestStreamNaabuHostPortsSkipsOversizedRecordAndContinues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "naabu.jsonl")
	content := strings.Repeat("x", maxResultArtifactRecordBytes+1) + "\n" +
		`{"host":"api.example.com","ip":"192.0.2.10","port":443}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write parser fixture: %v", err)
	}

	var items []contractresults.HostPort
	summary, err := StreamNaabuHostPortsWithSummary(context.Background(), []string{path}, func(item contractresults.HostPort) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamNaabuHostPortsWithSummary failed: %v", err)
	}
	if len(items) != 1 || items[0].Host != "api.example.com" {
		t.Fatalf("unexpected submitted items: %#v", items)
	}
	if summary != (ParseSummary{SourceRecords: 2, ParsedItems: 1, SkippedOversized: 1}) {
		t.Fatalf("unexpected parse summary: %#v", summary)
	}
}
