package directoryscanruntime

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestCandidatePlanPreservesBaselineFirstWebsiteOrderAndRawValues(t *testing.T) {
	facts := writeWebsiteURLFacts(t,
		"http://example.com",
		"https://example.com",
		"https://example.com/app?next=%00&payload=%0d%0a#frag",
		"HTTP://example.com",
		"https://example.com/%zz",
		"https://example.com/app?next=%00&payload=%0d%0a#frag",
	)
	plan, err := PreflightCandidates(context.Background(), enginecontract.Target{
		Type: enginecontract.TargetTypeDomain, Value: "example.com",
	}, facts)
	if err != nil {
		t.Fatalf("PreflightCandidates() error = %v", err)
	}
	if plan.CandidateCount() != 6 {
		t.Fatalf("CandidateCount() = %d, want 6", plan.CandidateCount())
	}

	candidates := collectCandidateReplay(t, plan, 2)
	want := []WebsiteCandidate{
		{Ordinal: 0, Website: "http://example.com"},
		{Ordinal: 1, Website: "https://example.com"},
		{Ordinal: 2, Website: "https://example.com/app?next=%00&payload=%0d%0a#frag"},
		{Ordinal: 3, Website: "HTTP://example.com"},
		{Ordinal: 4, Website: "https://example.com/%zz"},
		{Ordinal: 5, Website: "https://example.com/app?next=%00&payload=%0d%0a#frag"},
	}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("candidate replay = %#v, want %#v", candidates, want)
	}
}

