package portscanruntime

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

func TestPrepareTargetCandidatesPreservesBaselineOrderAndFactOrder(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "subdomains.txt")
	if err := os.WriteFile(facts, []byte("www.example.com\napi.example.com\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), portscancontract.Target{Type: portscancontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetCandidates() error = %v", err)
	}
	if count != 3 {
		t.Fatalf("candidate count = %d, want 3", count)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "example.com\nwww.example.com\napi.example.com\n"; string(got) != want {
		t.Fatalf("candidate bytes = %q, want %q", got, want)
	}
	if filepath.Dir(path) != workspace {
		t.Fatalf("candidate path = %q, want workspace %q", path, workspace)
	}
}

func TestWriteTargetBaselinePreservesIPv4Semantics(t *testing.T) {
	tests := []struct {
		name   string
		target portscancontract.Target
		want   []string
	}{
		{name: "domain", target: portscancontract.Target{Type: portscancontract.TargetTypeDomain, Value: "example.com"}, want: []string{"example.com"}},
		{name: "ip", target: portscancontract.Target{Type: portscancontract.TargetTypeIP, Value: "192.0.2.7"}, want: []string{"192.0.2.7"}},
		{name: "cidr includes network and broadcast", target: portscancontract.Target{Type: portscancontract.TargetTypeCIDR, Value: "192.0.2.0/30"}, want: []string{"192.0.2.0", "192.0.2.1", "192.0.2.2", "192.0.2.3"}},
		{name: "cidr /32", target: portscancontract.Target{Type: portscancontract.TargetTypeCIDR, Value: "192.0.2.9/32"}, want: []string{"192.0.2.9"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			if err := writeTargetBaseline(context.Background(), test.target, func(value string) error {
				got = append(got, value)
				return nil
			}); err != nil {
				t.Fatalf("writeTargetBaseline() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("baseline = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPrepareTargetCandidatesKeepsZeroFactInputSeparate(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "subdomains.txt")
	if err := os.WriteFile(facts, nil, 0o400); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), portscancontract.Target{Type: portscancontract.TargetTypeIP, Value: "192.0.2.7"}, facts, workspace)
	if err != nil {
		t.Fatalf("prepareTargetCandidates() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("candidate count = %d, want Target baseline only", count)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "192.0.2.7\n" {
		t.Fatalf("candidate bytes = %q, want baseline only", got)
	}
	if filepath.Clean(path) == filepath.Clean(facts) {
		t.Fatal("Engine passed the mounted fact path through as the tool input")
	}
}

func TestPrepareTargetCandidatesStopsBeforePublicationOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "subdomains.txt")
	if err := os.WriteFile(facts, []byte("api.example.com\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if path, _, err := prepareTargetCandidates(ctx, portscancontract.Target{Type: portscancontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil || path != "" {
		t.Fatalf("cancelled preparation = path %q err %v, want no candidate", path, err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "naabu-candidates.txt")); !os.IsNotExist(err) {
		t.Fatalf("cancelled candidate file still exists: %v", err)
	}
}

func TestPrepareTargetCandidatesDoesNotTruncateLargeInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "subdomains.txt")
	file, err := os.OpenFile(facts, os.O_CREATE|os.O_WRONLY, 0o400)
	if err != nil {
		t.Fatal(err)
	}
	const records = 20000
	for index := 0; index < records; index++ {
		if _, err := file.WriteString("sub" + strconv.Itoa(index) + ".example.com\n"); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	path, count, err := prepareTargetCandidates(context.Background(), portscancontract.Target{Type: portscancontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if count != records+1 {
		t.Fatalf("candidate count = %d, want %d", count, records+1)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(payload), "\n"); lines != records+1 {
		t.Fatalf("candidate lines = %d, want %d", lines, records+1)
	}
}

func TestPrepareTargetCandidatesRejectsMalformedLineBeforePublication(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(t.TempDir(), "subdomains.txt")
	if err := os.WriteFile(facts, []byte("api.example.com\r\n"), 0o400); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareTargetCandidates(context.Background(), portscancontract.Target{Type: portscancontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
		t.Fatal("CRLF subdomain fact was accepted")
	}
	if _, err := os.Stat(filepath.Join(workspace, "naabu-candidates.txt")); !os.IsNotExist(err) {
		t.Fatalf("incomplete Naabu candidate remains: %v", err)
	}
}

func TestPrepareTargetCandidatesRejectsUint64CounterOverflow(t *testing.T) {
	value := ^uint64(0)
	if err := incrementUint64(&value, "test counter"); err == nil {
		t.Fatal("counter overflow was accepted")
	}
}
