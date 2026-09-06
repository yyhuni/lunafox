package directoryscanruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestScanDeduplicateAndSubmitDirectoriesFinalizesGlobalWinnersBeforeSubmission(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 1)
	workspace := t.TempDir()
	sharedURL := "https://example.com/shared"
	artifacts := map[uint64][]byte{
		0: append(marshalFFUFRecord(t, ffufRecordFixture{URL: sharedURL, Status: 200, ContentLength: 1, ContentType: "first", Duration: 1}), '\n'),
		1: append(append([]byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}`+"\n"), marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/distinct", Status: 201, ContentLength: 2, ContentType: "distinct", Duration: 2})...), '\n'),
		2: append(marshalFFUFRecord(t, ffufRecordFixture{URL: sharedURL, Status: 299, ContentLength: 3, ContentType: "winner", Duration: 3}), '\n'),
	}
	runner := artifactWritingRunner(t, artifacts, nil)
	submitter := &captureDirectorySubmitter{}

	summary, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{
			scheduler: websiteSchedulerOptions{runner: runner, websiteTimeout: time.Second},
			dedup:     directoryDedupOptions{ChunkByteBudget: 1, MergeFanIn: 2},
		},
	)
	if err != nil {
		t.Fatalf("scanDeduplicateAndSubmitDirectories() error = %v", err)
	}
	if summary.Websites.NormalCompletedWebsites != 3 || summary.Records != (FFUFParseSummary{SourceRecords: 4, ParsedItems: 3, InvalidRecords: 1}) || summary.WinnerItems != 2 || summary.SkippedDuplicate != 1 {
		t.Fatalf("aggregation summary = %#v", summary)
	}
	if submitter.calls != 1 || len(submitter.items) != 2 {
		t.Fatalf("submission calls=%d items=%#v", submitter.calls, submitter.items)
	}
	if submitter.items[0].URL != "https://example.com/distinct" || submitter.items[1].URL != sharedURL || submitter.items[1].Status != 299 || submitter.items[1].ContentType != "winner" {
		t.Fatalf("submitted winners = %#v", submitter.items)
	}
	assertNoDirectoryDedupStaging(t, workspace)
	for ordinal := uint64(0); ordinal < plan.CandidateCount(); ordinal++ {
		path, pathErr := rawFFUFArtifactPath(workspace, ordinal)
		if pathErr != nil {
			t.Fatal(pathErr)
		}
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("raw artifact %d was not retained: %v", ordinal, statErr)
		}
	}
}

func TestScanDeduplicateAndSubmitDirectoriesSkipsEmptySubmission(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	submitter := &captureDirectorySubmitter{}
	summary, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{scheduler: websiteSchedulerOptions{
			runner: artifactWritingRunner(t, map[uint64][]byte{}, nil), websiteTimeout: time.Second,
		}},
	)
	if err != nil {
		t.Fatalf("zero-winner aggregation error = %v", err)
	}
	if summary.WinnerItems != 0 || submitter.calls != 0 {
		t.Fatalf("zero-winner summary=%#v calls=%d", summary, submitter.calls)
	}
	assertNoDirectoryDedupStaging(t, workspace)
}

func TestScanDeduplicateAndSubmitDirectoriesRetainsPartialAcknowledgementOnFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 1)
	workspace := t.TempDir()
	artifacts := make(map[uint64][]byte)
	for ordinal := uint64(0); ordinal < plan.CandidateCount(); ordinal++ {
		artifacts[ordinal] = append(marshalFFUFRecord(t, ffufRecordFixture{
			URL: "https://example.com/" + string(rune('a'+ordinal)), Status: 200, ContentLength: 1, ContentType: "", Duration: 1,
		}), '\n')
	}
	wantErr := errors.New("second batch rejected")
	submitter := &captureDirectorySubmitter{failAfterItems: 1, err: wantErr}
	summary, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{scheduler: websiteSchedulerOptions{
			runner: artifactWritingRunner(t, artifacts, nil), websiteTimeout: time.Second,
		}},
	)
	if !errors.Is(err, wantErr) || summary.WinnerItems != 3 {
		t.Fatalf("partial acknowledgement summary=%#v error=%v", summary, err)
	}
	assertNoDirectoryDedupStaging(t, workspace)
}

func TestScanDeduplicateAndSubmitDirectoriesWaitsForWinnerFileClose(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	artifacts := map[uint64][]byte{
		0: append(marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/a", Status: 200, ContentLength: 1, ContentType: "", Duration: 1}), '\n'),
	}
	var winnerClosed atomic.Bool
	var submittedBeforeClose atomic.Bool
	submitter := &captureDirectorySubmitter{onSubmit: func() {
		if !winnerClosed.Load() {
			submittedBeforeClose.Store(true)
		}
	}}
	create := func(path string) (directoryDedupWriteCloser, error) {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, err
		}
		if filepath.Base(path) == "winners.bin" {
			return &closeObservedDirectoryWriter{directoryDedupWriteCloser: file, closed: &winnerClosed}, nil
		}
		return file, nil
	}
	_, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{
			scheduler: websiteSchedulerOptions{runner: artifactWritingRunner(t, artifacts, nil), websiteTimeout: time.Second},
			dedup:     directoryDedupOptions{FileOps: directoryDedupFileOps{create: create}},
		},
	)
	if err != nil || !winnerClosed.Load() || submittedBeforeClose.Load() {
		t.Fatalf("winner close=%t submittedBeforeClose=%t error=%v", winnerClosed.Load(), submittedBeforeClose.Load(), err)
	}
}

func TestScanDeduplicateAndSubmitDirectoriesPreservesLatestDuplicateForResultAdapter(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	sharedURL := "https://example.com/shared"
	valid := append(marshalFFUFRecord(t, ffufRecordFixture{URL: sharedURL, Status: 200, ContentLength: 1, ContentType: "valid", Duration: 1}), '\n')
	invalidLater := []byte(`{"url":"https://example.com/shared","status":299,"length":-1,"content-type":"invalid","duration":2}` + "\n")
	submitter := &captureDirectorySubmitter{}
	summary, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{scheduler: websiteSchedulerOptions{
			runner: artifactWritingRunner(t, map[uint64][]byte{0: valid, 1: invalidLater}, nil), websiteTimeout: time.Second,
		}},
	)
	if err != nil || summary.Records.InvalidRecords != 0 || summary.WinnerItems != 1 || len(submitter.items) != 1 || submitter.items[0].Status != 299 || submitter.items[0].ContentLength != -1 {
		t.Fatalf("latest duplicate was changed or filtered before the result adapter: summary=%#v items=%#v error=%v", summary, submitter.items, err)
	}
}

func TestScanDeduplicateAndSubmitDirectoriesHandlesAllTimeoutAfterSubmission(t *testing.T) {
	for _, test := range []struct {
		name      string
		withItems bool
		wantCalls int
		wantBad   uint64
	}{
		{name: "non-empty", withItems: true, wantCalls: 1, wantBad: 2},
		{name: "empty"},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, _ := schedulerCandidatePlan(t, 0)
			workspace := t.TempDir()
			artifacts := make(map[uint64][]byte)
			modes := make(map[uint64]error)
			for ordinal := uint64(0); ordinal < plan.CandidateCount(); ordinal++ {
				if test.withItems {
					artifacts[ordinal] = append(marshalFFUFRecord(t, ffufRecordFixture{
						URL: "https://example.com/" + string(rune('a'+ordinal)), Status: 200, ContentLength: 1, ContentType: "", Duration: 1,
					}), []byte("\n{\"url\":")...)
				}
				modes[ordinal] = errInvocationTimeout
			}
			submitter := &captureDirectorySubmitter{}
			summary, err := scanDeduplicateAndSubmitDirectories(
				context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
				directoryAggregationOptions{scheduler: websiteSchedulerOptions{
					runner: artifactWritingRunner(t, artifacts, modes), websiteTimeout: 15 * time.Millisecond,
				}},
			)
			if !errors.Is(err, ErrAllWebsitesTimedOut) || summary.Records.MalformedRecords != test.wantBad || submitter.calls != test.wantCalls {
				t.Fatalf("all-timeout summary=%#v calls=%d error=%v", summary, submitter.calls, err)
			}
			assertNoDirectoryDedupStaging(t, workspace)
		})
	}
}

func TestScanDeduplicateAndSubmitDirectoriesDoesNotSubmitKnownFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload []byte
		mode    error
		wantErr error
	}{
		{name: "all rejected", payload: []byte(`{"url":null,"status":200,"length":1,"content-type":"","duration":1}` + "\n"), wantErr: ErrAllFFUFRecordsRejected},
		{name: "FFUF non-zero", mode: errors.New("exit code 7")},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, _ := schedulerCandidatePlan(t, 0)
			workspace := t.TempDir()
			artifacts := map[uint64][]byte{0: test.payload, 1: test.payload}
			modes := map[uint64]error{}
			if test.mode != nil {
				modes[0] = test.mode
			}
			submitter := &captureDirectorySubmitter{}
			_, err := scanDeduplicateAndSubmitDirectories(
				context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
				directoryAggregationOptions{scheduler: websiteSchedulerOptions{
					runner: artifactWritingRunner(t, artifacts, modes), websiteTimeout: time.Second,
				}},
			)
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("aggregation error = %v, want %v", err, test.wantErr)
			}
			if test.mode != nil && !errors.Is(err, test.mode) {
				t.Fatalf("aggregation error = %v, want %v", err, test.mode)
			}
			if submitter.calls != 0 {
				t.Fatalf("known failure submitted %d times", submitter.calls)
			}
			assertNoDirectoryDedupStaging(t, workspace)
		})
	}
}

func TestScanDeduplicateAndSubmitDirectoriesTurnsCleanupOnlyFailureIntoTaskFailure(t *testing.T) {
	plan, _ := schedulerCandidatePlan(t, 0)
	workspace := t.TempDir()
	wantErr := errors.New("cleanup failed")
	var privateDir string
	item := append(marshalFFUFRecord(t, ffufRecordFixture{URL: "https://example.com/a", Status: 200, ContentLength: 1, ContentType: "", Duration: 1}), '\n')
	submitter := &captureDirectorySubmitter{}
	summary, err := scanDeduplicateAndSubmitDirectories(
		context.Background(), plan, defaultFFUFConfig(), workspace, submitter,
		directoryAggregationOptions{
			scheduler: websiteSchedulerOptions{runner: artifactWritingRunner(t, map[uint64][]byte{0: item}, nil), websiteTimeout: time.Second},
			dedup: directoryDedupOptions{FileOps: directoryDedupFileOps{removeAll: func(path string) error {
				privateDir = path
				return wantErr
			}}},
		},
	)
	if !errors.Is(err, wantErr) || summary.Websites.NormalCompletedWebsites != plan.CandidateCount() || len(submitter.items) != 1 {
		t.Fatalf("cleanup-only summary=%#v error=%v", summary, err)
	}
	if privateDir == "" || filepath.Dir(privateDir) != workspace {
		t.Fatalf("cleanup path = %q", privateDir)
	}
	t.Cleanup(func() { _ = os.RemoveAll(privateDir) })
}

func TestIsOnlyDirectoryContextTerminationRejectsJoinedOperationalFailure(t *testing.T) {
	operationalErr := errors.New("disk failed")
	for _, test := range []struct {
		err  error
		want bool
	}{
		{err: context.Canceled, want: true},
		{err: errors.Join(context.Canceled, context.DeadlineExceeded), want: true},
		{err: errors.Join(context.Canceled, operationalErr)},
		{err: operationalErr},
	} {
		if got := isOnlyDirectoryContextTermination(test.err); got != test.want {
			t.Fatalf("isOnlyDirectoryContextTermination(%v) = %t, want %t", test.err, got, test.want)
		}
	}
}

var errInvocationTimeout = errors.New("wait for child deadline")

func artifactWritingRunner(t *testing.T, artifacts map[uint64][]byte, modes map[uint64]error) ffufRunnerFunc {
	t.Helper()
	return func(ctx context.Context, invocation ffufInvocation) error {
		path, err := rawFFUFArtifactPath(invocation.Workspace, invocation.Candidate.Ordinal)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, artifacts[invocation.Candidate.Ordinal], 0o600); err != nil {
			return err
		}
		mode := modes[invocation.Candidate.Ordinal]
		if errors.Is(mode, errInvocationTimeout) {
			<-ctx.Done()
			return ctx.Err()
		}
		return mode
	}
}

type captureDirectorySubmitter struct {
	mu             sync.Mutex
	items          []enginecontract.Directory
	calls          int
	failAfterItems int
	err            error
	onSubmit       func()
}

func (submitter *captureDirectorySubmitter) Submit(ctx context.Context, items <-chan enginecontract.Directory) error {
	if submitter.onSubmit != nil {
		submitter.onSubmit()
	}
	submitter.mu.Lock()
	submitter.calls++
	submitter.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-items:
			if !ok {
				return submitter.err
			}
			submitter.mu.Lock()
			submitter.items = append(submitter.items, item)
			submitter.mu.Unlock()
			if submitter.failAfterItems > 0 && len(submitter.items) == submitter.failAfterItems {
				return submitter.err
			}
		}
	}
}

type closeObservedDirectoryWriter struct {
	directoryDedupWriteCloser
	closed *atomic.Bool
}

func (writer *closeObservedDirectoryWriter) Close() error {
	err := writer.directoryDedupWriteCloser.Close()
	writer.closed.Store(true)
	return err
}

func assertNoDirectoryDedupStaging(t *testing.T, workspace string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(workspace, ".directory-result-dedup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("Directory dedup staging remains: %s", strings.Join(paths, ", "))
	}
}
