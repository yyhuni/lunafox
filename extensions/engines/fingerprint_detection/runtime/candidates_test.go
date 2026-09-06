package fingerprintdetectionruntime

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

func TestBuildCandidatesUsesBaselineFirstOrderAndPreservesExactOverlap(t *testing.T) {
	got, err := BuildCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, []string{
		"https://example.com",
		"https://example.com:8443/path?b=2&a=1#part",
		"https://example.com:8443/path?b=2&a=1#other",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Candidate{
		{OriginalURL: "http://example.com", ExecutionURL: "http://example.com"},
		{OriginalURL: "https://example.com", ExecutionURL: "https://example.com"},
		{OriginalURL: "https://example.com", ExecutionURL: "https://example.com"},
		{OriginalURL: "https://example.com:8443/path?b=2&a=1#part", ExecutionURL: "https://example.com:8443/path?b=2&a=1#part"},
		{OriginalURL: "https://example.com:8443/path?b=2&a=1#other", ExecutionURL: "https://example.com:8443/path?b=2&a=1#other"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
}

func TestBuildCandidatesExpandsIPv4CIDRInclusivelyIncluding32(t *testing.T) {
	for _, test := range []struct {
		target string
		want   []string
	}{
		{"192.0.2.0/30", []string{"http://192.0.2.0", "https://192.0.2.0", "http://192.0.2.1", "https://192.0.2.1", "http://192.0.2.2", "https://192.0.2.2", "http://192.0.2.3", "https://192.0.2.3"}},
		{"192.0.2.7/32", []string{"http://192.0.2.7", "https://192.0.2.7"}},
	} {
		t.Run(test.target, func(t *testing.T) {
			got, err := BuildCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: test.target}, nil)
			if err != nil {
				t.Fatal(err)
			}
			gotURLs := make([]string, 0, len(got))
			for _, candidate := range got {
				gotURLs = append(gotURLs, candidate.OriginalURL)
			}
			if !reflect.DeepEqual(gotURLs, test.want) {
				t.Fatalf("URLs = %#v, want %#v", gotURLs, test.want)
			}
		})
	}
}

func TestMaterializeCandidatesPreservesStableFactsAndWritesRawExecution(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	raw := "HTTPS://example.com:443/path%2Fpart?x=%00&next=%0d%0a#fragment"
	if err := os.WriteFile(facts, []byte(raw+"\nhttps://example.com/%zz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, count, err := MaterializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "http://example.com\nhttps://example.com\n"+raw+"\nhttps://example.com/%zz\n" {
		t.Fatalf("candidate file = %q", payload)
	}
	if count != 4 {
		t.Fatalf("candidate count = %d, want 4", count)
	}
}

func TestMaterializeCandidatesKeepsZeroFactProductDistinctFromMissingInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	path, count, err := MaterializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatalf("MaterializeCandidates() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("zero-fact candidate count = %d", count)
	}
	payload, err := os.ReadFile(path)
	if err != nil || string(payload) != "http://example.com\nhttps://example.com\n" {
		t.Fatalf("zero-fact candidate file = %q, %v", payload, err)
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
			if _, _, err := MaterializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace); err == nil {
				t.Fatalf("MaterializeCandidates() accepted %s websiteURLs framing", name)
			}
		})
	}
}

func TestMaterializeCandidatesDoesNotTruncateLargeInput(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	const records = 12000
	file, err := os.OpenFile(facts, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < records; index++ {
		if _, err := file.WriteString("https://sub" + strconv.Itoa(index) + ".example.com/path\n"); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	path, count, err := MaterializeCandidates(context.Background(), enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, facts, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if count != records+2 {
		t.Fatalf("candidate count = %d, want %d", count, records+2)
	}
	input, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(input)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		_ = input.Close()
		t.Fatal(err)
	}
	if err := input.Close(); err != nil {
		t.Fatal(err)
	}
	if lines != records+2 {
		t.Fatalf("candidate lines = %d, want %d", lines, records+2)
	}
}

func TestBuildCandidatesCancellationStopsCIDRExpansion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BuildCandidates(ctx, enginecontract.Target{Type: enginecontract.TargetTypeCIDR, Value: "192.0.2.0/24"}, nil); err == nil {
		t.Fatal("cancelled candidate build succeeded")
	}
}

// BuildCandidates remains a test-only characterization helper. Production
// materialization uses the streaming callback in candidates.go.
func BuildCandidates(ctx context.Context, target enginecontract.Target, websiteURLs []string) ([]Candidate, error) {
	result := make([]Candidate, 0)
	emit := func(raw string) error {
		if err := observedCandidateValidation(target, raw); err != nil {
			return err
		}
		result = append(result, Candidate{OriginalURL: raw, ExecutionURL: raw})
		return nil
	}
	if err := emitTargetBaselines(ctx, target, emit); err != nil {
		return nil, err
	}
	for _, raw := range websiteURLs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := emit(raw); err != nil {
			return nil, err
		}
	}
	return result, nil
}
