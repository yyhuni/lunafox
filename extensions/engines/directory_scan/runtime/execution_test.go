package directoryscanruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestRuntimeExecuteReportsExactSuccessfulLifecycle(t *testing.T) {
	workspace := t.TempDir()
	sharedURL := "https://example.com/private-result"
	artifacts := map[uint64][]byte{
		0: append(append(marshalFFUFRecord(t, ffufRecordFixture{
			URL: sharedURL, Status: 200, ContentLength: 1, ContentType: "first", Duration: 1,
		}), '\n'), []byte("not-json\n")...),
		1: append([]byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}`+"\n"), append(marshalFFUFRecord(t, ffufRecordFixture{
			URL: sharedURL, Status: 201, ContentLength: 2, ContentType: "winner", Duration: 2,
		}), '\n')...),
	}
	progress := &recordingDirectoryProgress{}
	var cleanupComplete atomic.Bool
	var completedAfterCleanup atomic.Bool
	progress.onReport = func(message string) {
		if strings.HasPrefix(message, "aggregation-completed ") {
			completedAfterCleanup.Store(cleanupComplete.Load())
		}
	}
	submitter := &captureDirectorySubmitter{}
	runtime := &Runtime{aggregation: directoryAggregationOptions{
		scheduler: websiteSchedulerOptions{
			runner: artifactWritingRunner(t, artifacts, nil), websiteTimeout: time.Second,
		},
		dedup: directoryDedupOptions{FileOps: directoryDedupFileOps{removeAll: func(path string) error {
			err := os.RemoveAll(path)
			if err == nil {
				cleanupComplete.Store(true)
			}
			return err
		}}},
	}}

	err := runtime.Execute(context.Background(), directoryExecutionFixture(t, workspace, progress, submitter))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := []string{
		"input_ready websiteCandidates=2",
		"scan_started websiteCandidates=2",
		"aggregation-completed websiteCandidates=2 normalCompletedWebsites=2 timedOutWebsites=0 skippedDuplicate=1 malformedRecords=1 invalidRecords=1 oversizedRecords=0",
	}
	if got := progress.Messages(); !reflect.DeepEqual(got, want) {
		t.Fatalf("progress messages = %#v, want %#v", got, want)
	}
	if !completedAfterCleanup.Load() {
		t.Fatal("aggregation-completed was reported before private dedup cleanup")
	}
	if len(submitter.items) != 1 || submitter.items[0].URL != sharedURL || submitter.items[0].Status != 201 {
		t.Fatalf("submitted items = %#v", submitter.items)
	}
	assertNoDirectoryDedupStaging(t, workspace)
}

func TestRuntimeExecuteReportsZeroFindingAndPartialTimeoutSuccess(t *testing.T) {
	valid := append(marshalFFUFRecord(t, ffufRecordFixture{
		URL: "https://example.com/pre-deadline", Status: 200, ContentLength: 0, ContentType: "", Duration: 0,
	}), '\n')
	tests := []struct {
		name      string
		artifacts map[uint64][]byte
		modes     map[uint64]error
		wantFinal string
		wantItems int
	}{
		{
			name:      "zero findings",
			artifacts: map[uint64][]byte{},
			wantFinal: "aggregation-completed websiteCandidates=2 normalCompletedWebsites=2 timedOutWebsites=0 skippedDuplicate=0 malformedRecords=0 invalidRecords=0 oversizedRecords=0",
		},
		{
			name:      "partial Website timeout",
			artifacts: map[uint64][]byte{1: valid},
			modes:     map[uint64]error{1: errInvocationTimeout},
			wantFinal: "aggregation-completed websiteCandidates=2 normalCompletedWebsites=1 timedOutWebsites=1 skippedDuplicate=0 malformedRecords=0 invalidRecords=0 oversizedRecords=0",
			wantItems: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			progress := &recordingDirectoryProgress{}
			submitter := &captureDirectorySubmitter{}
			runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
				runner: artifactWritingRunner(t, test.artifacts, test.modes), websiteTimeout: 15 * time.Millisecond,
			}}}
			execution := directoryExecutionFixture(t, workspace, progress, submitter)
			execution.Config.Ffuf.Concurrency = 2
			if err := runtime.Execute(context.Background(), execution); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			want := []string{
				"input_ready websiteCandidates=2",
				"scan_started websiteCandidates=2",
				test.wantFinal,
			}
			if got := progress.Messages(); !reflect.DeepEqual(got, want) {
				t.Fatalf("progress messages = %#v, want %#v", got, want)
			}
			if len(submitter.items) != test.wantItems {
				t.Fatalf("submitted items = %d, want %d", len(submitter.items), test.wantItems)
			}
		})
	}
}

func TestRuntimeExecuteReportsOnlyAggregationFailedForAllRejectedRecords(t *testing.T) {
	oversized := append(bytes.Repeat([]byte{'x'}, maximumFFUFPhysicalRecordBytes+1), '\n')
	tests := []struct {
		name        string
		payload     []byte
		failedEvent string
	}{
		{
			name:        "malformed",
			payload:     []byte("{\n"),
			failedEvent: "aggregation_failed malformedRecords=2 invalidRecords=0 oversizedRecords=0",
		},
		{
			name:        "invalid",
			payload:     []byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}` + "\n"),
			failedEvent: "aggregation_failed malformedRecords=0 invalidRecords=2 oversizedRecords=0",
		},
		{
			name:        "oversized",
			payload:     oversized,
			failedEvent: "aggregation_failed malformedRecords=0 invalidRecords=0 oversizedRecords=2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			progress := &recordingDirectoryProgress{}
			submitter := &captureDirectorySubmitter{}
			runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
				runner:         artifactWritingRunner(t, map[uint64][]byte{0: test.payload, 1: test.payload}, nil),
				websiteTimeout: time.Second,
			}}}
			err := runtime.Execute(context.Background(), directoryExecutionFixture(t, workspace, progress, submitter))
			if !errors.Is(err, ErrAllFFUFRecordsRejected) {
				t.Fatalf("Execute() error = %v, want ErrAllFFUFRecordsRejected", err)
			}
			want := []string{
				"input_ready websiteCandidates=2",
				"scan_started websiteCandidates=2",
				test.failedEvent,
			}
			if got := progress.Messages(); !reflect.DeepEqual(got, want) {
				t.Fatalf("progress messages = %#v, want %#v", got, want)
			}
			if submitter.calls != 0 {
				t.Fatalf("all-rejected path submitted %d batches", submitter.calls)
			}
			assertNoDirectoryDedupStaging(t, workspace)
		})
	}
}

