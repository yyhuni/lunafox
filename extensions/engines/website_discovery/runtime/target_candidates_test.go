package websitediscoveryruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

func TestPrepareTargetCandidatesConvertsHostPortFactsDeterministically(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	content := "{" + `"host":"z.example.com","ip":"192.0.2.30","port":8443` + "}\n" +
		"{" + `"host":"api.example.com","ip":"192.0.2.31","port":443` + "}\n" +
		"{" + `"host":"api.example.com","ip":"192.0.2.32","port":443` + "}\n" +
		"{" + `"host":"192.0.2.20","ip":"192.0.2.20","port":80` + "}\n"
	if err := os.WriteFile(facts, []byte(content), 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetCandidates() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://example.com\nhttps://example.com\nhttp://192.0.2.20\nhttps://api.example.com\nhttp://z.example.com:8443\nhttps://z.example.com:8443\n"
	if string(got) != want {
		t.Fatalf("candidate bytes = %q, want %q", got, want)
	}
	if count != 6 {
		t.Fatalf("candidate count = %d, want 6", count)
	}
}

func TestPrepareTargetCandidatesExpandsCIDRAndKeepsBaselineNonEmpty(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	if err := os.WriteFile(facts, nil, 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeCIDR, Value: "192.0.2.0/31"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetCandidates() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://192.0.2.0\nhttps://192.0.2.0\nhttp://192.0.2.1\nhttps://192.0.2.1\n"
	if string(got) != want || count != 4 {
		t.Fatalf("CIDR candidates = %q count=%d, want %q count=4", got, count, want)
	}
}

func TestPrepareTargetCandidatesRejectsMalformedFactBeforeHTTPX(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	if err := os.WriteFile(facts, []byte(`{"host":"api.example.com","ip":"192.0.2.10"}`+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	path, _, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeIP, Value: "192.0.2.1"}, facts, workspace)
	if err == nil || path != "" {
		t.Fatalf("malformed preparation = path %q err %v, want failure", path, err)
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "httpx-candidates.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("incomplete HTTPX candidate file remains: %v", statErr)
	}
}

func TestPrepareTargetCandidatesRejectsIncompleteHostPortBeforeHTTPX(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	if err := os.WriteFile(facts, []byte(`{"host":"","ip":"192.0.2.10","port":443}`+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	path, _, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeIP, Value: "192.0.2.1"}, facts, workspace)
	if err == nil || path != "" {
		t.Fatalf("incomplete HostPort preparation = path %q err %v, want failure", path, err)
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "httpx-candidates.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("incomplete HTTPX candidate file remains: %v", statErr)
	}
}

func TestPrepareTargetCandidatesProcessesLargeCrossChunkInputWithoutTruncation(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	file, err := os.OpenFile(facts, os.O_CREATE|os.O_WRONLY, 0o400)
	if err != nil {
		t.Fatal(err)
	}
	const records = 90000
	for index := 0; index < records; index++ {
		host := fmt.Sprintf("host%05d.example.com", index%40000)
		line := fmt.Sprintf(`{"host":%q,"ip":"192.0.2.%d","port":%d}`+"\n", host, (index%250)+1, 8000+(index%3))
		if _, err := file.WriteString(line); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 || count > uint64(records*2+2) {
		t.Fatalf("unexpected candidate count %d for %d facts", count, records)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(payload), "http://example.com\nhttps://example.com\n") {
		t.Fatalf("baseline order was lost: %q", payload[:min(len(payload), 80)])
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".website-discovery-sort-") {
			t.Fatalf("sort workspace leaked: %s", entry.Name())
		}
	}
}

func TestMergeHostPortChunksDeduplicatesAcrossChunksAndRejectsUnframedChunk(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "first.txt")
	second := filepath.Join(directory, "second.txt")
	if err := os.WriteFile(first, []byte("a.example.com\t08080\nb.example.com\t08080\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("a.example.com\t08080\nc.example.com\t08443\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	merged, err := mergeHostPortChunks(context.Background(), []string{first, second}, directory)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(merged)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "a.example.com\t08080\nb.example.com\t08080\nc.example.com\t08443\n"; got != want {
		t.Fatalf("merged keys = %q, want %q", got, want)
	}
	unframed := filepath.Join(directory, "unframed.txt")
	if err := os.WriteFile(unframed, []byte("a.example.com\t08080"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := mergeHostPortChunks(context.Background(), []string{unframed}, directory); err == nil {
		t.Fatal("unframed HostPort chunk was accepted")
	}
}

func TestPrepareTargetCandidatesRemovesSortStateAfterMalformedInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "host-ports.jsonl")
	if err := os.WriteFile(facts, []byte(`{"host":"bad","ip":"192.0.2.1","port":443}`+"\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareTargetCandidates(context.Background(), websitediscoverycontract.Target{Type: websitediscoverycontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
		t.Fatal("malformed HostPort input was accepted")
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".website-discovery-sort-") {
			t.Fatalf("sort workspace leaked after failure: %s", entry.Name())
		}
	}
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
