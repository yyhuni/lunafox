package directoryscanruntime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeDirectoryTarget(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		want      string
		wantErr   bool
	}{
		{name: "empty path", candidate: "http://example.com", want: "http://example.com/"},
		{name: "path without slash", candidate: "https://example.com/root", want: "https://example.com/root/"},
		{name: "path with slash", candidate: "https://example.com/root/", want: "https://example.com/root/"},
		{name: "query only", candidate: "http://example.com?path=/admin", want: "http://example.com/?path=/admin"},
		{name: "fragment only", candidate: "http://example.com#fragment", want: "http://example.com/#fragment"},
		{name: "query and fragment", candidate: "http://example.com?path=/admin#fragment", want: "http://example.com/?path=/admin#fragment"},
		{name: "double slash path", candidate: "http://example.com//", want: "http://example.com//"},
		{name: "port and query", candidate: "http://127.0.0.1:8080/a?b=1", want: "http://127.0.0.1:8080/a/?b=1"},
		{name: "userinfo", candidate: "https://user:pass@example.com/root", want: "https://user:pass@example.com/root/"},
		{name: "empty", candidate: "", wantErr: true},
		{name: "relative", candidate: "example.com/root", wantErr: true},
		{name: "empty authority", candidate: "http:///root", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeDirectoryTarget(test.candidate)
			if test.wantErr {
				if err == nil {
					t.Fatalf("normalizeDirectoryTarget(%q) = %q, want error", test.candidate, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDirectoryTarget(%q) error = %v", test.candidate, err)
			}
			if got != test.want {
				t.Fatalf("normalizeDirectoryTarget(%q) = %q, want %q", test.candidate, got, test.want)
			}
			idempotent, err := normalizeDirectoryTarget(got)
			if err != nil || idempotent != got {
				t.Fatalf("normalizeDirectoryTarget(%q) = %q, %v; want idempotent %q", got, idempotent, err, got)
			}
		})
	}
}

func TestNormalizeWordlistFileNormalizesOneLeadingSeparator(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.txt")
	payload := []byte("admin\n/admin\n//admin\n/\n\nvalue # inline\n")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "normalized.txt")
	if err := normalizeWordlistFile(context.Background(), source, destination); err != nil {
		t.Fatalf("normalizeWordlistFile() error = %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("admin\nadmin\n/admin\n/\n\nvalue # inline\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("normalized wordlist = %q, want %q", got, want)
	}
	sourceAfter, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceAfter, payload) {
		t.Fatalf("source wordlist changed: %q", sourceAfter)
	}
}

func TestNormalizeWordlistFilePreservesFramingAndZeroBytes(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{name: "zero bytes", payload: nil, want: nil},
		{name: "crlf", payload: []byte("/admin\r\n/admin2\r\n"), want: []byte("admin\r\nadmin2\r\n")},
		{name: "unterminated final line", payload: []byte("/admin\n/admin2"), want: []byte("admin\nadmin2")},
		{name: "empty lines", payload: []byte("\n\n"), want: []byte("\n\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "source.txt")
			if err := os.WriteFile(source, test.payload, 0o600); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(t.TempDir(), "normalized.txt")
			if err := normalizeWordlistFile(context.Background(), source, destination); err != nil {
				t.Fatalf("normalizeWordlistFile() error = %v", err)
			}
			got, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, test.want) {
				t.Fatalf("normalized wordlist = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeWordlistFileRejectsOversizedLine(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.txt")
	oversized := append(bytes.Repeat([]byte("a"), directoryWordlistLineLimit+1), '\n')
	if err := os.WriteFile(source, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "normalized.txt")
	err := normalizeWordlistFile(context.Background(), source, destination)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("normalizeWordlistFile() error = %v, want line-limit error", err)
	}
}

func TestPrepareNormalizedWordlistStagesAndRemovesPrivateCopy(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(t.TempDir(), "wordlist.txt")
	payload := []byte("/admin\n")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	wordlist, err := prepareNormalizedWordlist(context.Background(), source, workspace)
	if err != nil {
		t.Fatalf("prepareNormalizedWordlist() error = %v", err)
	}
	if wordlist.path != filepath.Join(workspace, directoryNormalizedListName) {
		t.Fatalf("normalized wordlist path = %q, want deterministic workspace path", wordlist.path)
	}
	got, err := os.ReadFile(wordlist.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "admin\n" {
		t.Fatalf("normalized wordlist = %q", got)
	}
	if err := wordlist.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(wordlist.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("normalized wordlist file still exists: %v", err)
	}
}

func TestRuntimeExecutePassesNormalizedWordlistAndCleansItUp(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(t.TempDir(), "wordlist.txt")
	payload := []byte("/admin\n//admin\n")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	var capturedPath string
	var capturedBytes []byte
	runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return err
		}
		capturedPath = invocation.Config.Wordlist
		capturedBytes, err = os.ReadFile(capturedPath)
		return err
	})
	runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	}}}
	execution := directoryExecutionFixture(t, workspace, &recordingDirectoryProgress{}, &captureDirectorySubmitter{})
	execution.Config.Ffuf.Wordlist = source
	execution.Config.Ffuf.Concurrency = 1
	if err := runtime.Execute(context.Background(), execution); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if string(capturedBytes) != "admin\n/admin\n" {
		t.Fatalf("runtime passed wordlist bytes %q, want normalized copy", capturedBytes)
	}
	sourceAfter, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceAfter, payload) {
		t.Fatalf("source wordlist changed: %q", sourceAfter)
	}
	if _, err := os.Stat(capturedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("normalized wordlist file was not cleaned up: %v", err)
	}
}