func TestRuntimeExecuteDoesNotReportCompletionForKnownScanFailures(t *testing.T) {
	tests := []struct {
		name    string
		modes   map[uint64]error
		wantErr error
	}{
		{
			name: "all Website timeout",
			modes: map[uint64]error{
				0: errInvocationTimeout,
				1: errInvocationTimeout,
			},
			wantErr: ErrAllWebsitesTimedOut,
		},
		{
			name:    "FFUF non-zero",
			modes:   map[uint64]error{0: errors.New("exit status 7")},
			wantErr: errors.New("exit status 7"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			progress := &recordingDirectoryProgress{}
			submitter := &captureDirectorySubmitter{}
			runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
				runner: artifactWritingRunner(t, map[uint64][]byte{}, test.modes), websiteTimeout: 15 * time.Millisecond,
			}}}
			execution := directoryExecutionFixture(t, workspace, progress, submitter)
			execution.Config.Ffuf.Concurrency = 1
			err := runtime.Execute(context.Background(), execution)
			if test.name == "FFUF non-zero" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr.Error()) {
					t.Fatalf("Execute() error = %v, want %q", err, test.wantErr)
				}
			} else if !errors.Is(err, test.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, test.wantErr)
			}
			want := []string{"input_ready websiteCandidates=2", "scan_started websiteCandidates=2"}
			if got := progress.Messages(); !reflect.DeepEqual(got, want) {
				t.Fatalf("failure progress = %#v, want %#v", got, want)
			}
			if submitter.calls != 0 {
				t.Fatalf("known failure submitted %d batches", submitter.calls)
			}
		})
	}
}

