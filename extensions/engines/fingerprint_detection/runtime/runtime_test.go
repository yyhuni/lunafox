package fingerprintdetectionruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	contractresults "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

type fakeObserverWardExecutor struct {
	lines [][]byte
	err   error
	cmd   observerWardCommand
}

func (fake *fakeObserverWardExecutor) Run(_ context.Context, command observerWardCommand, visit func([]byte) error) error {
	fake.cmd = command
	if fake.err != nil {
		return fake.err
	}
	for _, line := range fake.lines {
		if err := visit(line); err != nil {
			return err
		}
	}
	return nil
}

type captureProgress struct{ messages []string }

func (capture *captureProgress) Report(_ context.Context, message string) error {
	capture.messages = append(capture.messages, message)
	return nil
}

type captureWebsiteTechnologySubmitter struct {
	items []contractresults.WebsiteTechnology
	err   error
}

func (capture *captureWebsiteTechnologySubmitter) Submit(_ context.Context, items <-chan contractresults.WebsiteTechnology) error {
	for item := range items {
		capture.items = append(capture.items, item)
	}
	return capture.err
}

func TestRuntimeFailsClosedBeforeObserverWardForEmptyCorpus(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(corpus, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeObserverWardExecutor{}
	progress := &captureProgress{}
	submitter := &captureWebsiteTechnologySubmitter{}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          progress,
		Results:           enginecontract.Results{WebsiteTechnologies: submitter},
	}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err == nil {
		t.Fatal("empty corpus execution succeeded")
	}
	if executor.cmd.Name != "" || len(submitter.items) != 0 || len(progress.messages) != 0 {
		t.Fatalf("empty corpus side effects command=%#v items=%#v progress=%#v", executor.cmd, submitter.items, progress.messages)
	}
}

func TestRuntimeSubmitsTrustedAndZeroMatchObservationsWithAggregateProgress(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, []byte("https://example.com/app#fragment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(corpus, []byte(`[{"id":"one","http":[{"matchers":[{"type":"word","words":["one"]}]}],"info":{"name":"one"}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeObserverWardExecutor{lines: [][]byte{
		[]byte(`{"input_target":"http://example.com","target":"http://example.com","success":true,"matched":[{"base_url":"http://example.com","result":{"status":200,"name":[]}}]}`),
		[]byte(`{"input_target":"https://example.com","target":"https://example.com","success":false,"matched":[]}`),
		[]byte(`{"input_target":"https://example.com/app#fragment","target":"https://example.com/app","success":true,"matched":[{"base_url":"https://other.example/redirect","result":{"status":204,"name":["Vue","Vue"]}}]}`),
	}}
	progress := &captureProgress{}
	submitter := &captureWebsiteTechnologySubmitter{}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          progress,
		Results:           enginecontract.Results{WebsiteTechnologies: submitter},
	}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err != nil {
		t.Fatal(err)
	}
	wantItems := []contractresults.WebsiteTechnology{
		{URL: "http://example.com", Tech: []string{}},
		{URL: "https://example.com/app", Tech: []string{"Vue", "Vue"}},
	}
	if !reflect.DeepEqual(submitter.items, wantItems) {
		t.Fatalf("submitted items = %#v, want %#v", submitter.items, wantItems)
	}
	if len(progress.messages) != 1 || progress.messages[0] != "complete result reporting candidates=3 outputRecords=3 malformedRecords=0 trustedResponses=2 failedRequests=1 matchedCandidates=1 zeroMatchCandidates=1 submittedItems=2" {
		t.Fatalf("progress = %#v", progress.messages)
	}
	if _, err := os.Stat(filepath.Join(workspace, "observer-ward-candidates.txt")); !os.IsNotExist(err) {
		t.Fatalf("candidate artifact was not cleaned up: %v", err)
	}
	// No candidate identity index is created by the observed-result runtime.
}

