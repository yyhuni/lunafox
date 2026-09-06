package urlcollectionruntime

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

func TestPrepareTargetSeedsUsesExactDomainOverlapAndRetainsConfirmedVariants(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	content := "http://example.com\nhttps://example.com\nhttp://example.com:80\nhttps://example.com/path\nhttps://example.com?x=1\n"
	if err := os.WriteFile(facts, []byte(content), 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetSeeds() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://example.com\nhttps://example.com\nhttp://example.com:80\nhttps://example.com/path\nhttps://example.com?x=1\n"
	if string(got) != want || count != 5 {
		t.Fatalf("seed bytes = %q count=%d, want %q count=5", got, count, want)
	}
}

func TestPrepareTargetSeedsCIDROverlapKeepsPathQueryAndExplicitPort(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	content := "http://192.0.2.0\nhttps://192.0.2.1/path\nhttp://192.0.2.1:80\nhttps://192.0.2.3?x=1\n"
	if err := os.WriteFile(facts, []byte(content), 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/30"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetSeeds() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://192.0.2.0\nhttps://192.0.2.0\nhttp://192.0.2.1\nhttps://192.0.2.1\nhttp://192.0.2.2\nhttps://192.0.2.2\nhttp://192.0.2.3\nhttps://192.0.2.3\nhttps://192.0.2.1/path\nhttp://192.0.2.1:80\nhttps://192.0.2.3?x=1\n"
	if string(got) != want || count != 11 {
		t.Fatalf("CIDR seed bytes = %q count=%d, want %q count=11", got, count, want)
	}
}

func TestPrepareTargetSeedsCIDROverlapDoesNotParseOrRewritePayloadURLs(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	content := "HTTP://192.0.2.1\nhttps://192.0.2.1/%zz?x=%0d%0a#fragment\nhttp://192.0.2.1%zz\n"
	if err := os.WriteFile(facts, []byte(content), 0o400); err != nil {
		t.Fatal(err)
	}
	path, _, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/30"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetSeeds() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(got), content) {
		t.Fatalf("payload URLs were changed or skipped: %q", got)
	}
}

func TestPrepareTargetSeedsKeepsZeroFactProductDistinct(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeIP, Value: "192.0.2.7"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetSeeds() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "http://192.0.2.7\nhttps://192.0.2.7\n" || count != 2 {
		t.Fatalf("zero-fact seed = %q count=%d", got, count)
	}
	if filepath.Clean(path) == filepath.Clean(facts) {
		t.Fatal("Engine passed the mounted fact path through as the tool input")
	}
}

func TestPrepareTargetSeedsRejectsUnrecoverableWebsiteURLLineFraming(t *testing.T) {
	for name, payload := range map[string]string{
		"CRLF":         "https://example.com/\r\n",
		"unterminated": "https://example.com/",
	} {
		t.Run(name, func(t *testing.T) {
			workspace := t.TempDir()
			facts := filepath.Join(t.TempDir(), "website-urls.txt")
			if err := os.WriteFile(facts, []byte(payload), 0o400); err != nil {
				t.Fatal(err)
			}
			if _, _, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
				t.Fatalf("prepareTargetSeeds() accepted %s websiteURLs framing", name)
			}
		})
	}
}

func TestPrepareTargetSeedsDoesNotTruncateLargeInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	file, err := os.OpenFile(facts, os.O_CREATE|os.O_WRONLY, 0o400)
	if err != nil {
		t.Fatal(err)
	}
	const records = 20000
	for index := 0; index < records; index++ {
		if _, err := file.WriteString("https://sub" + strconv.Itoa(index) + ".example.com/path\n"); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if count != records+2 {
		t.Fatalf("seed count = %d, want %d", count, records+2)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(payload), "\n"); lines != records+2 {
		t.Fatalf("seed lines = %d, want %d", lines, records+2)
	}
}

func TestPrepareTargetSeedsFailsBeforePublicationWhenOutputIsOccupied(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o400); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(workspace, "katana-seeds.txt")
	if err := os.WriteFile(output, []byte("sentinel\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareTargetSeeds(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
		t.Fatal("occupied seed path was overwritten")
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != "sentinel\n" {
		t.Fatalf("occupied seed path changed: %q, %v", got, err)
	}
}

func TestIncrementUint64RejectsOverflow(t *testing.T) {
	value := ^uint64(0)
	if err := incrementUint64(&value, "test counter"); err == nil {
		t.Fatal("counter overflow was accepted")
	}
}