func TestRuntimeExecuteTreatsEveryProgressAcknowledgementFailureAsFatal(t *testing.T) {
	wantErr := errors.New("progress acknowledgement lost")
	tests := []struct {
		name            string
		failAt          int
		wantInvocations int32
	}{
		{name: "input_ready", failAt: 1},
		{name: "scan_started", failAt: 2},
		{name: "aggregation-completed", failAt: 3, wantInvocations: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			progress := &recordingDirectoryProgress{failAt: test.failAt, err: wantErr}
			var invocations atomic.Int32
			runner := ffufRunnerFunc(func(_ context.Context, invocation ffufInvocation) error {
				invocations.Add(1)
				path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
				if err != nil {
					return err
				}
				return os.WriteFile(path, nil, 0o600)
			})
			runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
				runner: runner, websiteTimeout: time.Second,
			}}}
			err := runtime.Execute(context.Background(), directoryExecutionFixture(t, workspace, progress, &captureDirectorySubmitter{}))
			if !errors.Is(err, wantErr) {
				t.Fatalf("Execute() error = %v, want acknowledgement failure", err)
			}
			if got := invocations.Load(); got != test.wantInvocations {
				t.Fatalf("FFUF invocations = %d, want %d", got, test.wantInvocations)
			}
			if len(progress.Messages()) != test.failAt {
				t.Fatalf("persisted progress count = %d, want %d", len(progress.Messages()), test.failAt)
			}
			assertNoDirectoryDedupStaging(t, workspace)
		})
	}
}

func TestRuntimeExecuteRetainsAllRejectedFailureWhenProgressAcknowledgementIsLost(t *testing.T) {
	wantProgressErr := errors.New("aggregation_failed acknowledgement lost")
	progress := &recordingDirectoryProgress{failAt: 3, err: wantProgressErr}
	workspace := t.TempDir()
	payload := []byte("not-json\n")
	runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
		runner: artifactWritingRunner(t, map[uint64][]byte{0: payload, 1: payload}, nil), websiteTimeout: time.Second,
	}}}
	err := runtime.Execute(context.Background(), directoryExecutionFixture(t, workspace, progress, &captureDirectorySubmitter{}))
	if !errors.Is(err, ErrAllFFUFRecordsRejected) || !errors.Is(err, wantProgressErr) {
		t.Fatalf("Execute() error = %v, want both aggregation and acknowledgement failures", err)
	}
	wantLast := "aggregation_failed malformedRecords=2 invalidRecords=0 oversizedRecords=0"
	if messages := progress.Messages(); len(messages) != 3 || messages[2] != wantLast {
		t.Fatalf("persisted aggregation failure = %#v", messages)
	}
}

func TestRuntimeExecuteProducesNoHeartbeatOrPerWebsiteProgressWhileRunning(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	runner := ffufRunnerFunc(func(ctx context.Context, invocation ffufInvocation) error {
		once.Do(func() { close(started) })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		return os.WriteFile(path, nil, 0o600)
	})
	progress := &recordingDirectoryProgress{}
	runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	}}}
	execution := directoryExecutionFixture(t, t.TempDir(), progress, &captureDirectorySubmitter{})
	execution.Config.Ffuf.Concurrency = 1
	done := make(chan error, 1)
	go func() { done <- runtime.Execute(context.Background(), execution) }()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("FFUF invocation did not start")
	}
	time.Sleep(25 * time.Millisecond)
	wantRunning := []string{"input_ready websiteCandidates=2", "scan_started websiteCandidates=2"}
	if got := progress.Messages(); !reflect.DeepEqual(got, wantRunning) {
		t.Fatalf("progress while running = %#v, want %#v", got, wantRunning)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if messages := progress.Messages(); len(messages) != 3 || !strings.HasPrefix(messages[2], "aggregation-completed ") {
		t.Fatalf("final progress = %#v", messages)
	}
}

