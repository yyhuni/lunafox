package screenshotruntime

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

func TestMaterializeCandidatesUsesTargetBaselineAndStableWebsiteURLs(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, []byte("https://example.com/a\nhttp://example.com\nhttps://example.com/a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, count, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("materializeCandidates() error = %v", err)
	}
	want := []string{"http://example.com", "https://example.com", "https://example.com/a", "http://example.com", "https://example.com/a"}
	if count != uint64(len(want)) {
		t.Fatalf("candidate count = %d, want %d", count, len(want))
	}
}

func TestMaterializeCandidatesPreservesRawWebsiteURLFacts(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	raw := "HTTPS://example.com:443/a%2Fb?x=%zz#fragment"
	if err := os.WriteFile(facts, []byte(raw+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, count, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("materializeCandidates() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("raw candidate was changed or omitted: count=%d", count)
	}
}

func TestMaterializeCandidatesPreservesRawPercentSpellingInPath(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	raw := "https://example.com/%zz"
	if err := os.WriteFile(facts, []byte(raw+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, count, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil || count != 3 {
		t.Fatalf("raw percent spelling was rejected or changed: count=%d error=%v", count, err)
	}
}

func TestMaterializeCandidatesRejectsOutOfTargetFactsWithoutRewriting(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, []byte("https://outside.example/%zz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
		t.Fatal("materializeCandidates accepted an out-of-Target raw URL")
	}
}

func TestMaterializeCandidatesRejectsUnrecoverableWebsiteURLLineFraming(t *testing.T) {
	for name, payload := range map[string]string{
		"CRLF":         "https://example.com/\r\n",
		"unterminated": "https://example.com/",
	} {
		t.Run(name, func(t *testing.T) {
			workspace := t.TempDir()
			facts := filepath.Join(workspace, "website-urls.txt")
			if err := os.WriteFile(facts, []byte(payload), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
				t.Fatalf("materializeCandidates() accepted %s websiteURLs framing", name)
			}
		})
	}
}

func TestMaterializeCandidatesExpandsIPv4CIDRIncludingNetworkAndBroadcast(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	path, count, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/30"}, facts, workspace)
	if err != nil {
		t.Fatalf("materializeCandidates() error = %v", err)
	}
	if count != 8 {
		t.Fatalf("CIDR candidate count = %d", count)
	}
	payload, err := os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(payload), "http://192.0.2.0\nhttps://192.0.2.0\n") || !strings.HasSuffix(string(payload), "http://192.0.2.3\nhttps://192.0.2.3\n") {
		t.Fatalf("CIDR candidates = %q, error=%v", payload, err)
	}
}

func TestMaterializeCandidatesDoesNotTruncateLargeInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	file, err := os.OpenFile(facts, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	const records = 12000
	for index := 0; index < records; index++ {
		if _, err := file.WriteString("https://sub" + strconv.Itoa(index) + ".example.com/path\n"); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	path, count, err := materializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if count != records+2 {
		t.Fatalf("candidate count = %d, want %d", count, records+2)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(payload), "\n"); lines != records+2 {
		t.Fatalf("candidate lines = %d, want %d", lines, records+2)
	}
}