func TestRuntimeExecuteCleansNormalizedWordlistOnFailure(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(t.TempDir(), "wordlist.txt")
	if err := os.WriteFile(source, []byte("/admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var capturedPath string
	runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
		capturedPath = invocation.Config.Wordlist
		return errors.New("ffuf failed")
	})
	runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	}}}
	execution := directoryExecutionFixture(t, workspace, &recordingDirectoryProgress{}, &captureDirectorySubmitter{})
	execution.Config.Ffuf.Wordlist = source
	execution.Config.Ffuf.Concurrency = 1
	if err := runtime.Execute(context.Background(), execution); err == nil {
		t.Fatal("Execute() succeeded despite runner failure")
	}
	if capturedPath == "" {
		t.Fatal("runner did not receive a wordlist path")
	}
	if _, err := os.Stat(capturedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("normalized wordlist file was not cleaned up after failure: %v", err)
	}
}

func TestNormalizeWordlistFileAppendsDedupedVariants(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		variants []string
		want     []byte
	}{
		{
			name:     "dedup against source",
			payload:  []byte("www.zip\r\n/admin\n"),
			variants: []string{"www.zip", "example.zip"},
			want:     []byte("www.zip\r\nadmin\nexample.zip\n"),
		},
		{
			name:     "source without trailing newline gains one",
			payload:  []byte("/admin"),
			variants: []string{"admin.zip"},
			want:     []byte("admin\nadmin.zip\n"),
		},
		{
			name:     "zero byte source",
			payload:  nil,
			variants: []string{"example.zip"},
			want:     []byte("example.zip\n"),
		},
		{
			name:     "no variants preserves source bytes",
			payload:  []byte("/admin\r\n"),
			variants: nil,
			want:     []byte("admin\r\n"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "source.txt")
			if err := os.WriteFile(source, test.payload, 0o600); err != nil {
				t.Fatal(err)
			}
			destination := filepath.Join(t.TempDir(), "normalized.txt")
			if err := normalizeWordlistFileWithVariants(context.Background(), source, destination, test.variants); err != nil {
				t.Fatalf("normalizeWordlistFileWithVariants() error = %v", err)
			}
			got, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, test.want) {
				t.Fatalf("normalized wordlist = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPrepareCandidateWordlistUsesDeterministicPath(t *testing.T) {
	workspace := t.TempDir()
	source := filepath.Join(t.TempDir(), "wordlist.txt")
	if err := os.WriteFile(source, []byte("admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := prepareCandidateWordlist(context.Background(), source, "https://www.example.com/", workspace, 7)
	if err != nil {
		t.Fatalf("prepareCandidateWordlist() error = %v", err)
	}
	wantPath := filepath.Join(workspace, directoryWordlistSubdir, "00000000000000000007.txt")
	if path != wantPath {
		t.Fatalf("candidate wordlist path = %q, want %q", path, wantPath)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"admin\n", "www.example.com.zip\n", "example.com.zip\n", "www.zip\n"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Fatalf("candidate wordlist missing %q: %q", want, got)
		}
	}
}