func TestRuntimeFFUFNonZeroKeepsToolStreamsOutOfProgressAndResults(t *testing.T) {
	const sensitive = "authorization=super-secret"
	rawStdout := append(marshalFFUFRecord(t, ffufRecordFixture{
		URL: "https://example.com/secret-path", Status: 200, ContentLength: 1, ContentType: "", Duration: 1,
	}), '\n')
	var stderr bytes.Buffer
	runner := newContainerFFUFProcessRunner()
	runner.stderr = &stderr
	runner.newProcess = func(_ context.Context, _ string, _ []string, stderrWriter io.Writer) ffufProcess {
		_, _ = io.WriteString(stderrWriter, sensitive+"\n")
		return &fakeFFUFProcess{
			stdout:  io.NopCloser(bytes.NewReader(rawStdout)),
			waitErr: errors.New("exit status 7"),
		}
	}
	progress := &recordingDirectoryProgress{}
	submitter := &captureDirectorySubmitter{}
	workspace := t.TempDir()
	runtime := &Runtime{aggregation: directoryAggregationOptions{scheduler: websiteSchedulerOptions{
		runner: runner, websiteTimeout: time.Second,
	}}}
	execution := directoryExecutionFixture(t, workspace, progress, submitter)
	execution.Config.Ffuf.Concurrency = 1
	err := runtime.Execute(context.Background(), execution)
	if err == nil || strings.Contains(err.Error(), sensitive) {
		t.Fatalf("Execute() error leaked stderr: %v", err)
	}
	if stderr.String() != sensitive+"\n" {
		t.Fatalf("container stderr = %q", stderr.String())
	}
	path, pathErr := rawFFUFArtifactPath(workspace, 0)
	if pathErr != nil {
		t.Fatal(pathErr)
	}
	gotRaw, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(gotRaw, rawStdout) {
		t.Fatalf("raw FFUF artifact = %q, error=%v", gotRaw, readErr)
	}
	for _, message := range progress.Messages() {
		if strings.Contains(message, sensitive) || strings.Contains(message, "secret-path") {
			t.Fatalf("progress leaked FFUF payload: %q", message)
		}
	}
	if submitter.calls != 0 || len(submitter.items) != 0 {
		t.Fatalf("non-zero FFUF submitted results: calls=%d items=%#v", submitter.calls, submitter.items)
	}
}

func directoryExecutionFixture(
	t *testing.T,
	workspace string,
	progress enginecontract.Progress,
	submitter enginecontract.DirectorySubmitter,
) *enginecontract.Execution {
	t.Helper()
	config := defaultFFUFConfig()
	config.Wordlist = writeDirectoryTestFile(t, "wordlist.txt", nil)
	return &enginecontract.Execution{
		Target:    enginecontract.Target{Type: enginecontract.TargetTypeDomain, Value: "example.com"},
		Input:     enginecontract.Input{WebsiteURLs: testInputPath(writeWebsiteURLFacts(t))},
		Config:    enginecontract.Config{Ffuf: config},
		Workspace: workspace,
		Progress:  progress,
		Results:   enginecontract.Results{Directories: submitter},
	}
}

func writeDirectoryTestFile(t *testing.T, name string, payload []byte) string {
	t.Helper()
	path := t.TempDir() + "/" + name
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

type recordingDirectoryProgress struct {
	mu       sync.Mutex
	messages []string
	failAt   int
	err      error
	onReport func(string)
}

func (progress *recordingDirectoryProgress) Report(_ context.Context, message string) error {
	progress.mu.Lock()
	progress.messages = append(progress.messages, message)
	call := len(progress.messages)
	failAt := progress.failAt
	err := progress.err
	onReport := progress.onReport
	progress.mu.Unlock()
	if onReport != nil {
		onReport(message)
	}
	if call == failAt {
		if err == nil {
			return fmt.Errorf("progress acknowledgement %d failed", call)
		}
		return err
	}
	return nil
}

func (progress *recordingDirectoryProgress) Messages() []string {
	progress.mu.Lock()
	defer progress.mu.Unlock()
	return append([]string(nil), progress.messages...)
}