func TestRuntimeAllowsDuplicateAndUnknownInputTarget(t *testing.T) {
	for name, lines := range map[string][][]byte{
		"duplicate": {
			[]byte(`{"input_target":"http://example.com","target":"http://example.com/","success":true,"matched":[{"base_url":"http://example.com","result":{"status":200,"name":[]}}]}`),
			[]byte(`{"input_target":"http://example.com","target":"http://example.com/","success":true,"matched":[{"base_url":"http://example.com","result":{"status":200,"name":["duplicate"]}}]}`),
		},
		"unknown": {
			[]byte(`{"input_target":"https://example.com/not-attempted","target":"https://example.com/","success":true,"matched":[{"base_url":"https://example.com/","result":{"status":200,"name":["wrong"]}}]}`),
		},
	} {
		t.Run(name, func(t *testing.T) {
			workspace := t.TempDir()
			facts := filepath.Join(workspace, "website-urls.txt")
			corpus := filepath.Join(workspace, "fingerprinthub_web.json")
			if err := os.WriteFile(facts, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(corpus, []byte(`[{"id":"one","http":[{"matchers":[{"type":"word","words":["one"]}]}],"info":{"name":"one"}}]`), 0o600); err != nil {
				t.Fatal(err)
			}
			executor := &fakeObserverWardExecutor{lines: lines}
			submitter := &captureWebsiteTechnologySubmitter{}
			execution := &enginecontract.Execution{
				Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
				Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
				Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
				PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
				Workspace:         workspace,
				Progress:          &captureProgress{},
				Results:           enginecontract.Results{WebsiteTechnologies: submitter},
			}
			if err := NewWithExecutor(executor).Execute(context.Background(), execution); err != nil {
				t.Fatalf("Execute() rejected optional attribution: %v", err)
			}
			wantItems := len(lines)
			if len(submitter.items) != wantItems {
				t.Fatalf("valid duplicate/unknown rows were dropped: got=%d want=%d items=%#v", len(submitter.items), wantItems, submitter.items)
			}
			if _, err := os.Stat(filepath.Join(workspace, "observer-ward-candidates.txt")); !os.IsNotExist(err) {
				t.Fatalf("failed execution left candidate artifact: %v", err)
			}
		})
	}
}

func TestRuntimeUsesObservedTargetWithoutInputTarget(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(facts, []byte("https://example.com/app#fragment\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corpus, []byte(`[{"id":"one","http":[{"matchers":[{"type":"word","words":["one"]}]}],"info":{"name":"one"}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	submitter := &captureWebsiteTechnologySubmitter{}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          &captureProgress{},
		Results:           enginecontract.Results{WebsiteTechnologies: submitter},
	}
	executor := &fakeObserverWardExecutor{lines: [][]byte{
		[]byte(`{"target":"https://example.com/app","success":true,"matched":[{"base_url":"https://example.com/app","result":{"status":200,"name":["wrong-attribution"]}}]}`),
	}}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err != nil {
		t.Fatalf("missing input_target should be accepted: %v", err)
	}
	if len(submitter.items) != 1 || submitter.items[0].URL != "https://example.com/app" {
		t.Fatalf("observed target submission = %#v", submitter.items)
	}
}

func TestRuntimeSkipsMalformedRowsWhenAValidNoFindingRowExists(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corpus, []byte(`[{"id":"one","http":[{"matchers":[{"type":"word"}]}],"info":{"name":"one"}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeObserverWardExecutor{lines: [][]byte{
		[]byte(`not-json`),
		[]byte(`{"input_target":"http://example.com","target":"http://example.com","success":false,"matched":[]}`),
	}}
	progress := &captureProgress{}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          progress,
		Results:           enginecontract.Results{WebsiteTechnologies: &captureWebsiteTechnologySubmitter{}},
	}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err != nil {
		t.Fatalf("mixed malformed/no-finding output failed: %v", err)
	}
	if len(progress.messages) != 1 || !strings.Contains(progress.messages[0], "malformedRecords=1") {
		t.Fatalf("progress did not report malformed row: %#v", progress.messages)
	}
}

func TestRuntimeFailsWhenNonEmptyObserverWardOutputIsAllMalformed(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corpus, []byte(`[{"id":"one","http":[{"matchers":[{"type":"word"}]}],"info":{"name":"one"}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          &captureProgress{},
		Results:           enginecontract.Results{WebsiteTechnologies: &captureWebsiteTechnologySubmitter{}},
	}
	executor := &fakeObserverWardExecutor{lines: [][]byte{[]byte(`not-json`), []byte(`[]`)}}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err == nil || !strings.Contains(err.Error(), "no valid record") {
		t.Fatalf("all-malformed output error = %v", err)
	}
}

func TestRuntimePropagatesSubmissionFailure(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	for path, payload := range map[string][]byte{facts: nil, corpus: []byte(`[{"id":"one","http":[{"matchers":[{"type":"word","words":["one"]}]}],"info":{"name":"one"}}]`)} {
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	submissionErr := errors.New("submit failed")
	submitter := &captureWebsiteTechnologySubmitter{err: submissionErr}
	execution := &enginecontract.Execution{
		Target: enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"}, Input: enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config: enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}}, PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus}, Workspace: workspace, Progress: &captureProgress{}, Results: enginecontract.Results{WebsiteTechnologies: submitter},
	}
	executor := &fakeObserverWardExecutor{lines: [][]byte{[]byte(`{"input_target":"http://example.com","target":"http://example.com","success":true,"matched":[{"base_url":"http://example.com","result":{"status":200,"name":[]}}]}`)}}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); !errors.Is(err, submissionErr) {
		t.Fatalf("error = %v, want %v", err, submissionErr)
	}
}

func TestRuntimeFailsClosedBeforeObserverWardForCorpusWithoutValidTemplates(t *testing.T) {
	workspace := t.TempDir()
	facts := filepath.Join(workspace, "website-urls.txt")
	if err := os.WriteFile(facts, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(workspace, "fingerprinthub_web.json")
	if err := os.WriteFile(corpus, []byte(`[{"id":"invalid","info":{"name":"Invalid"},"http":[{}]}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeObserverWardExecutor{}
	progress := &captureProgress{}
	submitter := &captureWebsiteTechnologySubmitter{}
	execution := &enginecontract.Execution{
		Target:            enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:             enginecontract.Input{WebsiteURLs: testInputPath(facts)},
		Config:            enginecontract.Config{ObserverWard: enginecontract.ObserverWardConfig{Enabled: true, Timeout: 3600, RequestTimeout: 10, Threads: 25}},
		PlatformResources: enginecontract.PlatformResources{FingerprintLibraryFingerPrintHub: corpus},
		Workspace:         workspace,
		Progress:          progress,
		Results:           enginecontract.Results{WebsiteTechnologies: submitter},
	}
	if err := NewWithExecutor(executor).Execute(context.Background(), execution); err == nil {
		t.Fatal("invalid FingerprintHub corpus execution succeeded")
	}
	if executor.cmd.Name != "" || len(submitter.items) != 0 || len(progress.messages) != 0 {
		t.Fatalf("invalid corpus side effects command=%#v items=%#v progress=%#v", executor.cmd, submitter.items, progress.messages)
	}
}