func TestCandidatePlanBuildsDomainAndIPv4Baselines(t *testing.T) {
	tests := []struct {
		name   string
		target enginecontract.Target
		want   []WebsiteCandidate
	}{
		{
			name:   "domain",
			target: enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
			want: []WebsiteCandidate{
				{Ordinal: 0, Website: "http://example.com"},
				{Ordinal: 1, Website: "https://example.com"},
			},
		},
		{
			name:   "ip",
			target: enginecontract.Target{Type: enginecontract.TargetTypeIP, Value: "192.0.2.10"},
			want: []WebsiteCandidate{
				{Ordinal: 0, Website: "http://192.0.2.10"},
				{Ordinal: 1, Website: "https://192.0.2.10"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PreflightCandidates(context.Background(), test.target, writeWebsiteURLFacts(t))
			if err != nil {
				t.Fatalf("PreflightCandidates() error = %v", err)
			}
			if got := collectCandidateReplay(t, plan, 1); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("candidate replay = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCandidatePlanExpandsIPv4CIDRInclusivelyWithoutTerminalOverflow(t *testing.T) {
	tests := []struct {
		name  string
		value string
		facts []string
		want  []string
	}{
		{
			name:  "network and broadcast",
			value: "192.0.2.0/30",
			facts: []string{"http://192.0.2.0", "https://192.0.2.3", "https://192.0.2.3/path"},
			want: []string{
				"http://192.0.2.0", "https://192.0.2.0",
				"http://192.0.2.1", "https://192.0.2.1",
				"http://192.0.2.2", "https://192.0.2.2",
				"http://192.0.2.3", "https://192.0.2.3",
				"https://192.0.2.3/path",
			},
		},
		{
			name:  "slash 32 at terminal address",
			value: "255.255.255.255/32",
			want:  []string{"http://255.255.255.255", "https://255.255.255.255"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PreflightCandidates(context.Background(), enginecontract.Target{
				Type: enginecontract.TargetTypeCIDR, Value: test.value,
			}, writeWebsiteURLFacts(t, test.facts...))
			if err != nil {
				t.Fatalf("PreflightCandidates() error = %v", err)
			}
			candidates := collectCandidateReplay(t, plan, 3)
			got := make([]string, len(candidates))
			for index, candidate := range candidates {
				got[index] = candidate.Website
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("CIDR candidates = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPreflightCandidatesPreservesLiteralFUZZBeforeReplay(t *testing.T) {
	plan, err := PreflightCandidates(context.Background(), enginecontract.Target{
		Type: enginecontract.TargetTypeDomain, Value: "example.com",
	}, writeWebsiteURLFacts(t, "https://example.com/ok", "https://example.com/FUZZ-late"))
	if err != nil {
		t.Fatalf("PreflightCandidates() error = %v", err)
	}
	candidates := collectCandidateReplay(t, plan, 2)
	if len(candidates) != 4 || candidates[3].Website != "https://example.com/FUZZ-late" {
		t.Fatalf("literal FUZZ candidate was changed or omitted: %#v", candidates)
	}
}

func TestPreflightCandidatesRejectsUnrecoverableWebsiteURLLineFraming(t *testing.T) {
	for name, payload := range map[string]string{
		"CRLF":         "https://example.com/\r\n",
		"unterminated": "https://example.com/",
	} {
		t.Run(name, func(t *testing.T) {
			facts := t.TempDir() + "/website-urls.txt"
			if err := os.WriteFile(facts, []byte(payload), 0o600); err != nil {
				t.Fatal(err)
			}
			if plan, err := PreflightCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts); err == nil || plan != nil {
				t.Fatalf("PreflightCandidates() accepted %s websiteURLs framing: plan=%#v err=%v", name, plan, err)
			}
		})
	}
}

func TestPreflightCandidatesRejectsInvalidTargetAndCanceledContext(t *testing.T) {
	tests := []enginecontract.Target{
		{Type: enginecontract.TargetTypeIP, Value: "192.0.2.010"},
		{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.1/24"},
		{Type: enginecontract.TargetType("unknown"), Value: "example.com"},
	}
	for _, target := range tests {
		if plan, err := PreflightCandidates(context.Background(), target, writeWebsiteURLFacts(t)); err == nil || plan != nil {
			t.Fatalf("PreflightCandidates(%#v) = plan %#v error %v, want rejection", target, plan, err)
		}
	}

	plan, err := PreflightCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "Example.COM"}, writeWebsiteURLFacts(t))
	if err != nil {
		t.Fatalf("PreflightCandidates() rejected the target value instead of preserving it: %v", err)
	}
	candidates := collectCandidateReplay(t, plan, 1)
	if len(candidates) != 2 || candidates[0].Website != "http://Example.COM" || candidates[1].Website != "https://Example.COM" {
		t.Fatalf("domain target value was normalized: %#v", candidates)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PreflightCandidates(ctx, enginecontract.Target{
		Type: enginecontract.TargetTypeCIDR, Value: "0.0.0.0/0",
	}, writeWebsiteURLFacts(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("PreflightCandidates(canceled) error = %v, want context.Canceled", err)
	}
}

func TestCIDRBaselineChecksContextBetweenHTTPAndHTTPS(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	enumerator := candidateEnumerator{
		targetType: enginecontract.TargetTypeCIDR,
		cidrStart:  binary.BigEndian.Uint32([]byte{192, 0, 2, 1}),
		cidrEnd:    binary.BigEndian.Uint32([]byte{192, 0, 2, 1}),
	}
	var emitted []string
	err := enumerator.forEachBaseline(ctx, func(candidate string) error {
		emitted = append(emitted, candidate)
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("forEachBaseline() error = %v, want context.Canceled", err)
	}
	if want := []string{"http://192.0.2.1"}; !reflect.DeepEqual(emitted, want) {
		t.Fatalf("emitted candidates = %#v, want %#v", emitted, want)
	}
}

func TestCandidateReplayUsesBoundedQueueAndStopsOnCancellation(t *testing.T) {
	websites := make([]string, 128)
	for index := range websites {
		websites[index] = fmt.Sprintf("https://example.com/%03d", index)
	}
	plan, err := PreflightCandidates(context.Background(), enginecontract.Target{
		Type: enginecontract.TargetTypeDomain, Value: "example.com",
	}, writeWebsiteURLFacts(t, websites...))
	if err != nil {
		t.Fatalf("PreflightCandidates() error = %v", err)
	}
	replay, err := plan.StartReplay(context.Background(), 1)
	if err != nil {
		t.Fatalf("StartReplay() error = %v", err)
	}
	if cap(replay.Candidates) != 1 {
		t.Fatalf("candidate queue capacity = %d, want 1", cap(replay.Candidates))
	}
	replay.Stop()
	waitResult := make(chan error, 1)
	go func() { waitResult <- replay.Wait() }()
	select {
	case waitErr := <-waitResult:
		if !errors.Is(waitErr, context.Canceled) {
			t.Fatalf("replay cancellation error = %v, want context.Canceled", waitErr)
		}
	case <-time.After(time.Second):
		t.Fatal("candidate replay did not stop after cancellation")
	}
}

func TestCandidateReplayFailsWhenImmutableInputContractDrifts(t *testing.T) {
	for _, test := range []struct {
		name        string
		replacement string
	}{
		{name: "count changes", replacement: "https://example.com/one\nhttps://example.com/two\n"},
		{name: "content changes at same count", replacement: "https://example.com/replaced\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := writeWebsiteURLFacts(t, "https://example.com/one")
			plan, err := PreflightCandidates(context.Background(), enginecontract.Target{
				Type: enginecontract.TargetTypeDomain, Value: "example.com",
			}, facts)
			if err != nil {
				t.Fatalf("PreflightCandidates() error = %v", err)
			}
			if err := os.WriteFile(facts, []byte(test.replacement), 0o600); err != nil {
				t.Fatal(err)
			}
			replay, err := plan.StartReplay(context.Background(), 2)
			if err != nil {
				t.Fatalf("StartReplay() error = %v", err)
			}
			for range replay.Candidates {
			}
			if err := replay.Wait(); err == nil || !strings.Contains(err.Error(), "changed between passes") {
				t.Fatalf("candidate replay drift error = %v", err)
			}
		})
	}
}

func collectCandidateReplay(t *testing.T, plan *CandidatePlan, capacity int) []WebsiteCandidate {
	t.Helper()
	replay, err := plan.StartReplay(context.Background(), capacity)
	if err != nil {
		t.Fatalf("StartReplay() error = %v", err)
	}
	var candidates []WebsiteCandidate
	for candidate := range replay.Candidates {
		candidates = append(candidates, candidate)
	}
	if err := replay.Wait(); err != nil {
		t.Fatalf("CandidateReplay.Wait() error = %v", err)
	}
	return candidates
}

func writeWebsiteURLFacts(t *testing.T, values ...string) string {
	t.Helper()
	path := t.TempDir() + "/website-urls.txt"
	payload := ""
	if len(values) > 0 {
		payload = strings.Join(values, "\n") + "\n"
	}
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
